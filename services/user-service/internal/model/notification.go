package model

import (
	"time"
)

// Notification represents a user notification
type Notification struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Type      string    `json:"type" db:"type"`
	Title     string    `json:"title" db:"title"`
	Message   string    `json:"message" db:"message"`
	IsRead    bool      `json:"is_read" db:"is_read"`
	Link      string    `json:"link,omitempty" db:"link"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// NotificationCreate represents data for creating a notification
type NotificationCreate struct {
	UserID  int    `json:"user_id" binding:"required"`
	Type    string `json:"type" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Message string `json:"message" binding:"required"`
	Link    string `json:"link,omitempty"`
}

// NotificationListResponse represents a paginated list of notifications with metadata
type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	Total         int            `json:"total"`
	Unread        int            `json:"unread"`
}

// NotificationCountResponse represents the count of unread notifications
type NotificationCountResponse struct {
	Count int `json:"count"`
}

// NotificationMarkResponse represents the response after marking notifications as read
type NotificationMarkResponse struct {
	Success     bool `json:"success"`
	MarkedCount int  `json:"marked_count"`
}
