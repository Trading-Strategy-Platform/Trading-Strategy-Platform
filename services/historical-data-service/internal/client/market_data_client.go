package client

import (
	"context"
	"fmt"
	"time"

	"services/historical-data-service/internal/model"

	"github.com/yourorg/trading-platform/shared/go/httpclient"
	"go.uber.org/zap"
)

// MarketDataClient handles communication with external market data providers
type MarketDataClient struct {
	client *httpclient.Client
	logger *zap.Logger
}

// NewMarketDataClient creates a new market data client
func NewMarketDataClient(baseURL string, apiKey string, logger *zap.Logger) *MarketDataClient {
	client := httpclient.New(httpclient.Config{
		BaseURL:       baseURL,
		Timeout:       60 * time.Second,
		ServiceKey:    "historical-service-key",
		RetryAttempts: 3,
	}, logger)

	return &MarketDataClient{
		client: client,
		logger: logger,
	}
}

// FetchHistoricalData retrieves historical market data from the provider
func (c *MarketDataClient) FetchHistoricalData(
	ctx context.Context,
	symbol string,
	interval string,
	startDate time.Time,
	endDate time.Time,
) ([]model.OHLCV, error) {
	path := fmt.Sprintf("/historical/%s", symbol)

	// Add query parameters
	params := map[string]string{
		"interval":  interval,
		"startDate": startDate.Format(time.RFC3339),
		"endDate":   endDate.Format(time.RFC3339),
	}

	// Add params to the path
	path = c.addQueryParams(path, params)

	var response struct {
		Data []model.OHLCV `json:"data"`
	}

	err := c.client.Get(ctx, path, &response)
	if err != nil {
		c.logger.Error("Failed to fetch historical data",
			zap.String("symbol", symbol),
			zap.String("interval", interval),
			zap.Time("startDate", startDate),
			zap.Time("endDate", endDate),
			zap.Error(err))
		return nil, err
	}

	return response.Data, nil
}

// addQueryParams is a helper function to add query parameters to a URL
func (c *MarketDataClient) addQueryParams(path string, params map[string]string) string {
	if len(params) == 0 {
		return path
	}

	path += "?"
	first := true
	for key, value := range params {
		if !first {
			path += "&"
		}
		path += fmt.Sprintf("%s=%s", key, value)
		first = false
	}

	return path
}
