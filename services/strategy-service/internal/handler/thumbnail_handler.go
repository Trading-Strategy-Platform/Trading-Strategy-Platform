package handler

import (
	"io"
	"net/http"
	"strconv"

	"services/strategy-service/internal/client"
	"services/strategy-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ThumbnailHandler handles strategy thumbnail operations
type ThumbnailHandler struct {
	strategyService *service.StrategyService
	mediaClient     *client.MediaClient
	logger          *zap.Logger
}

// NewThumbnailHandler creates a new thumbnail handler
func NewThumbnailHandler(strategyService *service.StrategyService, mediaClient *client.MediaClient, logger *zap.Logger) *ThumbnailHandler {
	return &ThumbnailHandler{
		strategyService: strategyService,
		mediaClient:     mediaClient,
		logger:          logger,
	}
}

// UploadThumbnail handles uploading a strategy thumbnail
// POST /api/v1/strategies/:id/thumbnail
func (h *ThumbnailHandler) UploadThumbnail(c *gin.Context) {
	// Get strategy ID from URL path
	idStr := c.Param("id")
	strategyID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Verify strategy ownership
	strategy, err := h.strategyService.GetStrategy(c.Request.Context(), strategyID, userID.(int))
	if err != nil {
		h.logger.Error("Failed to get strategy", zap.Error(err), zap.Int("strategy_id", strategyID))
		c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		return
	}

	if strategy.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to modify this strategy"})
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
		h.logger.Error("Failed to parse multipart form", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("thumbnail")
	if err != nil {
		h.logger.Error("Failed to get file from form", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > 5*1024*1024 { // 5MB limit
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 5MB)"})
		return
	}

	// Check content type
	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/gif" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type (allowed: jpeg, png, gif)"})
		return
	}

	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read file content", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Upload to media service
	mediaFile, err := h.mediaClient.UploadStrategyThumbnail(c, strategyID, fileContent, header.Filename, contentType)
	if err != nil {
		h.logger.Error("Failed to upload file to media service", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// Update strategy with new thumbnail URL
	err = h.strategyService.UpdateThumbnail(c.Request.Context(), strategyID, userID.(int), mediaFile.URL)
	if err != nil {
		h.logger.Error("Failed to update strategy thumbnail", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update strategy"})
		return
	}

	// Return success response with URLs
	response := gin.H{
		"message":    "Thumbnail uploaded successfully",
		"url":        mediaFile.URL,
		"thumbnails": mediaFile.Thumbnails,
	}

	c.JSON(http.StatusOK, response)
}
