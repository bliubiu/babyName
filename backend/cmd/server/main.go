package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/namemaster/backend/internal/application/handlers"
	"github.com/namemaster/backend/internal/application/services"
	"github.com/namemaster/backend/internal/infrastructure/database/memory"
	"github.com/namemaster/backend/internal/infrastructure/logger"
	"github.com/namemaster/backend/internal/infrastructure/middleware"
)

var (
	port    = flag.Int("port", 8080, "Server port")
	host    = flag.String("host", "localhost", "Server host")
	mode    = flag.String("mode", "backend", "Run mode: backend or all")
	static  = flag.String("static", "", "Static files directory for frontend (default: frontend/out)")
	help    = flag.Bool("help", false, "Show help")
	logPath = flag.String("log", "", "Log file path")
)

func main() {
	flag.Parse()

	if *static == "" {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		*static = filepath.Join(cwd, "..", "frontend", "out")
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

	logger.Info("Starting NameMaster server",
		logger.String("host", *host),
		logger.Int("port", *port),
	)

	store := memory.NewStore()

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

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.ZapLogger())
	router.Use(middleware.ZapRecovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.CORSMiddleware())

	api := router.Group("/api/v1")
	{
		api.POST("/names/generate", nameHandler.Generate)
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

		api.POST("/report/pdf", reportHandler.GeneratePDF)
		api.POST("/report/html", reportHandler.GenerateHTML)

		api.GET("/namestat/:name", nameStatHandler.GetNameStats)
		api.GET("/namestat", nameStatHandler.GetTopNames)
		api.GET("/namestat/:name/province", nameStatHandler.GetProvinceStats)
	}

	if *mode == "all" {
		logger.Info("Static files directory", logger.String("path", *static))
		router.Static("/static", *static)
		router.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Status(200)
			filePath := *static + "/index.html"
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
	if err := router.Run(addr); err != nil {
		logger.Fatal("Failed to start server",
			logger.Error(err),
		)
	}
}
