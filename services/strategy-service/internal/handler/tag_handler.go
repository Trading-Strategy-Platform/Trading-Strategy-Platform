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
