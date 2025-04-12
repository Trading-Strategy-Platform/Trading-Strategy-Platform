package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

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
	TagIDs       []int      `json:"-" db:"tag_ids"`
	Tags         []Tag      `json:"tags,omitempty" db:"-"`
	Structure    *Structure `json:"structure,omitempty" db:"-"`
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
		// Structure would need to be loaded separately
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

// Structure represents the rule structure of a strategy
type Structure struct {
	BuyRules  []Rule `json:"buyRules"`
	SellRules []Rule `json:"sellRules"`
}

// Value implements the driver.Valuer interface for Structure
func (s Structure) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface for Structure
func (s *Structure) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, s)
}

// Rule represents a single rule or group of rules in a strategy
type Rule struct {
	Type              string                 `json:"type"` // "rule" or "group"
	Indicator         *Indicator             `json:"indicator,omitempty"`
	Condition         *Condition             `json:"condition,omitempty"`
	Value             string                 `json:"value,omitempty"`
	IndicatorSettings map[string]interface{} `json:"indicatorSettings,omitempty"`
	Operator          string                 `json:"operator"`
	Rules             []Rule                 `json:"rules,omitempty"` // For type="group"
}

// Indicator represents a technical indicator reference in a rule
type Indicator struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Condition represents a comparison condition in a rule
type Condition struct {
	Symbol string `json:"symbol"` // ">=", "<=", ">", "<", "==", etc.
}

// BacktestRequest represents a request to backtest a strategy
type BacktestRequest struct {
	StrategyID     int       `json:"strategy_id" binding:"required"`
	SymbolID       int       `json:"symbol_id" binding:"required"`
	TimeframeID    int       `json:"timeframe_id" binding:"required"`
	StartDate      time.Time `json:"start_date" binding:"required"`
	EndDate        time.Time `json:"end_date" binding:"required"`
	InitialCapital float64   `json:"initial_capital" binding:"required,min=1"`
}

// Signal represents a trading signal generated by a strategy
type Signal struct {
	Type     string    `json:"type"` // "buy" or "sell"
	Time     time.Time `json:"time"`
	Price    float64   `json:"price"`
	BarIndex int       `json:"bar_index"`
}
