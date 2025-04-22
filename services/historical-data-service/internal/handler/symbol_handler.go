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

// GetAllSymbols handles retrieving all symbols with filtering, pagination and sorting
// GET /api/v1/symbols
func (h *SymbolHandler) GetAllSymbols(c *gin.Context) {
	// Get query parameters for filtering
	searchTerm := c.Query("search")
	assetType := c.Query("asset_type")
	exchange := c.Query("exchange")

	// Parse sorting parameters
	sortBy := c.DefaultQuery("sort_by", "symbol")
	sortDirection := c.DefaultQuery("sort_direction", "ASC")

	// Parse pagination parameters
	params := utils.ParsePaginationParams(c, 20, 100) // default limit: 20, max limit: 100

	// If no filters and no pagination/sorting specified, use the simple method
	if searchTerm == "" && assetType == "" && exchange == "" &&
		sortBy == "symbol" && sortDirection == "ASC" &&
		params.Page == 1 && params.Limit == 20 {
		symbols, err := h.symbolService.GetAllSymbols(c.Request.Context())
		if err != nil {
			h.logger.Error("Failed to get all symbols", zap.Error(err))
			utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve symbols")
			return
		}
		c.JSON(http.StatusOK, symbols)
		return
	}

	// Use the paginated method
	symbols, total, err := h.symbolService.GetSymbolsWithPagination(
		c.Request.Context(),
		searchTerm,
		assetType,
		exchange,
		sortBy,
		sortDirection,
		params.Page,
		params.Limit,
	)

	if err != nil {
		h.logger.Error("Failed to get filtered symbols", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve symbols")
		return
	}

	// Use standardized pagination response
	utils.SendPaginatedResponse(c, http.StatusOK, symbols, total, params.Page, params.Limit)
}

// GetSymbol handles retrieving a symbol by ID
// GET /api/v1/symbols/:id
func (h *SymbolHandler) GetSymbol(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid symbol ID")
		return
	}

	// Get symbol
	symbol, err := h.symbolService.GetSymbolByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get symbol", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve symbol")
		return
	}

	if symbol == nil {
		utils.SendErrorResponse(c, http.StatusNotFound, "Symbol not found")
		return
	}

	c.JSON(http.StatusOK, symbol)
}

// CreateSymbol handles creating a new symbol
// POST /api/v1/symbols
func (h *SymbolHandler) CreateSymbol(c *gin.Context) {
	var symbol model.Symbol
	if err := c.ShouldBindJSON(&symbol); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Create symbol
	id, err := h.symbolService.CreateSymbol(c.Request.Context(), &symbol)
	if err != nil {
		h.logger.Error("Failed to create symbol", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to create symbol: "+err.Error())
		return
	}

	// Set the ID in the response
	symbol.ID = id

	c.JSON(http.StatusCreated, symbol)
}

// UpdateSymbol handles updating an existing symbol
// PUT /api/v1/symbols/:id
func (h *SymbolHandler) UpdateSymbol(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid symbol ID")
		return
	}

	var symbol model.Symbol
	if err := c.ShouldBindJSON(&symbol); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Ensure ID matches path parameter
	symbol.ID = id

	// Update symbol
	success, err := h.symbolService.UpdateSymbol(c.Request.Context(), &symbol)
	if err != nil {
		h.logger.Error("Failed to update symbol", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to update symbol: "+err.Error())
		return
	}

	if !success {
		utils.SendErrorResponse(c, http.StatusNotFound, "Symbol not found")
		return
	}

	c.JSON(http.StatusOK, symbol)
}

// DeleteSymbol handles deleting a symbol (marking as inactive)
// DELETE /api/v1/symbols/:id
func (h *SymbolHandler) DeleteSymbol(c *gin.Context) {
	// Parse path parameter
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid symbol ID")
		return
	}

	// Delete symbol
	success, err := h.symbolService.DeleteSymbol(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete symbol", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to delete symbol: "+err.Error())
		return
	}

	if !success {
		utils.SendErrorResponse(c, http.StatusNotFound, "Symbol not found")
		return
	}

	c.Status(http.StatusNoContent)
}
