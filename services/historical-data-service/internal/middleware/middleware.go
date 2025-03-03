package middleware

import (
	"net/http"
	"strings"
	"time"

	"services/historical-data-service/internal/client"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger creates a middleware for logging HTTP requests
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log after the request is processed
		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		userID, _ := c.Get("userID")

		if query != "" {
			path = path + "?" + query
		}

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("client_ip", clientIP),
			zap.Duration("latency", latency),
		}

		if userID != nil {
			fields = append(fields, zap.Int("user_id", userID.(int)))
		}

		// Log with appropriate level based on status code
		if status >= 500 {
			logger.Error("Server error", fields...)
		} else if status >= 400 {
			logger.Warn("Client error", fields...)
		} else {
			logger.Info("Request completed", fields...)
		}
	}
}

// AuthMiddleware creates middleware for authenticating users
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

		// Validate token with User Service
		userID, err := userClient.ValidateToken(c.Request.Context(), token)
		if err != nil {
			logger.Debug("Invalid token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set user ID in context
		c.Set("userID", userID)
		c.Set("token", token)
		c.Next()
	}
}

// ServiceAuthMiddleware creates middleware for authenticating service-to-service requests
func ServiceAuthMiddleware(expectedKey string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the service key header
		serviceKey := c.GetHeader("X-Service-Key")
		if serviceKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Service authentication required"})
			c.Abort()
			return
		}

		// Validate service key
		if serviceKey != expectedKey {
			logger.Warn("Invalid service key received", zap.String("received_key", serviceKey))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid service key"})
			c.Abort()
			return
		}

		// Service is authenticated
		c.Next()
	}
}
