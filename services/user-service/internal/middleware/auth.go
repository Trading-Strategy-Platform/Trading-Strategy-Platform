package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/yourorg/trading-platform/shared/go/auth"
	sharedErrors "github.com/yourorg/trading-platform/shared/go/errors"
	"go.uber.org/zap"
)

// AuthMiddleware creates middleware for JWT authentication
func AuthMiddleware(authService *service.AuthService, logger *zap.Logger) gin.HandlerFunc {
	return auth.Middleware(auth.Config{
		JWTSecret: authService.GetJWTSecret(),
		Logger:    logger,
	})
}

// RequireRole middleware checks if the user has the required role
func RequireRole(requiredRoles ...string) gin.HandlerFunc {
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

// Auth middleware validates JWT tokens
func Auth(jwtSecret string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			err := sharedErrors.NewAuthError("Authorization header is required")
			c.Error(err)
			c.Abort()
			return
		}

		// Extract token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			err := sharedErrors.NewAuthError("Invalid authorization format, expected 'Bearer {token}'")
			c.Error(err)
			c.Abort()
			return
		}

		token := parts[1]

		// Validate JWT token
		claims, err := validateToken(token, jwtSecret)
		if err != nil {
			logger.Debug("Failed to validate token", zap.Error(err))
			authErr := sharedErrors.NewAuthError("Invalid or expired token")
			c.Error(authErr)
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AdminRequired middleware ensures the user has admin role
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			err := sharedErrors.NewAuthError("User role not found in context")
			c.Error(err)
			c.Abort()
			return
		}

		// Check if user has admin role
		if role != "admin" {
			err := sharedErrors.NewAuthError("Admin privileges required")
			c.Error(err)
			c.Abort()
			return
		}

		c.Next()
	}
}

// Claims represents JWT claims with standard claims embedded
type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.StandardClaims
}

// validateToken validates a JWT token and returns the claims
func validateToken(tokenString string, jwtSecret string) (*Claims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Return the secret key
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// Validate the token and extract claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if time.Now().Unix() > claims.ExpiresAt {
			return nil, sharedErrors.NewAuthError("Token has expired")
		}
		return claims, nil
	}

	return nil, sharedErrors.NewAuthError("Invalid token")
}
