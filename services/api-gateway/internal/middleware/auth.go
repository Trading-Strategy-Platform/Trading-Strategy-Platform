package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/yourorg/trading-platform/shared/go/auth"
	"github.com/yourorg/trading-platform/shared/go/response"
	"go.uber.org/zap"
)

// AuthConfig contains authentication configuration
type AuthConfig struct {
	JWTSecret            string
	ExcludedPaths        []string
	PublicPaths          []string
	AdminRequiredPaths   []string
	EnableAuthentication bool
}

// Map to store paths that require admin access
var adminPaths map[string]bool

// Initialize admin paths map based on configuration
func initAdminPaths(paths []string) {
	adminPaths = make(map[string]bool)
	for _, path := range paths {
		// Remove wildcard for exact matches
		if path[len(path)-1] == '*' {
			path = path[:len(path)-1]
		}
		adminPaths[path] = true
	}
}

// AuthMiddleware creates middleware to authenticate requests
func AuthMiddleware(config AuthConfig, logger *zap.Logger) gin.HandlerFunc {
	// Create path map for quick lookup
	excludedPaths := make(map[string]bool)
	for _, path := range config.ExcludedPaths {
		excludedPaths[path] = true
	}

	publicPaths := make(map[string]bool)
	for _, path := range config.PublicPaths {
		publicPaths[path] = true
	}

	adminPaths := make(map[string]bool)
	for _, path := range config.AdminRequiredPaths {
		adminPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip authentication for paths that don't require it
		path := c.Request.URL.Path
		if isExcludedPath(path, config.ExcludedPaths) {
			c.Next()
			return
		}

		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "No authorization token provided")
			c.Abort()
			return
		}

		// Extract token from Bearer format
		token := extractToken(authHeader)
		if token == "" {
			response.Unauthorized(c, "Invalid authorization format")
			c.Abort()
			return
		}

		// Validate token
		claims, err := auth.ValidateToken(token, config.JWTSecret)
		if err != nil {
			logger.Debug("Token validation failed", zap.Error(err))
			response.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Set("token", token)

		// Check admin role for admin paths
		if adminPaths[path] {
			role, exists := c.Get("userRole")
			if !exists || role.(string) != "admin" {
				response.Forbidden(c, "Admin access required")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// Helper function to extract token from Authorization header
func extractToken(authHeader string) string {
	// Check if the header starts with "Bearer "
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

// Helper function to check if a path is in the excluded paths list
func isExcludedPath(path string, excludedPaths []string) bool {
	for _, excludedPath := range excludedPaths {
		if path == excludedPath {
			return true
		}
		// Support wildcard paths
		if len(excludedPath) > 0 && excludedPath[len(excludedPath)-1] == '*' {
			if len(path) >= len(excludedPath)-1 && path[:len(excludedPath)-1] == excludedPath[:len(excludedPath)-1] {
				return true
			}
		}
	}
	return false
}
