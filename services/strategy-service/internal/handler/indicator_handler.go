package handler

import (
	"net/http"
	"strconv"
	"strings"

	"services/strategy-service/internal/model"
	"services/strategy-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// IndicatorHandler handles indicator-related HTTP requests
type IndicatorHandler struct {
	indicatorService *service.IndicatorService
	logger           *zap.Logger
}

// NewIndicatorHandler creates a new indicator handler
func NewIndicatorHandler(indicatorService *service.IndicatorService, logger *zap.Logger) *IndicatorHandler {
	return &IndicatorHandler{
		indicatorService: indicatorService,
		logger:           logger,
	}
}

// GetAllIndicators handles retrieving all indicators with filtering options
// GET /api/v1/indicators
func (h *IndicatorHandler) GetAllIndicators(c *gin.Context) {
	// Parse search parameter
	searchTerm := c.Query("search")

	// Parse categories filter (comma-separated)
	var categories []string
	if categoriesStr := c.Query("categories"); categoriesStr != "" {
		categories = strings.Split(categoriesStr, ",")
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	indicators, total, err := h.indicatorService.GetAllIndicators(c.Request.Context(), searchTerm, categories, page, limit)
	if err != nil {
		h.logger.Error("Failed to get indicators", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch indicators"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"indicators": indicators,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (total + limit - 1) / limit,
		},
	})
}

// AddParameter handles adding a parameter to an indicator
// POST /api/v1/indicators/{id}/parameters
func (h *IndicatorHandler) AddParameter(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid indicator ID"})
		return
	}

	var request struct {
		ParameterName string   `json:"parameter_name" binding:"required"`
		ParameterType string   `json:"parameter_type" binding:"required"`
		IsRequired    bool     `json:"is_required" binding:"required"`
		MinValue      *float64 `json:"min_value"`
		MaxValue      *float64 `json:"max_value"`
		DefaultValue  string   `json:"default_value"`
		Description   string   `json:"description"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parameter, err := h.indicatorService.AddParameter(
		c.Request.Context(),
		id,
		request.ParameterName,
		request.ParameterType,
		request.IsRequired,
		request.MinValue,
		request.MaxValue,
		request.DefaultValue,
		request.Description,
	)

	if err != nil {
		h.logger.Error("Failed to add parameter", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, parameter)
}

// AddEnumValue handles adding an enum value to a parameter
// POST /api/v1/parameters/{id}/enum-values
func (h *IndicatorHandler) AddEnumValue(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameter ID"})
		return
	}

	var request struct {
		EnumValue   string `json:"enum_value" binding:"required"`
		DisplayName string `json:"display_name"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	enumValue, err := h.indicatorService.AddEnumValue(
		c.Request.Context(),
		id,
		request.EnumValue,
		request.DisplayName,
	)

	if err != nil {
		h.logger.Error("Failed to add enum value", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, enumValue)
}

// CreateIndicator handles creating a new indicator
// POST /api/v1/indicators
func (h *IndicatorHandler) CreateIndicator(c *gin.Context) {
	// Get user ID from context
	userID, _ := c.Get("userID")
	h.logger.Info("Creating indicator by user", zap.Int("userID", userID.(int)))

	// Define request struct to handle parameters
	var request struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description" binding:"required"`
		Category    string `json:"category" binding:"required"`
		Formula     string `json:"formula"`
		Parameters  []struct {
			Name         string   `json:"name" binding:"required"`
			Type         string   `json:"type" binding:"required"`
			IsRequired   bool     `json:"is_required"`
			MinValue     *float64 `json:"min_value"`
			MaxValue     *float64 `json:"max_value"`
			DefaultValue string   `json:"default_value"`
			Description  string   `json:"description"`
			EnumValues   []struct {
				Value       string `json:"value" binding:"required"`
				DisplayName string `json:"display_name"`
			} `json:"enum_values"`
		} `json:"parameters"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log what we're trying to insert
	h.logger.Info("Creating indicator",
		zap.String("name", request.Name),
		zap.String("category", request.Category),
		zap.Int("parameters_count", len(request.Parameters)))

	// Convert request parameters to model parameters
	var parameters []model.IndicatorParameterCreate
	for _, param := range request.Parameters {
		// Convert enum values if present
		var enumValues []model.ParameterEnumValueCreate
		for _, enum := range param.EnumValues {
			enumValues = append(enumValues, model.ParameterEnumValueCreate{
				EnumValue:   enum.Value,
				DisplayName: enum.DisplayName,
			})
		}

		// Create parameter
		parameters = append(parameters, model.IndicatorParameterCreate{
			ParameterName: param.Name,
			ParameterType: param.Type,
			IsRequired:    param.IsRequired,
			MinValue:      param.MinValue,
			MaxValue:      param.MaxValue,
			DefaultValue:  param.DefaultValue,
			Description:   param.Description,
			EnumValues:    enumValues,
		})
	}

	// Create the indicator using the service
	indicator, err := h.indicatorService.CreateIndicator(
		c.Request.Context(),
		request.Name,
		request.Description,
		request.Category,
		request.Formula,
		parameters,
	)

	if err != nil {
		h.logger.Error("Failed to create indicator", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create indicator: " + err.Error()})
		return
	}

	h.logger.Info("Successfully created indicator", zap.Int("id", indicator.ID))
	c.JSON(http.StatusCreated, indicator)
}

// GetIndicator handles retrieving a specific indicator
// GET /api/v1/indicators/{id}
func (h *IndicatorHandler) GetIndicator(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid indicator ID"})
		return
	}

	indicator, err := h.indicatorService.GetIndicator(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get indicator", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, indicator)
}

// GetCategories handles retrieving indicator categories
// GET /api/v1/indicators/categories
func (h *IndicatorHandler) GetCategories(c *gin.Context) {
	categories, err := h.indicatorService.GetCategories(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get indicator categories", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch indicator categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}
