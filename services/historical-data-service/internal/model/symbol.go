package model

import (
	"time"
)

// Symbol represents a tradable market symbol
type Symbol struct {
	ID            int        `json:"id" db:"id"`
	Symbol        string     `json:"symbol" db:"symbol"`
	Name          string     `json:"name" db:"name"`
	Exchange      string     `json:"exchange" db:"exchange"`
	AssetType     string     `json:"asset_type" db:"asset_type"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	DataAvailable bool       `json:"data_available" db:"data_available"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// SymbolFilter represents filter parameters for symbol queries
type SymbolFilter struct {
	SearchTerm string `json:"search_term" form:"search_term"`
	AssetType  string `json:"asset_type" form:"asset_type"`
	Exchange   string `json:"exchange" form:"exchange"`
}

// Timeframe represents a data timeframe (1m, 5m, 1h, 1d, etc)
type Timeframe struct {
	ID          int        `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Minutes     int        `json:"minutes" db:"minutes"`
	DisplayName string     `json:"display_name" db:"display_name"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}
