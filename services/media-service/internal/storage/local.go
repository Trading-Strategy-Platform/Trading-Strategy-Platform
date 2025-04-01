package storage

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"services/media-service/internal/config"
	"services/media-service/internal/model"

	"github.com/google/uuid"
	"golang.org/x/image/draw"
)

// LocalStorage implements the Storage interface for local filesystem
type LocalStorage struct {
	basePath       string
	baseURL        string
	permissions    os.FileMode
	thumbnailSizes []config.ThumbnailSize
}

// NewLocalStorage creates a new LocalStorage
func NewLocalStorage(cfg *config.LocalStorageConfig, thumbnailSizes []config.ThumbnailSize) (*LocalStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(cfg.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Parse permissions string
	perms, err := strconv.ParseUint(cfg.Permissions, 8, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid permissions format: %w", err)
	}

	return &LocalStorage{
		basePath:       cfg.BasePath,
		baseURL:        cfg.BaseURL,
		permissions:    os.FileMode(perms),
		thumbnailSizes: thumbnailSizes,
	}, nil
}

// Store saves a file to the local filesystem
func (s *LocalStorage) Store(ctx context.Context, file *multipart.FileHeader, purpose string, entityID string) (*model.MediaFile, error) {
	// Generate unique ID
	id := uuid.New().String()

	// Create directory structure
	dirPath := filepath.Join(s.basePath, purpose, entityID)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Extract file extension
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".bin" // Default extension if none provided
	}

	// Generate final filename and path
	filename := fmt.Sprintf("%s%s", id, ext)
	filePath := filepath.Join(dirPath, filename)

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Create the destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file content
	if _, err = io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Set file permissions
	if err = os.Chmod(filePath, s.permissions); err != nil {
		return nil, fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Determine image dimensions if it's an image
	var width, height int
	if isImage(file.Header.Get("Content-Type")) {
		// Reopen the file to read dimensions
		src.Seek(0, 0)
		img, _, err := image.Decode(src)
		if err == nil {
			bounds := img.Bounds()
			width = bounds.Dx()
			height = bounds.Dy()
		}
	}

	// Construct relative path for URL
	relPath := filepath.Join(purpose, entityID, filename)
	// Replace Windows backslashes with forward slashes for URLs
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	// Create media file metadata
	media := &model.MediaFile{
		ID:          id,
		FileName:    file.Filename,
		ContentType: file.Header.Get("Content-Type"),
		Size:        file.Size,
		URL:         fmt.Sprintf("%s/%s", s.baseURL, relPath),
		Width:       width,
		Height:      height,
		CreatedAt:   time.Now(),
		StorageType: "local",
		StoragePath: filePath,
	}

	return media, nil
}

// Get retrieves a file from the local filesystem
func (s *LocalStorage) Get(ctx context.Context, id string) (io.ReadCloser, *model.MediaFile, error) {
	// Find the file in the filesystem
	// This is a simplistic implementation - in a real system you would have
	// a database to look up the file path from the ID
	var filePath string
	var media *model.MediaFile

	// Walk the base path to find the file with the matching ID
	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if the filename starts with the ID
		if strings.HasPrefix(info.Name(), id) {
			filePath = path

			// Extract information about the file
			relPath, _ := filepath.Rel(s.basePath, path)
			relPath = strings.ReplaceAll(relPath, "\\", "/")

			media = &model.MediaFile{
				ID:          id,
				FileName:    info.Name(),
				Size:        info.Size(),
				URL:         fmt.Sprintf("%s/%s", s.baseURL, relPath),
				CreatedAt:   info.ModTime(),
				StorageType: "local",
				StoragePath: path,
			}

			return filepath.SkipDir // Stop the walk once we found the file
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("error searching for file: %w", err)
	}

	if filePath == "" {
		return nil, nil, fmt.Errorf("file not found: %s", id)
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, media, nil
}

// Delete removes a file from the local filesystem
func (s *LocalStorage) Delete(ctx context.Context, id string) error {
	// Find and delete the file
	var found bool

	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if the filename starts with the ID
		if strings.HasPrefix(info.Name(), id) {
			// Delete the file
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete file: %w", err)
			}

			found = true
			return filepath.SkipDir // Stop the walk once we deleted the file
		}

		return nil
	})

	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("file not found: %s", id)
	}

	return nil
}

// GenerateThumbnails creates thumbnails for an image and returns their metadata
func (s *LocalStorage) GenerateThumbnails(ctx context.Context, originalFile *model.MediaFile) ([]model.Thumbnail, error) {
	// Check if the file is an image
	if !isImage(originalFile.ContentType) {
		return nil, nil // Non-image files don't get thumbnails
	}

	// Extract directory and filename parts
	dir := filepath.Dir(originalFile.StoragePath)
	filename := filepath.Base(originalFile.StoragePath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Open the original image
	originalImg, err := os.Open(originalFile.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open original image: %w", err)
	}
	defer originalImg.Close()

	// Decode the image
	img, _, err := image.Decode(originalImg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	thumbnails := make([]model.Thumbnail, 0, len(s.thumbnailSizes))

	// Generate each thumbnail
	for _, size := range s.thumbnailSizes {
		// Calculate dimensions while preserving aspect ratio
		width, height := calculateDimensions(originalFile.Width, originalFile.Height, size.Width, size.Height)

		// Create the thumbnail image
		thumbImg := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.BiLinear.Scale(thumbImg, thumbImg.Bounds(), img, img.Bounds(), draw.Over, nil)

		// Generate thumbnail filename
		thumbFilename := fmt.Sprintf("%s_%s%s", nameWithoutExt, size.Name, ext)
		thumbPath := filepath.Join(dir, thumbFilename)

		// Create thumbnail file
		thumbFile, err := os.Create(thumbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create thumbnail file: %w", err)
		}

		// Encode the thumbnail based on original format
		switch ext {
		case ".jpeg":
			if err := jpeg.Encode(thumbFile, thumbImg, &jpeg.Options{Quality: 85}); err != nil {
				thumbFile.Close()
				return nil, fmt.Errorf("failed to encode JPEG thumbnail: %w", err)
			}
		case ".png":
			if err := png.Encode(thumbFile, thumbImg); err != nil {
				thumbFile.Close()
				return nil, fmt.Errorf("failed to encode PNG thumbnail: %w", err)
			}
		default:
			// Default to JPEG for other formats
			if err := jpeg.Encode(thumbFile, thumbImg, &jpeg.Options{Quality: 85}); err != nil {
				thumbFile.Close()
				return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
			}
		}

		thumbFile.Close()

		// Set file permissions
		if err = os.Chmod(thumbPath, s.permissions); err != nil {
			return nil, fmt.Errorf("failed to set thumbnail permissions: %w", err)
		}

		// Calculate relative path for URL
		relPath, _ := filepath.Rel(s.basePath, thumbPath)
		relPath = strings.ReplaceAll(relPath, "\\", "/")

		// Add thumbnail to list
		thumbnails = append(thumbnails, model.Thumbnail{
			Name:   size.Name,
			URL:    fmt.Sprintf("%s/%s", s.baseURL, relPath),
			Width:  width,
			Height: height,
		})
	}

	return thumbnails, nil
}

// Helper functions

// isImage checks if a content type represents an image
func isImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// calculateDimensions calculates new dimensions while preserving aspect ratio
func calculateDimensions(origWidth, origHeight, maxWidth, maxHeight int) (int, int) {
	if origWidth <= 0 || origHeight <= 0 {
		return maxWidth, maxHeight
	}

	// Calculate aspect ratio
	ratio := float64(origWidth) / float64(origHeight)

	// Calculate new dimensions while preserving aspect ratio
	width := maxWidth
	height := int(float64(width) / ratio)

	// If height is too large, recalculate width based on maxHeight
	if height > maxHeight {
		height = maxHeight
		width = int(float64(height) * ratio)
	}

	return width, height
}
