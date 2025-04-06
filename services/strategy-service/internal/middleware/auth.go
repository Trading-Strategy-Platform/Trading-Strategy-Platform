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
	CheckUserRole(ctx context.Context, userID int, role string, token string) (bool, error)
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

		// Get the authorization token from the request
		token := extractTokenFromHeader(c.GetHeader("Authorization"))

		// Log that we're checking roles
		logger, _ := zap.NewProduction()
		logger.Info("Checking user role",
			zap.Int("userID", userID.(int)),
			zap.String("role", requiredRole),
			zap.Bool("has_token", token != ""))

		// Get user roles from user service
		hasRole, err := userClient.CheckUserRole(c.Request.Context(), userID.(int), requiredRole, token)
		if err != nil {
			logger.Error("Failed to verify user role", zap.Error(err))

			// FALLBACK FOR DEVELOPMENT ONLY
			// If there's an error checking roles, check if this is user ID 1 (admin)
			if userID.(int) == 1 && (requiredRole == "admin" || requiredRole == "user") {
				logger.Warn("Using fallback admin check", zap.Int("userID", userID.(int)))
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

// extractTokenFromHeader extracts the token from the Authorization header
func extractTokenFromHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		// Validate token with user service
		valid, err := userClient.ValidateUserAccess(c.Request.Context(), userId, token)

		if err != nil {
			// Log the specific error
			logger.Error("Failed to validate token", zap.Error(err), zap.String("token", token[:10]+"..."))

			// Check if it's a connection error
			if strings.Contains(err.Error(), "failed to connect to user service") {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Authentication service unavailable"})
			} else {
				// For development/testing, you might want to allow bypassing auth if user service is down
				if strings.Contains(authHeader, "BYPASS_AUTH_FOR_DEVELOPMENT_ONLY") {
					logger.Warn("Using development bypass for authentication",
						zap.Int("user_id", userId),
						zap.String("path", c.Request.URL.Path))
					c.Set("userID", userId)
					c.Next()
					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication error. Please try again later."})
			}

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
	// Split the token into its parts (header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid token format")
	}

	// Decode the payload (the middle part)
	// Add padding if needed
	payload := parts[1]
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", 4-len(payload)%4)
	}

	decodedPayload, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		// Try with RawURLEncoding if standard URLEncoding fails
		decodedPayload, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return 0, fmt.Errorf("failed to decode payload: %w", err)
		}
	}

	// Parse the JSON payload
	var claims struct {
		Sub  int    `json:"sub"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal(decodedPayload, &claims); err != nil {
		return 0, fmt.Errorf("failed to parse token payload: %w", err)
	}

	// Check if this is an access token
	if claims.Type != "access" && claims.Type != "" {
		return 0, fmt.Errorf("not an access token")
	}

	return claims.Sub, nil
}
