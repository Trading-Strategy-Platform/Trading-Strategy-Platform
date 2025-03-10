package handler

import (
	"net/http"
	"strconv"

	"services/user-service/internal/client"
	"services/user-service/internal/model"
	"services/user-service/internal/service"

	sharedModel "github.com/yourorg/trading-platform/shared/go/model"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/trading-platform/shared/go/response"
	"go.uber.org/zap"
)

// UserHandler handles user requests
type UserHandler struct {
	userService     *service.UserService
	validatorClient *client.ValidatorClient
	logger          *zap.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *service.UserService, validatorClient *client.ValidatorClient, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userService:     userService,
		validatorClient: validatorClient,
		logger:          logger,
	}
}

// GetCurrentUser handles fetching current user profile
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
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "")
		return
	}

	// Parse request body
	var request model.UserUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		validationErrors := parseValidationErrors(err)
		if validationErrors != nil {
			response.BadRequest(c, "Validation failed")
		} else {
			response.BadRequest(c, err.Error())
		}
		return
	}

	// Use shared validator for custom validation
	if err := h.validatorClient.Validate(request); err != nil {
		c.Error(err) // Let middleware handle validation errors
		return
	}

	// Update user
	err := h.userService.Update(c.Request.Context(), userID.(int), &request)
	if err != nil {
		h.logger.Error("Failed to update user", zap.Error(err), zap.Int("id", userID.(int)))
		response.Error(c, err)
		return
	}

	// Get updated user
	user, err := h.userService.GetByID(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("Failed to get updated user", zap.Error(err), zap.Int("id", userID.(int)))
		response.Error(c, err)
		return
	}

	response.Success(c, user)
}

// ChangePassword handles changing user's password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var changePassword model.UserChangePassword
	if err := c.ShouldBindJSON(&changePassword); err != nil {
		response.BadRequest(c, "Invalid request data")
		return
	}

	// Use shared validator for password strength validation
	if err := h.validatorClient.Validate(changePassword); err != nil {
		c.Error(err)
		return
	}

	if err := h.userService.ChangePassword(c.Request.Context(), userID.(int), &changePassword); err != nil {
		h.logger.Error("Failed to change password", zap.Error(err))
		c.Error(err)
		return
	}

	response.Success(c, gin.H{"message": "Password changed successfully"})
}

// GetUserByID handles fetching a user by ID (admin only)
func (h *UserHandler) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to get user", zap.Error(err))
		response.InternalError(c, "Failed to get user")
		return
	}

	if user == nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, user)
}

// UpdateUser handles updating a user (admin only)
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var userUpdate model.UserUpdate
	if err := c.ShouldBindJSON(&userUpdate); err != nil {
		response.BadRequest(c, "Invalid request data")
		return
	}

	// Use shared validator for custom validation
	if err := h.validatorClient.Validate(userUpdate); err != nil {
		c.Error(err) // Let middleware handle validation errors
		return
	}

	if err := h.userService.Update(c.Request.Context(), userID.(int), &userUpdate); err != nil {
		h.logger.Error("Failed to update user", zap.Error(err))
		c.Error(err)
		return
	}

	response.Success(c, gin.H{"message": "User updated successfully"})
}

// ListUsers handles listing users (admin only)
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Get pagination from context (set by middleware)
	pagination, exists := c.Get("pagination")
	if !exists {
		pagination = sharedModel.PaginationDefaults
	}

	users, meta, err := h.userService.ListUsers(c, pagination.(*sharedModel.Pagination))
	if err != nil {
		h.logger.Error("Failed to list users", zap.Error(err))
		c.Error(err)
		return
	}

	// Use standard response.Success instead of non-existent SuccessWithPagination
	response.Success(c, gin.H{
		"data": users,
		"meta": meta,
	})
}

// Helper function to create standard error responses
func standardError(code, message string) sharedModel.ErrorResponse {
	errorResponse := sharedModel.ErrorResponse{}
	errorResponse.Error.Type = "https://example.com/errors/" + code
	errorResponse.Error.Code = code
	errorResponse.Error.Message = message
	return errorResponse
}
