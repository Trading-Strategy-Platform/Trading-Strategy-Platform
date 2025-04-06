package model

import (
	"time"
)

// BinanceExchangeInfo represents the exchange information from Binance API
type BinanceExchangeInfo struct {
	Timezone   string          `json:"timezone"`
	ServerTime int64           `json:"serverTime"`
	Symbols    []BinanceSymbol `json:"symbols"`
}

// BinanceSymbol represents a trading symbol from Binance API
type BinanceSymbol struct {
	Symbol                 string   `json:"symbol"`
	Status                 string   `json:"status"`
	BaseAsset              string   `json:"baseAsset"`
	QuoteAsset             string   `json:"quoteAsset"`
	BaseAssetPrecision     int      `json:"baseAssetPrecision"`
	QuotePrecision         int      `json:"quotePrecision"`
	QuoteAssetPrecision    int      `json:"quoteAssetPrecision"`
	OrderTypes             []string `json:"orderTypes"`
	IsSpotTradingAllowed   bool     `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed bool     `json:"isMarginTradingAllowed"`
}

// BinanceKline represents a candlestick from Binance API
type BinanceKline struct {
	OpenTime  time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime time.Time
}

// SymbolDataStatus represents the status of a symbol's data
type SymbolDataStatus struct {
	Symbol        string      `json:"symbol"`
	SymbolID      int         `json:"symbol_id"`
	HasData       bool        `json:"has_data"`
	AvailableData []DateRange `json:"available_data"`
	MissingData   []DateRange `json:"missing_data"`
}

// BinanceDownloadRequest represents a request to download data from Binance
type BinanceDownloadRequest struct {
	Symbol    string    `json:"symbol" binding:"required"`
	Timeframe string    `json:"timeframe" binding:"required"`
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
}

// BinanceDownloadJob represents a job to download data from Binance
type BinanceDownloadJob struct {
	ID        int       `json:"id" db:"id"`
	Symbol    string    `json:"symbol" db:"symbol"`
	SymbolID  int       `json:"symbol_id" db:"symbol_id"`
	Timeframe string    `json:"timeframe" db:"timeframe"`
	StartDate time.Time `json:"start_date" db:"start_date"`
	EndDate   time.Time `json:"end_date" db:"end_date"`
	Status    string    `json:"status" db:"status"`
	Progress  float64   `json:"progress" db:"progress"`
	Error     string    `json:"error,omitempty" db:"error"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// BinanceDownloadStatus represents the status of a download job
type BinanceDownloadStatus struct {
	JobID     int       `json:"job_id"`
	Symbol    string    `json:"symbol"`
	Status    string    `json:"status"`
	Progress  float64   `json:"progress"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at"`
	Timeframe string    `json:"timeframe"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}
