package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"services/strategy-service/internal/model"
	"services/strategy-service/internal/service"
	"services/strategy-service/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// StrategyHandler handles strategy-related HTTP requests
type StrategyHandler struct {
	strategyService *service.StrategyService
	userClient      UserClient
	logger          *zap.Logger
}

// UserClient defines the interface for user service client
type UserClient interface {
	GetUserByID(ctx context.Context, userID int) (string, error)
}

// NewStrategyHandler creates a new strategy handler
func NewStrategyHandler(strategyService *service.StrategyService, userClient UserClient, logger *zap.Logger) *StrategyHandler {
	return &StrategyHandler{
		strategyService: strategyService,
		userClient:      userClient,
		logger:          logger,
	}
}

// GetAllStrategies handles listing strategies for a user
// GET /api/v1/strategies
func (h *StrategyHandler) GetAllStrategies(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse pagination parameters using the utility function
	params := utils.ParsePaginationParams(c, 10, 100) // default limit: 10, max limit: 100

	// Parse search term
	searchTerm := c.Query("search")

	// Parse purchased_only filter
	purchasedOnly := c.Query("purchased_only") == "true"

	// Parse tag filters
	var tagIDs []int
	if tagIDsStr := c.Query("tag_ids"); tagIDsStr != "" {
		for _, idStr := range strings.Split(tagIDsStr, ",") {
			if id, err := strconv.Atoi(idStr); err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}

	// Parse sorting parameters
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortDirection := c.DefaultQuery("sort_direction", "DESC")

	// Validate sort field
	validSortFields := map[string]bool{
		"name":       true,
		"created_at": true,
		"updated_at": true,
		"version":    true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at" // Default to creation date
	}

	// Normalize sort direction
	sortDirection = strings.ToUpper(sortDirection)
	if sortDirection != "ASC" && sortDirection != "DESC" {
		sortDirection = "DESC"
	}

	strategies, total, err := h.strategyService.GetAllStrategies(
		c.Request.Context(),
		userID.(int),
		searchTerm,
		purchasedOnly,
		tagIDs,
		sortBy,
		sortDirection,
		params.Page,
		params.Limit,
	)

	if err != nil {
		h.logger.Error("Failed to get user strategies", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to fetch strategies")
		return
	}

	// Use standardized pagination response
	utils.SendPaginatedResponse(c, http.StatusOK, strategies, total, params.Page, params.Limit)
}

// CreateStrategy handles creating a new strategy
// POST /api/v1/strategies
func (h *StrategyHandler) CreateStrategy(c *gin.Context) {
	var request model.StrategyCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Set default IsActive to true
	request.IsActive = true

	strategy, err := h.strategyService.CreateStrategy(c.Request.Context(), &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to create strategy", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": strategy})
}

// GetStrategyByID handles retrieving a strategy by ID
// GET /api/v1/strategies/{id}
func (h *StrategyHandler) GetStrategyByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	strategy, err := h.strategyService.GetStrategyByID(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to get strategy", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	// Get creator username
	creatorName, err := h.userClient.GetUserByID(c.Request.Context(), strategy.UserID)
	if err != nil {
		h.logger.Warn("Failed to get creator name", zap.Error(err))
		creatorName = "Unknown"
	}

	// Create response with additional metadata
	response := gin.H{
		"strategy":     strategy,
		"creator_name": creatorName,
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// UpdateStrategy handles updating a strategy
// PUT /api/v1/strategies/{id}
func (h *StrategyHandler) UpdateStrategy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var request model.StrategyUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get the original strategy to compare versions later
	originalStrategy, err := h.strategyService.GetStrategyByID(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to get original strategy", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	originalVersion := originalStrategy.Version

	// Update the strategy
	strategy, err := h.strategyService.UpdateStrategy(c.Request.Context(), id, &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to update strategy", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Check if a new version was created
	versionCreated := strategy.Version > originalVersion

	response := gin.H{
		"data": strategy,
	}

	if versionCreated {
		response["message"] = fmt.Sprintf("Strategy updated. New version created: v%d", strategy.Version)
	}

	c.JSON(http.StatusOK, response)
}

// DeleteStrategy handles deleting a strategy
// DELETE /api/v1/strategies/{id}
func (h *StrategyHandler) DeleteStrategy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err = h.strategyService.DeleteStrategy(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to delete strategy", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// GetVersions handles retrieving all versions of a strategy
// GET /api/v1/strategies/{id}/versions
func (h *StrategyHandler) GetVersions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse pagination parameters using the utility function
	params := utils.ParsePaginationParams(c, 20, 100) // default limit: 20, max limit: 100

	versions, total, err := h.strategyService.GetVersions(c.Request.Context(), id, userID.(int), params.Page, params.Limit)
	if err != nil {
		h.logger.Error("Failed to get versions", zap.Error(err), zap.Int("strategy_id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Use standardized pagination response
	utils.SendPaginatedResponse(c, http.StatusOK, versions, total, params.Page, params.Limit)
}

// GetVersionByID handles retrieving a specific version of a strategy
// GET /api/v1/strategies/{id}/versions/{version}
func (h *StrategyHandler) GetVersionByID(c *gin.Context) {
	// Parse strategy ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	// Parse version number
	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid version number")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get the specific version
	strategyVersion, err := h.strategyService.GetVersionByID(c.Request.Context(), id, version, userID.(int))
	if err != nil {
		h.logger.Error("Failed to get strategy version",
			zap.Error(err),
			zap.Int("strategy_id", id),
			zap.Int("version", version))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if strategyVersion == nil {
		utils.SendErrorResponse(c, http.StatusNotFound, "Strategy version not found or not accessible")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": strategyVersion})
}

// UpdateActiveVersion handles updating the active version of a strategy for a user
// PUT /api/v1/strategies/{id}/active-version
func (h *StrategyHandler) UpdateActiveVersion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	var request struct {
		Version int `json:"version" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err = h.strategyService.UpdateUserStrategyVersion(
		c.Request.Context(),
		userID.(int),
		id,
		request.Version,
	)

	if err != nil {
		h.logger.Error("Failed to update active version",
			zap.Error(err),
			zap.Int("strategy_id", id),
			zap.Int("version", request.Version))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}
