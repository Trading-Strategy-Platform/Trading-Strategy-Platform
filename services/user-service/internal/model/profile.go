package model

// ProfilePhotoUpload represents a profile photo upload request
type ProfilePhotoUpload struct {
	UserID int    `json:"user_id" binding:"required"`
	URL    string `json:"url" binding:"required"`
}

// ProfilePhotoResponse represents a response after uploading a profile photo
type ProfilePhotoResponse struct {
	URL        string      `json:"url"`
	Thumbnails []Thumbnail `json:"thumbnails,omitempty"`
}

// Thumbnail represents a generated thumbnail
type Thumbnail struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
