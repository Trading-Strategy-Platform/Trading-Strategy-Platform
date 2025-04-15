package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"services/media-service/internal/model"
	"services/media-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MediaHandler handles media-related HTTP requests
type MediaHandler struct {
	mediaService *service.MediaService
	logger       *zap.Logger
}

// NewMediaHandler creates a new media handler
func NewMediaHandler(mediaService *service.MediaService, logger *zap.Logger) *MediaHandler {
	return &MediaHandler{
		mediaService: mediaService,
		logger:       logger,
	}
}

// Upload handles file uploads
// POST /api/v1/media/upload
func (h *MediaHandler) Upload(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
		h.logger.Error("Failed to parse multipart form", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.logger.Error("Failed to get file from form", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Bind request parameters
	var req model.UploadRequest
	if err := c.ShouldBind(&req); err != nil {
		h.logger.Error("Failed to bind request parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// Set default purpose if not provided
	if req.Purpose == "" {
		req.Purpose = "general"
	}

	// Set default entity ID if not provided
	if req.EntityID == "" {
		req.EntityID = "default"
	}

	// Upload the file
	mediaFile, err := h.mediaService.Upload(c, header, req.Purpose, req.EntityID, req.GenerateThumbnails)
	if err != nil {
		h.logger.Error("Failed to upload file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload file: %v", err)})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, model.UploadResponse{
		Success: true,
		Message: "File uploaded successfully",
		Media:   *mediaFile,
	})
}

// Get handles file retrieval
// GET /api/v1/media/:id
func (h *MediaHandler) Get(c *gin.Context) {
	id := c.Param("id")

	// If the ID is empty, check if there's a wildcard path
	if id == "" {
		// This handles the case when using a wildcard path
		fullPath := c.Param("fullpath")
		if fullPath != "" {
			// Remove leading slash if present
			fullPath = strings.TrimPrefix(fullPath, "/")

			// Extract the file ID from the path
			pathParts := strings.Split(fullPath, "/")
			if len(pathParts) > 0 {
				id = pathParts[len(pathParts)-1]
			}
		}
	}

	// Extract the filename part from the ID if it contains a path
	parts := strings.Split(id, "/")
	id = parts[len(parts)-1]

	// If ID contains a filename with UUID, extract just the UUID part
	if strings.Contains(id, "-") {
		// This is likely a filename with UUID format like "123e4567-e89b-12d3-a456-426614174000.jpg"
		// Extract just the UUID part (before extension if there is one)
		if dotIndex := strings.LastIndex(id, "."); dotIndex != -1 {
			// Has extension
			id = id[:dotIndex]
		}
	}

	// Get the file
	file, mediaFile, err := h.mediaService.Get(c, id)
	if err != nil {
		h.logger.Error("Failed to get file", zap.Error(err), zap.String("id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	defer file.Close()

	// Set content type
	c.Header("Content-Type", mediaFile.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", mediaFile.FileName))

	// Add caching headers
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Header("Expires", "Thu, 31 Dec 2099 23:59:59 GMT")

	// Stream the file
	if _, err := io.Copy(c.Writer, file); err != nil {
		h.logger.Error("Failed to stream file", zap.Error(err))
		// Cannot send a JSON response here since we've already started writing the response
		return
	}
}

// GetByPath handles file retrieval by full path
// GET /api/v1/media/by-path/*fullpath
func (h *MediaHandler) GetByPath(c *gin.Context) {
	fullPath := c.Param("fullpath")

	// Remove leading slash if present
	fullPath = strings.TrimPrefix(fullPath, "/")

	// Extract the file ID from the path
	pathParts := strings.Split(fullPath, "/")
	if len(pathParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}

	// The last part should be the filename with the ID
	filename := pathParts[len(pathParts)-1]

	// Extract ID from filename (before extension if there is one)
	id := filename
	if dotIndex := strings.LastIndex(filename, "."); dotIndex != -1 {
		id = filename[:dotIndex]
	}

	// Get the file using extracted ID
	file, mediaFile, err := h.mediaService.Get(c, id)
	if err != nil {
		h.logger.Error("Failed to get file", zap.Error(err), zap.String("path", fullPath))
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	defer file.Close()

	// Set content type
	c.Header("Content-Type", mediaFile.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", mediaFile.FileName))

	// Add caching headers
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Header("Expires", "Thu, 31 Dec 2099 23:59:59 GMT")

	// Stream the file
	if _, err := io.Copy(c.Writer, file); err != nil {
		h.logger.Error("Failed to stream file", zap.Error(err))
		return
	}
}

// Delete handles file deletion
// DELETE /api/v1/media/:id
func (h *MediaHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// Extract the filename part from the ID if it contains a path
	parts := strings.Split(id, "/")
	id = parts[len(parts)-1]

	// Delete the file
	if err := h.mediaService.Delete(c, id); err != nil {
		h.logger.Error("Failed to delete file", zap.Error(err), zap.String("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete file: %v", err)})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "File deleted successfully",
	})
}
