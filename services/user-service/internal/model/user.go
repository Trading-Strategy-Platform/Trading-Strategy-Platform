package model

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID           int        `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Role         string     `json:"role" db:"role"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	LastLogin    *time.Time `json:"last_login,omitempty" db:"last_login"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// UserCreate represents data needed to create a new user
type UserCreate struct {
	Username string `json:"username" binding:"required,min=3,max=50,alphanum_dash"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,strong_password"`
	Role     string `json:"role"`
}

// UserLogin represents data needed for user login
type UserLogin struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserUpdate represents user update data
type UserUpdate struct {
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
	Role     *string `json:"role,omitempty"`
}

// UserChangePassword represents data for changing password
type UserChangePassword struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,strong_password"`
}

// UserPreference represents user preferences
type UserPreference struct {
	ID                   int        `json:"id" db:"id"`
	UserID               int        `json:"user_id" db:"user_id"`
	Theme                string     `json:"theme" db:"theme"`
	DefaultTimeframe     string     `json:"default_timeframe" db:"default_timeframe"`
	ChartPreferences     []byte     `json:"chart_preferences" db:"chart_preferences"`
	NotificationSettings []byte     `json:"notification_settings" db:"notification_settings"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty" db:"updated_at"`
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
