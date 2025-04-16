package model

import (
	"time"
)

// Candle represents a price candle returned by the get_candles function
type Candle struct {
	SymbolID int       `json:"symbol_id" db:"symbol_id"`
	Time     time.Time `json:"time" db:"candle_time"`
	Open     float64   `json:"open" db:"open"`
	High     float64   `json:"high" db:"high"`
	Low      float64   `json:"low" db:"low"`
	Close    float64   `json:"close" db:"close"`
	Volume   float64   `json:"volume" db:"volume"`
}

// CandleBatch represents a batch of candles for database import
type CandleBatch struct {
	SymbolID int       `json:"symbol_id"`
	Time     time.Time `json:"candle_time"` // Changed from "time" to "candle_time" to match SQL function
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Close    float64   `json:"close"`
	Volume   float64   `json:"volume"`
}

// MarketDataQuery represents a query for candle data
type MarketDataQuery struct {
	SymbolID  int        `json:"symbol_id" form:"symbol_id" binding:"required"`
	Timeframe string     `json:"timeframe" form:"timeframe" binding:"required"`
	StartDate *time.Time `json:"start_date" form:"start_date"`
	EndDate   *time.Time `json:"end_date" form:"end_date"`
	Limit     *int       `json:"limit" form:"limit"`
}

// DateRange represents a range of dates
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
