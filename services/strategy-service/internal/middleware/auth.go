// services/strategy-service/internal/middleware/auth.go
package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserClient defines the interface for user service client
type UserClient interface {
	ValidateUserAccess(ctx context.Context, userID int, token string) (bool, error)
	CheckUserRole(ctx context.Context, userID int, role string) (bool, error)
}

// RequireRole checks if the user has the specified role
func RequireRole(userClient UserClient, requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Get user roles from user service
		hasRole, err := userClient.CheckUserRole(c.Request.Context(), userID.(int), requiredRole)
		if err != nil {
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

// AuthMiddleware authenticates requests against the user service
func AuthMiddleware(userClient UserClient, logger *zap.Logger) gin.HandlerFunc {
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

		// Parse token to get user ID (JWT validation happens in user service)
		userId, err := extractUserIdFromToken(token)
		if err != nil {
			logger.Debug("Failed to extract user ID from token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Validate token with user service
		valid, err := userClient.ValidateUserAccess(c.Request.Context(), userId, token)
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

		// Set user ID in context
		c.Set("userID", userId)
		c.Next()
	}
}

// extractUserIdFromToken extracts the user ID from a JWT token
func extractUserIdFromToken(token string) (int, error) {
	// For this example, we'll use a simplified method that doesn't validate the token
	// In a real implementation, you would use a JWT library to verify and parse the token

	// Split the token into its parts (header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid token format")
	}

	// Decode the payload (the middle part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, err
	}

	// Parse the JSON payload
	var claims struct {
		Sub  int    `json:"sub"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, err
	}

	// Check if this is an access token
	if claims.Type != "access" {
		return 0, fmt.Errorf("not an access token")
	}

	return claims.Sub, nil
}
