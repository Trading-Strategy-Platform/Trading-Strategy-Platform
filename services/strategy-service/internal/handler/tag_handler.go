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
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}

	// Parse search parameter
	searchTerm := c.Query("search")

	tags, total, err := h.tagService.GetAllTags(c.Request.Context(), searchTerm, page, limit)
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

// UpdateTag handles updating an existing tag
// PUT /api/v1/strategy-tags/{id}
func (h *TagHandler) UpdateTag(c *gin.Context) {
	// Parse tag ID from URL path
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	// Parse request body
	var request struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update tag
	tag, err := h.tagService.UpdateTag(c.Request.Context(), id, request.Name)
	if err != nil {
		h.logger.Error("Failed to update tag", zap.Error(err), zap.Int("id", id))

		// Return appropriate status code based on error
		if err.Error() == "tag not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, tag)
}

// DeleteTag handles deleting a tag
// DELETE /api/v1/strategy-tags/{id}
func (h *TagHandler) DeleteTag(c *gin.Context) {
	// Parse tag ID from URL path
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	// Delete tag
	err = h.tagService.DeleteTag(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete tag", zap.Error(err), zap.Int("id", id))

		// Return appropriate status code based on error
		if err.Error() == "tag not found or is in use" {
			if err.Error() == "tag not found or is in use" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete tag because it's in use or doesn't exist"})
			} else {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tag"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
