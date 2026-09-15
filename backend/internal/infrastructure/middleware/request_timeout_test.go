package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// newTimeoutRouter 装配一个只挂 RequestTimeout 的路由，便于各用例复用。
func newTimeoutRouter(timeout time.Duration, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestTimeout(timeout))
	r.GET("/t", handler)
	return r
}

// TestRequestTimeoutNormalPassThrough 未超时时行为与不带该中间件完全一致
func TestRequestTimeoutNormalPassThrough(t *testing.T) {
	r := newTimeoutRouter(time.Second, func(c *gin.Context) {
		c.String(http.StatusOK, "ok-body")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))

	if w.Code != http.StatusOK {
		t.Errorf("状态码 = %d，期望 200", w.Code)
	}
	if w.Body.String() != "ok-body" {
		t.Errorf("响应体 = %q，期望 %q", w.Body.String(), "ok-body")
	}
}

// TestRequestTimeoutDeadlineWithoutWrite 处理链因 ctx 到期而返回且未写响应 → 补 503
//
// 模拟「尊重 ctx 的 handler」：等到 ctx.Done 后直接返回、不写任何响应。
func TestRequestTimeoutDeadlineWithoutWrite(t *testing.T) {
	r := newTimeoutRouter(30*time.Millisecond, func(c *gin.Context) {
		<-c.Request.Context().Done()
		// 故意不写响应：交由中间件补 503
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d，期望 503", w.Code)
	}
	if !strings.Contains(w.Body.String(), "请求超时") {
		t.Errorf("响应体 = %q，期望包含「请求超时」", w.Body.String())
	}
}

// TestRequestTimeoutNoMixedResponse 回归：超时后不得出现「两个响应被拼在一起」
//
// 历史实现把 c.Next() 放进 goroutine，超时分支直接返回，而 handler 仍在写 c.Writer；
// 同时 gin 会把该 *gin.Context 放回复用池交给下一个请求 → 两边交替写同一份响应。
// 用 httptest.Recorder 可以稳定观察到「503 JSON + handler 正文」拼接这种脏数据。
//
// 本用例的 handler 刻意**不理会** ctx（睡过 deadline 再写），这是最坏情况：
// 修复后中间件不会提前返回，因此只可能看到 handler 自己的正文。
func TestRequestTimeoutNoMixedResponse(t *testing.T) {
	r := newTimeoutRouter(10*time.Millisecond, func(c *gin.Context) {
		time.Sleep(40 * time.Millisecond) // 忽略 ctx，睡过 deadline
		c.String(http.StatusOK, "LATE-BODY")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))

	body := w.Body.String()
	hasTimeoutJSON := strings.Contains(body, "请求超时") || strings.Contains(body, `"code":503`)
	hasLateBody := strings.Contains(body, "LATE-BODY")

	if hasTimeoutJSON && hasLateBody {
		t.Fatalf("响应被拼接（超时 JSON 与 handler 正文同时出现）：%q", body)
	}
	if !hasLateBody {
		t.Fatalf("handler 正文丢失，实际响应：code=%d body=%q", w.Code, body)
	}
	if hasTimeoutJSON {
		t.Fatalf("handler 已写出正文时不应再补 503，实际响应：code=%d body=%q", w.Code, body)
	}
}

// TestRequestTimeoutChainFinishesBeforeReturn 回归：中间件返回时处理链必须已结束
//
// 这是「不再 fork goroutine」这一修复的直接断言：中间件函数的返回时刻
// 晚于 handler 的结束时刻。历史实现里两者顺序相反（超时分支先返回）。
func TestRequestTimeoutChainFinishesBeforeReturn(t *testing.T) {
	var handlerFinished atomic.Bool
	var middlewareReturnedAfterHandler atomic.Bool

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestTimeout(10 * time.Millisecond))
	// 在 RequestTimeout 之后插入一个中间件，其函数体在 c.Next() 返回后执行，
	// 即「RequestTimeout 内部 c.Next() 已返回」。用 handlerFinished 判断先后。
	r.Use(func(c *gin.Context) {
		c.Next()
		if handlerFinished.Load() {
			middlewareReturnedAfterHandler.Store(true)
		}
	})
	r.GET("/t", func(c *gin.Context) {
		time.Sleep(30 * time.Millisecond)
		handlerFinished.Store(true)
		c.String(http.StatusOK, "done")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))

	if !middlewareReturnedAfterHandler.Load() {
		t.Error("后续中间件恢复执行时 handler 尚未结束——说明 c.Next() 仍在异步分支中运行")
	}
	if w.Body.String() != "done" {
		t.Errorf("响应体 = %q，期望 %q", w.Body.String(), "done")
	}
}

// TestRequestTimeoutNonPositiveDisables 超时 <= 0 时不做任何干预
func TestRequestTimeoutNonPositiveDisables(t *testing.T) {
	r := newTimeoutRouter(0, func(c *gin.Context) {
		// 该 handler 只挂载 deadline 语义会被跳过；这里验证仍能拿到 ctx
		if _, ok := c.Request.Context().Deadline(); ok {
			t.Error("timeout<=0 时不应设置 deadline")
		}
		c.String(http.StatusOK, "no-timeout")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))

	if w.Code != http.StatusOK || w.Body.String() != "no-timeout" {
		t.Errorf("code=%d body=%q，期望 200/no-timeout", w.Code, w.Body.String())
	}
}

// TestRequestTimeoutSetsDeadline 正常路径下必须把 deadline 传给下游
func TestRequestTimeoutSetsDeadline(t *testing.T) {
	var gotDeadline bool
	r := newTimeoutRouter(time.Second, func(c *gin.Context) {
		_, gotDeadline = c.Request.Context().Deadline()
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/t", nil))

	if !gotDeadline {
		t.Error("下游未拿到带 deadline 的 context")
	}
}

// TestRequestTimeoutUpstreamCancelPropagates 上游取消（客户端断开）同样体现在 ctx 上
func TestRequestTimeoutUpstreamCancelPropagates(t *testing.T) {
	var sawErr atomic.Value
	r := newTimeoutRouter(time.Second, func(c *gin.Context) {
		<-c.Request.Context().Done()
		sawErr.Store(c.Request.Context().Err())
	})

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/t", nil).WithContext(ctx)
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got, _ := sawErr.Load().(error); got != context.Canceled {
		t.Errorf("ctx.Err() = %v，期望 context.Canceled", got)
	}
	// 客户端已断开，中间件不应再写 503（deadline 未到期）
	if w.Code == http.StatusServiceUnavailable {
		t.Errorf("上游取消（非 deadline）时不应补 503，实际 code=%d", w.Code)
	}
}
