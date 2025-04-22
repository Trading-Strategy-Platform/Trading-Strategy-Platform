package handler

import (
	"net/http"
	"strconv"
	"time"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/service"
	"services/historical-data-service/internal/utils"

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

// GetCandles handles retrieving candle data with dynamic timeframe and pagination
// GET /api/v1/market-data/candles
func (h *MarketDataHandler) GetCandles(c *gin.Context) {
	// Parse query parameters
	var query model.MarketDataQuery

	// Parse symbol_id
	symbolIDStr := c.Query("symbol_id")
	symbolID, err := strconv.Atoi(symbolIDStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid symbol ID")
		return
	}
	query.SymbolID = symbolID

	// Parse timeframe
	timeframe := c.Query("timeframe")
	if timeframe == "" {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Timeframe is required")
		return
	}
	query.Timeframe = timeframe

	// Parse optional parameters
	if startStr := c.Query("start_date"); startStr != "" {
		startDate, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			// Try an alternate format
			startDate, err = time.Parse("2006-01-02", startStr)
			if err != nil {
				utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid start_date format. Use YYYY-MM-DD or RFC3339")
				return
			}
		}
		query.StartDate = &startDate
	}

	if endStr := c.Query("end_date"); endStr != "" {
		endDate, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			// Try an alternate format
			endDate, err = time.Parse("2006-01-02", endStr)
			if err != nil {
				utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid end_date format. Use YYYY-MM-DD or RFC3339")
				return
			}
		}
		query.EndDate = &endDate
	}

	// Parse pagination parameters
	params := utils.ParsePaginationParams(c, 1000, 5000) // default: 1000, max: 5000

	// Get candle data with pagination
	candles, total, err := h.marketDataService.GetCandles(
		c.Request.Context(),
		&query,
		params.Page,
		params.Limit,
	)

	if err != nil {
		h.logger.Error("Failed to get candles",
			zap.Error(err),
			zap.Int("symbolID", query.SymbolID),
			zap.String("timeframe", query.Timeframe))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to get candle data")
		return
	}

	// Use standardized pagination response
	utils.SendPaginatedResponse(c, http.StatusOK, candles, total, params.Page, params.Limit)
}

// BatchImportCandles handles batch importing of candle data
// POST /api/v1/market-data/candles/batch
func (h *MarketDataHandler) BatchImportCandles(c *gin.Context) {
	var request struct {
		Candles []model.CandleBatch `json:"candles" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate candle data
	if len(request.Candles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No candle data provided"})
		return
	}

	// Import candles
	count, err := h.marketDataService.BatchImportCandles(c.Request.Context(), request.Candles)
	if err != nil {
		h.logger.Error("Failed to batch import candles", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import candles: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully imported candles",
		"count":   count,
	})
}

// GetAssetTypes handles retrieving available asset types
// GET /api/v1/market-data/asset-types
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
// GET /api/v1/market-data/exchanges
func (h *MarketDataHandler) GetExchanges(c *gin.Context) {
	exchanges, err := h.marketDataService.GetExchanges(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get exchanges", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exchanges"})
		return
	}

	c.JSON(http.StatusOK, exchanges)
}

// BatchImportMarketData handles batch importing of market data for internal service use
// POST /api/v1/service/market-data/batch
func (h *MarketDataHandler) BatchImportMarketData(c *gin.Context) {
	var request []struct {
		SymbolID  int                 `json:"symbol_id" binding:"required"`
		Timeframe string              `json:"timeframe" binding:"required"`
		Candles   []model.CandleBatch `json:"candles" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(request) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No import requests provided"})
		return
	}

	// Process each set of candles
	totalImported := 0
	for _, batch := range request {
		count, err := h.marketDataService.BatchImportCandles(c.Request.Context(), batch.Candles)
		if err != nil {
			h.logger.Error("Failed to import batch",
				zap.Error(err),
				zap.Int("symbolID", batch.SymbolID),
				zap.String("timeframe", batch.Timeframe))
			continue
		}
		totalImported += count
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Market data batch imported successfully",
		"count":   totalImported,
	})
}
