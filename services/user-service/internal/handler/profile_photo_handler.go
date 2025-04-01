package handler

import (
	"io"
	"net/http"

	"services/user-service/internal/client"
	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProfilePhotoHandler handles profile photo operations
type ProfilePhotoHandler struct {
	userService *service.UserService
	mediaClient *client.MediaClient
	logger      *zap.Logger
}

// NewProfilePhotoHandler creates a new profile photo handler
func NewProfilePhotoHandler(userService *service.UserService, mediaClient *client.MediaClient, logger *zap.Logger) *ProfilePhotoHandler {
	return &ProfilePhotoHandler{
		userService: userService,
		mediaClient: mediaClient,
		logger:      logger,
	}
}

// UploadProfilePhoto handles uploading a user profile photo
// POST /api/v1/users/me/profile-photo
func (h *ProfilePhotoHandler) UploadProfilePhoto(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
		h.logger.Error("Failed to parse multipart form", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("photo")
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
	mediaFile, err := h.mediaClient.UploadProfilePhoto(c, userID.(int), fileContent, header.Filename, contentType)
	if err != nil {
		h.logger.Error("Failed to upload file to media service", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// Update user profile with new photo URL
	err = h.userService.UpdateProfilePhoto(c, userID.(int), mediaFile.URL)
	if err != nil {
		h.logger.Error("Failed to update profile photo URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	// Return success response with URLs
	response := gin.H{
		"message":    "Profile photo uploaded successfully",
		"url":        mediaFile.URL,
		"thumbnails": mediaFile.Thumbnails,
	}

	c.JSON(http.StatusOK, response)
}
