package handler

import (
	"strconv"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/trading-platform/shared/go/response"
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

// GetAllSymbols handles fetching all symbols
func (h *SymbolHandler) GetAllSymbols(c *gin.Context) {
	symbols, err := h.symbolService.GetAllSymbols(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get symbols", zap.Error(err))
		response.Error(c, err)
		return
	}

	response.Success(c, symbols)
}

// GetSymbolByID handles fetching a symbol by ID
func (h *SymbolHandler) GetSymbolByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid symbol ID")
		return
	}

	symbol, err := h.symbolService.GetSymbolByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get symbol", zap.Error(err), zap.Int("id", id))
		response.Error(c, err)
		return
	}

	if symbol == nil {
		response.NotFound(c, "Symbol not found")
		return
	}

	response.Success(c, symbol)
}

// CreateSymbol handles creating a new symbol
func (h *SymbolHandler) CreateSymbol(c *gin.Context) {
	var symbol model.Symbol
	if err := c.ShouldBindJSON(&symbol); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Only admin users should be able to create symbols
	userRole, exists := c.Get("userRole")
	if !exists || userRole.(string) != "admin" {
		response.Forbidden(c, "Admin access required")
		return
	}

	id, err := h.symbolService.CreateSymbol(c.Request.Context(), &symbol)
	if err != nil {
		h.logger.Error("Failed to create symbol", zap.Error(err))
		response.Error(c, err)
		return
	}

	symbol.ID = id
	response.Created(c, symbol)
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

// GetAllTimeframes handles fetching all timeframes
func (h *TimeframeHandler) GetAllTimeframes(c *gin.Context) {
	timeframes, err := h.timeframeService.GetAllTimeframes(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get timeframes", zap.Error(err))
		response.Error(c, err)
		return
	}

	response.Success(c, timeframes)
}

// GetTimeframeByID handles fetching a timeframe by ID
func (h *TimeframeHandler) GetTimeframeByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid timeframe ID")
		return
	}

	timeframe, err := h.timeframeService.GetTimeframeByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get timeframe", zap.Error(err), zap.Int("id", id))
		response.Error(c, err)
		return
	}

	if timeframe == nil {
		response.NotFound(c, "Timeframe not found")
		return
	}

	response.Success(c, timeframe)
}

// CreateTimeframe handles creating a new timeframe
func (h *TimeframeHandler) CreateTimeframe(c *gin.Context) {
	var timeframe model.Timeframe
	if err := c.ShouldBindJSON(&timeframe); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Only admin users should be able to create timeframes
	userRole, exists := c.Get("userRole")
	if !exists || userRole.(string) != "admin" {
		response.Forbidden(c, "Admin access required")
		return
	}

	id, err := h.timeframeService.CreateTimeframe(c.Request.Context(), &timeframe)
	if err != nil {
		h.logger.Error("Failed to create timeframe", zap.Error(err))
		response.Error(c, err)
		return
	}

	timeframe.ID = id
	response.Created(c, timeframe)
}
