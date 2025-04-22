// services/strategy-service/internal/middleware/auth.go
package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserClient defines the interface for user service client
type UserClient interface {
	ValidateUserAccess(ctx context.Context, userID int, token string) (bool, error)
}

// AuthMiddleware authenticates requests against the user service
func AuthMiddleware(userClient UserClient, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		logger.Info("Auth request received",
			zap.String("path", c.Request.URL.Path),
			zap.String("auth_header_exists", strconv.FormatBool(authHeader != "")))

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			logger.Warn("Invalid authorization format",
				zap.String("auth_header", authHeader))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		// Extract token
		token := headerParts[1]
		// Only log a portion of the token for security reasons
		tokenPreview := token
		if len(token) > 10 {
			tokenPreview = token[:10] + "..."
		}
		logger.Info("Token received", zap.String("token_preview", tokenPreview))

		// Parse token to get user ID and role (JWT validation happens in extractUserInfoFromToken)
		userId, userRole, err := extractUserInfoFromToken(token)
		if err != nil {
			logger.Error("Failed to extract user info from token",
				zap.Error(err),
				zap.String("token_preview", tokenPreview))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		logger.Info("User info extracted from token",
			zap.Int("extracted_userID", userId),
			zap.String("extracted_userRole", userRole))

		// Set user ID and role in context
		c.Set("userID", userId)
		c.Set("userRole", userRole)
		c.Next()
	}
}

// RequireRole checks if the user has the specified role
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the user is authenticated
		_, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Get the role directly from the context (set by AuthMiddleware)
		userRole, exists := c.Get("userRole")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role information missing"})
			c.Abort()
			return
		}

		// Check if user has the required role
		if userRole.(string) != requiredRole {
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

// extractUserInfoFromToken extracts both user ID and role from a JWT token
func extractUserInfoFromToken(token string) (int, string, error) {
	// Split the token into its parts (header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, "", fmt.Errorf("invalid token format")
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
			return 0, "", fmt.Errorf("failed to decode payload: %w", err)
		}
	}

	// Parse the JSON payload
	var claims struct {
		Sub  int    `json:"sub"`
		Type string `json:"type"`
		Role string `json:"role"`
	}

	if err := json.Unmarshal(decodedPayload, &claims); err != nil {
		return 0, "", fmt.Errorf("failed to parse token payload: %w", err)
	}

	// Check if this is an access token
	if claims.Type != "access" && claims.Type != "" {
		return 0, "", fmt.Errorf("not an access token")
	}

	// If role is empty, default to "user"
	if claims.Role == "" {
		claims.Role = "user"
	}

	return claims.Sub, claims.Role, nil
}
