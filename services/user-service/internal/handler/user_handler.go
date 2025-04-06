package handler

import (
	"net/http"
	"strconv"

	"services/user-service/internal/model"
	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserHandler handles user requests
type UserHandler struct {
	userService *service.UserService
	logger      *zap.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *service.UserService, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// GetCurrentUser handles fetching current user profile
// GET /api/v1/users/me
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("userID")
	user, err := h.userService.GetCurrentUser(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("failed to get current user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user data"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateCurrentUser handles updating current user profile
// PUT /api/v1/users/me
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	var request model.UserUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")
	err := h.userService.Update(c.Request.Context(), userID.(int), &request)
	if err != nil {
		h.logger.Error("failed to update user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Return updated user
	user, err := h.userService.GetCurrentUser(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("failed to get updated user", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ChangePassword handles changing user's password
// PUT /api/v1/users/me/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var request model.UserChangePassword
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")
	err := h.userService.ChangePassword(c.Request.Context(), userID.(int), &request)
	if err != nil {
		h.logger.Debug("password change failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// DeleteCurrentUser handles deleting the current user (deactivating)
// DELETE /api/v1/users/me
func (h *UserHandler) DeleteCurrentUser(c *gin.Context) {
	userID, _ := c.Get("userID")

	err := h.userService.DeleteUser(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("failed to delete user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User successfully deactivated"})
}

// GetUserPreferences handles fetching user preferences
// GET /api/v1/users/me/preferences
func (h *UserHandler) GetUserPreferences(c *gin.Context) {
	userID, _ := c.Get("userID")

	preferences, err := h.userService.GetPreferences(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("failed to get user preferences", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get preferences"})
		return
	}

	c.JSON(http.StatusOK, preferences)
}

// UpdateUserPreferences handles updating user preferences
// PUT /api/v1/users/me/preferences
func (h *UserHandler) UpdateUserPreferences(c *gin.Context) {
	var request model.PreferencesUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")
	err := h.userService.UpdatePreferences(c.Request.Context(), userID.(int), &request)
	if err != nil {
		h.logger.Error("failed to update preferences", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
		return
	}

	// Return updated preferences
	preferences, err := h.userService.GetPreferences(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("failed to get updated preferences", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"message": "Preferences updated successfully"})
		return
	}

	c.JSON(http.StatusOK, preferences)
}

// GetUserByID handles fetching a user by ID (admin only)
// GET /api/v1/admin/users/{id}
func (h *UserHandler) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser handles updating a user (admin only)
// PUT /api/v1/admin/users/{id}
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var request model.UserUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.userService.Update(c.Request.Context(), id, &request)
	if err != nil {
		h.logger.Error("failed to update user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Return updated user
	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to get updated user", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ListUsers handles listing users (admin only)
// GET /api/v1/admin/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	users, total, err := h.userService.ListUsers(c.Request.Context(), page, limit)
	if err != nil {
		h.logger.Error("failed to list users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GET /api/v1/admin/users/:id/roles
func (h *UserHandler) GetUserRoles(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Return the user's role in an array format as expected by the client
	c.JSON(http.StatusOK, gin.H{
		"roles": []string{user.Role},
	})
}
