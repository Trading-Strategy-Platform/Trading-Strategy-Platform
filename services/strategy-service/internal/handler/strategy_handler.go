// services/strategy-service/internal/handler/strategy_handler.go
package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"services/strategy-service/internal/model"
	"services/strategy-service/internal/service"

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

// ListUserStrategies handles listing strategies for a user
// GET /api/v1/strategies
func (h *StrategyHandler) ListUserStrategies(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

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

	strategies, total, err := h.strategyService.GetUserStrategies(
		c.Request.Context(),
		userID.(int),
		searchTerm,
		purchasedOnly,
		tagIDs,
		page,
		limit,
	)

	if err != nil {
		h.logger.Error("Failed to get user strategies", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch strategies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"strategies": strategies,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (total + limit - 1) / limit, // Calculate total pages
		},
	})
}

// CreateStrategy handles creating a new strategy
// POST /api/v1/strategies
func (h *StrategyHandler) CreateStrategy(c *gin.Context) {
	var request model.StrategyCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Set default IsActive to true
	request.IsActive = true

	strategy, err := h.strategyService.CreateStrategy(c.Request.Context(), &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to create strategy", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, strategy)
}

// GetStrategy handles retrieving a strategy by ID
// GET /api/v1/strategies/{id}
func (h *StrategyHandler) GetStrategy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	strategy, err := h.strategyService.GetStrategy(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to get strategy", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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

	c.JSON(http.StatusOK, response)
}

// UpdateStrategy handles updating a strategy
// PUT /api/v1/strategies/{id}
func (h *StrategyHandler) UpdateStrategy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var request model.StrategyUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	strategy, err := h.strategyService.UpdateStrategy(c.Request.Context(), id, &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to update strategy", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, strategy)
}

// DeleteStrategy handles deleting a strategy
// DELETE /api/v1/strategies/{id}
func (h *StrategyHandler) DeleteStrategy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err = h.strategyService.DeleteStrategy(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to delete strategy", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	versions, total, err := h.strategyService.GetVersions(c.Request.Context(), id, userID.(int), page, limit)
	if err != nil {
		h.logger.Error("Failed to get versions", zap.Error(err), zap.Int("strategy_id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"versions": versions,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (total + limit - 1) / limit,
		},
	})
}

// UpdateActiveVersion handles updating the active version of a strategy for a user
// PUT /api/v1/strategies/{id}/active-version
func (h *StrategyHandler) UpdateActiveVersion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	var request struct {
		Version int `json:"version" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
