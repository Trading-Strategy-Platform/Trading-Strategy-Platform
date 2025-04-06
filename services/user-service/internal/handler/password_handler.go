package handler

import (
	"net/http"

	"services/user-service/internal/model"
	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PasswordHandler handles password-related operations
type PasswordHandler struct {
	authService *service.AuthService
	logger      *zap.Logger
}

// NewPasswordHandler creates a new password handler
func NewPasswordHandler(authService *service.AuthService, logger *zap.Logger) *PasswordHandler {
	return &PasswordHandler{
		authService: authService,
		logger:      logger,
	}
}

// ChangePassword handles changing user's password
// PUT /api/v1/users/me/password
func (h *PasswordHandler) ChangePassword(c *gin.Context) {
	var request model.UserChangePassword
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")
	err := h.authService.ChangePassword(c.Request.Context(), userID.(int), &request)
	if err != nil {
		h.logger.Debug("password change failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}
