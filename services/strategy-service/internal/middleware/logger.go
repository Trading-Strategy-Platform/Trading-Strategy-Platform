// services/strategy-service/internal/middleware/logger.go
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/trading-platform/shared/go/logging"
	"go.uber.org/zap"
)

// LoggerMiddleware logs request information
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		statusCode := c.Writer.Status()

		logger.Info("Request processed",
			zap.String("path", path),
			zap.String("method", method),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
		)
	}
}

// NewLogger creates a new logger for the Strategy Service
func NewLogger(config *logging.Config) (*zap.Logger, error) {
	return logging.NewLogger(config)
}
