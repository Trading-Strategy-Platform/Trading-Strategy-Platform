package handler

import (
	"net/http"
	"strconv"

	"services/strategy-service/internal/service"
	"services/strategy-service/internal/utils"

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
	// Parse pagination parameters using the utility function
	params := utils.ParsePaginationParams(c, 100, 500) // default limit: 100, max limit: 500

	// Parse search parameter
	searchTerm := c.Query("search")

	tags, total, err := h.tagService.GetAllTags(c.Request.Context(), searchTerm, params.Page, params.Limit)
	if err != nil {
		h.logger.Error("Failed to get tags", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to fetch tags")
		return
	}

	// Use standardized pagination response
	utils.SendPaginatedResponse(c, http.StatusOK, tags, total, params.Page, params.Limit)
}

// CreateTag handles creating a new tag
// POST /api/v1/strategy-tags
func (h *TagHandler) CreateTag(c *gin.Context) {
	var request struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	tag, err := h.tagService.CreateTag(c.Request.Context(), request.Name)
	if err != nil {
		h.logger.Error("Failed to create tag", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": tag})
}

// UpdateTag handles updating an existing tag
// PUT /api/v1/strategy-tags/{id}
func (h *TagHandler) UpdateTag(c *gin.Context) {
	// Parse tag ID from URL path
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid tag ID")
		return
	}

	// Parse request body
	var request struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Update tag
	tag, err := h.tagService.UpdateTag(c.Request.Context(), id, request.Name)
	if err != nil {
		h.logger.Error("Failed to update tag", zap.Error(err), zap.Int("id", id))

		// Return appropriate status code based on error
		if err.Error() == "tag not found" {
			utils.SendErrorResponse(c, http.StatusNotFound, err.Error())
		} else {
			utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tag})
}

// DeleteTag handles deleting a tag
// DELETE /api/v1/strategy-tags/{id}
func (h *TagHandler) DeleteTag(c *gin.Context) {
	// Parse tag ID from URL path
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid tag ID")
		return
	}

	// Delete tag
	err = h.tagService.DeleteTag(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete tag", zap.Error(err), zap.Int("id", id))

		// Return appropriate status code based on error
		if err.Error() == "tag not found or is in use" {
			utils.SendErrorResponse(c, http.StatusBadRequest, "Cannot delete tag because it's in use or doesn't exist")
		} else {
			utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to delete tag")
		}
		return
	}

	c.Status(http.StatusNoContent)
}
