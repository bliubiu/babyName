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

func init() {
	_ = logger.Init(&logger.Config{Level: "error", OutputPath: ""})
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
