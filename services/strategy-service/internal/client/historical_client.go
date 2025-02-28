// services/strategy-service/internal/client/historical_client.go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"services/strategy-service/internal/model"

	"go.uber.org/zap"
)

// HistoricalClient handles communication with the Historical Data Service
type HistoricalClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewHistoricalClient creates a new Historical Data Service client
func NewHistoricalClient(baseURL string, logger *zap.Logger) *HistoricalClient {
	return &HistoricalClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// CreateBacktest sends a backtest request to the Historical Data Service
func (c *HistoricalClient) CreateBacktest(ctx context.Context, request *model.BacktestRequest, userID int) (int, error) {
	url := fmt.Sprintf("%s/api/v1/backtests", c.baseURL)

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
		Strategy        json.RawMessage `json:"strategy"`
	}{
		StrategyID:      request.StrategyID,
		StrategyVersion: 1, // Default to latest version
		SymbolID:        request.SymbolID,
		TimeframeID:     request.TimeframeID,
		StartDate:       request.StartDate,
		EndDate:         request.EndDate,
		InitialCapital:  request.InitialCapital,
		UserID:          userID,
		// Strategy structure would be set later
	}

	// Serialize the payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		c.logger.Error("Failed to marshal backtest request", zap.Error(err))
		return 0, err
	}

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return 0, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Key", "strategy-service-key")

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to send backtest request", zap.Error(err))
		return 0, err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusAccepted {
		c.logger.Error("Historical service returned unexpected status",
			zap.Int("status_code", resp.StatusCode))
		return 0, fmt.Errorf("historical service returned status code %d", resp.StatusCode)
	}

	// Parse response
	var response struct {
		BacktestID int    `json:"backtest_id"`
		Message    string `json:"message"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		c.logger.Error("Failed to decode backtest response", zap.Error(err))
		return 0, err
	}

	return response.BacktestID, nil
}

// GetSymbols retrieves available trading symbols
func (c *HistoricalClient) GetSymbols(ctx context.Context) ([]struct {
	ID       int    `json:"id"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
}, error) {
	url := fmt.Sprintf("%s/api/v1/symbols", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Service-Key", "strategy-service-key")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get symbols", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("historical service returned status code %d", resp.StatusCode)
	}

	var symbols []struct {
		ID       int    `json:"id"`
		Symbol   string `json:"symbol"`
		Name     string `json:"name"`
		Exchange string `json:"exchange"`
	}

	err = json.NewDecoder(resp.Body).Decode(&symbols)
	if err != nil {
		c.logger.Error("Failed to decode symbols response", zap.Error(err))
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
	url := fmt.Sprintf("%s/api/v1/timeframes", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Service-Key", "strategy-service-key")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get timeframes", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("historical service returned status code %d", resp.StatusCode)
	}

	var timeframes []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Minutes     int    `json:"minutes"`
		DisplayName string `json:"display_name"`
	}

	err = json.NewDecoder(resp.Body).Decode(&timeframes)
	if err != nil {
		c.logger.Error("Failed to decode timeframes response", zap.Error(err))
		return nil, err
	}

	return timeframes, nil
}
