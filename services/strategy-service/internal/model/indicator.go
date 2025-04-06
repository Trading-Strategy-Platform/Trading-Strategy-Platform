package model

import (
	"time"
)

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

// IndicatorParameterCreate represents the data needed to create a parameter
type IndicatorParameterCreate struct {
	IndicatorID   int                        `json:"indicator_id"`
	ParameterName string                     `json:"parameter_name" binding:"required"`
	ParameterType string                     `json:"parameter_type" binding:"required"`
	IsRequired    bool                       `json:"is_required"`
	MinValue      *float64                   `json:"min_value,omitempty"`
	MaxValue      *float64                   `json:"max_value,omitempty"`
	DefaultValue  string                     `json:"default_value,omitempty"`
	Description   string                     `json:"description,omitempty"`
	EnumValues    []ParameterEnumValueCreate `json:"enum_values,omitempty"`
}

// ParameterEnumValueCreate represents the data needed to create an enum value
type ParameterEnumValueCreate struct {
	ParameterID int    `json:"parameter_id"`
	EnumValue   string `json:"enum_value" binding:"required"`
	DisplayName string `json:"display_name,omitempty"`
}
