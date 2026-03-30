package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/bliubiu/babyName/internal/application/handlers"
	"github.com/bliubiu/babyName/internal/application/services"
	"github.com/bliubiu/babyName/internal/infrastructure/cache"
	"github.com/bliubiu/babyName/internal/infrastructure/database/memory"
	"github.com/bliubiu/babyName/internal/infrastructure/logger"
	"github.com/bliubiu/babyName/internal/infrastructure/middleware"
)

var (
	port    = flag.Int("port", 8080, "Server port")
	host    = flag.String("host", "localhost", "Server host")
	mode    = flag.String("mode", "all", "Run mode: backend or all")
	static  = flag.String("static", "", "Static files directory for frontend (default: frontend/out)")
	help    = flag.Bool("help", false, "Show help")
	logPath = flag.String("log", "name.runlog", "Log file path")
)

func main() {
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// 优先使用环境变量配置静态文件路径
	if envStatic := os.Getenv("FRONTEND_STATIC_PATH"); envStatic != "" {
		*static = envStatic
	} else if *static == "" {
		// 检查standalone输出目录
		standalonePath := filepath.Join(cwd, "..", "frontend", ".next", "standalone")
		if _, err := os.Stat(standalonePath); err == nil {
			*static = standalonePath
		} else {
			// 回退到传统out目录
			*static = filepath.Join(cwd, "..", "frontend", "out")
		}
	}

	if *help {
		fmt.Println("宝宝起名大师 - NameMaster")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  namemaster.exe [options]")
		fmt.Println("")
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if err := logger.Init(&logger.Config{
		Level:      "debug",
		Format:     "console",
		OutputPath: *logPath,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// 初始化缓存
	cacheConfig := cache.Config{
		Type:     os.Getenv("CACHE_TYPE"),     // 从环境变量获取缓存类型
		RedisURL: os.Getenv("REDIS_URL"), // 从环境变量获取Redis连接URL
	}
	cache.Init(cacheConfig)

	// 检查静态文件目录是否存在
	if _, err := os.Stat(*static); os.IsNotExist(err) {
		logger.Warn("Frontend static files not found", logger.String("path", *static))
		logger.Info("Creating static files directory", logger.String("path", *static))
		if err := os.MkdirAll(*static, 0755); err != nil {
			logger.Fatal("Failed to create static files directory", logger.Error(err))
		}
	}

	// 检查关键静态文件是否存在
	indexHtmlPath := filepath.Join(*static, "index.html")
	if _, err := os.Stat(indexHtmlPath); os.IsNotExist(err) {
		logger.Warn("index.html not found", logger.String("path", indexHtmlPath))
	}

	logger.Info("Using static files directory", logger.String("path", *static))
	logger.Info("Current working directory", logger.String("cwd", cwd))

	logger.Info("Starting NameMaster server",
		logger.String("host", *host),
		logger.Int("port", *port),
	)

	// 使用内存存储
	store := memory.NewStore()
	logger.Info("Using memory store")

	nameService := services.NewNameService(store)
	baziService := services.NewBaziService(store)
	yijingService := services.NewYijingService(store)
	zodiacService := services.NewZodiacService(store)
	historyService := services.NewHistoryService(store)
	favoriteService := services.NewFavoriteService(store)
	reportService := services.NewReportService(store)

	nameHandler := handlers.NewNameHandler(nameService)
	baziHandler := handlers.NewBaziHandler(baziService)
	yijingHandler := handlers.NewYijingHandler(yijingService)
	zodiacHandler := handlers.NewZodiacHandler(zodiacService)
	historyHandler := handlers.NewHistoryHandler(historyService)
	favoriteHandler := handlers.NewFavoriteHandler(favoriteService)
	reportHandler := handlers.NewReportHandler(reportService)
	nameStatHandler := handlers.NewNameStatHandler()
	huangliHandler := handlers.NewHuangliHandler()

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.ZapLogger())
	router.Use(middleware.ZapRecovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.IPRateLimit(100, 10))

	api := router.Group("/api/v1")
	{
		api.POST("/names/generate", nameHandler.Generate)
		api.POST("/names/generate/analysis", nameHandler.GenerateWithAnalysis)
		api.GET("/names/:id", nameHandler.GetByID)

		api.POST("/bazi/analyze", baziHandler.Analyze)

		api.GET("/yijing/hexagram/:id", yijingHandler.GetHexagram)
		api.GET("/yijing/hexagram", yijingHandler.GetAllHexagrams)

		api.GET("/zodiac/:animal", zodiacHandler.GetZodiac)
		api.GET("/zodiac", zodiacHandler.GetAllZodiacs)

		api.GET("/history", historyHandler.GetHistory)
		api.POST("/history", historyHandler.SaveHistory)
		api.DELETE("/history/:id", historyHandler.DeleteHistory)

		api.GET("/favorites", favoriteHandler.GetFavorites)
		api.POST("/favorites", favoriteHandler.SaveFavorite)
		api.DELETE("/favorites/:id", favoriteHandler.DeleteFavorite)
		api.GET("/favorites/check", favoriteHandler.CheckFavorite)
		// 批量操作API
		api.POST("/history/batch", historyHandler.BatchSaveHistory)
		api.POST("/favorites/batch", favoriteHandler.BatchSaveFavorite)
		api.DELETE("/favorites/batch", favoriteHandler.BatchDeleteFavorite)

		api.POST("/report/pdf", reportHandler.GeneratePDF)
		api.POST("/report/html", reportHandler.GenerateHTML)

		api.GET("/namestat/:name", nameStatHandler.GetNameStats)
		api.GET("/namestat", nameStatHandler.GetTopNames)
		api.GET("/namestat/:name/province", nameStatHandler.GetProvinceStats)

		// 黄历相关API
		api.GET("/huangli", huangliHandler.GetHuangli)
		api.GET("/lunar", huangliHandler.GetLunarCalendar)
	}

	if *mode == "all" {
		// 处理Next.js的静态文件，添加缓存控制
		staticGroup := router.Group("/")
		staticGroup.Use(func(c *gin.Context) {
			// 对HTML文件和根路径禁用缓存，解决旧版页面问题
			if strings.HasSuffix(c.Request.URL.Path, ".html") || c.Request.URL.Path == "/" || 
			   strings.HasSuffix(c.Request.URL.Path, "/compare") || 
			   strings.HasSuffix(c.Request.URL.Path, "/favorites") || 
			   strings.HasSuffix(c.Request.URL.Path, "/history") || 
			   strings.HasSuffix(c.Request.URL.Path, "/result") || 
			   strings.HasSuffix(c.Request.URL.Path, "/stat") {
				c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
				c.Header("Pragma", "no-cache")
				c.Header("Expires", "0")
			} else {
				// 对静态资源设置合理的缓存
				c.Header("Cache-Control", "public, max-age=31536000")
			}
			c.Next()
		})
		
		// 检查构建模式：standalone模式有.next目录，export模式有_next目录
		isStandalone := false
		if _, err := os.Stat(filepath.Join(*static, ".next")); err == nil {
			isStandalone = true
		}
		
		// 检查是否为export模式（HTML文件直接在根目录）
		isExport := false
		if _, err := os.Stat(filepath.Join(*static, "_next")); err == nil {
			if _, err := os.Stat(filepath.Join(*static, "compare.html")); err == nil {
				isExport = true
			}
		}
		
		// 根据模式设置静态文件路径
		if isStandalone {
			// Standalone模式
			staticGroup.Static("/_next", filepath.Join(*static, ".next"))
			staticGroup.Static("/static", filepath.Join(*static, "public", "static"))
		} else if isExport {
			// Export模式 - HTML文件在根目录
			staticGroup.Static("/_next", filepath.Join(*static, "_next"))
			staticGroup.Static("/static", filepath.Join(*static, "static"))
		} else {
			// 传统out模式
			staticGroup.Static("/_next", filepath.Join(*static, "_next"))
			staticGroup.Static("/static", filepath.Join(*static, "static"))
		}
		
		// 辅助函数：获取HTML文件路径
		getHtmlPath := func(page string) string {
			if isExport {
				// Export模式：HTML文件直接在根目录，如 compare.html
				return filepath.Join(*static, page+".html")
			} else if isStandalone {
				// Standalone模式
				path := filepath.Join(*static, "public", page, "index.html")
				if _, err := os.Stat(path); os.IsNotExist(err) {
					path = filepath.Join(*static, page, "index.html")
				}
				return path
			} else {
				// 传统out模式
				return filepath.Join(*static, page, "index.html")
			}
		}
		
		// 处理根路径
		staticGroup.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			filePath := getHtmlPath("index")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
		
		// 处理其他页面路由
		staticGroup.GET("/compare", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			filePath := getHtmlPath("compare")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
		
		staticGroup.GET("/favorites", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			filePath := getHtmlPath("favorites")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
		
		staticGroup.GET("/history", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			filePath := getHtmlPath("history")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
		
		staticGroup.GET("/result", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			filePath := getHtmlPath("result")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
		
		staticGroup.GET("/stat", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			filePath := getHtmlPath("stat")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
		
		// 处理Next.js静态导出生成的.txt文件请求
		staticGroup.GET("/index.txt", func(c *gin.Context) {
			c.Header("Content-Type", "text/plain; charset=utf-8")
			filePath := filepath.Join(*static, "index.txt")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
		
		staticGroup.GET("/compare.txt", func(c *gin.Context) {
			c.Header("Content-Type", "text/plain; charset=utf-8")
			filePath := filepath.Join(*static, "compare.txt")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
		
		staticGroup.GET("/favorites.txt", func(c *gin.Context) {
			c.Header("Content-Type", "text/plain; charset=utf-8")
			filePath := filepath.Join(*static, "favorites.txt")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
		
		staticGroup.GET("/history.txt", func(c *gin.Context) {
			c.Header("Content-Type", "text/plain; charset=utf-8")
			filePath := filepath.Join(*static, "history.txt")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
		
		staticGroup.GET("/result.txt", func(c *gin.Context) {
			c.Header("Content-Type", "text/plain; charset=utf-8")
			filePath := filepath.Join(*static, "result.txt")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
		
		staticGroup.GET("/stat.txt", func(c *gin.Context) {
			c.Header("Content-Type", "text/plain; charset=utf-8")
			filePath := filepath.Join(*static, "stat.txt")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(404, "Not Found")
				return
			}
			defer file.Close()
			io.Copy(c.Writer, file)
		})
	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	logger.Info("Server starting", logger.String("addr", addr))
	
	// 启用 HTTP/2 支持
	// 注意：HTTP/2 需要 HTTPS 支持
	// 这里使用自签名证书，生产环境应该使用真实的 TLS 证书
	certFile := filepath.Join(cwd, "cert.pem")
	keyFile := filepath.Join(cwd, "key.pem")
	
	// 检查证书文件是否存在
	_, certErr := os.Stat(certFile)
	_, keyErr := os.Stat(keyFile)
	if os.IsNotExist(certErr) || os.IsNotExist(keyErr) {
		logger.Warn("TLS certificates not found, running in HTTP mode")
		// 在没有证书的情况下，使用 HTTP
		if err := router.Run(addr); err != nil {
			logger.Fatal("Failed to start server",
				logger.Error(err),
			)
		}
	} else {
		logger.Info("Running in HTTPS mode with HTTP/2 support")
		// 使用 HTTPS 启用 HTTP/2
		if err := router.RunTLS(addr, certFile, keyFile); err != nil {
			logger.Fatal("Failed to start server with TLS",
				logger.Error(err),
			)
		}
	}
}
