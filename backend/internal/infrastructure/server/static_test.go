package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"name/internal/infrastructure/logger"
)

// logTestDir 测试用日志目录：固定 logs 目录后 Init 总会落盘，
// 测试侧统一重定向到临时缓存区，避免测试产物写入真实 logs/。
var logTestDir = func() string {
	dir, err := os.MkdirTemp("", "namer-server-logs")
	if err != nil {
		return os.TempDir()
	}
	return dir
}()

func init() {
	_ = logger.Init(&logger.Config{Level: "error", OutputPath: logTestDir})
}

func TestSetupStaticFileServer_EvaluateStatRoutes(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建基础页面文件
	for name, content := range map[string]string{
		"index.html":    "<html>index</html>",
		"evaluate.html": "<html>evaluate</html>",
		"evaluate.txt":  "rsc-evaluate",
		"stat.html":     "<html>stat</html>",
		"stat.txt":      "rsc-stat",
	} {
		_ = os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
	}
	// 需要 _next 让 detectBuildMode 识别为 export 模式
	_ = os.MkdirAll(filepath.Join(tmpDir, "_next", "static", "chunks"), 0755)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupStaticFileServer(router, StaticConfig{Root: tmpDir})

	tests := []struct {
		path   string
		code   int
		expect string
	}{
		{"/", 200, "index"},
		{"/evaluate", 200, "evaluate"},
		{"/evaluate.txt", 200, "rsc-evaluate"},
		{"/stat", 200, "stat"},
		{"/stat.txt", 200, "rsc-stat"},
		{"/notexist", 404, "404"},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", tt.path, nil)
		router.ServeHTTP(rec, req)
		if rec.Code != tt.code {
			t.Errorf("GET %s want %d got %d", tt.path, tt.code, rec.Code)
			continue
		}
		if tt.expect != "" && !strings.Contains(rec.Body.String(), tt.expect) {
			t.Errorf("GET %s body missing %q got %q", tt.path, tt.expect, rec.Body.String())
		}
	}
}

// TestSetupStaticFileServer_JsonViewerAndRootAssets 守护 /json-viewer 页面路由与
// 根级静态资源（public/ 复制到产物根目录的文件）可访问。
// 历史缺口：json-viewer 未注册路由、json4matter.html 等不在 _next//static/ 下，
// 导致前端导航的 JSON 查看器 404。
func TestSetupStaticFileServer_JsonViewerAndRootAssets(t *testing.T) {
	tmpDir := t.TempDir()

	// 页面产物
	for name, content := range map[string]string{
		"index.html":       "<html>index</html>",
		"json-viewer.html": "<html>json-viewer</html>",
		"json-viewer.txt":  "rsc-json-viewer",
	} {
		_ = os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
	}
	// 根级静态资源（对应 public/ 复制到 out/ 根目录）
	_ = os.WriteFile(filepath.Join(tmpDir, "json4matter.html"), []byte("<html>viewer</html>"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "manifest.json"), []byte(`{"name":"namer"}`), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "sw.js"), []byte("// sw"), 0644)
	// 根级目录
	_ = os.MkdirAll(filepath.Join(tmpDir, "gate-data"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "gate-data", "a.json"), []byte("{}"), 0644)
	// 需要 _next 让 detectBuildMode 识别为 export 模式
	_ = os.MkdirAll(filepath.Join(tmpDir, "_next", "static", "chunks"), 0755)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupStaticFileServer(router, StaticConfig{Root: tmpDir})

	tests := []struct {
		path   string
		code   int
		expect string
	}{
		{"/json-viewer", 200, "json-viewer"},
		{"/json-viewer.txt", 200, "rsc-json-viewer"},
		{"/json4matter.html", 200, "viewer"},
		{"/manifest.json", 200, "namer"},
		{"/sw.js", 200, "// sw"},
		{"/gate-data/a.json", 200, "{}"},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", tt.path, nil)
		router.ServeHTTP(rec, req)
		if rec.Code != tt.code {
			t.Errorf("GET %s want %d got %d", tt.path, tt.code, rec.Code)
			continue
		}
		if tt.expect != "" && !strings.Contains(rec.Body.String(), tt.expect) {
			t.Errorf("GET %s body missing %q got %q", tt.path, tt.expect, rec.Body.String())
		}
	}
}
