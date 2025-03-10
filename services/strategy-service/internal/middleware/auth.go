// services/strategy-service/internal/middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/trading-platform/shared/go/auth"
	"go.uber.org/zap"
)

// UserClient defines the interface for user service client
type UserClient interface {
	ValidateUserAccess(ctx context.Context, userID int, token string) (bool, error)
}

// AuthMiddleware authenticates requests against the user service
func AuthMiddleware(userClient UserClient, logger *zap.Logger, jwtSecret string) gin.HandlerFunc {
	// Use the shared auth middleware if JWT secret is provided
	if jwtSecret != "" {
		return auth.Middleware(auth.Config{
			JWTSecret: jwtSecret,
			Logger:    logger,
		})
	}

	// Fall back to the user service validation if no JWT secret
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		// Extract token
		token := headerParts[1]

		// Parse token to get user ID
		claims, err := auth.ValidateToken(token, jwtSecret)
		if err != nil {
			logger.Debug("Failed to extract user ID from token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Validate token with user service as a double-check
		valid, err := userClient.ValidateUserAccess(c.Request.Context(), claims.UserID, token)
		if err != nil {
			logger.Error("Failed to validate token", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication service unavailable"})
			c.Abort()
			return
		}

		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set user ID and role in context
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Next()
	}
}

// RequireRole middleware checks if the user has the required role
func RequireRole(requiredRoles ...string) gin.HandlerFunc {
	return auth.RequireRole(requiredRoles...)
}
