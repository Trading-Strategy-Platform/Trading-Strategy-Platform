package model

import (
	"encoding/json"
	"time"
)

// User represents a user in the system
type User struct {
	ID              int        `json:"id" db:"id"`
	Username        string     `json:"username" db:"username"`
	Email           string     `json:"email" db:"email"`
	PasswordHash    string     `json:"-" db:"password_hash"`
	Role            string     `json:"role" db:"role"`
	ProfilePhotoURL string     `json:"profile_photo_url,omitempty" db:"profile_photo_url"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	LastLogin       *time.Time `json:"last_login,omitempty" db:"last_login"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// UserDetails represents detailed user information including preferences and notification counts
type UserDetails struct {
	ID                       int             `json:"id" db:"id"`
	Username                 string          `json:"username" db:"username"`
	Email                    string          `json:"email" db:"email"`
	Role                     string          `json:"role" db:"role"`
	ProfilePhotoURL          string          `json:"profile_photo_url,omitempty" db:"profile_photo_url"`
	IsActive                 bool            `json:"is_active" db:"is_active"`
	LastLogin                *time.Time      `json:"last_login,omitempty" db:"last_login"`
	CreatedAt                time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt                *time.Time      `json:"updated_at,omitempty" db:"updated_at"`
	UnreadNotificationsCount int             `json:"unread_notifications_count" db:"unread_notifications_count"`
	Theme                    string          `json:"theme,omitempty" db:"theme"`
	DefaultTimeframe         string          `json:"default_timeframe,omitempty" db:"default_timeframe"`
	ChartPreferences         json.RawMessage `json:"chart_preferences,omitempty" db:"chart_preferences"`
	NotificationSettings     json.RawMessage `json:"notification_settings,omitempty" db:"notification_settings"`
}

// UserCreate represents data needed to create a new user
type UserCreate struct {
	Username        string `json:"username" binding:"required,min=3,max=50"`
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8"`
	Role            string `json:"role"`
	ProfilePhotoURL string `json:"profile_photo_url,omitempty"`
}

// UserLogin represents data needed for user login
type UserLogin struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserUpdate represents data for updating user profile
type UserUpdate struct {
	Username        *string `json:"username,omitempty"`
	Email           *string `json:"email,omitempty"`
	ProfilePhotoURL *string `json:"profile_photo_url,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
}

// PreferencesUpdate represents data for updating user preferences
type PreferencesUpdate struct {
	Theme                *string         `json:"theme,omitempty"`
	DefaultTimeframe     *string         `json:"default_timeframe,omitempty"`
	ChartPreferences     json.RawMessage `json:"chart_preferences,omitempty"`
	NotificationSettings json.RawMessage `json:"notification_settings,omitempty"`
}

// UserPreferences represents user preferences
type UserPreferences struct {
	Theme                string          `json:"theme,omitempty" db:"theme"`
	DefaultTimeframe     string          `json:"default_timeframe,omitempty" db:"default_timeframe"`
	ChartPreferences     json.RawMessage `json:"chart_preferences,omitempty" db:"chart_preferences"`
	NotificationSettings json.RawMessage `json:"notification_settings,omitempty" db:"notification_settings"`
}

// UserChangePassword represents data for changing password
type UserChangePassword struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

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

// UserSession represents a user session
type UserSession struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	IPAddress *string   `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent *string   `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// TokenResponse represents the response sent after successful authentication
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         User      `json:"user"`
}
