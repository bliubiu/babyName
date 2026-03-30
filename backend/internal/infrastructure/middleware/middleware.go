package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/bliubiu/babyName/internal/infrastructure/logger"
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
			logger.LogError("Server Error", fields...)
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
				logger.LogError("Panic Recovered", fields...)
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
		c.Set("request_id", requestID)
		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

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
	mu         sync.RWMutex
	limiters   map[string]*RateLimiter
	capacity   float64
	rate       float64
	cleanupInt time.Duration
}

// NewIPRateLimiter 创建IP级别限流器
func NewIPRateLimiter(capacity, rate float64) *IPRateLimiter {
	limiter := &IPRateLimiter{
		limiters:   make(map[string]*RateLimiter),
		capacity:   capacity,
		rate:       rate,
		cleanupInt: 5 * time.Minute,
	}
	go limiter.cleanup()
	return limiter
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

	limiter = NewRateLimiter(il.capacity, il.rate)
	il.limiters[ip] = limiter
	return limiter
}

// cleanup 定期清理不活跃的限流器
func (il *IPRateLimiter) cleanup() {
	ticker := time.NewTicker(il.cleanupInt)
	for range ticker.C {
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

// IPRateLimit IP级别速率限制中间件
func IPRateLimit(capacity, rate float64) gin.HandlerFunc {
	limiter := NewIPRateLimiter(capacity, rate)

	return func(c *gin.Context) {
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
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
