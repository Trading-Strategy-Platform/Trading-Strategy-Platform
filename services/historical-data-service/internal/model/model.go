package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// MarketData represents a single OHLCV candle for a symbol/timeframe
type MarketData struct {
	ID          int        `json:"id" db:"id"`
	SymbolID    int        `json:"symbol_id" db:"symbol_id"`
	TimeframeID int        `json:"timeframe_id" db:"timeframe_id"`
	Timestamp   time.Time  `json:"timestamp" db:"timestamp"`
	Open        float64    `json:"open" db:"open"`
	High        float64    `json:"high" db:"high"`
	Low         float64    `json:"low" db:"low"`
	Close       float64    `json:"close" db:"close"`
	Volume      float64    `json:"volume" db:"volume"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" db:"updated_at"`
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

// BacktestStatus represents the status of a backtest
type BacktestStatus string

const (
	BacktestStatusQueued    BacktestStatus = "queued"
	BacktestStatusRunning   BacktestStatus = "running"
	BacktestStatusCompleted BacktestStatus = "completed"
	BacktestStatusFailed    BacktestStatus = "failed"
)

// Backtest represents a strategy backtest
type Backtest struct {
	ID              int              `json:"id" db:"id"`
	UserID          int              `json:"user_id" db:"user_id"`
	StrategyID      int              `json:"strategy_id" db:"strategy_id"`
	StrategyName    string           `json:"strategy_name" db:"strategy_name"`
	StrategyVersion int              `json:"strategy_version" db:"strategy_version"`
	SymbolID        int              `json:"symbol_id" db:"symbol_id"`
	TimeframeID     int              `json:"timeframe_id" db:"timeframe_id"`
	StartDate       time.Time        `json:"start_date" db:"start_date"`
	EndDate         time.Time        `json:"end_date" db:"end_date"`
	InitialCapital  float64          `json:"initial_capital" db:"initial_capital"`
	Status          BacktestStatus   `json:"status" db:"status"`
	Results         *BacktestResults `json:"results,omitempty" db:"results"`
	ErrorMessage    *string          `json:"error_message,omitempty" db:"error_message"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       *time.Time       `json:"updated_at,omitempty" db:"updated_at"`
	CompletedAt     *time.Time       `json:"completed_at,omitempty" db:"completed_at"`

	// Additional fields populated on response
	SymbolInfo    *Symbol    `json:"symbol_info,omitempty" db:"-"`
	TimeframeInfo *Timeframe `json:"timeframe_info,omitempty" db:"-"`
}

// BacktestResults represents the results of a backtest
type BacktestResults struct {
	NetProfit          float64   `json:"net_profit"`
	ProfitFactor       float64   `json:"profit_factor"`
	TotalTrades        int       `json:"total_trades"`
	WinningTrades      int       `json:"winning_trades"`
	LosingTrades       int       `json:"losing_trades"`
	WinRate            float64   `json:"win_rate"`
	MaxDrawdown        float64   `json:"max_drawdown"`
	MaxDrawdownPercent float64   `json:"max_drawdown_percent"`
	CAGR               float64   `json:"cagr"`
	SharpeRatio        float64   `json:"sharpe_ratio"`
	SortinoRatio       float64   `json:"sortino_ratio"`
	Trades             []Trade   `json:"trades"`
	EquityCurve        []float64 `json:"equity_curve"`
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

// Trade represents a single trade in a backtest
type Trade struct {
	EntryTime     time.Time `json:"entry_time"`
	ExitTime      time.Time `json:"exit_time"`
	EntryPrice    float64   `json:"entry_price"`
	ExitPrice     float64   `json:"exit_price"`
	Direction     string    `json:"direction"` // "long" or "short"
	Quantity      float64   `json:"quantity"`
	ProfitLoss    float64   `json:"profit_loss"`
	ProfitLossPct float64   `json:"profit_loss_pct"`
}

// BacktestRequest represents the input parameters for a backtest
type BacktestRequest struct {
	StrategyID      int       `json:"strategy_id" binding:"required"`
	StrategyVersion int       `json:"strategy_version,omitempty"`
	SymbolID        int       `json:"symbol_id" binding:"required"`
	TimeframeID     int       `json:"timeframe_id" binding:"required"`
	StartDate       time.Time `json:"start_date" binding:"required"`
	EndDate         time.Time `json:"end_date" binding:"required"`
	InitialCapital  float64   `json:"initial_capital" binding:"required,min=1"`
}

// MarketDataImport represents data for importing market data
type MarketDataImport struct {
	SymbolID    int     `json:"symbol_id" binding:"required"`
	TimeframeID int     `json:"timeframe_id" binding:"required"`
	Data        []OHLCV `json:"data" binding:"required"`
}

// OHLCV represents a single candlestick record
type OHLCV struct {
	Timestamp time.Time `json:"timestamp" binding:"required"`
	Open      float64   `json:"open" binding:"required"`
	High      float64   `json:"high" binding:"required"`
	Low       float64   `json:"low" binding:"required"`
	Close     float64   `json:"close" binding:"required"`
	Volume    float64   `json:"volume" binding:"required"`
}

// MarketDataQuery represents a query for market data
type MarketDataQuery struct {
	SymbolID    int        `json:"symbol_id" form:"symbol_id" binding:"required"`
	TimeframeID int        `json:"timeframe_id" form:"timeframe_id" binding:"required"`
	StartDate   *time.Time `json:"start_date" form:"start_date"`
	EndDate     *time.Time `json:"end_date" form:"end_date"`
	Limit       *int       `json:"limit" form:"limit"`
}
