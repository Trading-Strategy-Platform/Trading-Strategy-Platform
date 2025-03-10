// services/strategy-service/internal/model/strategy.go
package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Strategy represents a trading strategy
type Strategy struct {
	ID          int        `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	UserID      int        `json:"user_id" db:"user_id"`
	Description string     `json:"description" db:"description"`
	Structure   Structure  `json:"structure" db:"structure"`
	IsPublic    bool       `json:"is_public" db:"is_public"`
	Version     int        `json:"version" db:"version"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	Tags        []Tag      `json:"tags,omitempty" db:"-"`
}

// StrategyResponse represents the response for a strategy with additional metadata
type StrategyResponse struct {
	Strategy       Strategy `json:"strategy"`
	AuthorUsername string   `json:"author_username"`
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

// Condition represents a condition in a rule
type Condition struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

// StrategyCreate represents data needed to create a new strategy
type StrategyCreate struct {
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	Structure   Structure `json:"structure" binding:"required"`
	IsPublic    bool      `json:"is_public"`
	Tags        []int     `json:"tags,omitempty"`
}

// StrategyUpdate represents the fields that can be updated for a strategy
type StrategyUpdate struct {
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	IsPublic    *bool      `json:"is_public,omitempty"`
	Structure   *Structure `json:"structure,omitempty"`
	Tags        []int      `json:"tags,omitempty"`
	Notes       string     `json:"notes,omitempty"` // Change notes for version history
}

// StrategyVersion represents a version of a strategy
type StrategyVersion struct {
	ID          int        `json:"id" db:"id"`
	StrategyID  int        `json:"strategy_id" db:"strategy_id"`
	Version     int        `json:"version" db:"version"`
	Structure   Structure  `json:"structure" db:"structure"`
	ChangeNotes string     `json:"change_notes" db:"change_notes"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// VersionCreate represents data needed to create a new strategy version
type VersionCreate struct {
	Structure   Structure `json:"structure" binding:"required"`
	ChangeNotes string    `json:"change_notes"`
}

// Tag represents a strategy tag
type Tag struct {
	ID   int    `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// TagCreate represents data needed to create a new tag
type TagCreate struct {
	Name string `json:"name" binding:"required"`
}

// IndicatorParameter represents a parameter for a technical indicator
type IndicatorParameter struct {
	ID            int                  `json:"id" db:"id"`
	IndicatorID   int                  `json:"indicator_id" db:"indicator_id"`
	Name          string               `json:"name" db:"name"`
	ParameterType string               `json:"parameter_type" db:"parameter_type"`
	DefaultValue  string               `json:"default_value" db:"default_value"`
	MinValue      string               `json:"min_value,omitempty" db:"min_value"`
	MaxValue      string               `json:"max_value,omitempty" db:"max_value"`
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

// MarketplaceCreate represents data needed to create a marketplace listing
type MarketplaceCreate struct {
	StrategyID         int     `json:"strategy_id" binding:"required"`
	Price              float64 `json:"price" binding:"required,min=0"`
	IsSubscription     bool    `json:"is_subscription"`
	SubscriptionPeriod string  `json:"subscription_period,omitempty"`
	Description        string  `json:"description"`
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

// Signal represents a trading signal generated by a strategy
type Signal struct {
	Type     string    `json:"type"` // "buy" or "sell"
	Time     time.Time `json:"time"`
	Price    float64   `json:"price"`
	BarIndex int       `json:"bar_index"`
}

// ReviewCreate represents data needed to create a strategy review
type ReviewCreate struct {
	MarketplaceID int    `json:"marketplace_id" binding:"required"`
	Rating        int    `json:"rating" binding:"required,min=1,max=5"`
	Comment       string `json:"comment"`
}

// StrategyReview represents a review of a strategy
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
