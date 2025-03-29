package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Candle represents a price candle returned by the get_candles function
type Candle struct {
	SymbolID int       `json:"symbol_id" db:"symbol_id"`
	Time     time.Time `json:"time" db:"time"`
	Open     float64   `json:"open" db:"open"`
	High     float64   `json:"high" db:"high"`
	Low      float64   `json:"low" db:"low"`
	Close    float64   `json:"close" db:"close"`
	Volume   float64   `json:"volume" db:"volume"`
}

// CandleBatch represents a batch of candles for database import
type CandleBatch struct {
	SymbolID int       `json:"symbol_id"`
	Time     time.Time `json:"time"`
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Close    float64   `json:"close"`
	Volume   float64   `json:"volume"`
}

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

// Timeframe represents a data timeframe (1m, 5m, 1h, 1d, etc)
type Timeframe struct {
	ID          int        `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Minutes     int        `json:"minutes" db:"minutes"`
	DisplayName string     `json:"display_name" db:"display_name"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// BacktestSummary represents the summary view of a backtest
type BacktestSummary struct {
	BacktestID    int             `json:"backtest_id" db:"backtest_id"`
	Name          string          `json:"name" db:"name"`
	StrategyName  string          `json:"strategy_name" db:"strategy_name"`
	Date          time.Time       `json:"date" db:"date"`
	Status        string          `json:"status" db:"status"`
	SymbolResults json.RawMessage `json:"symbol_results" db:"symbol_results"`
	CompletedRuns int             `json:"completed_runs" db:"completed_runs"`
	TotalRuns     int             `json:"total_runs" db:"total_runs"`
}

// BacktestDetails represents the detailed view of a backtest
type BacktestDetails struct {
	BacktestID      int             `json:"backtest_id" db:"backtest_id"`
	Name            string          `json:"name" db:"name"`
	Description     string          `json:"description" db:"description"`
	StrategyName    string          `json:"strategy_name" db:"strategy_name"`
	StrategyID      int             `json:"strategy_id" db:"strategy_id"`
	StrategyVersion int             `json:"strategy_version" db:"strategy_version"`
	Timeframe       string          `json:"timeframe" db:"timeframe"`
	StartDate       time.Time       `json:"start_date" db:"start_date"`
	EndDate         time.Time       `json:"end_date" db:"end_date"`
	InitialCapital  float64         `json:"initial_capital" db:"initial_capital"`
	Status          string          `json:"status" db:"status"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	RunResults      json.RawMessage `json:"run_results" db:"run_results"`
}

// BacktestResults represents the performance results of a backtest
type BacktestResults struct {
	TotalTrades      int             `json:"total_trades" binding:"required"`
	WinningTrades    int             `json:"winning_trades" binding:"required"`
	LosingTrades     int             `json:"losing_trades" binding:"required"`
	ProfitFactor     float64         `json:"profit_factor" binding:"required"`
	SharpeRatio      float64         `json:"sharpe_ratio" binding:"required"`
	MaxDrawdown      float64         `json:"max_drawdown" binding:"required"`
	FinalCapital     float64         `json:"final_capital" binding:"required"`
	TotalReturn      float64         `json:"total_return" binding:"required"`
	AnnualizedReturn float64         `json:"annualized_return" binding:"required"`
	ResultsJSON      json.RawMessage `json:"results_json" binding:"required"`
}

// Value implements the driver.Valuer interface for BacktestResults
func (r BacktestResults) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// Scan implements the sql.Scanner interface for BacktestResults
func (r *BacktestResults) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &r)
}

// BacktestTrade represents a single trade in a backtest run
type BacktestTrade struct {
	ID                int        `json:"id,omitempty" db:"id"`
	BacktestRunID     int        `json:"backtest_run_id" db:"backtest_run_id"`
	SymbolID          int        `json:"symbol_id" db:"symbol_id" binding:"required"`
	Symbol            string     `json:"symbol,omitempty" db:"symbol"`
	EntryTime         time.Time  `json:"entry_time" db:"entry_time" binding:"required"`
	ExitTime          *time.Time `json:"exit_time,omitempty" db:"exit_time"`
	PositionType      string     `json:"position_type" db:"position_type" binding:"required"`
	EntryPrice        float64    `json:"entry_price" db:"entry_price" binding:"required"`
	ExitPrice         *float64   `json:"exit_price,omitempty" db:"exit_price"`
	Quantity          float64    `json:"quantity" db:"quantity" binding:"required"`
	ProfitLoss        *float64   `json:"profit_loss,omitempty" db:"profit_loss"`
	ProfitLossPercent *float64   `json:"profit_loss_percent,omitempty" db:"profit_loss_percent"`
	ExitReason        *string    `json:"exit_reason,omitempty" db:"exit_reason"`
}

// BacktestRequest represents the input parameters for a backtest
type BacktestRequest struct {
	StrategyID      int       `json:"strategy_id" binding:"required"`
	StrategyVersion int       `json:"strategy_version,omitempty"`
	Name            string    `json:"name,omitempty"`
	Description     string    `json:"description,omitempty"`
	Timeframe       string    `json:"timeframe" binding:"required"`
	SymbolIDs       []int     `json:"symbol_ids" binding:"required,min=1"`
	StartDate       time.Time `json:"start_date" binding:"required"`
	EndDate         time.Time `json:"end_date" binding:"required"`
	InitialCapital  float64   `json:"initial_capital" binding:"required,min=1"`
}

// MarketDataQuery represents a query for candle data
type MarketDataQuery struct {
	SymbolID  int        `json:"symbol_id" form:"symbol_id" binding:"required"`
	Timeframe string     `json:"timeframe" form:"timeframe" binding:"required"`
	StartDate *time.Time `json:"start_date" form:"start_date"`
	EndDate   *time.Time `json:"end_date" form:"end_date"`
	Limit     *int       `json:"limit" form:"limit"`
}

// SymbolFilter represents filter parameters for symbol queries
type SymbolFilter struct {
	SearchTerm string `json:"search_term" form:"search_term"`
	AssetType  string `json:"asset_type" form:"asset_type"`
	Exchange   string `json:"exchange" form:"exchange"`
}
