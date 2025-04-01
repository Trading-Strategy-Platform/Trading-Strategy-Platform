package storage

import (
	"context"
	"io"
	"mime/multipart"

	"services/media-service/internal/config"
	"services/media-service/internal/model"
)

// Storage defines the interface for media storage operations
type Storage interface {
	// Store saves a file to storage and returns metadata about the stored file
	Store(ctx context.Context, file *multipart.FileHeader, purpose string, entityID string) (*model.MediaFile, error)

	// Get retrieves a file from storage
	Get(ctx context.Context, id string) (io.ReadCloser, *model.MediaFile, error)

	// Delete removes a file from storage
	Delete(ctx context.Context, id string) error

	// GenerateThumbnails creates thumbnails for an image and returns their metadata
	GenerateThumbnails(ctx context.Context, originalFile *model.MediaFile) ([]model.Thumbnail, error)
}

// NewStorage creates a new storage implementation based on the configuration
func NewStorage(cfg *config.Config) (Storage, error) {
	switch cfg.Storage.Type {
	case "local":
		return NewLocalStorage(&cfg.Storage.Local, cfg.Upload.ThumbnailSizes)
	case "s3":
		return NewS3Storage(&cfg.Storage.S3, cfg.Upload.ThumbnailSizes)
	default:
		// Default to local storage
		return NewLocalStorage(&cfg.Storage.Local, cfg.Upload.ThumbnailSizes)
	}
}
