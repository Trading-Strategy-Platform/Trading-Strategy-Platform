package middleware

import (
	"context"
	"time"

	"services/api-gateway/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MetricsMiddleware creates middleware for tracking API metrics via Kafka
func MetricsMiddleware(kafkaService *service.KafkaService, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Extract user ID if available
		var userID *int
		if id, exists := c.Get("userID"); exists {
			if userIDValue, ok := id.(int); ok {
				userID = &userIDValue
			}
		}

		// Publish metrics asynchronously to not block the response
		go func() {
			ctx := context.Background()
			err := kafkaService.PublishAPIMetric(
				ctx,
				c.Request.URL.Path,
				c.Request.Method,
				c.Writer.Status(),
				duration,
				userID,
			)

			if err != nil {
				logger.Warn("Failed to publish API metrics",
					zap.Error(err),
					zap.String("path", c.Request.URL.Path),
					zap.Int("status", c.Writer.Status()))
			}
		}()
	}
}
