package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"services/media-service/internal/config"
	"services/media-service/internal/model"
	"services/media-service/internal/storage"

	"go.uber.org/zap"
)

// MediaService handles media operations
type MediaService struct {
	storage storage.Storage
	config  *config.Config
	logger  *zap.Logger
}

// NewMediaService creates a new media service
func NewMediaService(storage storage.Storage, config *config.Config, logger *zap.Logger) *MediaService {
	return &MediaService{
		storage: storage,
		config:  config,
		logger:  logger,
	}
}

// Upload processes and stores a file
func (s *MediaService) Upload(ctx context.Context, file *multipart.FileHeader, purpose string, entityID string, generateThumbnails bool) (*model.MediaFile, error) {
	// Validate the file
	if err := s.validateFile(file); err != nil {
		return nil, err
	}

	// Store the file
	mediaFile, err := s.storage.Store(ctx, file, purpose, entityID)
	if err != nil {
		s.logger.Error("Failed to store file", zap.Error(err))
		return nil, err
	}

	// Generate thumbnails if needed
	if generateThumbnails && isImage(mediaFile.ContentType) {
		thumbnails, err := s.storage.GenerateThumbnails(ctx, mediaFile)
		if err != nil {
			s.logger.Error("Failed to generate thumbnails", zap.Error(err))
			// Continue without thumbnails
		} else {
			mediaFile.Thumbnails = thumbnails
		}
	}

	return mediaFile, nil
}

// Get retrieves a file by ID
func (s *MediaService) Get(ctx context.Context, id string) (io.ReadCloser, *model.MediaFile, error) {
	return s.storage.Get(ctx, id)
}

// Delete removes a file
func (s *MediaService) Delete(ctx context.Context, id string) error {
	return s.storage.Delete(ctx, id)
}

// validateFile checks if a file meets the requirements
func (s *MediaService) validateFile(file *multipart.FileHeader) error {
	// Check file size
	if file.Size > s.config.Upload.MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d bytes)", file.Size, s.config.Upload.MaxFileSize)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if len(s.config.Upload.AllowedExtensions) > 0 {
		allowed := false
		for _, allowedExt := range s.config.Upload.AllowedExtensions {
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file extension not allowed: %s", ext)
		}
	}

	return nil
}

// isImage checks if a content type represents an image
func isImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}
