package main

import (
	"context"
	"encoding/json"
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
	"name/internal/domain/namestatistics"
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
	dataDir := filepath.Join(cwd, "data")

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
	name      *services.NameService
	bazi      *services.BaziService
	yijing    *services.YijingService
	zodiac    *services.ZodiacService
	history   *services.HistoryService
	favorite  *services.FavoriteService
	report    *services.ReportService
	feedback  *services.FeedbackService
	namestats namestatistics.NameStatisticsService
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
	fateSvc := services.NewFateNameService(fateEngine, services.WithFateBaziAnalyzer(baziAdapter))

	nameSvc := services.NewNameService(
		services.WithBaziAnalyzer(baziAdapter),
		services.WithHexagramFinder(hexagramAdapter),
		services.WithZiweiAnalyzer(ziweiAdapter),
		services.WithNameDB(ndb),
		services.WithZodiacFinder(zodiacAdapter),
		services.WithCache(cacheInst),
		services.WithFateService(fateSvc), // fate 引擎优先（新引擎更完善）
	)

	// 创建收藏服务（需要 NameDB 实现自学习）
	favSvc := services.NewFavoriteService(store)
	if ndb := nameSvc.GetNameDB(); ndb != nil {
		favSvc.SetNameDB(ndb)
	}

	// 姓名统计服务
	namestatsSvc := namestatistics.NewNameStatisticsService(store)

	return &appServices{
		name:      nameSvc,
		bazi:      services.NewBaziService(cacheInst),
		report:    services.NewReportService(),
		yijing:    services.NewYijingService(store, cacheInst),
		zodiac:    services.NewZodiacService(store),
		history:   services.NewHistoryService(store),
		favorite:  favSvc,
		feedback:  services.NewFeedbackService(store, store),
		namestats: namestatsSvc,
	}
}

// --- 处理器装配 ---

type appHandlers struct {
	name      *handlers.NameHandler
	bazi      *handlers.BaziHandler
	yijing    *handlers.YijingHandler
	zodiac    *handlers.ZodiacHandler
	history   *handlers.HistoryHandler
	favorite  *handlers.FavoriteHandler
	report    *handlers.ReportHandler
	feedback  *handlers.FeedbackHandler
	stat      *handlers.NameStatHandler
	huangli   *handlers.HuangliHandler
	character *handlers.CharacterHandler
	namestats *handlers.NameStatisticsHandler
}

func newHandlers(svc *appServices, dataDir string) *appHandlers {
	return &appHandlers{
		name:      handlers.NewNameHandler(svc.name),
		bazi:      handlers.NewBaziHandler(svc.bazi),
		yijing:    handlers.NewYijingHandler(svc.yijing),
		zodiac:    handlers.NewZodiacHandler(svc.zodiac),
		history:   handlers.NewHistoryHandler(svc.history),
		favorite:  handlers.NewFavoriteHandler(svc.favorite),
		report:    handlers.NewReportHandler(svc.report),
		feedback:  handlers.NewFeedbackHandler(svc.feedback),
		stat:      handlers.NewNameStatHandler(),
		huangli:   handlers.NewHuangliHandler(),
		character: handlers.NewCharacterHandler(svc.name.GetNameDB()),
		namestats: handlers.NewNameStatisticsHandler(svc.namestats),
	}
}

// --- 路由配置 ---

func setupRouter(h *appHandlers, runMode, staticDir string, store database.Store, rateCapacity, rateRate float64) (*gin.Engine, *middleware.IPRateLimiter) {
	router := gin.New()

	// 全局中间件
	router.Use(middleware.ZapLogger())
	router.Use(middleware.ZapRecovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.RequestTimeout(30 * time.Second))
	router.Use(middleware.CORSMiddleware())
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
		api.POST("/names/generate/analysis", h.name.GenerateWithAnalysis)
		api.GET("/names/:id", h.name.GetByID)

		api.POST("/bazi/analyze", h.bazi.Analyze)

		api.GET("/yijing/hexagram/:id", h.yijing.GetHexagram)
		api.GET("/yijing/hexagram", h.yijing.GetAllHexagrams)

		api.GET("/zodiac/:animal", h.zodiac.GetZodiac)
		api.GET("/zodiac", h.zodiac.GetAllZodiacs)

		api.GET("/history", h.history.GetHistory)
		api.POST("/history", h.history.SaveHistory)
		api.DELETE("/history/:id", h.history.DeleteHistory)

		api.GET("/favorites", h.favorite.GetFavorites)
		api.POST("/favorites", h.favorite.SaveFavorite)
		api.DELETE("/favorites/:id", h.favorite.DeleteFavorite)
		api.GET("/favorites/check", h.favorite.CheckFavorite)
		api.POST("/history/batch", h.history.BatchSaveHistory)
		api.POST("/favorites/batch", h.favorite.BatchSaveFavorite)
		api.DELETE("/favorites/batch", h.favorite.BatchDeleteFavorite)

		api.POST("/feedback", h.feedback.SaveFeedback)
		api.POST("/feedback/request", h.feedback.SaveRequest)
		api.GET("/feedback/algorithm-performance", h.feedback.GetAlgorithmPerformance)

		api.POST("/report/pdf", h.report.GeneratePDF)
		api.POST("/report/html", h.report.GenerateHTML)

		api.GET("/namestat/:name", h.stat.GetNameStats)
		api.GET("/namestat", h.stat.GetTopNames)

		// 姓名统计（基于 Chinese-Names-Corpus）
		api.GET("/namestats/surnames", h.namestats.GetSurnameStats)
		api.GET("/namestats/surnames/:surname", h.namestats.GetSurnameStat)
		api.GET("/namestats/surnames/:surname/given-names", h.namestats.GetGivenNameStats)
		api.GET("/namestats/surnames/:surname/full-names", h.namestats.GetFullNameStats)
		api.GET("/namestats/full-names/:full_name", h.namestats.GetFullNameStat)
		api.GET("/namestats/names/:name/gender", h.namestats.GetNameGenderStats)
		api.GET("/namestats/top", h.namestats.GetTopFullNames)
		api.GET("/namestats/total", h.namestats.GetTotalNameCount)

		api.GET("/huangli", h.huangli.GetHuangli)
		api.GET("/lunar", h.huangli.GetLunarCalendar)

		// 汉字/偏旁数据
		api.GET("/characters/groups", h.character.GetCharGroups)
		api.GET("/characters/radical", h.character.GetRadicalChars)
		api.GET("/characters/curated-names", h.character.GetCuratedNames)
		api.GET("/characters/styles", h.character.GetStyles)
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
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-quit:
		logger.Info("Shutting down server", logger.String("signal", sig.String()))
	case err := <-errCh:
		logger.Error("Failed to start server", logger.ErrField(err))
		return
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
	path := filepath.Join(dataDir, "curated_names.json")
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Warn("读取 curated_names.json 失败，fate 引擎无策展白名单", logger.ErrField(err))
		return nil
	}
	var entries []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		logger.Warn("解析 curated_names.json 失败，fate 引擎无策展白名单", logger.ErrField(err))
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
