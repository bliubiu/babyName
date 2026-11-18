package main

// 桌面端入口（Wails v3）
//
// 定位：替代 cmd/server 的 HTTP 部署形态，成为唯一的桌面交付物。
// 装配逻辑全部收敛在 internal/bootstrap（store + 物料 + 服务 + 绑定层），
// 本文件只做三件事：解析目录 → 装配 → 启动 Wails 应用。
//
// 与 v2 的差异（docs/48）：
//   - wails.Run(&options.App{...}) → application.New + Window.NewWithOptions + app.Run()
//   - Bind []interface{} → Services []application.Service（生命周期改由
//     desktop.App.ServiceStartup 承载）
//   - assetserver.Options{Assets} → application.BundledAssetFileServer（同时提供
//     /wails/runtime.js；自动生成 index.html 所在子目录前缀）
//   - Server 选项仅在 -tags server 构建下生效，供 Playwright E2E 直连真实 Go 服务
//
// 目录语义（按数据目录来源分叉，见 bootstrap.DataDirSource）：
//   - 物料：-data > DATA_DIR > 仓库根 data/（dev）> exe 同级 data/（打包）> cwd/data
//   - dev：库 = 仓库根 namer.db、日志 = 仓库根 logs/（历史布局不变）
//   - 打包：库与日志收敛到 exe 同级 data/ 内（namer.db、data/logs），
//     安装目录整体自包含、可整体拷贝迁移

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"name/internal/bootstrap"
	"name/internal/infrastructure/logger"
)

// assets 前端静态资源。构建时把 ../frontend/dist 同步到 backend/frontend/dist
// 供 embed（见 Taskfile.yml 的 build:frontend 任务）。
//
//go:embed all:frontend/dist
var assets embed.FS

var (
	help      = flag.Bool("help", false, "显示帮助")
	dataDirOp = flag.String("data", "", "物料数据目录 (默认: DATA_DIR 环境变量 > 仓库根 data/ > exe 同级 data/ > cwd/data)")
	dbPathOp  = flag.String("db", "", "SQLite 库路径 (默认: NAMER_DB_PATH 环境变量 > 按目录来源推导；指向新路径=自动建库播种，即干净库；指向既有文件=私有库)")
	logPathOp = flag.String("log", "", "日志目录 (默认: NAMER_LOG_PATH 环境变量 > 按目录来源推导)")
)

func main() {
	flag.Parse()
	if *help {
		printHelp()
		return
	}

	// 统一崩溃兜底：run() 之外任何未被转成 error 的 panic（WebView2 初始化、
	// 物料解析、GUI 回调等）在 -H windowsgui 下同样会静默退出。捕获后走
	// 与「返回 error」完全相同的出口，保证任何退出前都有说明（docs/48 §9.8）。
	defer func() {
		if r := recover(); r != nil {
			fatalStartup(nil, r)
		}
	}()

	if err := run(); err != nil {
		fatalStartup(err, nil)
	}
}

// fatalStartup 是所有启动失败 / 崩溃的唯一出口。
// 三路输出各司其职，互不混用：
//   - startup-error.log：完整技术诊断（原始错误、路径、环境、panic 栈），可反馈
//   - MessageBoxW：面向用户的友好话术 + 可操作步骤 + 错误代码，不暴露内部术语
//   - stderr：技术诊断，供 dev / CI 控制台排查（GUI 子系统下不可见）
//
// 注意 os.Exit 不执行 defer，故这里不会与 main 的 recover 互相触发。
func fatalStartup(err error, panicValue any) {
	f := classifyStartupFailure(err, panicValue)
	diagnostic := formatDiagnostic(f)

	writeStartupError(diagnostic)
	showStartupError(formatFriendlyMessage(f))
	fmt.Fprint(os.Stderr, diagnostic)
	os.Exit(1)
}

// writeStartupError 把诊断写到 exe 同级 startup-error.log。
// 早期失败发生在日志系统初始化之前，这是唯一可靠的排查留痕。
func writeStartupError(msg string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	_ = writeStartupErrorLog(filepath.Dir(exe), msg)
}

// writeStartupErrorLog 抽成纯函数便于单测：以追加方式写入带时间戳的记录。
// 追加而非覆盖，避免连续多次启动失败把上一次的诊断冲掉。
func writeStartupErrorLog(dir, msg string) error {
	line := fmt.Sprintf("%s %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	f, err := os.OpenFile(filepath.Join(dir, "startup-error.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

func run() error {
	// 目录解析一次并记住来源：物料 + 库 + 日志均支持环境变量与命令行覆盖
	dataDir, source, err := bootstrap.ResolveDataDirDetailed(*dataDirOp)
	if err != nil {
		return err
	}

	// 日志先于装配初始化，装配期日志才能落盘（默认位置依赖目录来源）
	if err := initLogger(dataDir, source); err != nil {
		return err
	}
	defer logger.Sync()

	// 库路径：-db > NAMER_DB_PATH > 按来源推导（dev=仓库根，打包=data 文件夹内）
	dbPath := *dbPathOp
	if strings.TrimSpace(dbPath) == "" {
		dbPath = os.Getenv("NAMER_DB_PATH")
	}
	if strings.TrimSpace(dbPath) == "" {
		dbPath = bootstrap.DefaultDBPath(dataDir, source)
	}

	// 装配：存储 + 物料 + 服务 + 桌面绑定层
	res, err := bootstrap.Run(bootstrap.Options{
		DBPath:       dbPath,
		DataDir:      dataDir,
		SeedFromJSON: true, // 首跑库为空时自动从 JSON 物料播种（幂等）
	})
	if err != nil {
		return fmt.Errorf("应用装配失败: %w", err)
	}
	defer func() { _ = res.Close() }()

	logger.Info("桌面端启动",
		logger.String("物料目录", dataDir),
		logger.String("数据库", dbPath),
	)

	app := application.New(application.Options{
		Name:        "起名 NameMaster",
		Description: "基于八字五行、易经卦象与音形义的智能起名工具",
		Services: []application.Service{
			// 生成器静态分析此调用，绑定 res.App 的全部导出方法（Startup 已改名
			// ServiceStartup，不会被暴露成前端可调用方法，docs/48 §4.1）
			application.NewService(res.App),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		// 仅在 -tags server 构建下生效：Playwright 直连 127.0.0.1:8080，
		// 不创建原生窗口（docs/48 阶段 3）。生产桌面构建自动忽略此字段。
		Server: application.ServerOptions{
			Host: "127.0.0.1",
			Port: 8080,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "起名 NameMaster",
		Width:  1280,
		Height: 860,
		URL:    "/",
	})

	return app.Run()
}

// initLogger 初始化日志（桌面端默认 INFO + 落盘，目录可由 -log/NAMER_LOG_PATH 覆盖）
//
// 默认位置按数据目录来源推导：dev=仓库根/logs（与 Web 端一致），
// 打包=exe 同级 data/logs（与库同处 data 文件夹内）。
func initLogger(dataDir string, source bootstrap.DataDirSource) error {
	logPath := strings.TrimSpace(*logPathOp)
	if logPath == "" {
		logPath = os.Getenv("NAMER_LOG_PATH")
	}
	if logPath == "" {
		logPath = bootstrap.DefaultLogDir(dataDir, source)
	}
	if err := logger.Init(&logger.Config{
		Level:      "info",
		Format:     "console",
		OutputPath: logPath,
	}); err != nil {
		// 日志失败不阻断启动：降级为标准输出
		fmt.Fprintf(os.Stderr, "警告: 日志系统初始化失败(%v)，使用标准输出\n", err)
	}
	return nil
}

func printHelp() {
	fmt.Println("起名 NameMaster 桌面端 (Wails v3)")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  namer-desktop.exe [options]")
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("启动说明:")
	fmt.Println("  - 首次启动需初始化数据，约 1~2 分钟内没有窗口属正常现象，请耐心等待")
	fmt.Println("  - 启动失败会弹出说明，并在 exe 同目录生成 startup-error.log")
	fmt.Println("  - 数据目录默认按 exe 位置自动探测，也可用 -data 或 DATA_DIR 环境变量指定")
	fmt.Println("")
	fmt.Println("私有/干净数据库 (-db):")
	fmt.Println("  namer-desktop.exe -db D:/my/private.db    指向既有文件 = 使用该私有库（收藏/历史独立）")
	fmt.Println("  namer-desktop.exe -db D:/new/clean.db     指向不存在路径 = 自动建库并播种（干净库）")
	fmt.Println("  常驻私有库可为 exe 建快捷方式，在快捷方式「目标」末尾追加 -db 参数；")
	fmt.Println("  注意 -db 优先于默认推导，首次建库约 1~2 分钟（播种）后正常使用")
}
