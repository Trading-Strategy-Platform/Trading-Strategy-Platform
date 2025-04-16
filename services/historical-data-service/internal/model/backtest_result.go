package model

import (
	"encoding/json"
	"time"
)

// BacktestResult represents the complete result of a backtest run
type BacktestResult struct {
	Trades      []BacktestTrade  `json:"trades"`
	EquityCurve []float64        `json:"equity_curve"`
	EquityTimes []string         `json:"equity_times"`
	Metrics     BacktestMetrics  `json:"metrics"`
	ResultsJSON *json.RawMessage `json:"results_json,omitempty"`
}

// BacktestMetrics represents performance metrics from a backtest
type BacktestMetrics struct {
	TotalTrades      int     `json:"total_trades"`
	WinningTrades    int     `json:"winning_trades"`
	LosingTrades     int     `json:"losing_trades"`
	WinRate          float64 `json:"win_rate"`
	ProfitFactor     float64 `json:"profit_factor"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	FinalCapital     float64 `json:"final_capital"`
	TotalReturn      float64 `json:"total_return"`
	AnnualizedReturn float64 `json:"annualized_return"`
	AverageTrade     float64 `json:"average_trade"`
	AverageWin       float64 `json:"average_win"`
	AverageLoss      float64 `json:"average_loss"`
	LargestWin       float64 `json:"largest_win"`
	LargestLoss      float64 `json:"largest_loss"`
}

// IndicatorParameter represents a parameter for a technical indicator
type IndicatorParameter struct {
	Name    string   `json:"name"`
	Default string   `json:"default"`
	Type    string   `json:"type"`
	Options []string `json:"options,omitempty"`
}

// Indicator represents a technical indicator available for strategies
type Indicator struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Parameters  []IndicatorParameter `json:"parameters"`
}

// BacktestParameters contains all parameters needed to run a backtest
type BacktestParameters struct {
	// Basic parameters
	SymbolID        int       `json:"symbol_id"`
	StrategyID      int       `json:"strategy_id"`
	StrategyVersion int       `json:"strategy_version"`
	InitialCapital  float64   `json:"initial_capital"`
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	Timeframe       string    `json:"timeframe"`

	// Trading parameters
	MarketType     string  `json:"market_type"`     // "spot" or "futures"
	Leverage       float64 `json:"leverage"`        // For futures trading
	CommissionRate float64 `json:"commission_rate"` // In percentage
	SlippageRate   float64 `json:"slippage_rate"`   // In percentage
	AllowShort     bool    `json:"allow_short"`     // Allow short positions

	// Risk management
	PositionSizing string  `json:"position_sizing"` // "fixed", "percentage", "risk_based"
	RiskPercentage float64 `json:"risk_percentage"` // For percentage and risk-based sizing
	StopLoss       float64 `json:"stop_loss"`       // In percentage
	TakeProfit     float64 `json:"take_profit"`     // In percentage
	TrailingStop   float64 `json:"trailing_stop"`   // In percentage
}
