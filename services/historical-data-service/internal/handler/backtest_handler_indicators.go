package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetIndicators returns all available technical indicators for strategies
// GET /api/v1/backtests/indicators
func (h *BacktestHandler) GetIndicators(c *gin.Context) {
	// Get available indicators from the backtest service
	indicators, err := h.backtestService.GetIndicators(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get indicators", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve indicators"})
		return
	}

	c.JSON(http.StatusOK, indicators)
}

// ValidateStrategy validates a strategy structure
// POST /api/v1/backtests/validate-strategy
func (h *BacktestHandler) ValidateStrategy(c *gin.Context) {
	var request struct {
		Strategy map[string]interface{} `json:"strategy" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the strategy
	valid, message, err := h.backtestService.ValidateStrategy(c.Request.Context(), request.Strategy)
	if err != nil {
		h.logger.Error("Failed to validate strategy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate strategy"})
		return
	}

	if !valid {
		c.JSON(http.StatusOK, gin.H{
			"valid":   false,
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": message,
	})
}

// GetBacktestServiceStatus checks the health of the backtesting service
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
