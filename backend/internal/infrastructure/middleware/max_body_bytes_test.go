package middleware

// max_body_bytes_test.go — 请求体大小上限中间件回归
//
// 背景（docs/24 P1-4）：全仓无 http.MaxBytesReader / LimitReader，
// handler 直接 ShouldBindJSON 读全量 body，单个大 body 即可造成内存放大。
//
// 本文件断言：
//   - Content-Length 超限 → 413 + 中文提示，且 handler 完全不执行（不读 body）
//   - 未超限 → 行为与不加中间件完全一致（body 完整可读）
//   - Content-Length 未知（分块/无长度头）→ 由 MaxBytesReader 兜底，读超限即报错
//   - limit <= 0 → 不限制（保留配置能力）
//   - 无 body 的请求不受影响

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// newBodyLimitRouter 装配一个只挂 MaxBodyBytes 的路由；handlerInvoked 记录处理链是否进入
func newBodyLimitRouter(limit int64, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodyBytes(limit))
	r.POST("/t", handler)
	return r
}

// TestMaxBodyBytesRejectsOversizedContentLength 长度已知且超限 → 413，处理链不执行
func TestMaxBodyBytesRejectsOversizedContentLength(t *testing.T) {
	invoked := false
	r := newBodyLimitRouter(16, func(c *gin.Context) {
		invoked = true
		c.String(http.StatusOK, "should-not-happen")
	})

	req := httptest.NewRequest(http.MethodPost, "/t", strings.NewReader(strings.Repeat("a", 64)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("状态码 = %d，期望 413", w.Code)
	}
	if !strings.Contains(w.Body.String(), "请求体过大") {
		t.Errorf("响应体 = %q，期望包含「请求体过大」", w.Body.String())
	}
	if invoked {
		t.Error("超限请求不应进入处理链（body 不应被读取/反序列化）")
	}
}

// TestMaxBodyBytesPassesNormalRequest 未超限时行为与不带中间件一致
func TestMaxBodyBytesPassesNormalRequest(t *testing.T) {
	const body = `{"surname":"张","gender":"male"}`
	r := newBodyLimitRouter(1<<20, func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Errorf("读取请求体失败: %v", err)
		}
		c.String(http.StatusOK, string(raw))
	})

	req := httptest.NewRequest(http.MethodPost, "/t", strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 200", w.Code)
	}
	if w.Body.String() != body {
		t.Errorf("响应体 = %q，期望 %q（body 应完整可读）", w.Body.String(), body)
	}
}

// TestMaxBodyBytesLimitsUnknownLengthBody Content-Length 未知时由 MaxBytesReader 兜底
//
// 模拟分块传输：长度头缺失，中间件无法预判，必须在「读」这一侧有界。
func TestMaxBodyBytesLimitsUnknownLengthBody(t *testing.T) {
	const limit = 16
	readErr := error(nil)
	readLen := 0

	r := newBodyLimitRouter(limit, func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		readErr, readLen = err, len(raw)
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/t", strings.NewReader(strings.Repeat("a", 256)))
	req.ContentLength = -1 // 长度未知
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if readErr == nil {
		t.Fatal("长度未知且超限时，读取 body 应当报错（否则内存无界）")
	}
	// MaxBytesReader 最多允许读出 limit 个字节，然后返回错误
	if readLen > limit {
		t.Errorf("读出字节数 = %d，超过上限 %d", readLen, limit)
	}
}

// TestMaxBodyBytesDisabledWhenNonPositive limit <= 0 表示不限制
func TestMaxBodyBytesDisabledWhenNonPositive(t *testing.T) {
	body := strings.Repeat("a", 4096)
	r := newBodyLimitRouter(0, func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Errorf("不限制时读取失败: %v", err)
		}
		c.String(http.StatusOK, "%d", len(raw))
	})

	req := httptest.NewRequest(http.MethodPost, "/t", strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 200", w.Code)
	}
	if w.Body.String() != "4096" {
		t.Errorf("读出长度 = %s，期望 4096", w.Body.String())
	}
}

// TestMaxBodyBytesNilBody 无 body 的请求（如 GET）不受影响
func TestMaxBodyBytesNilBody(t *testing.T) {
	r := newBodyLimitRouter(16, func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/t", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 200", w.Code)
	}
}
