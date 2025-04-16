package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"services/historical-data-service/internal/model"

	"go.uber.org/zap"
)

// BacktestClient handles communication with the Backtesting Service
type BacktestClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewBacktestClient creates a new backtesting service client
func NewBacktestClient(baseURL string, logger *zap.Logger) *BacktestClient {
	return &BacktestClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Longer timeout for backtests
		},
		logger: logger,
	}
}

// RunBacktest sends a backtest request to the backtesting service
func (c *BacktestClient) RunBacktest(
	ctx context.Context,
	candles []model.Candle,
	strategy json.RawMessage,
	params map[string]interface{},
) (*model.BacktestResult, error) {
	// Build request payload
	payload := map[string]interface{}{
		"candles":  candles,
		"strategy": strategy,
		"params":   params,
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backtest request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/backtest", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	c.logger.Info("Sending backtest request", zap.String("url", url))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to send request to backtesting service", zap.Error(err))
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("backtest service returned status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("backtest service error: %s", errorResp.Error)
	}

	// Parse response
	var result model.BacktestResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Error("Failed to decode backtest response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ValidateStrategy validates a strategy structure
func (c *BacktestClient) ValidateStrategy(ctx context.Context, strategy json.RawMessage) (bool, string, error) {
	// Build request payload
	payload := map[string]interface{}{
		"strategy": strategy,
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return false, "", fmt.Errorf("failed to marshal strategy validation request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/validate-strategy", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to send request to backtesting service", zap.Error(err))
		return false, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result struct {
		Valid   bool   `json:"valid"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Error("Failed to decode validation response", zap.Error(err))
		return false, "", fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Valid {
		message := result.Error
		if message == "" {
			message = "Strategy validation failed"
		}
		return false, message, nil
	}

	return true, result.Message, nil
}

// GetAvailableIndicators retrieves the list of available indicators
func (c *BacktestClient) GetAvailableIndicators(ctx context.Context) ([]model.Indicator, error) {
	// Create HTTP request
	url := fmt.Sprintf("%s/indicators", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to send request to backtesting service", zap.Error(err))
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("indicator service returned status %d", resp.StatusCode)
	}

	// Parse response
	var indicators []model.Indicator
	if err := json.NewDecoder(resp.Body).Decode(&indicators); err != nil {
		c.logger.Error("Failed to decode indicators response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return indicators, nil
}

// CheckHealth checks if the backtesting service is healthy
func (c *BacktestClient) CheckHealth(ctx context.Context) (bool, error) {
	// Create HTTP request
	url := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to send health check to backtesting service", zap.Error(err))
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	return resp.StatusCode == http.StatusOK, nil
}
