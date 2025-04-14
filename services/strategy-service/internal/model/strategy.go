package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

// Add this custom type for handling PostgreSQL integer arrays
type IntArray []int

// Scan implements the sql.Scanner interface for IntArray
func (a *IntArray) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return a.scanBytes(src)
	case string:
		return a.scanBytes([]byte(src))
	case nil:
		*a = nil
		return nil
	}

	return errors.New("cannot convert to []int")
}

// scanBytes handles the conversion from bytes to integer array
func (a *IntArray) scanBytes(src []byte) error {
	str := string(src)

	// Check for empty array
	if str == "{}" {
		*a = []int{}
		return nil
	}

	// Remove the curly braces
	str = strings.Trim(str, "{}")

	// Split by comma
	elements := strings.Split(str, ",")

	// Convert each element to integer
	*a = make([]int, len(elements))
	for i, element := range elements {
		val, err := strconv.Atoi(strings.TrimSpace(element))
		if err != nil {
			return err
		}
		(*a)[i] = val
	}

	return nil
}

// Strategy represents a trading strategy
type Strategy struct {
	ID           int        `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	UserID       int        `json:"user_id" db:"user_id"`
	Description  string     `json:"description" db:"description"`
	ThumbnailURL string     `json:"thumbnail_url" db:"thumbnail_url"`
	Structure    Structure  `json:"structure" db:"structure"`
	IsPublic     bool       `json:"is_public" db:"is_public"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	Version      int        `json:"version" db:"version"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	Tags         []Tag      `json:"tags,omitempty" db:"-"`
}

// ExtendedStrategy is used for the get_my_strategies SQL function output
type ExtendedStrategy struct {
	ID           int        `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Description  string     `json:"description" db:"description"`
	ThumbnailURL string     `json:"thumbnail_url" db:"thumbnail_url"`
	OwnerID      int        `json:"owner_id" db:"owner_id"`
	OwnerUserID  int        `json:"owner_username" db:"owner_user_id"`
	IsPublic     bool       `json:"is_public" db:"is_public"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	Version      int        `json:"version" db:"version"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	AccessType   string     `json:"access_type" db:"access_type"`
	PurchaseID   *int       `json:"purchase_id,omitempty" db:"purchase_id"`
	PurchaseDate *time.Time `json:"purchase_date,omitempty" db:"purchase_date"`
	TagIDs       IntArray   `json:"-" db:"tag_ids"`
	Tags         []Tag      `json:"tags,omitempty" db:"-"`
	Structure    Structure  `json:"structure" db:"structure"`
}

// ConvertToStrategy converts ExtendedStrategy to Strategy
func (es *ExtendedStrategy) ConvertToStrategy() *Strategy {
	return &Strategy{
		ID:           es.ID,
		Name:         es.Name,
		UserID:       es.OwnerID,
		Description:  es.Description,
		ThumbnailURL: es.ThumbnailURL,
		IsPublic:     es.IsPublic,
		IsActive:     es.IsActive,
		Version:      es.Version,
		CreatedAt:    es.CreatedAt,
		UpdatedAt:    es.UpdatedAt,
		Tags:         es.Tags,
		Structure:    es.Structure,
	}
}

// StrategyCreate represents data for creating a new strategy
type StrategyCreate struct {
	Name         string    `json:"name" binding:"required"`
	Description  string    `json:"description"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Structure    Structure `json:"structure" binding:"required"`
	IsPublic     bool      `json:"is_public"`
	IsActive     bool      `json:"is_active"` // Added field, defaulted to true in service
	Tags         []int     `json:"tags,omitempty"`
}

// StrategyUpdate represents data for updating an existing strategy
type StrategyUpdate struct {
	Name         *string    `json:"name"`
	Description  *string    `json:"description"`
	ThumbnailURL *string    `json:"thumbnail_url"`
	Structure    *Structure `json:"structure"`
	IsPublic     *bool      `json:"is_public"`
	IsActive     *bool      `json:"is_active"`
	Tags         []int      `json:"tags,omitempty"`
}

// StrategyResponse represents the response for a strategy with additional metadata
type StrategyResponse struct {
	Strategy       Strategy `json:"strategy"`
	VersionsCount  int      `json:"versions_count"`
	CreatorName    string   `json:"creator_name"`
	BacktestsCount int      `json:"backtests_count,omitempty"`
}

// Structure represents the raw JSON structure of a strategy
// It preserves the exact format and order of the original JSON
type Structure json.RawMessage

// MarshalJSON implements the json.Marshaler interface
// Simply returns the raw JSON bytes exactly as stored
func (s Structure) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		// Return default empty structure
		return []byte(`{"buyRules":{},"sellRules":{}}`), nil
	}

	// Return the raw JSON exactly as it was stored, preserving all order
	return []byte(s), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface
// Stores the raw JSON bytes exactly as provided
func (s *Structure) UnmarshalJSON(data []byte) error {
	// Store exact raw JSON data
	*s = make([]byte, len(data))
	copy(*s, data)
	return nil
}

// Value implements the driver.Valuer interface for database storage
func (s Structure) Value() (driver.Value, error) {
	if len(s) == 0 {
		// Return default empty structure
		return []byte(`{"buyRules":{},"sellRules":{}}`), nil
	}
	return []byte(s), nil
}

// Scan implements the sql.Scanner interface for database retrieval
func (s *Structure) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		// Store exact raw data
		*s = make([]byte, len(v))
		copy(*s, v)
		return nil
	case string:
		*s = []byte(v)
		return nil
	case nil:
		*s = []byte(`{"buyRules":{},"sellRules":{}}`)
		return nil
	default:
		return errors.New("unsupported type for Structure.Scan")
	}
}

// BacktestRequest represents a request to backtest a strategy
type BacktestRequest struct {
	StrategyID     int       `json:"strategy_id" binding:"required"`
	SymbolID       int       `json:"symbol_id" binding:"required"`
	TimeframeID    int       `json:"timeframe_id" binding:"required"`
	StartDate      time.Time `json:"start_date" binding:"required"`
	EndDate        time.Time `json:"end_date" binding:"required"`
	InitialCapital float64   `json:"initial_capital" binding:"required,min:1"`
}

// Signal represents a trading signal generated by a strategy
type Signal struct {
	Type     string    `json:"type"` // "buy" or "sell"
	Time     time.Time `json:"time"`
	Price    float64   `json:"price"`
	BarIndex int       `json:"bar_index"`
}
