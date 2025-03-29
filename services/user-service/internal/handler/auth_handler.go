package handler

import (
	"net/http"

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
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	err := h.authService.Logout(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("logout failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

// RefreshToken handles refreshing access tokens
// POST /api/v1/auth/refresh-token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
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
