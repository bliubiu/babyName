package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/namemaster/backend/internal/infrastructure/logger"
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
