package model

import (
	"encoding/json"
	"time"
)

// BacktestRequest represents a request to run a backtest
type BacktestRequest struct {
	StrategyID     int       `json:"strategy_id" validate:"required"`
	SymbolID       int       `json:"symbol_id" validate:"required"`
	TimeframeID    int       `json:"timeframe_id" validate:"required"`
	StartDate      time.Time `json:"start_date" validate:"required"`
	EndDate        time.Time `json:"end_date" validate:"required"`
	InitialCapital float64   `json:"initial_capital" validate:"required,gt=0"`
}

// BacktestResult represents the result of a backtest
type BacktestResult struct {
	ID             int             `json:"id"`
	StrategyID     int             `json:"strategy_id"`
	UserID         int             `json:"user_id"`
	SymbolID       int             `json:"symbol_id"`
	TimeframeID    int             `json:"timeframe_id"`
	StartDate      time.Time       `json:"start_date"`
	EndDate        time.Time       `json:"end_date"`
	InitialCapital float64         `json:"initial_capital"`
	FinalCapital   float64         `json:"final_capital"`
	TotalReturn    float64         `json:"total_return"`
	AnnualReturn   float64         `json:"annual_return"`
	Trades         int             `json:"trades"`
	WinRate        float64         `json:"win_rate"`
	Drawdown       float64         `json:"drawdown"`
	SharpeRatio    float64         `json:"sharpe_ratio"`
	Status         string          `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	TradeHistory   json.RawMessage `json:"trade_history,omitempty"`
	EquityCurve    json.RawMessage `json:"equity_curve,omitempty"`
}
