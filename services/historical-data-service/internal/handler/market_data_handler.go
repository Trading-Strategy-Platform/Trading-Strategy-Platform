package handler

import (
	"strconv"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/trading-platform/shared/go/response"
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

// GetMarketData handles requests for market data
func (h *MarketDataHandler) GetMarketData(c *gin.Context) {
	var query model.MarketDataQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	data, err := h.marketDataService.GetMarketData(c.Request.Context(), &query)
	if err != nil {
		h.logger.Error("Failed to get market data",
			zap.Error(err),
			zap.Int("symbolID", query.SymbolID),
			zap.Int("timeframeID", query.TimeframeID))
		response.Error(c, err)
		return
	}

	response.Success(c, data)
}

// ImportMarketData handles importing market data
func (h *MarketDataHandler) ImportMarketData(c *gin.Context) {
	var request model.MarketDataImport
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Only admin users should be able to import data
	userRole, exists := c.Get("userRole")
	if !exists || userRole.(string) != "admin" {
		response.Forbidden(c, "Admin access required")
		return
	}

	err := h.marketDataService.ImportMarketData(c.Request.Context(), &request)
	if err != nil {
		h.logger.Error("Failed to import market data",
			zap.Error(err),
			zap.Int("symbolID", request.SymbolID),
			zap.Int("timeframeID", request.TimeframeID))
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Market data imported successfully"})
}

// BatchImportMarketData handles batch importing market data
func (h *MarketDataHandler) BatchImportMarketData(c *gin.Context) {
	var requests []model.MarketDataImport
	if err := c.ShouldBindJSON(&requests); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Only admin users should be able to import data
	userRole, exists := c.Get("userRole")
	if !exists || userRole.(string) != "admin" {
		response.Forbidden(c, "Admin access required")
		return
	}

	err := h.marketDataService.BatchImportMarketData(c.Request.Context(), requests)
	if err != nil {
		h.logger.Error("Failed to batch import market data", zap.Error(err))
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Market data batch imported successfully"})
}

// GetDataAvailabilityRange handles checking data availability
func (h *MarketDataHandler) GetDataAvailabilityRange(c *gin.Context) {
	symbolID := c.Query("symbol_id")
	timeframeID := c.Query("timeframe_id")

	if symbolID == "" || timeframeID == "" {
		response.BadRequest(c, "Symbol ID and timeframe ID are required")
		return
	}

	// Convert to integers
	symbolIDInt, symbolErr := strconv.Atoi(symbolID)
	timeframeIDInt, timeframeErr := strconv.Atoi(timeframeID)

	if symbolErr != nil || timeframeErr != nil {
		response.BadRequest(c, "Invalid symbol ID or timeframe ID")
		return
	}

	startDate, endDate, err := h.marketDataService.GetDataAvailabilityRange(
		c.Request.Context(),
		symbolIDInt,
		timeframeIDInt,
	)

	if err != nil {
		h.logger.Error("Failed to get data availability range",
			zap.Error(err),
			zap.Int("symbolID", symbolIDInt),
			zap.Int("timeframeID", timeframeIDInt))
		response.Error(c, err)
		return
	}

	if startDate == nil || endDate == nil {
		response.Success(c, gin.H{
			"has_data": false,
		})
		return
	}

	response.Success(c, gin.H{
		"has_data":   true,
		"start_date": startDate,
		"end_date":   endDate,
	})
}
