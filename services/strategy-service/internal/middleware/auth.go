package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserClient defines the interface for user service client
type UserClient interface {
	ValidateUserAccess(ctx context.Context, userID int, token string) (bool, error)
	CheckUserRole(ctx context.Context, userID int, role string, token string) (bool, error)
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

		// Parse token to get user ID (JWT validation happens in user service)
		userId, err := extractUserIdFromToken(token)
		if err != nil {
			logger.Error("Failed to extract user ID from token",
				zap.Error(err),
				zap.String("token_preview", tokenPreview))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		logger.Info("User ID extracted from token", zap.Int("extracted_userID", userId))

		// DIRECT VALIDATION - Bypass UserClient.ValidateUserAccess
		// -----------------------------------------------------
		// Create a direct HTTP request to the user service
		baseURL := "http://user-service:8083" // Should match your config
		url := fmt.Sprintf("%s/api/v1/auth/validate", baseURL)

		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
		if err != nil {
			logger.Error("Failed to create validation request", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication error"})
			c.Abort()
			return
		}

		// Add headers
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("X-Service-Key", "strategy-service-key")

		// Send request
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)

		if err != nil {
			logger.Error("Failed to validate token with User Service",
				zap.Error(err),
				zap.String("url", url))

			// IMPORTANT: For debugging, allow requests through with the extracted userId
			logger.Warn("⚠️ BYPASSING AUTH FOR DEBUGGING - using extracted user ID",
				zap.Int("userId", userId))
			c.Set("userID", userId)
			c.Next()
			return
		}
		defer resp.Body.Close()

		// Log response for debugging
		logger.Info("User service validation response",
			zap.Int("status", resp.StatusCode),
			zap.String("method", req.Method),
			zap.String("url", url))

		// Check response status
		if resp.StatusCode == http.StatusOK {
			var response struct {
				Valid  bool `json:"valid"`
				UserID int  `json:"user_id"`
			}

			err = json.NewDecoder(resp.Body).Decode(&response)
			if err != nil {
				logger.Error("Failed to decode validation response", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication error"})
				c.Abort()
				return
			}

			logger.Info("Token validation result",
				zap.Bool("valid", response.Valid),
				zap.Int("response_userID", response.UserID),
				zap.Int("extracted_userID", userId),
				zap.Bool("userIDs_match", response.UserID == userId))

			// Verify the token belongs to the expected user
			if !response.Valid || response.UserID != userId {
				logger.Warn("Token validation failed",
					zap.Bool("valid", response.Valid),
					zap.Int("response_userID", response.UserID),
					zap.Int("extracted_userID", userId))
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
				c.Abort()
				return
			}
		} else if resp.StatusCode == http.StatusUnauthorized {
			logger.Warn("Token unauthorized by user service",
				zap.Int("status", resp.StatusCode),
				zap.Int("extracted_userID", userId))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		} else {
			// For any other status, log it and return a generic error
			bodyBytes, _ := io.ReadAll(resp.Body)
			logger.Error("Unexpected response from user service",
				zap.Int("status", resp.StatusCode),
				zap.String("body", string(bodyBytes)),
				zap.Int("extracted_userID", userId))

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication service error"})
			c.Abort()
			return
		}

		// Set user ID in context
		logger.Info("✅ Authentication successful - setting user ID in context",
			zap.Int("userID", userId))
		c.Set("userID", userId)
		c.Next()
	}
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
