package middleware

import (
	"context"
	"fmt"
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

// DefaultCORSConfig 默认 CORS 配置
// 注意：不使用 "*" 通配符与 AllowCredentials: true 同时存在（违反 W3C CORS 规范），
// 而是使用空列表让运行时回显请求的 Origin。
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{},
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
		// - 配置包含 "*" 且未启用凭据：使用通配符 *
		// - 配置包含 "*" 且启用凭据：必须回显具体 Origin（W3C CORS 规范禁止 * 与 credentials 同用）
		// - 配置为白名单：仅匹配时回显具体 Origin
		wildcard := false
		for _, o := range config.AllowOrigins {
			if o == "*" {
				wildcard = true
				break
			}
		}

		if wildcard && !config.AllowCredentials {
			allowOrigin = "*"
		} else if wildcard && config.AllowCredentials {
			// 凭据模式下回显请求 Origin（浏览器要求）
			if origin != "" {
				allowOrigin = origin
			} else {
				allowOrigin = "*"
			}
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
		c.Writer.Header().Set("Access-Control-Allow-Credentials", fmt.Sprintf("%v", config.AllowCredentials))
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
	maxIPs         int           // 最大 IP 记录数，防止内存耗尽
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

// RateLimit 全局速率限制中间件（已废弃，建议使用IPRateLimit）
func RateLimit(capacity, rate float64) gin.HandlerFunc {
	limiter := NewRateLimiter(capacity, rate)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatusJSON(429, gin.H{
				"code":    429,
				"success": false,
				"message": "请求过于频繁，请稍后再试",
			})
			return
		}

		c.Next()
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
// 超过指定时间未完成的请求会被取消并返回 503
// 注意：HTTP Server 层的 WriteTimeout 已提供基准保护，
// 此中间件提供更细粒度的每个请求超时控制，并确保 c.Request.Context() 携带 deadline
func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 替换请求上下文为带超时的上下文
		c.Request = c.Request.WithContext(ctx)

		// 等待完成或超时
		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			// 正常完成
		case <-ctx.Done():
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


