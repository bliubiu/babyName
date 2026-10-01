package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"name/internal/application/handlers"
	"name/internal/application/services"
	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/domain/name"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/config"
	"name/internal/infrastructure/data"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/database/memory"
	"name/internal/infrastructure/database/sqlite"
	"name/internal/infrastructure/logger"
	"name/internal/infrastructure/middleware"
	"name/internal/infrastructure/server"
)

var (
	port         = flag.Int("port", 0, "Server port (覆盖配置文件)")
	host         = flag.String("host", "", "Server host (覆盖配置文件)")
	mode         = flag.String("mode", "", "Run mode: backend or all (覆盖配置文件)")
	static       = flag.String("static", "", "Static files directory (覆盖配置文件)")
	help         = flag.Bool("help", false, "Show help")
	logPath      = flag.String("log", "", "Log file path (覆盖配置文件)")
	dbPath       = flag.String("db", "", "SQLite database file path (覆盖配置文件)")
	rateCapacity = flag.Float64("rate-capacity", 0, "Rate limiter capacity (覆盖配置文件)")
	rateRate     = flag.Float64("rate-rate", 0, "Rate limiter rate (覆盖配置文件)")
	configFile   = flag.String("config", "", "配置文件路径 (默认: application.yml)")
)

// cwdDataDir 解析传统文化数据的目录（docs/29 B12）
//
// 优先级：DATA_DIR 环境变量 > 工作目录下的 data/。
// docker-entrypoint.sh 按 DATA_DIR 播种字库，本函数与之保持同源；
// 分离成独立函数是为了可被测试直接覆盖。
func cwdDataDir(cwd string) string {
	if dir := strings.TrimSpace(os.Getenv("DATA_DIR")); dir != "" {
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

	// 初始化缓存（SQLite + 内存缓存，不使用 Redis）
	cache.Init()

	// 加载传统文化数据（从 JSON 文件）
	//
	// ★ docs/29 B12：原先固定 `filepath.Join(cwd, "data")`，但 compose 与
	// docker-entrypoint.sh 都按 DATA_DIR 环境变量播种字库。两者不一致时，
	// 字库被播种到 A 目录、应用却去 B 目录读 —— 服务正常启动、healthz 通过，
	// 生成接口全部返回空结果。故此处以 DATA_DIR 为准（缺省才回落到 cwd/data），
	// 让入口脚本与进程读同一个目录。
	dataDir := cwdDataDir(cwd)

	// 1. 加载传统文化数据（hanzi.json → 易经 → 诗词 → 生肖）
	if err := loadCulturalData(dataDir); err != nil {
		logger.Warn("Failed to load cultural data from JSON, using built-in data", logger.ErrField(err))
	} else {
		logger.Info("Cultural data loaded from JSON files", logger.String("dataDir", dataDir))
	}

	// 2. 加载 word.json 释义（候选字未覆盖的 fallback 释义）
	if err := hanzi.LoadWordData(dataDir); err != nil {
		logger.Warn("Failed to load word.json, some character meanings may be unavailable", logger.ErrField(err))
	} else {
		logger.Info("Word data loaded from word.json")
	}

	// 检查静态文件目录
	ensureStaticDir(cfg.Static)

	logger.Info("Starting NameMaster server",
		logger.String("host", cfg.Server.Host),
		logger.Int("port", cfg.Server.Port),
	)

	// 初始化数据存储 — 优先 SQLite 持久化，回退到内存存储
	store := initStore(cfg.Database.Path, dataDir)

	// 创建服务层
	cacheInst := cache.GetCache()
	svc := newServices(store, cacheInst, dataDir)

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
		logger.Warn("Frontend static files not found", logger.String("path", path))
		if err := os.MkdirAll(path, 0755); err != nil {
			logger.Fatal("Failed to create static files directory", logger.ErrField(err))
		}
	}
	indexPath := filepath.Join(path, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		logger.Warn("index.html not found in static directory", logger.String("path", indexPath))
	}
	logger.Info("Using static files directory", logger.String("path", path))
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
		logger.Info("Using SQLite store", logger.String("db", dbPath))
		return dbStore
	} else {
		logger.Warn("Failed to open SQLite, falling back to memory store", logger.ErrField(err))
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

func newServices(store database.Store, cacheInst cache.Cache, dataDir string) *appServices {
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
	if curatedNames := loadCuratedNamesForFate(dataDir); len(curatedNames) > 0 {
		fate.SetCuratedNames(curatedNames)
		logger.Info("已注入策展好名到 fate 引擎", logger.Int("count", len(curatedNames)))
	}
	// SQLite 下沉（P3 方案6 装配）：构造 HanziDataProvider 引用，
	// 若底层 store 支持 SQL 过滤则注入到 provider，fate 引擎沿用同一 provider。
	// 失败/不支持时降级到 Go 端全表过滤（向后兼容）。
	hanziProvider := &services.HanziDataProvider{}
	if store != nil && services.HasSQLiteCharStore(store) {
		hanziProvider.SetSQLFilter(services.NewSQLiteHanziFilter(services.AsSQLiteCharStore(store)))
		logger.Info("SQLite 汉字过滤已启用", logger.String("store", "sqlite"))
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
		services.WithCache(cacheInst),
		services.WithFateService(fateSvc), // fate 引擎优先（新引擎更完善）
	)

	// 异步生成任务（长任务体验：提交 task_id → 轮询进度/结果）
	taskSvc := services.NewTaskService(nameSvc)

	// 创建收藏服务（需要 NameDB 实现自学习）
	favSvc := services.NewFavoriteService(store)
	if ndb := nameSvc.GetNameDB(); ndb != nil {
		favSvc.SetNameDB(ndb)
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
		// 检查数据库是否可用
		if store != nil {
			// 尝试执行轻量查询验证连通性
			if store.GetHanziByChar("一") != nil {
				c.JSON(200, gin.H{"status": "ready", "db": "connected"})
				return
			}
		}
		c.JSON(503, gin.H{"status": "not ready", "db": "disconnected"})
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
	logger.Info("Server starting", logger.String("addr", addr))
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
			logger.Warn("TLS certificates not found, running in HTTP mode")
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		} else {
			logger.Info("Running in HTTPS mode with HTTP/2 support")
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}
	}()

	// 等待中断信号或启动失败
	select {
	case sig := <-quit:
		logger.Info("Shutting down server", logger.String("signal", sig.String()))
	case err := <-errCh:
		// 启动失败同样要走完整清理：此前直接 return，跳过 store 关闭、
		// 限流器停止与缓存释放，Windows 下残留句柄锁住 namer.db
		logger.Error("Failed to start server", logger.ErrField(err))
	}

	// 给予 10 秒时间完成正在处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", logger.ErrField(err))
	}

	// 关闭数据库连接
	if store != nil {
		if closer, ok := store.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				logger.Error("Failed to close database", logger.ErrField(err))
			}
		}
	}

	// 停止限流器清理协程
	if rateLimiter != nil {
		rateLimiter.Stop()
	}

	// 关闭缓存实例，释放后台清理协程
	if cacheInst := cache.GetCache(); cacheInst != nil {
		if closer, ok := cacheInst.(interface{ Close() }); ok {
			closer.Close()
		}
	}

	logger.Info("Server exited")
}

// loadCulturalData 从 JSON 文件加载传统文化数据
func loadCulturalData(dataDir string) error {
	return data.Init(dataDir)
}

// loadCuratedNamesForFate 读取 curated_names.json 的 name 字段，返回策展好名列表
// 注入 fate 引擎作为共现分白名单（与 verify_fate 的 loadCuratedNames 逻辑一致）
func loadCuratedNamesForFate(dataDir string) []string {
	entries, err := name.LoadCuratedNamesData(dataDir)
	if err != nil {
		logger.Warn("读取 curated_names.json 失败，fate 引擎无策展白名单", logger.ErrField(err))
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
