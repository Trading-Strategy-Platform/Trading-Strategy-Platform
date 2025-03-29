package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

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
	return json.Unmarshal(b, &s)
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

// StrategyVersion represents a version of a strategy
type StrategyVersion struct {
	ID          int       `json:"id" db:"id"`
	StrategyID  int       `json:"strategy_id" db:"strategy_id"`
	Version     int       `json:"version" db:"version"`
	Structure   Structure `json:"structure" db:"structure"`
	ChangeNotes string    `json:"change_notes" db:"change_notes"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// VersionCreate represents data needed to create a new version
type VersionCreate struct {
	Structure   Structure `json:"structure" binding:"required"`
	ChangeNotes string    `json:"change_notes"`
}

// Tag represents a strategy tag
type Tag struct {
	ID   int    `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// StrategyResponse represents the response for a strategy with additional metadata
type StrategyResponse struct {
	Strategy       Strategy `json:"strategy"`
	VersionsCount  int      `json:"versions_count"`
	CreatorName    string   `json:"creator_name"`
	BacktestsCount int      `json:"backtests_count,omitempty"`
}

// IndicatorParameter represents a parameter for a technical indicator
type IndicatorParameter struct {
	ID            int                  `json:"id" db:"id"`
	IndicatorID   int                  `json:"indicator_id" db:"indicator_id"`
	ParameterName string               `json:"parameter_name" db:"parameter_name"`
	ParameterType string               `json:"parameter_type" db:"parameter_type"`
	IsRequired    bool                 `json:"is_required" db:"is_required"`
	MinValue      *float64             `json:"min_value,omitempty" db:"min_value"`
	MaxValue      *float64             `json:"max_value,omitempty" db:"max_value"`
	DefaultValue  string               `json:"default_value,omitempty" db:"default_value"`
	Description   string               `json:"description,omitempty" db:"description"`
	EnumValues    []ParameterEnumValue `json:"enum_values,omitempty" db:"-"`
}

// ParameterEnumValue represents a predefined value for an enum parameter
type ParameterEnumValue struct {
	ID          int    `json:"id" db:"id"`
	ParameterID int    `json:"parameter_id" db:"parameter_id"`
	EnumValue   string `json:"enum_value" db:"enum_value"`
	DisplayName string `json:"display_name" db:"display_name"`
}

// TechnicalIndicator represents a technical indicator definition
type TechnicalIndicator struct {
	ID          int                  `json:"id" db:"id"`
	Name        string               `json:"name" db:"name"`
	Description string               `json:"description" db:"description"`
	Category    string               `json:"category" db:"category"`
	Formula     string               `json:"formula" db:"formula"`
	CreatedAt   time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time           `json:"updated_at,omitempty" db:"updated_at"`
	Parameters  []IndicatorParameter `json:"parameters,omitempty" db:"-"`
}

// MarketplaceItem represents a strategy listed in the marketplace
type MarketplaceItem struct {
	ID                 int        `json:"id" db:"id"`
	StrategyID         int        `json:"strategy_id" db:"strategy_id"`
	UserID             int        `json:"user_id" db:"user_id"`
	Price              float64    `json:"price" db:"price"`
	IsSubscription     bool       `json:"is_subscription" db:"is_subscription"`
	SubscriptionPeriod string     `json:"subscription_period,omitempty" db:"subscription_period"`
	IsActive           bool       `json:"is_active" db:"is_active"`
	Description        string     `json:"description" db:"description"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty" db:"updated_at"`

	// Additional fields for responses
	Strategy       *Strategy `json:"strategy,omitempty" db:"-"`
	CreatorName    string    `json:"creator_name,omitempty" db:"-"`
	AverageRating  float64   `json:"average_rating,omitempty" db:"-"`
	ReviewsCount   int       `json:"reviews_count,omitempty" db:"-"`
	PurchasesCount int       `json:"purchases_count,omitempty" db:"-"`
}

// StrategyPurchase represents a purchase of a strategy from the marketplace
type StrategyPurchase struct {
	ID              int        `json:"id" db:"id"`
	MarketplaceID   int        `json:"marketplace_id" db:"marketplace_id"`
	BuyerID         int        `json:"buyer_id" db:"buyer_id"`
	PurchasePrice   float64    `json:"purchase_price" db:"purchase_price"`
	SubscriptionEnd *time.Time `json:"subscription_end,omitempty" db:"subscription_end"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// StrategyReview represents a review of a purchased strategy
type StrategyReview struct {
	ID            int        `json:"id" db:"id"`
	MarketplaceID int        `json:"marketplace_id" db:"marketplace_id"`
	UserID        int        `json:"user_id" db:"user_id"`
	Rating        int        `json:"rating" db:"rating"`
	Comment       string     `json:"comment" db:"comment"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty" db:"updated_at"`

	// Additional fields for responses
	UserName string `json:"user_name,omitempty" db:"-"`
}

// ReviewCreate represents data needed to create a strategy review
type ReviewCreate struct {
	MarketplaceID int    `json:"marketplace_id" binding:"required"`
	Rating        int    `json:"rating" binding:"required,min=1,max=5"`
	Comment       string `json:"comment"`
}

// Signal represents a trading signal generated by a strategy
type Signal struct {
	Type     string    `json:"type"` // "buy" or "sell"
	Time     time.Time `json:"time"`
	Price    float64   `json:"price"`
	BarIndex int       `json:"bar_index"`
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

// Strategy - Make sure IsActive field is included
type Strategy struct {
	ID           int        `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	UserID       int        `json:"user_id" db:"user_id"`
	Description  string     `json:"description" db:"description"`
	ThumbnailURL string     `json:"thumbnail_url" db:"thumbnail_url"`
	Structure    Structure  `json:"structure" db:"structure"`
	IsPublic     bool       `json:"is_public" db:"is_public"`
	IsActive     bool       `json:"is_active" db:"is_active"` // Added field
	Version      int        `json:"version" db:"version"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	Tags         []Tag      `json:"tags,omitempty" db:"-"`
}

// StrategyCreate - Make sure IsActive field is included
type StrategyCreate struct {
	Name         string    `json:"name" binding:"required"`
	Description  string    `json:"description"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Structure    Structure `json:"structure" binding:"required"`
	IsPublic     bool      `json:"is_public"`
	IsActive     bool      `json:"is_active"` // Added field, defaulted to true in service
	Tags         []int     `json:"tags,omitempty"`
}

// StrategyUpdate - Make sure IsActive field is included
type StrategyUpdate struct {
	Name         *string    `json:"name"`
	Description  *string    `json:"description"`
	ThumbnailURL *string    `json:"thumbnail_url"`
	Structure    *Structure `json:"structure"`
	IsPublic     *bool      `json:"is_public"`
	IsActive     *bool      `json:"is_active"` // Added field
	Tags         []int      `json:"tags,omitempty"`
}

// ExtendedStrategy - Added to map SQL function output directly
type ExtendedStrategy struct {
	ID            int        `json:"id" db:"id"`
	Name          string     `json:"name" db:"name"`
	Description   string     `json:"description" db:"description"`
	ThumbnailURL  string     `json:"thumbnail_url" db:"thumbnail_url"`
	OwnerID       int        `json:"owner_id" db:"owner_id"`
	OwnerUsername string     `json:"owner_username" db:"owner_username"`
	IsPublic      bool       `json:"is_public" db:"is_public"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	Version       int        `json:"version" db:"version"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	AccessType    string     `json:"access_type" db:"access_type"`
	PurchaseID    *int       `json:"purchase_id,omitempty" db:"purchase_id"`
	PurchaseDate  *time.Time `json:"purchase_date,omitempty" db:"purchase_date"`
	TagIDs        []int      `json:"-" db:"tag_ids"`
	Tags          []Tag      `json:"tags,omitempty" db:"-"`
	Structure     *Structure `json:"structure,omitempty" db:"-"`
}

// MarketplaceCreate - Add missing VersionID field
type MarketplaceCreate struct {
	StrategyID         int     `json:"strategy_id" binding:"required"`
	VersionID          int     `json:"version_id" binding:"required"` // Added field
	Price              float64 `json:"price" binding:"required,min=0"`
	IsSubscription     bool    `json:"is_subscription"`
	SubscriptionPeriod string  `json:"subscription_period,omitempty"`
	Description        string  `json:"description"`
}
