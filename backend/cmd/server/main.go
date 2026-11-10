package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"name/internal/application/handlers"
	"name/internal/application/services"
	"name/internal/domain/classics"
	"name/internal/domain/fate"
	"name/internal/domain/log"
	"name/internal/domain/name"
	"name/internal/infrastructure/config"
	"name/internal/infrastructure/data"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/database/memory"
	"name/internal/infrastructure/database/sqlite"
	"name/internal/infrastructure/gateaudit"
	"name/internal/infrastructure/logger"
	"name/internal/infrastructure/middleware"
	"name/internal/infrastructure/server"
	"name/internal/paths"
)

var (
	port         = flag.Int("port", 0, "Server port (覆盖配置文件)")
	host         = flag.String("host", "", "Server host (覆盖配置文件)")
	mode         = flag.String("mode", "", "Run mode: backend or all (覆盖配置文件)")
	static       = flag.String("static", "", "Static files directory (覆盖配置文件)")
	help         = flag.Bool("help", false, "Show help")
	logPath      = flag.String("log", "", "Log dir (覆盖配置文件；空 = 项目根/logs，文件名 namer-YYYYMMDD.log 按天)")
	dbPath       = flag.String("db", "", "SQLite database file path (覆盖配置文件)")
	rateCapacity = flag.Float64("rate-capacity", 0, "Rate limiter capacity (覆盖配置文件)")
	rateRate     = flag.Float64("rate-rate", 0, "Rate limiter rate (覆盖配置文件)")
	configFile   = flag.String("config", "", "配置文件路径 (默认: application.yml)")
)

// mainSourceFile 定位本包源文件，作为 cwdDataDir 向上探测仓库根 data/ 的起点
//
// 提取为包级变量是为了可测：模拟容器内「无仓库结构、探测无果」的场景时，
// 测试可替换为临时目录中的路径，不必依赖真实仓库布局。
var mainSourceFile = func() string {
	_, file, _, _ := runtime.Caller(0)
	return file
}

// cwdDataDir 解析传统文化数据的目录（docs/29 B12）
//
// 优先级：DATA_DIR 环境变量 > 向上探测到的仓库根 data/ > 工作目录下的 data/。
//
// ★ docs/29 B12：compose 与 docker-entrypoint.sh 都按 DATA_DIR 环境变量播种
// 字库。两者不一致时，字库被播种到 A 目录、应用却去 B 目录读 —— 服务正常启动、
// healthz 通过，生成接口全部返回空结果。故此处以 DATA_DIR 为准。
//
// ★ 仓库根布局：物料源文件位于仓库根 data/ 而非 backend/data。无 DATA_DIR 时
// 若只回落 cwd/data，从 backend/ 启动即指向已不存在的 backend/data，并连带让
// fail-fast 的导入指引指向该空目录——用户照抄后导入失败。故先向上探测仓库根
// data/（与测试侧共用 paths.FindDataDir）；探测无果（容器内 /app 无仓库结构）
// 再回落 cwd/data，与入口脚本的 `DATA_DIR=${DATA_DIR:-/app/data}` 同源。
//
// 分离成独立函数是为了可被测试直接覆盖。
func cwdDataDir(cwd string) string {
	if dir := strings.TrimSpace(os.Getenv("DATA_DIR")); dir != "" {
		return dir
	}
	if dir := paths.FindDataDir(mainSourceFile()); dir != "" {
		return dir
	}
	return filepath.Join(cwd, "data")
}

func main() {
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	// 加载配置文件
	cfg, err := config.Load(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 命令行参数覆盖配置文件
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *mode != "" {
		cfg.Mode = *mode
	}
	if *static != "" {
		cfg.Static = *static
	}
	if *logPath != "" {
		cfg.Log.OutputPath = *logPath
	}
	if *dbPath != "" {
		cfg.Database.Path = *dbPath
	}
	if *rateCapacity != 0 {
		cfg.RateLimit.Capacity = *rateCapacity
	}
	if *rateRate != 0 {
		cfg.RateLimit.Rate = *rateRate
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// 解析静态文件路径
	cfg.Static = resolveStaticPath(cwd, cfg.Static)

	// 初始化日志
	if err := logger.Init(&logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		OutputPath: cfg.Log.OutputPath,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "警告: 日志系统初始化失败(%v)，使用标准输出\n", err)
	}
	defer logger.Sync()
	// 领域层日志端口注入（docs/43 P1-9）：domain 包统一经 domain/log 门面
	// 写日志，实现由基础设施适配；未注入时领域日志为 no-op。
	log.SetLogger(logger.AsDomainLogger())

	// 注册门禁命中审计落地器（docs/33 M3）：命中事件写成结构化日志
	gateaudit.Install()

	// 初始化缓存（SQLite + 内存缓存，不使用 Redis）

	// 加载传统文化数据（从 JSON 文件）
	//
	// ★ docs/29 B12：原先固定 `filepath.Join(cwd, "data")`，但 compose 与
	// docker-entrypoint.sh 都按 DATA_DIR 环境变量播种字库。两者不一致时，
	// 字库被播种到 A 目录、应用却去 B 目录读 —— 服务正常启动、healthz 通过，
	// 生成接口全部返回空结果。故此处以 DATA_DIR 为准（缺省才回落到 cwd/data），
	// 让入口脚本与进程读同一个目录。
	dataDir := cwdDataDir(cwd)

	// 1. 初始化数据存储 — 优先 SQLite 持久化，回退到内存存储
	//
	// 顺序说明：物料统一从 SQLite 读取（docs/31 ADR-008），因此必须先开库，
	// 再把库内物料灌进内存索引。initStore 不会写物料表（只建表与用户数据）。
	store := initStore(cfg.Database.Path, dataDir)

	// 2. 加载传统文化物料（以 SQLite 物料库为唯一真源，docs/31 ADR-008）
	//
	// ★ docs/29 B12：原先固定 `filepath.Join(cwd, "data")`，但 compose 与
	// docker-entrypoint.sh 都按 DATA_DIR 环境变量播种字库。两者不一致时，
	// 字库被播种到 A 目录、应用却去 B 目录读 —— 服务正常启动、healthz 通过，
	// 生成接口全部返回空结果。故此处以 DATA_DIR 为准（缺省才回落到 cwd/data）。
	//
	// ★ ADR-008 fail-fast：物料缺失一律拒绝启动并给出导入指引。此前此处只
	// Warn 后继续，后果是字库为空时服务照常监听、healthz 返回 200，但每个
	// 起名请求都 400 —— 监控看到「服务健康」而业务 100% 不可用，比启动失败
	// 难排查得多。
	if err := loadCulturalData(dataDir, cfg.Database.Path, store); err != nil {
		fmt.Fprintf(os.Stderr, "\n错误: 物料加载失败，服务拒绝启动\n  原因: %v\n\n", err)
		logger.Fatal("加载传统文化物料失败，服务拒绝启动", logger.ErrField(err))
	}
	logger.Info("传统文化物料已装载", logger.String("数据目录", dataDir))

	// 诗词索引预热（docs/46 P2 性能）：把首个 /generate 请求的索引构建尖峰前移到
	// 启动期。必须在 loadCulturalData 之后（物料已装载），否则索引会以空语料固化为空。
	classics.BuildPoetryIndex()

	// 经典字集提取完整性校验（docs/44 遗留④）：classics 物料在库但部分来源
	// 字集行数为 0 时，出典加分与 source_classic 筛选会静默降级（shijing/chuci
	// 曾长期处于此状态而无人发现——旧版 classics 指纹只取 shijing.json，其余
	// 源文件变更不触发重灌，直到 E2E 巡检发现）。classics 为可选物料
	// （Required=false），不阻断启动，但以 ERROR 级日志暴露并给出修复指引。
	warnClassicsExtractGaps(store)

	// 3. 检查静态文件目录（仅 all 模式需要：api 模式下静态页由前端容器/nginx
	// 提供，容器内对不存在的静态路径强制 mkdir 只会 Fatal——docs/30 §六 C10）
	if cfg.Mode == "all" {
		ensureStaticDir(cfg.Static)
	}

	logger.Info("服务启动中",
		logger.String("host", cfg.Server.Host),
		logger.Int("port", cfg.Server.Port),
	)

	// 创建服务层
	svc := newServices(store, dataDir)

	// 创建处理器
	h := newHandlers(svc, dataDir)

	// 路由配置
	router, rateLimiter := setupRouter(h, cfg.Mode, cfg.Static, store, cfg.RateLimit.Capacity, cfg.RateLimit.Rate)

	// 启动服务器（优雅关闭）
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	startServer(router, addr, cwd, store, rateLimiter, cfg.Server.ReadHeaderTimeout)
}

// --- 配置解析 ---

func resolveStaticPath(cwd, static string) string {
	if envStatic := os.Getenv("FRONTEND_STATIC_PATH"); envStatic != "" {
		return envStatic
	}
	if static != "" {
		return static
	}
	// 自动检测
	candidates := []string{
		filepath.Join(cwd, "..", "frontend", ".next", "standalone"),
		filepath.Join(cwd, "..", "frontend", "out"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(cwd, "..", "frontend", "out")
}

func ensureStaticDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		logger.Warn("未找到前端静态目录", logger.String("path", path))
		if err := os.MkdirAll(path, 0755); err != nil {
			logger.Fatal("创建静态文件目录失败", logger.ErrField(err))
		}
	}
	indexPath := filepath.Join(path, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		logger.Warn("静态目录中未找到 index.html", logger.String("path", indexPath))
	}
	logger.Info("使用静态文件目录", logger.String("path", path))
}

func printHelp() {
	fmt.Println("起名 - NameMaster")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  namer.exe [options]")
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

// --- 数据存储 ---

func initStore(dbPath, dataDir string) database.Store {
	if dbStore, err := sqlite.NewStore(dbPath, dataDir); err == nil {
		logger.Info("使用 SQLite 存储", logger.String("db", dbPath))
		return dbStore
	} else {
		logger.Warn("打开 SQLite 失败，回落内存存储", logger.ErrField(err))
		return memory.NewStore()
	}
}

// --- 服务装配 ---

type appServices struct {
	name     *services.NameService
	tasks    *services.TaskService
	history  *services.HistoryService
	favorite *services.FavoriteService
	report   *services.ReportService
}

func newServices(store database.Store, dataDir string) *appServices {
	baziAdapter := &services.BaziAdapter{}
	hexagramAdapter := &services.HexagramAdapter{}
	ziweiAdapter := &services.ZiweiAdapter{}
	zodiacAdapter := &services.ZodiacAdapter{}

	// 候选名库管理器（精选名 + 自学习 + 标准字表，供 API 层与收藏自学习共享）
	ndb, ndbErr := name.NewNameDB(dataDir, name.WithCuratedPersister(services.NewCuratedPersisterAdapter(store)))
	if ndbErr != nil {
		logger.Warn("构造候选名库失败，收藏自学习与候选名功能降级", logger.ErrField(ndbErr))
	}

	// 装配 fate 引擎（新一代引擎：完整 Rater 评分链、避讳长辈、负面反馈、更大候选池）
	// 前置条件：loadCulturalData 已通过 data.Init 将标准字表同步到 hanzi.HanziData
	services.SyncNamingIndexFromHanzi()
	if curatedNames := loadCuratedNamesForFate(store); len(curatedNames) > 0 {
		fate.SetCuratedNames(curatedNames)
		logger.Info("已注入策展好名到 fate 引擎", logger.Int("count", len(curatedNames)))
	}
	// SQLite 下沉（P3 方案6 装配）：构造 HanziDataProvider 引用，
	// 若底层 store 支持 SQL 过滤则注入到 provider，fate 引擎沿用同一 provider。
	// 失败/不支持时降级到 Go 端全表过滤（向后兼容）。
	//
	// ★ 库空甄别（docs/29 B8 孪生陷阱，空库回落实测修复）：hanzi_data 为
	// 0 行（未执行 materials import，或物料被清空）时 SQL 过滤「成功且
	// 零行」——与「有数据但无匹配」不可区分。若照样注入，回落场景
	// （dataloader 已把 JSON 字库装进内存）候选池恒空，全部生成请求
	// 失败且看不出是库没导入。故库空时不注入，让 provider 走 Go 端
	// 全表过滤，与 dataloader 的「逐物料回落」策略同口径。
	hanziProvider := &services.HanziDataProvider{}
	if store != nil && services.HasSQLiteCharStore(store) {
		charStore := services.AsSQLiteCharStore(store)
		switch rows, err := charStore.HanziRowCount(); {
		case err != nil:
			logger.Warn("查询 SQLite 字库行数失败，SQLite 汉字过滤未启用（走 Go 端过滤）", logger.ErrField(err))
		case rows == 0:
			logger.Warn("SQLite 字库为空（未执行 materials import？），SQLite 汉字过滤未启用，回退 Go 端全表过滤")
		default:
			hanziProvider.SetSQLFilter(services.NewSQLiteHanziFilter(charStore))
			logger.Info("SQLite 汉字过滤已启用", logger.String("store", "sqlite"), logger.Int("字库行数", rows))
		}
	}
	fateEngine := fate.NewEngine(hanziProvider, services.NewBaziAnalyzerAdapter(), fate.DefaultRaters())
	fateSvc := services.NewFateNameService(fateEngine,
		services.WithFateBaziAnalyzer(baziAdapter),
		services.WithFateHexagramFinder(hexagramAdapter),
		services.WithFateZiweiAnalyzer(ziweiAdapter),
	)

	nameSvc := services.NewNameService(
		services.WithBaziAnalyzer(baziAdapter),
		services.WithHexagramFinder(hexagramAdapter),
		services.WithZiweiAnalyzer(ziweiAdapter),
		services.WithNameDB(ndb),
		services.WithZodiacFinder(zodiacAdapter),
		services.WithFateService(fateSvc), // fate 引擎优先（新引擎更完善）
	)

	// 异步生成任务（长任务体验：提交 task_id → 轮询进度/结果）
	taskSvc := services.NewTaskService(nameSvc)

	// 创建收藏服务（需要 NameDB 实现自学习）
	favSvc := services.NewFavoriteService(store)
	if ndb := nameSvc.GetNameDB(); ndb != nil {
		favSvc.SetNameDB(ndb)
	}
	// 门禁复审回流（docs/33 M6）：收藏是隐式正反馈，把名字登记为复审候选
	if r, ok := store.(services.GateReviewRecorder); ok {
		favSvc.SetGateReviewRecorder(r)
	}
	// 门禁指标闭环（docs/33 M4/M6）：注入复审统计作为误报率/漏报率/复审时长的真实数据源
	if p, ok := store.(fate.GateMetricProvider); ok {
		fate.SetGateMetricProvider(p)
	}

	return &appServices{
		name:     nameSvc,
		tasks:    taskSvc,
		report:   services.NewReportService(),
		history:  services.NewHistoryService(store),
		favorite: favSvc,
	}
}

// --- 处理器装配 ---

type appHandlers struct {
	name      *handlers.NameHandler
	nameAsync *handlers.NameAsyncHandler
	history   *handlers.HistoryHandler
	favorite  *handlers.FavoriteHandler
	report    *handlers.ReportHandler
	stat      *handlers.NameStatHandler
	huangli   *handlers.HuangliHandler
	character *handlers.CharacterHandler
}

func newHandlers(svc *appServices, dataDir string) *appHandlers {
	return &appHandlers{
		name:      handlers.NewNameHandler(svc.name),
		nameAsync: handlers.NewNameAsyncHandler(handlers.NewNameHandler(svc.name), svc.tasks, svc.name),
		history:   handlers.NewHistoryHandler(svc.history),
		favorite:  handlers.NewFavoriteHandler(svc.favorite),
		report:    handlers.NewReportHandler(svc.report),
		stat:      handlers.NewNameStatHandler(),
		huangli:   handlers.NewHuangliHandler(),
		character: handlers.NewCharacterHandler(svc.name.GetNameDB()),
	}
}

// --- 路由配置 ---

func setupRouter(h *appHandlers, runMode, staticDir string, store database.Store, rateCapacity, rateRate float64) (*gin.Engine, *middleware.IPRateLimiter) {
	router := gin.New()

	// 可信代理（docs/29 A10）：gin 默认信任所有代理并从 X-Forwarded-For
	// 最左侧取 ClientIP，攻击者可伪造该头让每个请求落入新令牌桶，完全绕过
	// IP 限流并污染访问日志。这里仅信任回环与内网段（nginx/容器网络所在），
	// gin 会从右向左取第一个不可信地址——nginx 追加的真实客户端 IP 位于链首
	// 伪造值之后，伪造值永远不会被选中；直连（无代理）时取 TCP 对端地址。
	if err := router.SetTrustedProxies([]string{
		"127.0.0.0/8", "::1",
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
	}); err != nil {
		logger.Warn("设置可信代理失败，使用 gin 默认（信任所有代理）", logger.ErrField(err))
	}

	// 全局中间件。
	// CORS 必须位于所有可能自行中止响应的中间件（MaxBodyBytes 413 /
	// RequestTimeout 503）之前：否则跨源请求的错误响应不带 CORS 头，
	// 浏览器把 413/503 上报成含糊的 CORS 错误而非真实状态（docs/29 P3）。
	router.Use(middleware.ZapLogger())
	router.Use(middleware.ZapRecovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.CORSMiddleware())
	// 会话标识：收藏/历史按 nm_sid（浏览器）或 X-Session-ID（API 客户端）隔离。
	router.Use(middleware.SessionID())
	// 请求体上限：本站最大请求体是 /report/* 的报告数据（几十 KB 量级），
	// 1 MiB 留足余量；超限直接 413，避免大 body 造成内存放大。
	router.Use(middleware.MaxBodyBytes(1 << 20))
	router.Use(middleware.RequestTimeout(30 * time.Second))
	rateLimitHandler, rateLimiter := middleware.IPRateLimitWithLimiter(rateCapacity, rateRate)
	router.Use(rateLimitHandler)

	// 健康检查
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now().Unix()})
	})
	router.GET("/readyz", func(c *gin.Context) {
		// 探活口径（docs/40 A5）：优先 ReadPing 打持久层读连接 —— 此前的
		// GetHanziByChar 探的是启动期预载的内存 map，SQLite 读路径损坏/被锁
		// 依然 200，readyz 失真。内存库存储的 ReadPing 恒 nil（无磁盘依赖）。
		if store != nil {
			if err := store.ReadPing(); err == nil {
				c.JSON(200, gin.H{"status": "ready", "db": "connected"})
				return
			}
		}
		c.JSON(503, gin.H{"status": "not ready", "db": "disconnected"})
	})

	// 门禁指标端点（docs/33 M4）：命中率 / 命中分布 + 复审闭环指标（数据源 M6）
	router.GET("/metrics", func(c *gin.Context) {
		c.JSON(200, fate.GateMetricsSnapshot())
	})

	// API 路由
	api := router.Group("/api/v1")
	{
		api.POST("/names/generate", h.name.Generate)
		// 异步生成（长任务体验）：提交 task_id → 轮询进度/结果
		api.POST("/names/generate/async", h.nameAsync.GenerateAsync)
		api.GET("/names/task/:id", h.nameAsync.GetTask)
		// 探索模式（换一批）：与上次生成结果零交集
		api.POST("/names/generate/explore", h.nameAsync.Explore)
		// 测名：输入姓名 + 生辰 → 完整评分报告 + 风险体检
		api.POST("/names/evaluate", h.name.Evaluate)

		// 注：/names/generate/analysis、/bazi/analyze、/yijing/*、/zodiac/*、
		// /feedback*、/report/html、/namestats/*、/characters/radical|curated-names|styles、
		// /favorites/check、/history/batch、/favorites/batch 已下线——前端无页面调用
		// （docs/27 接口利用率审计：30 个接口仅 9 个在用）。领域层能力保留，
		// 由生成管线内部消费（八字/易卦/生肖已并入生成响应）。

		api.GET("/history", h.history.GetHistory)
		api.POST("/history", h.history.SaveHistory)
		api.DELETE("/history/:id", h.history.DeleteHistory)

		api.GET("/favorites", h.favorite.GetFavorites)
		api.POST("/favorites", h.favorite.SaveFavorite)
		api.DELETE("/favorites/:id", h.favorite.DeleteFavorite)

		api.POST("/report/pdf", h.report.GeneratePDF)

		api.GET("/namestat/:name", h.stat.GetNameStats)
		api.GET("/namestat", h.stat.GetTopNames)

		api.GET("/huangli", h.huangli.GetHuangli)
		api.GET("/lunar", h.huangli.GetLunarCalendar)
		api.GET("/lunar/month", h.huangli.GetLunarMonthCalendar)

		// 汉字/偏旁数据
		api.GET("/characters/groups", h.character.GetCharGroups)
	}

	// 静态文件服务（仅 all 模式）
	if runMode == "all" {
		server.SetupStaticFileServer(router, server.StaticConfig{Root: staticDir})
	}

	return router, rateLimiter
}

// --- 服务器启动 ---

func startServer(router *gin.Engine, addr, cwd string, store database.Store, rateLimiter *middleware.IPRateLimiter, readHeaderTimeout time.Duration) {
	logger.Info("服务开始监听", logger.String("addr", addr))
	displayAddr := addr
	if strings.HasPrefix(displayAddr, ":") {
		displayAddr = "localhost" + displayAddr
	}
	fmt.Printf("\n  🚀 起名 NameMaster 服务已启动\n  📡 监听地址: http://%s\n  ⏳ 打开浏览器访问 http://%s\n\n", addr, displayAddr)

	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	// errCh 用于将启动失败信号回传主流程，避免主流程在等信号时无法感知
	errCh := make(chan error, 1)

	// 信号注册必须先于 ListenAndServe 启动：否则启动早期到达的
	// SIGINT/SIGTERM 走默认处置直接杀进程，跳过下方全部清理
	// （Windows 上未关闭的 SQLite 句柄会锁住库文件，docs/29 P3）。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		certFile := filepath.Join(cwd, "cert.pem")
		keyFile := filepath.Join(cwd, "key.pem")

		_, certErr := os.Stat(certFile)
		_, keyErr := os.Stat(keyFile)
		if os.IsNotExist(certErr) || os.IsNotExist(keyErr) {
			logger.Warn("未找到 TLS 证书，以 HTTP 模式运行")
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		} else {
			logger.Info("以 HTTPS 模式运行（支持 HTTP/2）")
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}
	}()

	// 等待中断信号或启动失败
	select {
	case sig := <-quit:
		logger.Info("服务正在关闭", logger.String("signal", sig.String()))
	case err := <-errCh:
		// 启动失败同样要走完整清理：此前直接 return，跳过 store 关闭、
		// 限流器停止与缓存释放，Windows 下残留句柄锁住 namer.db
		logger.Error("服务启动失败", logger.ErrField(err))
	}

	// 给予 10 秒时间完成正在处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("服务强制关闭", logger.ErrField(err))
	}

	// 关闭数据库连接
	if store != nil {
		if closer, ok := store.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				logger.Error("关闭数据库失败", logger.ErrField(err))
			}
		}
	}

	// 停止限流器清理协程
	if rateLimiter != nil {
		rateLimiter.Stop()
	}

	logger.Info("服务已退出")
}

// loadCulturalData 装载传统文化物料
//
// db 为 nil 时走纯 JSON 路径；非 nil 时优先从 SQLite 物料库读（ADR-008）。
// SQLite 路径下必需物料缺失会返回错误并附带 materials import 恢复命令，
// 由调用方拒绝启动（ADR-008 fail-fast）。
func loadCulturalData(dataDir, dbPath string, db database.Store) error {
	if loader, ok := db.(data.DBLoader); ok {
		return loadCulturalDataFromDB(dataDir, dbPath, loader)
	}
	return data.Init(dataDir)
}

// warnClassicsExtractGaps 经典字集提取完整性校验（docs/44 遗留④）。
//
// 复用 materials status 的同源统计（Store.ClassicsStats + classics.RegisteredMaterials），
// 对「已登记字集物料在库中行数为 0」的静默遗漏打 ERROR 日志。不阻断启动：
// classics 为可选物料，核心生成链路不依赖；CI 侧由 e2e_check 的
// source_classic=shijing/chuci 场景断言兜底。
func warnClassicsExtractGaps(store database.Store) {
	sc, ok := store.(*sqlite.Store)
	if !ok {
		return // 内存存储无物料库概念
	}
	stats := sc.ClassicsStats()
	if stats.TotalExtracts == 0 {
		return // 经典语料未导入：必需物料缺失已由 loadCulturalData fail-fast 覆盖
	}
	var missing []string
	for _, m := range classics.RegisteredMaterials() {
		if stats.ExtractByMaterial[m] == 0 {
			missing = append(missing, m)
		}
	}
	if len(missing) > 0 {
		logger.Error("经典字集提取存在静默遗漏，出典加分与来源筛选将降级",
			logger.String("缺失来源", strings.Join(missing, ", ")),
			logger.String("修复指引", "在仓库根执行 go -C backend run ./cmd/materials import 后重启服务"),
		)
	}
}

// loadCulturalDataFromDB 从 SQLite 物料库装载，失败时在错误中附带恢复命令
//
// 单独拆出是为了可直接用 data.DBLoader 假实现测试，不必构造整个 database.Store。
func loadCulturalDataFromDB(dataDir, dbPath string, loader data.DBLoader) error {
	if err := data.InitWithDB(dataDir, loader); err != nil {
		return fmt.Errorf("%w；恢复命令: %s", err, materialsImportHint(dataDir, dbPath))
	}
	return nil
}

// materialsImportHint 构造物料导入指引命令
//
// 参数与 cmd/materials 的 import 子命令保持一致：--data 源文件目录、--db 目标库。
// 用普通双引号而非 %q：%q 是 Go 语法转义，会把 Windows 路径写成
// `D:\\19-...`，用户照抄到 shell 里是错的；此处产出的是 shell 命令。
func materialsImportHint(dataDir, dbPath string) string {
	return fmt.Sprintf(`go run ./cmd/materials import --data "%s" --db "%s"`, dataDir, dbPath)
}

// loadCuratedNamesForFate 从库读取策展名的 name 字段，返回策展好名列表
// 注入 fate 引擎作为共现分白名单（与 verify_fate 的 loadCuratedNames 逻辑一致）
//
// 运行时只读 SQLite（docs/31 ADR-008）：curated_names 表由 materials import 导入，
// 不再读 curated_names.json。库不可用/读取失败时返回 nil（fate 引擎无策展白名单）。
func loadCuratedNamesForFate(store database.Store) []string {
	if store == nil {
		return nil
	}
	entries, err := store.LoadAllCuratedNames()
	if err != nil {
		logger.Warn("从库读取策展名失败，fate 引擎无策展白名单", logger.ErrField(err))
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name != "" {
			names = append(names, e.Name)
		}
	}
	return names
}
