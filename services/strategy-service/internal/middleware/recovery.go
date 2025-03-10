package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RecoveryMiddleware recovers from any panics and logs the error
func RecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Request failed with panic",
					zap.Any("error", err),
					zap.String("url", c.Request.URL.String()),
					zap.String("method", c.Request.Method))

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("Internal server error: %v", err),
				})
			}
		}()
		c.Next()
	}
}
