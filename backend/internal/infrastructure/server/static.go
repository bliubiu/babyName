package server

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"name/internal/infrastructure/logger"
)

//go:embed frontend
var embeddedFiles embed.FS

// StaticConfig 静态文件服务配置
type StaticConfig struct {
	Root string // 静态文件根目录（仅开发模式使用文件系统时有效）
}

// SetupStaticFileServer 配置静态文件服务
// 优先使用嵌入的静态文件（生产模式），回退到文件系统（开发模式）
func SetupStaticFileServer(router *gin.Engine, cfg StaticConfig) {
	// 尝试从嵌入文件系统加载
	subFS, err := fs.Sub(embeddedFiles, "frontend")
	if err == nil {
		if _, err := fs.Stat(subFS, "index.html"); err == nil {
			logger.Info("静态文件模式: 嵌入 (生产)")
			setupEmbeddedStatic(router, subFS)
			return
		}
	}

	// 回退到文件系统（开发模式）
	logger.Info("静态文件模式: 文件系统 (开发)", logger.String("root", cfg.Root))
	setupFileSystemStatic(router, cfg)
}

// --- 嵌入模式（生产） ---

func setupEmbeddedStatic(router *gin.Engine, staticFS fs.FS) {
	group := router.Group("/")
	group.Use(staticCacheControl())
	group.StaticFS("/", http.FS(staticFS))
}

// staticCacheControl 嵌入模式的缓存控制
func staticCacheControl() gin.HandlerFunc {
	cacheRules := getCacheRules()
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if rule, ok := cacheRules[path]; ok {
			rule(c)
		} else if strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".txt") {
			noCache(c)
		} else {
			longCache(c)
		}
		c.Next()
	}
}

// --- 文件系统模式（开发） ---

func setupFileSystemStatic(router *gin.Engine, cfg StaticConfig) {
	staticGroup := router.Group("/")
	staticGroup.Use(fsCacheControl())

	mode := detectBuildMode(cfg.Root)
	logger.Info("检测到构建模式", logger.String("mode", mode))

	// 注册静态资源目录
	switch mode {
	case "standalone":
		staticGroup.Static("/_next", filepath.Join(cfg.Root, ".next"))
		staticGroup.Static("/static", filepath.Join(cfg.Root, "public", "static"))
	default:
		staticGroup.Static("/_next", filepath.Join(cfg.Root, "_next"))
		staticGroup.Static("/static", filepath.Join(cfg.Root, "static"))
	}

	// 页面路由
	pages := []string{"index", "compare", "evaluate", "favorites", "history", "result", "huangli", "stat", "json-viewer", "share"}
	for _, page := range pages {
		registerPageRoute(staticGroup, page, mode, cfg.Root)
	}

	// 根级静态资源（Next.js 静态导出产物直接落在 out/ 根目录）
	// 例：public/json4matter.html、manifest.json、sw.js、gate-data/ 等。
	// 这些不在 _next/ 或 static/ 下，必须单独注册，否则 404。
	registerRootStaticAssets(staticGroup, mode, cfg.Root)
}

// rootStaticFiles 根级静态资源文件名白名单
// 说明：只放行明确由 public/ 引入的产物，避免把 out/ 下的临时文件全部暴露出去。
var rootStaticFiles = []string{
	"json4matter.html", // 离线单文件 JSON 美化查看器（/json-viewer 页面的 iframe 内容）
	"manifest.json",    // PWA manifest
	"sw.js",            // service worker
}

// rootStaticDirs 根级静态资源目录
var rootStaticDirs = []string{
	"gate-data", // 门禁表 JSON 数据
}

// registerRootStaticAssets 注册 out/ 根目录下的静态资源与目录。
// Next.js output:'export' 会把 public/ 的内容原样复制到产物根目录，
// 但 gin 的 StaticFS("/") 语义会把根目录整体挂载；此处按文件名/目录精确注册，
// 避免误将 .txt 等内部产物暴露，同时保证 /json4matter.html 等可访问。
func registerRootStaticAssets(group *gin.RouterGroup, mode, root string) {
	base := root
	if mode == "standalone" {
		base = filepath.Join(root, "public")
	}

	for _, name := range rootStaticFiles {
		file := filepath.Join(base, name)
		if _, err := os.Stat(file); err != nil {
			continue // 产物中不存在该文件则跳过（如 public/ 未提供）
		}
		group.StaticFile("/"+name, file)
	}

	for _, dir := range rootStaticDirs {
		dirPath := filepath.Join(base, dir)
		if st, err := os.Stat(dirPath); err != nil || !st.IsDir() {
			continue
		}
		group.Static("/"+dir, dirPath)
	}
}

// fsCacheControl 文件系统模式的缓存控制
func fsCacheControl() gin.HandlerFunc {
	cacheRules := getCacheRules()
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if rule, ok := cacheRules[path]; ok {
			rule(c)
		} else if strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".txt") {
			noCache(c)
		} else {
			longCache(c)
		}
		c.Next()
	}
}

// --- 缓存控制工具 ---

type cacheRule func(*gin.Context)

func getCacheRules() map[string]cacheRule {
	return map[string]cacheRule{
		"/":            noCache,
		"/compare":     noCache,
		"/favorites":   noCache,
		"/history":     noCache,
		"/result":      noCache,
		"/huangli":     noCache,
		"/evaluate":    noCache,
		"/stat":        noCache,
		"/json-viewer": noCache,
		"/share":       noCache,
	}
}

func noCache(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

func longCache(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=31536000")
}

// --- 文件系统独有功能（检测构建模式、页面路由） ---

// detectBuildMode 检测前端构建模式
func detectBuildMode(root string) string {
	if _, err := os.Stat(filepath.Join(root, ".next")); err == nil {
		return "standalone"
	}
	if _, err := os.Stat(filepath.Join(root, "_next")); err == nil {
		return "export"
	}
	return "out"
}

// registerPageRoute 注册单个页面的 HTML 路由
func registerPageRoute(group *gin.RouterGroup, page, mode, root string) {
	// 确定 HTML 文件路径
	htmlPath := htmlFilePath(page, mode, root)

	// 注册 HTML 路由
	routePath := "/" + page
	if page == "index" {
		routePath = "/"
	}
	group.GET(routePath, serveHTML(htmlPath))

	// 注册 .txt 文件路由（Next.js 静态导出产物）
	txtPath := filepath.Join(root, page+".txt")
	group.GET("/"+page+".txt", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		f, err := os.Open(txtPath)
		if err != nil {
			c.String(404, "Not Found")
			return
		}
		defer f.Close()
		if _, err := io.Copy(c.Writer, f); err != nil {
			logger.Warn("静态文件传输失败", zap.String("path", txtPath), zap.Error(err))
		}
	})
}

// htmlFilePath 根据构建模式获取 HTML 文件的真实路径
func htmlFilePath(page, mode, root string) string {
	switch mode {
	case "standalone":
		path := filepath.Join(root, "public", page, "index.html")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			path = filepath.Join(root, page, "index.html")
		}
		return path
	default:
		return filepath.Join(root, page+".html")
	}
}

// serveHTML 返回一个处理 HTML 请求的 HandlerFunc
func serveHTML(filePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		f, err := os.Open(filePath)
		if err != nil {
			c.String(404, "Not Found")
			return
		}
		defer f.Close()
		if _, err := io.Copy(c.Writer, f); err != nil {
			logger.Warn("静态文件传输失败", zap.String("path", filePath), zap.Error(err))
		}
	}
}
