// services/strategy-service/internal/handler/tag_handler.go
package handler

import (
	"net/http"
	"strconv"

	"services/strategy-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TagHandler handles tag-related HTTP requests
type TagHandler struct {
	tagService *service.TagService
	logger     *zap.Logger
}

// NewTagHandler creates a new tag handler
func NewTagHandler(tagService *service.TagService, logger *zap.Logger) *TagHandler {
	return &TagHandler{
		tagService: tagService,
		logger:     logger,
	}
}

// GetAllTags handles retrieving all tags
// GET /api/v1/strategy-tags
func (h *TagHandler) GetAllTags(c *gin.Context) {
	// For tags, we typically don't need pagination since there are not many tags
	// However, we could add it if needed
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}

	tags, total, err := h.tagService.GetAllTags(c.Request.Context(), page, limit)
	if err != nil {
		h.logger.Error("Failed to get tags", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tags": tags,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (total + limit - 1) / limit,
		},
	})
}

// CreateTag handles creating a new tag
// POST /api/v1/strategy-tags
func (h *TagHandler) CreateTag(c *gin.Context) {
	var request struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tag, err := h.tagService.CreateTag(c.Request.Context(), request.Name)
	if err != nil {
		h.logger.Error("Failed to create tag", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tag)
}

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

// GetAllIndicators handles retrieving all indicators
// GET /api/v1/indicators
func (h *IndicatorHandler) GetAllIndicators(c *gin.Context) {
	// Parse category filter
	var category *string
	if categoryStr := c.Query("category"); categoryStr != "" {
		category = &categoryStr
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

	indicators, total, err := h.indicatorService.GetAllIndicators(c.Request.Context(), category, page, limit)
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
	var request struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description" binding:"required"`
		Category    string `json:"category" binding:"required"`
		Formula     string `json:"formula"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	indicator, err := h.indicatorService.CreateIndicator(
		c.Request.Context(),
		request.Name,
		request.Description,
		request.Category,
		request.Formula,
	)

	if err != nil {
		h.logger.Error("Failed to create indicator", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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
