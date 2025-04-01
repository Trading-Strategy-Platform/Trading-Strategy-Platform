package model

import (
	"time"
)

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
	UploadedBy  string      `json:"uploaded_by,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	StorageType string      `json:"storage_type"`
	StoragePath string      `json:"-"` // Internal use only, not exposed in API
}

// Thumbnail represents a generated thumbnail of a media file
type Thumbnail struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// UploadRequest represents a request to upload a file
type UploadRequest struct {
	Purpose            string `json:"purpose" form:"purpose"`     // e.g., "profile", "strategy", "post"
	EntityID           string `json:"entity_id" form:"entity_id"` // ID of related entity (user, strategy, etc.)
	GenerateThumbnails bool   `json:"generate_thumbnails" form:"generate_thumbnails"`
}

// UploadResponse represents the response after a successful upload
type UploadResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message,omitempty"`
	Media   MediaFile `json:"media"`
}

// DeleteRequest represents a request to delete a media file
type DeleteRequest struct {
	ID string `json:"id" uri:"id" binding:"required"`
}
