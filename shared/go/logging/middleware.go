package logging

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Middleware creates a gin middleware for logging HTTP requests
func Middleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Stop timer
		end := time.Now()
		latency := end.Sub(start)

		// Get status
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Log data
		if errorMessage != "" {
			logger.Info("HTTP request",
				zap.String("method", method),
				zap.String("path", path),
				zap.String("query", query),
				zap.Int("status", status),
				zap.Duration("latency", latency),
				zap.String("client_ip", clientIP),
				zap.String("error", errorMessage),
				zap.String("user_agent", c.Request.UserAgent()),
			)
		} else {
			logger.Info("HTTP request",
				zap.String("method", method),
				zap.String("path", path),
				zap.String("query", query),
				zap.Int("status", status),
				zap.Duration("latency", latency),
				zap.String("client_ip", clientIP),
				zap.String("user_agent", c.Request.UserAgent()),
			)
		}
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a new request ID
		requestID := generateRequestID()
		c.Set("RequestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// RecoveryMiddleware recovers from any panics and logs the error
func RecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("client_ip", c.ClientIP()),
				)
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple implementation - replace with UUID for production
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
