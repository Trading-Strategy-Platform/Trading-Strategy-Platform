package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"services/strategy-service/internal/client"
	"services/strategy-service/internal/model"
	"services/strategy-service/internal/service"
	"services/strategy-service/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// StrategyHandler handles strategy-related HTTP requests
type StrategyHandler struct {
	strategyService *service.StrategyService
	userClient      *client.UserClient
	logger          *zap.Logger
}

// NewStrategyHandler creates a new strategy handler
func NewStrategyHandler(strategyService *service.StrategyService, userClient *client.UserClient, logger *zap.Logger) *StrategyHandler {
	return &StrategyHandler{
		strategyService: strategyService,
		userClient:      userClient,
		logger:          logger,
	}
}

// GetAllStrategies handles retrieving all strategies for a user
// GET /api/v1/strategies
func (h *StrategyHandler) GetAllStrategies(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse query parameters
	searchTerm := c.Query("search")
	purchasedOnly := c.Query("purchased_only") == "true"

	// Parse tags (comma-separated)
	var tagIDs []int
	if tagsParam := c.Query("tags"); tagsParam != "" {
		tagParts := strings.Split(tagsParam, ",")
		for _, part := range tagParts {
			if tagID, err := strconv.Atoi(part); err == nil {
				tagIDs = append(tagIDs, tagID)
			}
		}
	}

	// Parse pagination parameters
	params := utils.ParsePaginationParams(c, 20, 100) // default limit: 20, max limit: 100

	// Parse sorting parameters
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortDirection := c.DefaultQuery("sort_direction", "DESC")

	// Get strategies from service
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
		h.logger.Error("Failed to get strategies", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to fetch strategies")
		return
	}

	// Return paginated response
	utils.SendPaginatedResponse(c, http.StatusOK, strategies, total, params.Page, params.Limit)
}

// GetStrategyByID handles retrieving a strategy by ID
// GET /api/v1/strategies/{id}
func (h *StrategyHandler) GetStrategyByID(c *gin.Context) {
	// Parse strategy ID from URL
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get strategy from service
	strategy, err := h.strategyService.GetStrategyByID(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to get strategy", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": strategy})
}

// CreateStrategy handles creating a new strategy
// POST /api/v1/strategies
func (h *StrategyHandler) CreateStrategy(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var request model.StrategyCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Ensure structure is valid JSON
	if len(request.Structure) == 0 {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Strategy structure cannot be empty")
		return
	}

	// Create strategy using service
	strategy, err := h.strategyService.CreateStrategy(c.Request.Context(), &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to create strategy", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": strategy})
}

// UpdateStrategy handles updating a strategy (creates a new version)
// PUT /api/v1/strategies/{id}
func (h *StrategyHandler) UpdateStrategy(c *gin.Context) {
	// Parse strategy ID from URL
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var request model.StrategyUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Ensure structure is valid JSON
	if len(request.Structure) == 0 {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Strategy structure cannot be empty")
		return
	}

	// Update strategy using service
	strategy, err := h.strategyService.UpdateStrategy(c.Request.Context(), id, userID.(int), &request)
	if err != nil {
		h.logger.Error("Failed to update strategy", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": strategy})
}

// DeleteStrategy handles deleting a strategy
// DELETE /api/v1/strategies/{id}
func (h *StrategyHandler) DeleteStrategy(c *gin.Context) {
	// Parse strategy ID from URL
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Delete strategy using service
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
	// Parse strategy ID from URL
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse pagination parameters
	params := utils.ParsePaginationParams(c, 20, 100) // default limit: 20, max limit: 100

	// Parse sorting parameters
	sortBy := c.DefaultQuery("sort_by", "version")
	sortDirection := c.DefaultQuery("sort_direction", "DESC")

	// Get strategy versions from service
	versions, total, err := h.strategyService.GetVersions(
		c.Request.Context(),
		id,
		userID.(int),
		sortBy,
		sortDirection,
		params.Page,
		params.Limit,
	)

	if err != nil {
		h.logger.Error("Failed to get strategy versions", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Return paginated response
	utils.SendPaginatedResponse(c, http.StatusOK, versions, total, params.Page, params.Limit)
}

// GetVersionByID handles retrieving a specific version of a strategy
// GET /api/v1/strategies/{id}/versions/{version}
func (h *StrategyHandler) GetVersionByID(c *gin.Context) {
	// Parse strategy ID and version ID from URL
	strategyIDStr := c.Param("id")
	strategyID, err := strconv.Atoi(strategyIDStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	versionIDStr := c.Param("version")
	versionID, err := strconv.Atoi(versionIDStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid version ID")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get strategy version from service
	version, err := h.strategyService.GetVersionByID(
		c.Request.Context(),
		strategyID,
		versionID,
		userID.(int),
	)

	if err != nil {
		h.logger.Error("Failed to get strategy version",
			zap.Error(err),
			zap.Int("strategy_id", strategyID),
			zap.Int("version_id", versionID))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": version})
}

// SetActiveVersion handles setting a strategy version as the active one for a user
// POST /api/v1/strategies/{id}/versions/{version}/activate
func (h *StrategyHandler) SetActiveVersion(c *gin.Context) {
	// Parse strategy ID and version ID from URL
	strategyIDStr := c.Param("id")
	strategyID, err := strconv.Atoi(strategyIDStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	versionIDStr := c.Param("version")
	versionID, err := strconv.Atoi(versionIDStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid version ID")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Set active version using service
	err = h.strategyService.SetActiveVersion(
		c.Request.Context(),
		userID.(int),
		strategyID,
		versionID,
	)

	if err != nil {
		h.logger.Error("Failed to set active version",
			zap.Error(err),
			zap.Int("strategy_id", strategyID),
			zap.Int("version_id", versionID))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// BacktestStrategy handles submitting a backtest for a strategy
// POST /api/v1/strategies/{id}/backtest
func (h *StrategyHandler) BacktestStrategy(c *gin.Context) {
	// Parse strategy ID from URL
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var request struct {
		SymbolID       int     `json:"symbol_id" binding:"required"`
		TimeframeID    int     `json:"timeframe_id" binding:"required"`
		StartDateStr   string  `json:"start_date" binding:"required"`
		EndDateStr     string  `json:"end_date" binding:"required"`
		InitialCapital float64 `json:"initial_capital" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Parse dates
	layout := "2006-01-02"
	startDate, err := time.Parse(layout, request.StartDateStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid start date format (should be YYYY-MM-DD)")
		return
	}

	endDate, err := time.Parse(layout, request.EndDateStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid end date format (should be YYYY-MM-DD)")
		return
	}

	// Create backtest request
	backtestRequest := &model.BacktestRequest{
		StrategyID:     id,
		SymbolID:       request.SymbolID,
		TimeframeID:    request.TimeframeID,
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: request.InitialCapital,
	}

	// Submit backtest using service
	backtestID, err := h.strategyService.CreateBacktest(c.Request.Context(), backtestRequest, userID.(int))
	if err != nil {
		h.logger.Error("Failed to submit backtest", zap.Error(err), zap.Int("strategy_id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "Backtest submitted successfully",
		"backtest_id": backtestID,
	})
}
