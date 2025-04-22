package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware creates middleware for JWT authentication
func AuthMiddleware(authService *service.AuthService, logger *zap.Logger) gin.HandlerFunc {
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

		// Validate the token
		tokenString := headerParts[1]
		userID, role, err := authService.ValidateToken(tokenString)
		if err != nil {
			logger.Debug("token validation failed", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set user ID and role in context
		c.Set("userID", userID)
		c.Set("userRole", role)
		c.Next()
	}
}

// RequireRole middleware checks if the user has the required role
func RequireRole(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get role from context (set by AuthMiddleware)
		userRole, exists := c.Get("userRole")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Check if user has one of the required roles
		hasRole := false
		for _, role := range requiredRoles {
			if userRole.(string) == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ServiceAuthMiddleware creates middleware for service-to-service authentication
func ServiceAuthMiddleware(expectedKey string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get service key from header
		serviceKey := c.GetHeader("X-Service-Key")
		if serviceKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Service key required"})
			c.Abort()
			return
		}

		// Simple comparison for service key
		if serviceKey != expectedKey {
			logger.Warn("Invalid service key in request",
				zap.String("IP", c.ClientIP()),
				zap.String("Path", c.Request.URL.Path))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid service key"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ExtractToken extracts the JWT token from authorization header
func ExtractToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header required")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization format")
	}

	return parts[1], nil
}
