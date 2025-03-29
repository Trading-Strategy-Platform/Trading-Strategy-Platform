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

// GetAssetTypes handles retrieving available asset types
func (h *MarketDataHandler) GetAssetTypes(c *gin.Context) {
	assetTypes, err := h.marketDataService.GetAssetTypes(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get asset types", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get asset types"})
		return
	}

	c.JSON(http.StatusOK, assetTypes)
}

// GetExchanges handles retrieving available exchanges
func (h *MarketDataHandler) GetExchanges(c *gin.Context) {
	exchanges, err := h.marketDataService.GetExchanges(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get exchanges", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exchanges"})
		return
	}

	c.JSON(http.StatusOK, exchanges)
}

// GetCandles handles retrieving candle data with dynamic timeframe
func (h *MarketDataHandler) GetCandles(c *gin.Context) {
	symbolID, err := strconv.Atoi(c.Query("symbol_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid symbol ID"})
		return
	}

	timeframe := c.Query("timeframe")
	if timeframe == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Timeframe is required"})
		return
	}

	var startDate, endDate *time.Time

	if startStr := c.Query("start_time"); startStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_time format"})
			return
		}
		startDate = &parsedTime
	}

	if endStr := c.Query("end_time"); endStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_time format"})
			return
		}
		endDate = &parsedTime
	}

	var limitPtr *int
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
			return
		}
		limitPtr = &limit
	}

	candles, err := h.marketDataService.GetCandles(
		c.Request.Context(),
		symbolID,
		timeframe,
		startDate,
		endDate,
		limitPtr,
	)

	if err != nil {
		h.logger.Error("Failed to get candles", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get candle data"})
		return
	}

	c.JSON(http.StatusOK, candles)
}

// BatchImportCandles handles batch importing of candle data
func (h *MarketDataHandler) BatchImportCandles(c *gin.Context) {
	var request struct {
		Candles []model.CandleBatch `json:"candles" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert to JSONB for the database function
	count, err := h.marketDataService.BatchImportCandles(c.Request.Context(), request.Candles)
	if err != nil {
		h.logger.Error("Failed to batch import candles", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import candles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully imported candles",
		"count":   count,
	})
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
