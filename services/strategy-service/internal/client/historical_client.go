// services/strategy-service/internal/client/historical_client.go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"services/strategy-service/internal/model"

	"github.com/yourorg/trading-platform/shared/go/httpclient"
	"go.uber.org/zap"
)

// HistoricalClient handles communication with the Historical Data Service
type HistoricalClient struct {
	client *httpclient.Client
	logger *zap.Logger
}

// NewHistoricalClient creates a new Historical Data Service client
func NewHistoricalClient(baseURL string, logger *zap.Logger) *HistoricalClient {
	client := httpclient.New(httpclient.Config{
		BaseURL:    baseURL,
		Timeout:    30 * time.Second,
		ServiceKey: "strategy-service-key",
	}, logger)

	return &HistoricalClient{
		client: client,
		logger: logger,
	}
}

// CreateBacktest sends a backtest request to the Historical Data Service
func (c *HistoricalClient) CreateBacktest(ctx context.Context, request *model.BacktestRequest, userID int) (int, error) {
	// Create the request payload
	payload := struct {
		StrategyID      int             `json:"strategy_id"`
		StrategyVersion int             `json:"strategy_version"`
		SymbolID        int             `json:"symbol_id"`
		TimeframeID     int             `json:"timeframe_id"`
		StartDate       time.Time       `json:"start_date"`
		EndDate         time.Time       `json:"end_date"`
		InitialCapital  float64         `json:"initial_capital"`
		UserID          int             `json:"user_id"`
		Strategy        json.RawMessage `json:"strategy,omitempty"`
	}{
		StrategyID:      request.StrategyID,
		StrategyVersion: 1, // Default to latest version
		SymbolID:        request.SymbolID,
		TimeframeID:     request.TimeframeID,
		StartDate:       request.StartDate,
		EndDate:         request.EndDate,
		InitialCapital:  request.InitialCapital,
		UserID:          userID,
	}

	// Add service key to context
	serviceCtx := context.WithValue(ctx, "service_key", "strategy-service-key")

	var response struct {
		ID int `json:"id"`
	}

	err := c.client.Post(serviceCtx, "/api/v1/backtests", payload, &response)
	if err != nil {
		c.logger.Error("Failed to create backtest request",
			zap.Int("strategyID", request.StrategyID),
			zap.Int("userID", userID),
			zap.Error(err))
		return 0, err
	}

	return response.ID, nil
}

// GetBacktestStatus retrieves the status of a backtest
func (c *HistoricalClient) GetBacktestStatus(ctx context.Context, backtestID int) (string, error) {
	path := fmt.Sprintf("/api/v1/backtests/%d/status", backtestID)

	var response struct {
		Status string `json:"status"`
	}

	err := c.client.Get(ctx, path, &response)
	if err != nil {
		c.logger.Error("Failed to get backtest status",
			zap.Int("backtestID", backtestID),
			zap.Error(err))
		return "", err
	}

	return response.Status, nil
}

// GetBacktestResults retrieves the results of a completed backtest
func (c *HistoricalClient) GetBacktestResults(ctx context.Context, backtestID int) (*model.BacktestResult, error) {
	path := fmt.Sprintf("/api/v1/backtests/%d/results", backtestID)

	var result model.BacktestResult

	err := c.client.Get(ctx, path, &result)
	if err != nil {
		c.logger.Error("Failed to get backtest results",
			zap.Int("backtestID", backtestID),
			zap.Error(err))
		return nil, err
	}

	return &result, nil
}

// GetSymbols retrieves available trading symbols
func (c *HistoricalClient) GetSymbols(ctx context.Context) ([]struct {
	ID       int    `json:"id"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
}, error) {
	path := "/api/v1/symbols"

	var symbols []struct {
		ID       int    `json:"id"`
		Symbol   string `json:"symbol"`
		Name     string `json:"name"`
		Exchange string `json:"exchange"`
	}

	err := c.client.Get(ctx, path, &symbols)
	if err != nil {
		c.logger.Error("Failed to get symbols", zap.Error(err))
		return nil, err
	}

	return symbols, nil
}

// GetTimeframes retrieves available timeframes
func (c *HistoricalClient) GetTimeframes(ctx context.Context) ([]struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Minutes     int    `json:"minutes"`
	DisplayName string `json:"display_name"`
}, error) {
	path := "/api/v1/timeframes"

	var timeframes []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Minutes     int    `json:"minutes"`
		DisplayName string `json:"display_name"`
	}

	err := c.client.Get(ctx, path, &timeframes)
	if err != nil {
		c.logger.Error("Failed to get timeframes", zap.Error(err))
		return nil, err
	}

	return timeframes, nil
}
