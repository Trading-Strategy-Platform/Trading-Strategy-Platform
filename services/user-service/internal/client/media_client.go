package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// MediaClient handles communication with the Media Service
type MediaClient struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
	logger     *zap.Logger
}

// MediaUploadResponse represents a response from the media service
type MediaUploadResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message"`
	Media   MediaFile `json:"media"`
}

// MediaFile represents metadata about a stored media file
type MediaFile struct {
	ID          string      `json:"id"`
	FileName    string      `json:"file_name"`
	ContentType string      `json:"content_type"`
	Size        int64       `json:"size"`
	URL         string      `json:"url"`
	Thumbnails  []Thumbnail `json:"thumbnails,omitempty"`
	Width       int         `json:"width,omitempty"`
	Height      int         `json:"height,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

// Thumbnail represents a generated thumbnail
type Thumbnail struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// NewMediaClient creates a new media client
func NewMediaClient(baseURL, serviceKey string, logger *zap.Logger) *MediaClient {
	return &MediaClient{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// UploadProfilePhoto uploads a profile photo for a user
func (c *MediaClient) UploadProfilePhoto(ctx context.Context, userID int, fileContent []byte, filename, contentType string) (*MediaFile, error) {
	// Prepare the multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(fileContent)); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add form fields
	if err := writer.WriteField("purpose", "profile"); err != nil {
		return nil, fmt.Errorf("failed to write purpose field: %w", err)
	}
	if err := writer.WriteField("entity_id", fmt.Sprintf("%d", userID)); err != nil {
		return nil, fmt.Errorf("failed to write entity_id field: %w", err)
	}
	if err := writer.WriteField("generate_thumbnails", "true"); err != nil {
		return nil, fmt.Errorf("failed to write generate_thumbnails field: %w", err)
	}

	writer.Close()

	// Create the request
	url := fmt.Sprintf("%s/api/v1/media/upload", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Service-Key", c.serviceKey)

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to send request to media service", zap.Error(err))
		return nil, fmt.Errorf("failed to send request to media service: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("media service returned error", zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("media service returned status code %d", resp.StatusCode)
	}

	// Parse response
	var uploadResponse MediaUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResponse); err != nil {
		c.logger.Error("failed to decode response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !uploadResponse.Success {
		return nil, fmt.Errorf("upload failed: %s", uploadResponse.Message)
	}

	return &uploadResponse.Media, nil
}

// DeleteMedia deletes a media file
func (c *MediaClient) DeleteMedia(ctx context.Context, mediaID string) error {
	// Create the request
	url := fmt.Sprintf("%s/api/v1/media/%s", c.baseURL, mediaID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set header
	req.Header.Set("X-Service-Key", c.serviceKey)

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to send delete request to media service", zap.Error(err))
		return fmt.Errorf("failed to send delete request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("media service returned error on delete", zap.Int("status", resp.StatusCode))
		return fmt.Errorf("media service returned status code %d", resp.StatusCode)
	}

	return nil
}
