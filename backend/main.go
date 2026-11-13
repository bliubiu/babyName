package main

// 桌面端入口（Wails v2）
//
// 定位：替代 cmd/server 的 HTTP 部署形态，成为唯一的桌面交付物。
// 装配逻辑全部收敛在 internal/bootstrap（store + 物料 + 服务 + 绑定层），
// 本文件只做三件事：解析目录 → 装配 → 启动 Wails 窗口。
//
// 目录语义（dev 与打包一致）：
//   - 物料：DATA_DIR 环境变量 > 仓库根 data/（dev）> cwd/data（便携打包）
//   - 库：NAMER_DB_PATH 环境变量 > 物料目录上一级的 namer.db（dev 复用仓库根库）
//
// 打包后的用户数据目录迁移（os.UserConfigDir）在后续切片处理。

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"name/internal/bootstrap"
	"name/internal/infrastructure/logger"
)

// assets 前端静态资源。wails build 会把 ../frontend/dist 自动同步到
// backend/frontend/dist（见 wails.json 的 frontend:dir），构建后清理还原。
//
//go:embed all:frontend/dist
var assets embed.FS

var (
	help      = flag.Bool("help", false, "显示帮助")
	dataDirOp = flag.String("data", "", "物料数据目录 (默认: DATA_DIR 环境变量 > 仓库根 data/ > cwd/data)")
	dbPathOp  = flag.String("db", "", "SQLite 库路径 (默认: NAMER_DB_PATH 环境变量 > 物料目录上一级 namer.db)")
	logPathOp = flag.String("log", "", "日志目录 (默认: 物料目录上一级 logs)")
)

func main() {
	flag.Parse()
	if *help {
		printHelp()
		return
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 日志先于装配初始化，装配期日志才能落盘
	if err := initLogger(); err != nil {
		return err
	}
	defer logger.Sync()

	// 目录解析：物料 + 库 + 日志，均支持环境变量与命令行覆盖
	dataDir, err := bootstrap.ResolveDataDir(*dataDirOp)
	if err != nil {
		return err
	}
	dbPath := *dbPathOp
	if strings.TrimSpace(dbPath) == "" {
		dbPath = os.Getenv("NAMER_DB_PATH")
	}
	if strings.TrimSpace(dbPath) == "" {
		dbPath = filepath.Join(filepath.Dir(dataDir), "namer.db")
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

	// 启动 Wails 窗口；Bind 的 App 即前端可直接调用的全部方法
	return wails.Run(&options.App{
		Title:  "起名 NameMaster",
		Width:  1280,
		Height: 860,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: res.App.Startup,
		Bind: []interface{}{
			res.App,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})
}

// initLogger 初始化日志（桌面端默认 INFO + 落盘，目录可由 -log 覆盖）
func initLogger() error {
	logPath := strings.TrimSpace(*logPathOp)
	if logPath == "" {
		logPath = os.Getenv("NAMER_LOG_PATH")
	}
	if logPath == "" {
		// 默认与库同层：物料目录上一级/logs（dev 即仓库根/logs，与 Web 端一致）
		dataDir, _ := bootstrap.ResolveDataDir("")
		logPath = filepath.Join(filepath.Dir(dataDir), "logs")
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
	fmt.Println("起名 NameMaster 桌面端 (Wails)")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  namer-desktop.exe [options]")
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
}
