package handler

import (
	"net/http"
	"strconv"
	"time"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MarketDataHandler handles market data HTTP requests
type MarketDataHandler struct {
	marketDataService *service.MarketDataService
	logger            *zap.Logger
}

// NewMarketDataHandler creates a new market data handler
func NewMarketDataHandler(marketDataService *service.MarketDataService, logger *zap.Logger) *MarketDataHandler {
	return &MarketDataHandler{
		marketDataService: marketDataService,
		logger:            logger,
	}
}

// GetMarketData handles retrieving market data
func (h *MarketDataHandler) GetMarketData(c *gin.Context) {
	// Parse path parameters
	symbolIDStr := c.Param("symbol_id")
	symbolID, err := strconv.Atoi(symbolIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid symbol ID"})
		return
	}

	timeframeIDStr := c.Param("timeframe_id")
	timeframeID, err := strconv.Atoi(timeframeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timeframe ID"})
		return
	}

	// Create query object
	query := &model.MarketDataQuery{
		SymbolID:    symbolID,
		TimeframeID: timeframeID,
	}

	// Parse optional query parameters
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format, use YYYY-MM-DD"})
			return
		}
		query.StartDate = &startDate
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format, use YYYY-MM-DD"})
			return
		}
		query.EndDate = &endDate
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
			return
		}
		query.Limit = &limit
	}

	// Get market data
	data, err := h.marketDataService.GetMarketData(c.Request.Context(), query)
	if err != nil {
		h.logger.Error("Failed to get market data",
			zap.Error(err),
			zap.Int("symbolID", symbolID),
			zap.Int("timeframeID", timeframeID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get market data"})
		return
	}

	c.JSON(http.StatusOK, data)
}

// ImportMarketData handles importing market data
func (h *MarketDataHandler) ImportMarketData(c *gin.Context) {
	var request model.MarketDataImport
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Import data
	err := h.marketDataService.ImportMarketData(c.Request.Context(), &request)
	if err != nil {
		h.logger.Error("Failed to import market data",
			zap.Error(err),
			zap.Int("userID", userID.(int)),
			zap.Int("symbolID", request.SymbolID),
			zap.Int("timeframeID", request.TimeframeID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import market data: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Market data imported successfully",
		"count":     len(request.Data),
		"symbol":    request.SymbolID,
		"timeframe": request.TimeframeID,
	})
}

// BatchImportMarketData handles batch importing market data
func (h *MarketDataHandler) BatchImportMarketData(c *gin.Context) {
	var requests []model.MarketDataImport
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(requests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No import requests provided"})
		return
	}

	// Import data
	err := h.marketDataService.BatchImportMarketData(c.Request.Context(), requests)
	if err != nil {
		h.logger.Error("Failed to batch import market data", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to batch import market data: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Market data batch imported successfully",
		"count":   len(requests),
	})
}
