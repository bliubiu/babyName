package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"name/internal/infrastructure/logger"
)

func ZapLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		cost := time.Since(start)

		fields := []logger.Field{
			logger.String("method", c.Request.Method),
			logger.String("path", path),
			logger.String("query", query),
			logger.Int("status", c.Writer.Status()),
			logger.Duration("cost", cost),
			logger.String("ip", c.ClientIP()),
			logger.String("user-agent", c.Request.UserAgent()),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, logger.String("errors", c.Errors.String()))
		}

		if c.Writer.Status() >= 500 {
			logger.Error("Server Error", fields...)
		} else if c.Writer.Status() >= 400 {
			logger.Warn("Client Error", fields...)
		} else {
			logger.Info("Request", fields...)
		}
	}
}

func ZapRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				fields := []logger.Field{
					logger.String("method", c.Request.Method),
					logger.String("path", c.Request.URL.Path),
					logger.String("ip", c.ClientIP()),
					logger.Any("error", err),
				}
				logger.Error("Panic Recovered", fields...)
				c.AbortWithStatusJSON(500, gin.H{
					"code":    500,
					"success": false,
					"message": "服务器内部错误",
				})
			}
		}()
		c.Next()
	}
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = c.GetHeader("X-Requestid")
		}
		// 客户端未提供时自动生成，确保日志可关联请求
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		// 回写响应头，便于客户端排查问题
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

// CORSConfig CORS 配置
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
}

// defaultCORSOrigins 默认允许的来源：本机同源的两个常用端口。
//
// 早前这里是 "*" 且 AllowCredentials=true，中间件会回显请求方的任意 Origin，
// 效果等于「任意站点 + 带凭据」跨域读取——用户访问任何恶意站点都能读到他的
// 起名历史与收藏。默认收敛为本地白名单；正式部署需要多域名时用
// NAMER_CORS_ORIGINS（逗号分隔）显式列出，不要退回通配。
var defaultCORSOrigins = []string{
	"http://localhost:8080",
	"http://127.0.0.1:8080",
	"http://localhost:3000",
	"http://127.0.0.1:3000",
}

// DefaultCORSConfig 默认 CORS 配置。
//
// 允许的来源可用环境变量 NAMER_CORS_ORIGINS 覆盖（逗号分隔），未设置时用本地白名单。
func DefaultCORSConfig() CORSConfig {
	origins := defaultCORSOrigins
	if env := strings.TrimSpace(os.Getenv("NAMER_CORS_ORIGINS")); env != "" {
		origins = nil
		for _, part := range strings.Split(env, ",") {
			if o := strings.TrimSpace(part); o != "" {
				origins = append(origins, o)
			}
		}
	}
	return CORSConfig{
		AllowOrigins:     origins,
		AllowMethods:     []string{"POST", "OPTIONS", "GET", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "origin", "Cache-Control", "X-Requested-With", "X-Request-ID"},
		AllowCredentials: true,
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return CORSMiddlewareWithConfig(DefaultCORSConfig())
}

func CORSMiddlewareWithConfig(config CORSConfig) gin.HandlerFunc {
	defaultAllowMethods := "POST, OPTIONS, GET, PUT, DELETE, PATCH"
	if len(config.AllowMethods) > 0 {
		methods := ""
		for i, m := range config.AllowMethods {
			if i > 0 {
				methods += ", "
			}
			methods += m
		}
		defaultAllowMethods = methods
	}

	defaultAllowHeaders := "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID"
	if len(config.AllowHeaders) > 0 {
		headers := ""
		for i, h := range config.AllowHeaders {
			if i > 0 {
				headers += ", "
			}
			headers += h
		}
		defaultAllowHeaders = headers
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowOrigin := ""

		// 计算允许的 Origin：
		// - 配置包含 "*"：一律使用通配符 *。通配 + 凭据本是 docs/28 W1 的
		//   漏洞形态（任意源可带凭据跨域读写），此处不再为它开「回显任意
		//   Origin」的口子——通配模式下凭据响应头一律不发送（下方 ACAC 分支）。
		// - 配置为白名单：仅匹配时回显具体 Origin
		wildcard := false
		for _, o := range config.AllowOrigins {
			if o == "*" {
				wildcard = true
				break
			}
		}

		if wildcard {
			allowOrigin = "*"
		} else {
			for _, o := range config.AllowOrigins {
				if o == origin && origin != "" {
					allowOrigin = origin
					break
				}
			}
		}

		if allowOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		}
		// 仅在确实放行了具体 Origin 时才声明允许凭据：
		// 无 ACAO 时发 ACAC 属规范不符；ACAO:* 与 ACAC:true 同发违反 W3C
		// （浏览器会忽略，但会误导运维以为凭据跨域可用，docs/29 P3#7）
		if allowOrigin != "" && allowOrigin != "*" && config.AllowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			c.Writer.Header().Del("Access-Control-Allow-Credentials")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", defaultAllowHeaders)
		c.Writer.Header().Set("Access-Control-Allow-Methods", defaultAllowMethods)
		// 配合 Allow-Credentials 时，Vary 必须包含 Origin，避免 CDN/代理缓存错乱
		c.Writer.Header().Add("Vary", "Origin")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RateLimiter 令牌桶限流器
type RateLimiter struct {
	mu           sync.Mutex
	tokens       float64
	capacity     float64
	rate         float64
	lastRefilled time.Time
}

// NewRateLimiter 创建一个新的限流器
func NewRateLimiter(capacity, rate float64) *RateLimiter {
	return &RateLimiter{
		capacity:     capacity,
		tokens:       capacity,
		rate:         rate,
		lastRefilled: time.Now(),
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	duration := now.Sub(rl.lastRefilled)

	newTokens := duration.Seconds() * rl.rate
	rl.tokens = min(rl.tokens+newTokens, rl.capacity)
	rl.lastRefilled = now

	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}

	return false
}

// IPRateLimiter IP级别限流器
type IPRateLimiter struct {
	mu             sync.RWMutex
	limiters       map[string]*RateLimiter
	defaultLimiter *RateLimiter // 超过 IP 上限时复用的全局限流器
	capacity       float64
	rate           float64
	maxIPs         int // 最大 IP 记录数，防止内存耗尽
	cleanupInt     time.Duration
	stopCh         chan struct{}
}

// NewIPRateLimiter 创建IP级别限流器
func NewIPRateLimiter(capacity, rate float64) *IPRateLimiter {
	return NewIPRateLimiterWithMaxIPs(capacity, rate, 10000)
}

// NewIPRateLimiterWithMaxIPs 创建带 IP 上限的限流器
func NewIPRateLimiterWithMaxIPs(capacity, rate float64, maxIPs int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		limiters:       make(map[string]*RateLimiter),
		defaultLimiter: NewRateLimiter(capacity, rate),
		capacity:       capacity,
		rate:           rate,
		maxIPs:         maxIPs,
		cleanupInt:     5 * time.Minute,
		stopCh:         make(chan struct{}),
	}
	go limiter.cleanup()
	return limiter
}

// Stop 停止清理 goroutine
func (il *IPRateLimiter) Stop() {
	close(il.stopCh)
}

// GetLimiter 获取指定IP的限流器
func (il *IPRateLimiter) GetLimiter(ip string) *RateLimiter {
	il.mu.RLock()
	limiter, exists := il.limiters[ip]
	il.mu.RUnlock()

	if exists {
		return limiter
	}

	il.mu.Lock()
	defer il.mu.Unlock()

	if limiter, exists = il.limiters[ip]; exists {
		return limiter
	}

	// 超过上限时，复用全局默认限流器（不新增条目，避免每次请求都新建导致 token 重置）
	if len(il.limiters) >= il.maxIPs {
		return il.defaultLimiter
	}

	limiter = NewRateLimiter(il.capacity, il.rate)
	il.limiters[ip] = limiter
	return limiter
}

// cleanup 定期清理不活跃的限流器
func (il *IPRateLimiter) cleanup() {
	ticker := time.NewTicker(il.cleanupInt)
	defer ticker.Stop()
	for {
		select {
		case <-il.stopCh:
			return
		case <-ticker.C:
			il.mu.Lock()
			now := time.Now()
			for ip, limiter := range il.limiters {
				limiter.mu.Lock()
				if now.Sub(limiter.lastRefilled) > 10*time.Minute {
					delete(il.limiters, ip)
				}
				limiter.mu.Unlock()
			}
			il.mu.Unlock()
		}
	}
}



// IPRateLimitWithLimiter 创建IP级别速率限制中间件并返回限流器实例
func IPRateLimitWithLimiter(capacity, rate float64) (gin.HandlerFunc, *IPRateLimiter) {
	limiter := NewIPRateLimiter(capacity, rate)

	handler := func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.GetLimiter(ip).Allow() {
			logger.Warn("IP rate limit exceeded",
				logger.String("ip", ip),
				logger.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(429, gin.H{
				"code":    429,
				"success": false,
				"message": "请求过于频繁，请稍后再试",
			})
			return
		}

		c.Next()
	}

	return handler, limiter
}

// IPRateLimit IP级别速率限制中间件
func IPRateLimit(capacity, rate float64) gin.HandlerFunc {
	handler, _ := IPRateLimitWithLimiter(capacity, rate)
	return handler
}

// RequestTimeout 请求超时中间件
//
// 语义：把带 deadline 的 context 挂到请求上，让下游（尤其是起名引擎的 N² 枚举分片，
// 其内层循环会检查 ctx.Done）在超时后尽早退出；并在「处理链已返回、且一个字都没写出」
// 时补一个 503。
//
// ⚠ 为什么**不能**在 goroutine 里跑 c.Next()（这是历史实现的缺陷）：
// 一旦超时分支直接返回，gin 的 handleHTTPRequest 就会结束本次处理、把 *gin.Context
// 放回 sync.Pool，而此时那个 goroutine 仍在执行 c.Next()——它会继续改写**同一个**
// Context（c.index / c.Keys / c.Errors / c.Writer），并与从池里取到该对象的**下一个请求**
// 并发读写。后果不是「优雅超时」，而是两个请求的响应被交替写进同一个连接。
// gin 的 Context 明确不支持并发使用，因此这里不 fork goroutine：
// 超时通过 ctx 传播给业务代码，而不是强行中断 handler。
//
// 残留限制：完全不理会 ctx 的 handler 仍会阻塞到 HTTP Server 的 WriteTimeout
// （见 cmd/server/main.go 的 Server 配置）。这是 gin 上下文体用池模型的固有约束，
// 无法在不破坏 Context 复用安全的前提下从中间件强制中断它。
func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if timeout <= 0 {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 替换请求上下文为带超时的上下文
		c.Request = c.Request.WithContext(ctx)

		// 串行执行后续处理链（不 fork goroutine，见上面的说明）
		c.Next()

		// 处理链已返回。若因 deadline 到期而提前退出、且没有任何响应写出，
		// 补一个 503——此时仍处于 gin 的串行流程内，写响应是安全的。
		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			logger.Warn("Request timeout",
				logger.String("method", c.Request.Method),
				logger.String("path", c.Request.URL.Path),
				logger.Duration("timeout", timeout),
			)
			c.AbortWithStatusJSON(503, gin.H{
				"code":    503,
				"success": false,
				"message": "请求超时，请稍后重试",
			})
		}
	}
}

// MaxBodyBytes 请求体大小上限中间件
//
// 背景：此前全仓没有任何 body 上限（handler 直接 ShouldBindJSON 读全量 body），
// 单个大 body 即可造成内存放大；而全局 IP 限流是按「请求数」计量的，
// 挡不住「少请求、大体量」的形态。
//
// 两层防护：
//  1. Content-Length 已知且超限 → 直接 413 中止，不读 body、不进 handler
//     （这样客户端拿到明确的「请求体过大」语义，而不是被反序列化错误误导成参数非法）；
//  2. Content-Length 未知（分块传输 / 无长度头）→ 用 http.MaxBytesReader 兜底，
//     读取超限时下游读 body 会得到错误，不会无界地占内存。
//
// limit <= 0 表示不限制（保留配置能力，便于测试或特殊部署）。
func MaxBodyBytes(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limit <= 0 || c.Request.Body == nil {
			c.Next()
			return
		}

		if c.Request.ContentLength > limit {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"code":    http.StatusRequestEntityTooLarge,
				"success": false,
				"message": "请求体过大，请精简后重试",
			})
			return
		}

		// 长度未知或未超限：包一层有界读取，作为第二道防线
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}
