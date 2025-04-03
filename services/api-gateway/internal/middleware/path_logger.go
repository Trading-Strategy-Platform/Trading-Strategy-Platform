package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DuplicatePathLogger logs requests with duplicate API version paths
func DuplicatePathLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if strings.Contains(path, "/api/v1/v1/") {
			logger.Warn("Received request with duplicate API version path",
				zap.String("path", path),
				zap.String("method", c.Request.Method),
				zap.String("client_ip", c.ClientIP()),
				zap.String("user_agent", c.Request.UserAgent()))
		}

		c.Next()
	}
}
