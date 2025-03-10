package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"services/historical-data-service/internal/model"

	"github.com/yourorg/trading-platform/shared/go/httpclient"
	"go.uber.org/zap"
)

// BacktestExecutionClient handles communication with the backtest execution engine
type BacktestExecutionClient struct {
	client *httpclient.Client
	logger *zap.Logger
}

// NewBacktestExecutionClient creates a new backtest execution client
func NewBacktestExecutionClient(baseURL string, logger *zap.Logger) *BacktestExecutionClient {
	client := httpclient.New(httpclient.Config{
		BaseURL:       baseURL,
		Timeout:       120 * time.Second, // Longer timeout for backtest execution
		ServiceKey:    "historical-service-key",
		RetryAttempts: 1, // Don't retry backtest executions automatically
	}, logger)

	return &BacktestExecutionClient{
		client: client,
		logger: logger,
	}
}

// ExecuteBacktest sends a backtest request to the execution engine
func (c *BacktestExecutionClient) ExecuteBacktest(
	ctx context.Context,
	strategy json.RawMessage,
	marketData []model.OHLCV,
	initialCapital float64,
) (*model.BacktestResults, error) {
	path := "/api/v1/execute"

	payload := struct {
		Strategy       json.RawMessage `json:"strategy"`
		MarketData     []model.OHLCV   `json:"market_data"`
		InitialCapital float64         `json:"initial_capital"`
	}{
		Strategy:       strategy,
		MarketData:     marketData,
		InitialCapital: initialCapital,
	}

	// Add service key to context
	serviceCtx := context.WithValue(ctx, "service_key", "historical-service-key")

	var results model.BacktestResults

	err := c.client.Post(serviceCtx, path, payload, &results)
	if err != nil {
		c.logger.Error("Failed to execute backtest",
			zap.Error(err),
			zap.Float64("initialCapital", initialCapital))
		return nil, err
	}

	return &results, nil
}

// ValidateStrategy checks if a strategy is valid before running a backtest
func (c *BacktestExecutionClient) ValidateStrategy(
	ctx context.Context,
	strategy json.RawMessage,
) error {
	path := "/api/v1/validate"

	payload := struct {
		Strategy json.RawMessage `json:"strategy"`
	}{
		Strategy: strategy,
	}

	// Add service key to context
	serviceCtx := context.WithValue(ctx, "service_key", "historical-service-key")

	var response struct {
		Valid bool   `json:"valid"`
		Error string `json:"error,omitempty"`
	}

	err := c.client.Post(serviceCtx, path, payload, &response)
	if err != nil {
		c.logger.Error("Failed to validate strategy", zap.Error(err))
		return err
	}

	if !response.Valid {
		return fmt.Errorf("invalid strategy: %s", response.Error)
	}

	return nil
}
