package handler

import (
	"net/http"
	"strconv"
	"strings"

	"services/user-service/internal/model"
	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserHandler handles user-related HTTP requests
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

// BatchGetServiceUsers handles fetching multiple users by ID for service-to-service communication
// GET /api/v1/service/users/batch?ids=1,2,3
func (h *UserHandler) BatchGetServiceUsers(c *gin.Context) {
	// Verify service key
	serviceKey := c.GetHeader("X-Service-Key")
	if serviceKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Service key required"})
		return
	}

	// Validate service key
	valid, err := h.userService.ValidateServiceKey(c.Request.Context(), "strategy-service", serviceKey)
	if err != nil || !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid service key"})
		return
	}

	// Parse ids from query parameter
	idsParam := c.Query("ids")
	if idsParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User IDs required"})
		return
	}

	// Split the comma-separated list
	idStrings := strings.Split(idsParam, ",")

	// Convert to integers
	var ids []int
	for _, idStr := range idStrings {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			h.logger.Warn("Invalid user ID in batch request",
				zap.String("id", idStr),
				zap.Error(err))
			continue // Skip invalid IDs
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid user IDs provided"})
		return
	}

	// Get users by their IDs
	users, err := h.userService.GetUsersByIDs(c.Request.Context(), ids)
	if err != nil {
		h.logger.Error("failed to get users batch", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	// Format the response to include only the needed fields
	type UserResponse struct {
		ID              int    `json:"id"`
		Username        string `json:"username"`
		ProfilePhotoURL string `json:"profile_photo_url,omitempty"`
	}

	result := make([]UserResponse, 0, len(users))
	for _, user := range users {
		result = append(result, UserResponse{
			ID:              user.ID,
			Username:        user.Username,
			ProfilePhotoURL: user.ProfilePhotoURL,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": result})
}
