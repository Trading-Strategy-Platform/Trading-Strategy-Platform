package middleware

import (
	"net/http"
	"strings"

	"services/historical-data-service/internal/client"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/trading-platform/shared/go/auth"
	"go.uber.org/zap"
)

// AuthMiddleware creates middleware to authenticate users
func AuthMiddleware(userClient *client.UserClient, logger *zap.Logger, jwtSecret string) gin.HandlerFunc {
	// Use the shared auth middleware if JWT secret is provided
	if jwtSecret != "" {
		return auth.Middleware(auth.Config{
			JWTSecret: jwtSecret,
			Logger:    logger,
		})
	}

	// Fall back to the user service validation if no JWT secret
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token (remove "Bearer " prefix)
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			// No prefix was removed, invalid format
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		// Validate token with User Service
		userID, err := userClient.ValidateToken(c.Request.Context(), token)
		if err != nil {
			logger.Debug("Invalid token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set user ID and token in context
		c.Set("userID", userID)
		c.Set("token", token)
		c.Next()
	}
}

// RequireRole middleware checks if the user has the required role
func RequireRole(userClient *client.UserClient, requiredRoles ...string) gin.HandlerFunc {
	return auth.RequireRole(requiredRoles...)
}

// ServiceAuthMiddleware creates middleware to authenticate service-to-service calls
func ServiceAuthMiddleware(serviceKey string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get service key from header
		headerKey := c.GetHeader("X-Service-Key")
		if headerKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Service key required"})
			c.Abort()
			return
		}

		// Validate service key
		if headerKey != serviceKey {
			logger.Warn("Invalid service key")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid service key"})
			c.Abort()
			return
		}

		// Service is authenticated
		c.Next()
	}
}
