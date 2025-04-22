package handler

import (
	"net/http"
	"strconv"
	"strings"

	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ServiceHandler handles service-to-service requests
type ServiceHandler struct {
	userService *service.UserService
	logger      *zap.Logger
}

// NewServiceHandler creates a new service handler
func NewServiceHandler(userService *service.UserService, logger *zap.Logger) *ServiceHandler {
	return &ServiceHandler{
		userService: userService,
		logger:      logger,
	}
}

// BatchGetUsers handles fetching multiple users by ID for service-to-service communication
// GET /api/v1/service/users/batch?ids=1,2,3
func (h *ServiceHandler) BatchGetUsers(c *gin.Context) {
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
		h.logger.Error("Failed to get users batch", zap.Error(err))
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

// GetUserByID handles fetching a user by ID for service-to-service communication
// GET /api/v1/service/users/:id
func (h *ServiceHandler) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}
