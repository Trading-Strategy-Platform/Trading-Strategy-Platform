package handler

import (
	"net/http"
	"strconv"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SymbolHandler handles symbol HTTP requests
type SymbolHandler struct {
	symbolService *service.SymbolService
	logger        *zap.Logger
}

// NewSymbolHandler creates a new symbol handler
func NewSymbolHandler(symbolService *service.SymbolService, logger *zap.Logger) *SymbolHandler {
	return &SymbolHandler{
		symbolService: symbolService,
		logger:        logger,
	}
}

// GetAllSymbols handles retrieving all symbols
func (h *SymbolHandler) GetAllSymbols(c *gin.Context) {
	symbols, err := h.symbolService.GetAllSymbols(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get all symbols", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve symbols"})
		return
	}

	c.JSON(http.StatusOK, symbols)
}

// GetSymbol handles retrieving a symbol by ID
func (h *SymbolHandler) GetSymbol(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid symbol ID"})
		return
	}

	// Get symbol
	symbol, err := h.symbolService.GetSymbolByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get symbol", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve symbol"})
		return
	}

	if symbol == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Symbol not found"})
		return
	}

	c.JSON(http.StatusOK, symbol)
}

// CreateSymbol handles creating a new symbol
func (h *SymbolHandler) CreateSymbol(c *gin.Context) {
	var symbol model.Symbol
	if err := c.ShouldBindJSON(&symbol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create symbol
	id, err := h.symbolService.CreateSymbol(c.Request.Context(), &symbol)
	if err != nil {
		h.logger.Error("Failed to create symbol", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create symbol: " + err.Error()})
		return
	}

	// Set the ID in the response
	symbol.ID = id

	c.JSON(http.StatusCreated, symbol)
}

// TimeframeHandler handles timeframe HTTP requests
type TimeframeHandler struct {
	timeframeService *service.TimeframeService
	logger           *zap.Logger
}

// NewTimeframeHandler creates a new timeframe handler
func NewTimeframeHandler(timeframeService *service.TimeframeService, logger *zap.Logger) *TimeframeHandler {
	return &TimeframeHandler{
		timeframeService: timeframeService,
		logger:           logger,
	}
}

// GetAllTimeframes handles retrieving all timeframes
func (h *TimeframeHandler) GetAllTimeframes(c *gin.Context) {
	timeframes, err := h.timeframeService.GetAllTimeframes(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get all timeframes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve timeframes"})
		return
	}

	c.JSON(http.StatusOK, timeframes)
}

// GetTimeframe handles retrieving a timeframe by ID
func (h *TimeframeHandler) GetTimeframe(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timeframe ID"})
		return
	}

	// Get timeframe
	timeframe, err := h.timeframeService.GetTimeframeByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get timeframe", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve timeframe"})
		return
	}

	if timeframe == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Timeframe not found"})
		return
	}

	c.JSON(http.StatusOK, timeframe)
}

// CreateTimeframe handles creating a new timeframe
func (h *TimeframeHandler) CreateTimeframe(c *gin.Context) {
	var timeframe model.Timeframe
	if err := c.ShouldBindJSON(&timeframe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create timeframe
	id, err := h.timeframeService.CreateTimeframe(c.Request.Context(), &timeframe)
	if err != nil {
		h.logger.Error("Failed to create timeframe", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create timeframe: " + err.Error()})
		return
	}

	// Set the ID in the response
	timeframe.ID = id

	c.JSON(http.StatusCreated, timeframe)
}
