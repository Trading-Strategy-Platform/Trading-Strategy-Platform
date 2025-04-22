package model

import (
	"time"
)

// DataSource represents a source of market data
type DataSource string

const (
	// Data sources
	SourceBinance DataSource = "BINANCE"
	SourceYahoo   DataSource = "YAHOO"
	SourceIEX     DataSource = "IEX"
	// Add more sources as needed
)

// MarketDataDownloadRequest represents a request to download market data
type MarketDataDownloadRequest struct {
	Symbol    string    `json:"symbol" binding:"required"`
	Source    string    `json:"source" binding:"required"`
	Timeframe string    `json:"timeframe" binding:"required"`
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
}

// MarketDataDownloadJob represents a job to download market data
type MarketDataDownloadJob struct {
	ID                int        `json:"id" db:"id"`
	Symbol            string     `json:"symbol" db:"symbol"`
	SymbolID          int        `json:"symbol_id" db:"symbol_id"`
	Source            string     `json:"source" db:"source"`
	Timeframe         string     `json:"timeframe" db:"timeframe"`
	StartDate         time.Time  `json:"start_date" db:"start_date"`
	EndDate           time.Time  `json:"end_date" db:"end_date"`
	Status            string     `json:"status" db:"status"`
	Progress          float64    `json:"progress" db:"progress"`
	TotalCandles      int        `json:"total_candles" db:"total_candles"`
	ProcessedCandles  int        `json:"processed_candles" db:"processed_candles"`
	Retries           int        `json:"retries" db:"retries"`
	Error             string     `json:"error,omitempty" db:"error"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
	LastProcessedTime *time.Time `json:"last_processed_time,omitempty" db:"last_processed_time"`
}

// MarketDataDownloadStatus represents the status of a download job
type MarketDataDownloadStatus struct {
	JobID             int        `json:"job_id"`
	Symbol            string     `json:"symbol"`
	Source            string     `json:"source"`
	Status            string     `json:"status"`
	Progress          float64    `json:"progress"`
	ProcessedCandles  int        `json:"processed_candles"`
	TotalCandles      int        `json:"total_candles"`
	Retries           int        `json:"retries"`
	Error             string     `json:"error,omitempty"`
	StartedAt         time.Time  `json:"started_at"`
	Timeframe         string     `json:"timeframe"`
	StartDate         time.Time  `json:"start_date"`
	EndDate           time.Time  `json:"end_date"`
	LastProcessedTime *time.Time `json:"last_processed_time,omitempty"`
}

// SymbolDataStatus represents the status of a symbol's data
type SymbolDataStatus struct {
	Symbol        string      `json:"symbol"`
	SymbolID      int         `json:"symbol_id"`
	HasData       bool        `json:"has_data"`
	AvailableData []DateRange `json:"available_data"`
	MissingData   []DateRange `json:"missing_data"`
}

// DataInventoryItem represents an item in the data inventory
type DataInventoryItem struct {
	SymbolID            int       `db:"symbol_id" json:"symbol_id"`
	Symbol              string    `db:"symbol" json:"symbol"`
	Name                string    `db:"name" json:"name"`
	AssetType           string    `db:"asset_type" json:"asset_type"`
	Exchange            string    `db:"exchange" json:"exchange"`
	CandleCount         int64     `db:"candle_count" json:"candle_count"`
	EarliestDate        time.Time `db:"earliest_date" json:"earliest_date"`
	LatestDate          time.Time `db:"latest_date" json:"latest_date"`
	AvailableTimeframes []string  `db:"available_timeframes" json:"available_timeframes"`
}
