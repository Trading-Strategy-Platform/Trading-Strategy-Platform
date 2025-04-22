package handler

import (
	"net/http"
	"strconv"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/service"
	"services/historical-data-service/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BacktestHandler handles backtest HTTP requests
type BacktestHandler struct {
	backtestService *service.BacktestService
	logger          *zap.Logger
}

// NewBacktestHandler creates a new backtest handler
func NewBacktestHandler(backtestService *service.BacktestService, logger *zap.Logger) *BacktestHandler {
	return &BacktestHandler{
		backtestService: backtestService,
		logger:          logger,
	}
}

// CreateBacktest handles creating a new backtest
// POST /api/v1/backtests
func (h *BacktestHandler) CreateBacktest(c *gin.Context) {
	var request model.BacktestRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get user ID and token from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	token, _ := c.Get("token")
	tokenStr, _ := token.(string)

	// Create backtest
	backtestID, err := h.backtestService.CreateBacktest(
		c.Request.Context(),
		&request,
		userID.(int),
		tokenStr,
	)

	if err != nil {
		h.logger.Error("Failed to create backtest",
			zap.Error(err),
			zap.Int("userID", userID.(int)),
			zap.Int("strategyID", request.StrategyID))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"backtest_id": backtestID,
		"message":     "Backtest created and queued for processing",
	})
}

// GetBacktest handles retrieving a backtest by ID
// GET /api/v1/backtests/:id
func (h *BacktestHandler) GetBacktest(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid backtest ID")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get backtest
	backtest, err := h.backtestService.GetBacktest(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to get backtest",
			zap.Error(err),
			zap.Int("id", id),
			zap.Int("userID", userID.(int)))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, backtest)
}

// ListBacktests handles listing backtests for a user with filtering, sorting, and pagination
// GET /api/v1/backtests
func (h *BacktestHandler) ListBacktests(c *gin.Context) {
	// Parse query parameters for filtering
	searchTerm := c.Query("search")
	status := c.Query("status")

	// Parse sorting parameters
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortDirection := c.DefaultQuery("sort_direction", "DESC")

	// Parse pagination parameters using the utility function
	params := utils.ParsePaginationParams(c, 10, 100) // default limit: 10, max limit: 100

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// List backtests
	backtests, total, err := h.backtestService.ListBacktests(
		c.Request.Context(),
		userID.(int),
		searchTerm,
		status,
		sortBy,
		sortDirection,
		params.Page,
		params.Limit,
	)

	if err != nil {
		h.logger.Error("Failed to list backtests",
			zap.Error(err),
			zap.Int("userID", userID.(int)))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve backtests")
		return
	}

	// Use standardized pagination response
	utils.SendPaginatedResponse(c, http.StatusOK, backtests, total, params.Page, params.Limit)
}

// UpdateBacktestRunStatus handles updating the status of a backtest run
// PUT /api/v1/backtest-runs/:id/status
func (h *BacktestHandler) UpdateBacktestRunStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid backtest run ID")
		return
	}

	var request struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Validate status
	validStatuses := map[string]bool{
		"pending":   true,
		"running":   true,
		"completed": true,
		"failed":    true,
	}

	if !validStatuses[request.Status] {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid status. Must be one of: pending, running, completed, failed")
		return
	}

	success, err := h.backtestService.UpdateBacktestRunStatus(c.Request.Context(), id, request.Status)
	if err != nil {
		h.logger.Error("Failed to update backtest run status",
			zap.Error(err),
			zap.Int("run_id", id))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to update status")
		return
	}

	if !success {
		utils.SendErrorResponse(c, http.StatusNotFound, "Backtest run not found")
		return
	}

	c.Status(http.StatusNoContent)
}

// SaveBacktestResults handles saving results for a backtest run
// POST /api/v1/backtest-runs/:id/results
func (h *BacktestHandler) SaveBacktestResults(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid backtest run ID")
		return
	}

	var request model.BacktestResults
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	resultID, err := h.backtestService.SaveBacktestResults(c.Request.Context(), id, &request)
	if err != nil {
		h.logger.Error("Failed to save backtest results",
			zap.Error(err),
			zap.Int("run_id", id))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to save results")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result_id": resultID,
		"message":   "Backtest results saved successfully",
	})
}

// AddBacktestTrade handles adding a trade to a backtest run
// POST /api/v1/backtest-runs/:id/trades
func (h *BacktestHandler) AddBacktestTrade(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid backtest run ID")
		return
	}

	var request model.BacktestTrade
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Set the backtest run ID from the path parameter
	request.BacktestRunID = id

	tradeID, err := h.backtestService.AddBacktestTrade(c.Request.Context(), &request)
	if err != nil {
		h.logger.Error("Failed to add backtest trade",
			zap.Error(err),
			zap.Int("run_id", id))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to add trade")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"trade_id": tradeID,
		"message":  "Trade added successfully",
	})
}

// GetBacktestTrades handles retrieving trades for a backtest run with sorting and pagination
// GET /api/v1/backtest-runs/:id/trades
func (h *BacktestHandler) GetBacktestTrades(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid backtest run ID")
		return
	}

	// Parse sorting parameters
	sortBy := c.DefaultQuery("sort_by", "entry_time")
	sortDirection := c.DefaultQuery("sort_direction", "ASC")

	// Parse pagination parameters
	params := utils.ParsePaginationParams(c, 100, 1000) // default limit: 100, max limit: 1000

	trades, total, err := h.backtestService.GetBacktestTrades(
		c.Request.Context(),
		id,
		sortBy,
		sortDirection,
		params.Limit,
		utils.CalculateOffset(params.Page, params.Limit),
	)

	if err != nil {
		h.logger.Error("Failed to get backtest trades",
			zap.Error(err),
			zap.Int("run_id", id))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve trades")
		return
	}

	// Use standardized pagination response
	utils.SendPaginatedResponse(c, http.StatusOK, trades, total, params.Page, params.Limit)
}

// DeleteBacktest handles deleting a backtest
// DELETE /api/v1/backtests/:id
func (h *BacktestHandler) DeleteBacktest(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid backtest ID")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Delete backtest
	err = h.backtestService.DeleteBacktest(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to delete backtest",
			zap.Error(err),
			zap.Int("id", id),
			zap.Int("userID", userID.(int)))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// GetBacktestRuns handles retrieving all runs for a backtest with sorting and pagination
// GET /api/v1/backtests/:id/runs
func (h *BacktestHandler) GetBacktestRuns(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid backtest ID")
		return
	}

	// Parse sorting parameters
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortDirection := c.DefaultQuery("sort_direction", "DESC")

	// Parse pagination parameters
	params := utils.ParsePaginationParams(c, 20, 100) // default limit: 20, max limit: 100

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// First verify that the user owns this backtest
	backtest, err := h.backtestService.GetBacktest(c.Request.Context(), id, userID.(int))
	if err != nil || backtest == nil {
		utils.SendErrorResponse(c, http.StatusForbidden, "Access denied or backtest not found")
		return
	}

	// Get backtest runs
	runs, total, err := h.backtestService.GetBacktestRuns(
		c.Request.Context(),
		id,
		sortBy,
		sortDirection,
		params.Page,
		params.Limit,
	)

	if err != nil {
		h.logger.Error("Failed to get backtest runs",
			zap.Error(err),
			zap.Int("backtest_id", id))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve backtest runs")
		return
	}

	// Use standardized pagination response
	utils.SendPaginatedResponse(c, http.StatusOK, runs, total, params.Page, params.Limit)
}

// NotifyBacktestComplete handles notifications about completed backtests
// This is used by background workers or other services
// POST /api/v1/service/backtests/notify
func (h *BacktestHandler) NotifyBacktestComplete(c *gin.Context) {
	var request struct {
		BacktestID int    `json:"backtest_id" binding:"required"`
		StrategyID int    `json:"strategy_id" binding:"required"`
		UserID     int    `json:"user_id" binding:"required"`
		Status     string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Log the notification
	h.logger.Info("Received backtest completion notification",
		zap.Int("backtestID", request.BacktestID),
		zap.Int("strategyID", request.StrategyID),
		zap.Int("userID", request.UserID),
		zap.String("status", request.Status))

	// In a real implementation, you might update a message queue or notify the user

	c.JSON(http.StatusOK, gin.H{"message": "Notification received"})
}

// GetBacktestServiceStatus checks if the backtesting service is healthy
// GET /api/v1/backtests/service-status
func (h *BacktestHandler) GetBacktestServiceStatus(c *gin.Context) {
	// Check if the backtesting service is healthy
	healthy, err := h.backtestService.CheckBacktestServiceHealth(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to check backtest service health", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to check backtest service health",
			"error":   err.Error(),
		})
		return
	}

	if !healthy {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unavailable",
			"message": "Backtesting service is not available",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Backtesting service is operational",
	})
}
