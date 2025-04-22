package model

import (
	"encoding/json"
	"time"
)

// Strategy represents a trading strategy with version information
type Strategy struct {
	ID              int             `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	UserID          int             `json:"user_id" db:"user_id"`
	Description     string          `json:"description" db:"description"`
	ThumbnailURL    string          `json:"thumbnail_url" db:"thumbnail_url"`
	Structure       json.RawMessage `json:"structure" db:"structure"`
	IsPublic        bool            `json:"is_public" db:"is_public"`
	IsActive        bool            `json:"is_active" db:"is_active"`
	Version         int             `json:"version" db:"version"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       *time.Time      `json:"updated_at,omitempty" db:"updated_at"`
	StrategyGroupID int             `json:"strategy_group_id" db:"strategy_group_id"`

	// Additional fields not in DB but used in responses
	Username         string     `json:"username,omitempty" db:"-"`
	Tags             []Tag      `json:"tags,omitempty" db:"-"`
	TagIDs           []int      `json:"tag_ids,omitempty" db:"-"`
	IsCurrentVersion bool       `json:"is_current_version,omitempty" db:"-"`
	AccessType       string     `json:"access_type,omitempty" db:"-"`
	PurchaseID       *int       `json:"purchase_id,omitempty" db:"-"`
	PurchaseDate     *time.Time `json:"purchase_date,omitempty" db:"-"`
}

// StrategyCreate represents the data needed to create a new strategy
type StrategyCreate struct {
	Name         string          `json:"name" binding:"required"`
	Description  string          `json:"description"`
	ThumbnailURL string          `json:"thumbnail_url"`
	Structure    json.RawMessage `json:"structure" binding:"required"`
	IsPublic     bool            `json:"is_public"`
	TagIDs       []int           `json:"tag_ids,omitempty"`
}

// StrategyUpdate represents the data needed to update a strategy (create new version)
type StrategyUpdate struct {
	Name         string          `json:"name" binding:"required"`
	Description  string          `json:"description"`
	ThumbnailURL string          `json:"thumbnail_url"`
	Structure    json.RawMessage `json:"structure" binding:"required"`
	IsPublic     bool            `json:"is_public"`
	ChangeNotes  string          `json:"change_notes"`
	TagIDs       []int           `json:"tag_ids,omitempty"`
}

// BacktestRequest represents the data needed to backtest a strategy
type BacktestRequest struct {
	StrategyID     int       `json:"strategy_id" binding:"required"`
	SymbolID       int       `json:"symbol_id" binding:"required"`
	TimeframeID    int       `json:"timeframe_id" binding:"required"`
	StartDate      time.Time `json:"start_date" binding:"required"`
	EndDate        time.Time `json:"end_date" binding:"required"`
	InitialCapital float64   `json:"initial_capital" binding:"required"`
}
