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
	pages := []string{"index", "compare", "favorites", "history", "result", "huangli"}
	for _, page := range pages {
		registerPageRoute(staticGroup, page, mode, cfg.Root)
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
		"/":           noCache,
		"/compare":    noCache,
		"/favorites":  noCache,
		"/history":    noCache,
		"/result":     noCache,
		"/huangli":    noCache,
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
		io.Copy(c.Writer, f)
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
		io.Copy(c.Writer, f)
	}
}
