package handler

import (
	"net/http"
	"strconv"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/service"

	sharedModel "github.com/yourorg/trading-platform/shared/go/model"
	"github.com/yourorg/trading-platform/shared/go/response"

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
func (h *BacktestHandler) CreateBacktest(c *gin.Context) {
	var request model.BacktestRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		validationErrors := parseValidationErrors(err)
		if validationErrors != nil {
			c.JSON(http.StatusBadRequest, validationErrors)
		} else {
			response.BadRequest(c, err.Error())
		}
		return
	}

	// Get user ID and token from context
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
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
		response.BadRequest(c, err.Error())
		return
	}

	response.Accepted(c, gin.H{
		"backtest_id": backtestID,
		"message":     "Backtest created and queued for processing",
	})
}

// GetBacktest handles retrieving a backtest by ID
func (h *BacktestHandler) GetBacktest(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid backtest ID")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "")
		return
	}

	// Get backtest
	backtest, err := h.backtestService.GetBacktest(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to get backtest",
			zap.Error(err),
			zap.Int("id", id),
			zap.Int("userID", userID.(int)))
		response.Error(c, err)
		return
	}

	if backtest == nil {
		response.NotFound(c, "Backtest not found")
		return
	}

	response.Success(c, backtest)
}

// ListBacktests handles listing backtests for a user
func (h *BacktestHandler) ListBacktests(c *gin.Context) {
	// Get pagination from context (assuming this is set by a middleware)
	pagination, exists := c.Get("pagination")
	if !exists {
		// If not set by middleware, extract it manually
		pagination = &sharedModel.Pagination{
			Page:    sharedModel.PaginationDefaults.Page,
			PerPage: sharedModel.PaginationDefaults.PerPage,
		}

		// Extract page parameter
		if pageStr := c.Query("page"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
				pagination.(*sharedModel.Pagination).Page = page
			}
		}

		// Extract per_page parameter
		if perPageStr := c.Query("per_page"); perPageStr != "" {
			if perPage, err := strconv.Atoi(perPageStr); err == nil && perPage > 0 {
				pagination.(*sharedModel.Pagination).PerPage = perPage
			}
		}
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// List backtests
	backtests, total, err := h.backtestService.ListBacktests(
		c.Request.Context(),
		userID.(int),
		pagination.(*sharedModel.Pagination).Page,
		pagination.(*sharedModel.Pagination).PerPage,
	)

	if err != nil {
		h.logger.Error("Failed to list backtests",
			zap.Error(err),
			zap.Int("userID", userID.(int)))
		response.Error(c, err)
		return
	}

	meta := sharedModel.NewPaginationMeta(pagination.(*sharedModel.Pagination), total)

	response.Success(c, gin.H{
		"backtests": backtests,
		"meta":      meta,
	})
}

// DeleteBacktest handles deleting a backtest
func (h *BacktestHandler) DeleteBacktest(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid backtest ID")
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "")
		return
	}

	// Delete backtest
	err = h.backtestService.DeleteBacktest(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to delete backtest",
			zap.Error(err),
			zap.Int("id", id),
			zap.Int("userID", userID.(int)))
		response.Error(c, err)
		return
	}

	response.NoContent(c)
}

// Helper function to parse validation errors from binding errors
func parseValidationErrors(err error) *sharedModel.ValidationErrors {
	// Implementation depends on the validation library used
	// For gin's default validator, you would extract field errors
	// This is a simplified example
	return nil
}
