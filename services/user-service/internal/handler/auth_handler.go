package handler

import (
	"fmt"
	"net/http"
	"strings"

	"services/user-service/internal/model"
	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService *service.AuthService
	logger      *zap.Logger
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register handles user registration
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var request model.UserCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.authService.Register(c.Request.Context(), &request)
	if err != nil {
		h.logger.Error("registration failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// Login handles user login
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var request model.UserLogin
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.authService.Login(c.Request.Context(), &request)
	if err != nil {
		h.logger.Debug("login failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Logout handles user logout
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get the refresh token from the request body
	var request struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token is required"})
		return
	}

	// Invalidate the refresh token
	err := h.authService.Logout(c.Request.Context(), request.RefreshToken)
	if err != nil {
		h.logger.Error("logout failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

// LogoutAll handles logging out all devices for a user
// POST /api/v1/auth/logout-all
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	count, err := h.authService.LogoutAll(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("logout all failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout from all devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Successfully logged out from all devices",
		"session_count": count,
	})
}

// Validate handles token validation
// GET /api/v1/auth/validate
func (h *AuthHandler) Validate(c *gin.Context) {
	// The AuthMiddleware has already validated the token and set the userID and userRole in the context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Get the role from context (set by AuthMiddleware)
	userRole, exists := c.Get("userRole")
	if !exists {
		// Default to user if role not found (should not happen with updated middleware)
		userRole = "user"
	}

	// Set headers for Nginx auth_request module
	c.Header("X-User-ID", fmt.Sprintf("%d", userID))
	c.Header("X-User-Role", userRole.(string))

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"user_id": userID,
		"role":    userRole,
	})
}

// RefreshToken handles refreshing access tokens
// POST /api/v1/auth/refresh-token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var request model.RefreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.authService.RefreshToken(c.Request.Context(), request.RefreshToken)
	if err != nil {
		h.logger.Debug("token refresh failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// AuthHeader extracts the token from the Authorization header
func AuthHeader(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", nil
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", nil
	}

	return parts[1], nil
}
