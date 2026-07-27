package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// newTestIPRateLimiter 创建测试用限流器，自动 Stop 清理 goroutine
func newTestIPRateLimiter(t *testing.T, capacity, rate float64) *IPRateLimiter {
	t.Helper()
	il := NewIPRateLimiter(capacity, rate)
	t.Cleanup(il.Stop)
	return il
}

// --- RateLimiter 单元测试 ---

// TestRateLimiterAllowsWithinCapacity 容量内应全部允许
func TestRateLimiterAllowsWithinCapacity(t *testing.T) {
	rl := NewRateLimiter(3, 0) // 容量3，速率0（不恢复）
	for i := 0; i < 3; i++ {
		if !rl.Allow() {
			t.Fatalf("第 %d 次应允许", i+1)
		}
	}
}

// TestRateLimiterRejectsWhenExhausted 耗尽后应拒绝
func TestRateLimiterRejectsWhenExhausted(t *testing.T) {
	rl := NewRateLimiter(2, 0)
	rl.Allow()
	rl.Allow()
	if rl.Allow() {
		t.Error("耗尽后应拒绝")
	}
}

// TestRateLimiterRefillsOverTime 速率恢复后应重新允许
func TestRateLimiterRefillsOverTime(t *testing.T) {
	rl := NewRateLimiter(1, 1000) // 速率高，快速恢复
	if !rl.Allow() {
		t.Fatal("首次应允许")
	}
	if rl.Allow() {
		t.Fatal("耗尽后应拒绝")
	}
	time.Sleep(20 * time.Millisecond) // 1000/s * 0.02s ≈ 20 令牌
	if !rl.Allow() {
		t.Error("等待后应恢复允许")
	}
}

// --- IPRateLimiter 单元测试 ---

// TestIPRateLimiterIsolatesByIP 不同 IP 应独立限流
func TestIPRateLimiterIsolatesByIP(t *testing.T) {
	il := newTestIPRateLimiter(t, 1, 0)
	il.GetLimiter("1.1.1.1").Allow() // 1.1.1.1 耗尽
	if !il.GetLimiter("2.2.2.2").Allow() {
		t.Error("不同 IP 应独立限流")
	}
}

// TestIPRateLimiterReusesLimiter 相同 IP 应返回同一限流器
func TestIPRateLimiterReusesLimiter(t *testing.T) {
	il := newTestIPRateLimiter(t, 5, 1)
	l1 := il.GetLimiter("1.1.1.1")
	l2 := il.GetLimiter("1.1.1.1")
	if l1 != l2 {
		t.Error("相同 IP 应返回同一限流器实例")
	}
}

// TestIPRateLimiterMaxIPsFallback 超过 IP 上限应复用全局限流器
func TestIPRateLimiterMaxIPsFallback(t *testing.T) {
	il := NewIPRateLimiterWithMaxIPs(5, 1, 2) // 最多2个IP
	t.Cleanup(il.Stop)
	il.GetLimiter("1.1.1.1")
	il.GetLimiter("2.2.2.2")
	// 第3个 IP 超过上限，应复用 defaultLimiter 而非新增
	l3 := il.GetLimiter("3.3.3.3")
	if l3 != il.defaultLimiter {
		t.Error("超过 maxIPs 应复用全局限流器")
	}
}

// --- RequestID 中间件测试 ---

func setupGin(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

// TestRequestIDGeneratesWhenMissing 缺少请求头时自动生成并回写
func TestRequestIDGeneratesWhenMissing(t *testing.T) {
	r := setupGin(t)
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		rid, _ := c.Get("request_id")
		c.String(200, rid.(string))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码期望 200，实际 %d", w.Code)
	}
	rid := w.Body.String()
	if rid == "" {
		t.Error("应自动生成 request_id")
	}
	if w.Header().Get("X-Request-ID") != rid {
		t.Error("响应头应回写 X-Request-ID")
	}
}

// TestRequestIDReusesHeader 携带请求头时应复用
func TestRequestIDReusesHeader(t *testing.T) {
	r := setupGin(t)
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		rid, _ := c.Get("request_id")
		c.String(200, rid.(string))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "my-id-123")
	r.ServeHTTP(w, req)

	if w.Body.String() != "my-id-123" {
		t.Errorf("应复用请求头 request_id，实际 %s", w.Body.String())
	}
	if w.Header().Get("X-Request-ID") != "my-id-123" {
		t.Error("响应头应回写复用的 request_id")
	}
}

// --- CORS 中间件测试 ---

// TestCORSMiddlewareSetsHeaders 普通 GET 应设置凭据与 Vary 头
func TestCORSMiddlewareSetsHeaders(t *testing.T) {
	r := setupGin(t)
	r.Use(CORSMiddleware())
	r.GET("/", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("应设置 Access-Control-Allow-Credentials: true")
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Error("Vary 应包含 Origin")
	}
}

// TestCORSPreflightReturns204 OPTIONS 预检应返回 204
func TestCORSPreflightReturns204(t *testing.T) {
	r := setupGin(t)
	r.Use(CORSMiddleware())
	r.GET("/", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS 预检应返回 204，实际 %d", w.Code)
	}
}
