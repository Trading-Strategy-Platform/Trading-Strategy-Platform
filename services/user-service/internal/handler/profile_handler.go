package handler

import (
	"io"
	"net/http"

	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProfileHandler handles profile-related HTTP requests
type ProfileHandler struct {
	profileService *service.ProfileService
	logger         *zap.Logger
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(profileService *service.ProfileService, logger *zap.Logger) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
		logger:         logger,
	}
}

// GetProfilePhoto handles retrieving a user's profile photo URL
// GET /api/v1/users/me/profile-photo
func (h *ProfileHandler) GetProfilePhoto(c *gin.Context) {
	userID, _ := c.Get("userID")

	photoURL, err := h.profileService.GetProfilePhotoURL(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("failed to get profile photo URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profile photo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": photoURL,
	})
}

// UploadProfilePhoto handles uploading a user profile photo
// POST /api/v1/users/me/profile-photo
func (h *ProfileHandler) UploadProfilePhoto(c *gin.Context) {
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

	// Upload profile photo
	result, err := h.profileService.UploadProfilePhoto(
		c.Request.Context(),
		userID.(int),
		fileContent,
		header.Filename,
		contentType,
	)
	if err != nil {
		h.logger.Error("Failed to upload profile photo", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload profile photo"})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"message":    "Profile photo uploaded successfully",
		"url":        result.URL,
		"thumbnails": result.Thumbnails,
	})
}

// DeleteProfilePhoto handles deleting a user's profile photo
// DELETE /api/v1/users/me/profile-photo
func (h *ProfileHandler) DeleteProfilePhoto(c *gin.Context) {
	userID, _ := c.Get("userID")

	err := h.profileService.ClearProfilePhoto(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("Failed to delete profile photo", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete profile photo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile photo deleted successfully",
	})
}
