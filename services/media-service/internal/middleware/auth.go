package middleware

import (
	"net/http"

	"services/media-service/internal/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware creates middleware for service authentication
func AuthMiddleware(cfg *config.Config, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication if disabled
		if !cfg.Auth.Enabled {
			c.Next()
			return
		}

		// Get the service key header
		serviceKey := c.GetHeader("X-Service-Key")
		if serviceKey == "" {
			logger.Warn("Missing service key")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Service key required"})
			c.Abort()
			return
		}

		// Validate the service key
		if serviceKey != cfg.Auth.ServiceKey {
			logger.Warn("Invalid service key", zap.String("provided_key", serviceKey))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid service key"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PublicRoute creates middleware that skips authentication for public routes
func PublicRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Mark the route as public
		c.Set("public", true)
		c.Next()
	}
}

// ConditionalAuth creates middleware that only applies authentication if the route is not public
func ConditionalAuth(cfg *config.Config, logger *zap.Logger) gin.HandlerFunc {
	authMiddleware := AuthMiddleware(cfg, logger)

	return func(c *gin.Context) {
		// Check if the route is marked as public
		_, isPublic := c.Get("public")
		if isPublic {
			c.Next()
			return
		}

		// Apply authentication middleware
		authMiddleware(c)
	}
}
