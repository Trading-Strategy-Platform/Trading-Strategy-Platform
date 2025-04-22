package model

import (
	"time"
)

// StrategyVersion represents a version of a strategy
type StrategyVersion struct {
	ID              int       `json:"id" db:"id"`
	StrategyID      int       `json:"strategy_id" db:"strategy_id"`
	Version         int       `json:"version" db:"version"`
	Structure       Structure `json:"structure" db:"structure"`
	ChangeNotes     string    `json:"change_notes" db:"change_notes"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	IsActiveVersion bool      `json:"is_active_version,omitempty"`
}

// VersionCreate represents data needed to create a new version
type VersionCreate struct {
	Structure   Structure `json:"structure" binding:"required"`
	ChangeNotes string    `json:"change_notes"`
}
