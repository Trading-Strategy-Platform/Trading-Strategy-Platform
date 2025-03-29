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
	// Get query parameters for filtering
	searchTerm := c.Query("search")
	assetType := c.Query("asset_type")
	exchange := c.Query("exchange")

	// If no filters provided, get all symbols
	if searchTerm == "" && assetType == "" && exchange == "" {
		symbols, err := h.symbolService.GetAllSymbols(c.Request.Context())
		if err != nil {
			h.logger.Error("Failed to get all symbols", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve symbols"})
			return
		}
		c.JSON(http.StatusOK, symbols)
		return
	}

	// Apply filters
	filter := &model.SymbolFilter{
		SearchTerm: searchTerm,
		AssetType:  assetType,
		Exchange:   exchange,
	}

	symbols, err := h.symbolService.GetSymbolsByFilter(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to get filtered symbols", zap.Error(err))
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

// UpdateSymbol handles updating an existing symbol
func (h *SymbolHandler) UpdateSymbol(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid symbol ID"})
		return
	}

	var symbol model.Symbol
	if err := c.ShouldBindJSON(&symbol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure ID matches path parameter
	symbol.ID = id

	// Update symbol
	success, err := h.symbolService.UpdateSymbol(c.Request.Context(), &symbol)
	if err != nil {
		h.logger.Error("Failed to update symbol", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update symbol: " + err.Error()})
		return
	}

	if !success {
		c.JSON(http.StatusNotFound, gin.H{"error": "Symbol not found"})
		return
	}

	c.JSON(http.StatusOK, symbol)
}

// DeleteSymbol handles deleting a symbol (marking as inactive)
func (h *SymbolHandler) DeleteSymbol(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid symbol ID"})
		return
	}

	// Delete symbol
	success, err := h.symbolService.DeleteSymbol(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete symbol", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete symbol: " + err.Error()})
		return
	}

	if !success {
		c.JSON(http.StatusNotFound, gin.H{"error": "Symbol not found"})
		return
	}

	c.Status(http.StatusNoContent)
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
