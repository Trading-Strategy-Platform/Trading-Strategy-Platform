package middleware

import (
	"net/http"
	"strings"

	"services/historical-data-service/internal/client"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware creates middleware to authenticate users
func AuthMiddleware(userClient *client.UserClient, logger *zap.Logger) gin.HandlerFunc {
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

		// First, try to extract userID from token to verify format
		userId, err := client.ExtractUserIDFromToken(token)
		if err != nil {
			logger.Debug("Failed to extract user ID from token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		// Validate token with User Service
		validatedUserID, err := userClient.ValidateToken(c.Request.Context(), token)
		if err != nil {
			logger.Debug("Invalid token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Verify the token belongs to the expected user
		if validatedUserID != userId {
			logger.Warn("Token validation failed - userIDs don't match",
				zap.Int("extracted_userID", userId),
				zap.Int("validated_userID", validatedUserID))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user ID and token in context
		c.Set("userID", userId)
		c.Set("token", token)
		c.Next()
	}
}

// RequireRole checks if the user has the specified role
func RequireRole(userClient *client.UserClient, requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Get the authorization token from the request
		token, _ := c.Get("token")
		tokenStr, _ := token.(string)

		// Check if user has the required role
		hasRole, err := userClient.CheckUserRole(c.Request.Context(), userID.(int), requiredRole, tokenStr)
		if err != nil {
			// FALLBACK FOR DEVELOPMENT ONLY
			// If there's an error checking roles, check if this is user ID 1 (admin)
			if userID.(int) == 1 && (requiredRole == "admin" || requiredRole == "user") {
				c.Next()
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify user role"})
			c.Abort()
			return
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
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
