package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"services/historical-data-service/internal/model"

	"go.uber.org/zap"
)

const (
	BinanceAPIBaseURL      = "https://api.binance.com"
	BinanceAPIExchangeInfo = "/api/v3/exchangeInfo"
	BinanceAPIKlines       = "/api/v3/klines"
	MaxKlinesLimit         = 1000
)

// BinanceClient handles communication with the Binance API
type BinanceClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewBinanceClient creates a new Binance API client
func NewBinanceClient(logger *zap.Logger) *BinanceClient {
	return &BinanceClient{
		baseURL: BinanceAPIBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// GetExchangeInfo retrieves all available symbols from Binance
func (c *BinanceClient) GetExchangeInfo(ctx context.Context) (*model.BinanceExchangeInfo, error) {
	url := c.baseURL + BinanceAPIExchangeInfo

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to fetch exchange info from Binance", zap.Error(err))
		return nil, fmt.Errorf("failed to fetch exchange info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Binance API returned status code %d", resp.StatusCode)
	}

	var exchangeInfo model.BinanceExchangeInfo
	if err := json.NewDecoder(resp.Body).Decode(&exchangeInfo); err != nil {
		c.logger.Error("Failed to decode Binance exchange info", zap.Error(err))
		return nil, fmt.Errorf("failed to decode exchange info: %w", err)
	}

	return &exchangeInfo, nil
}

// GetKlines retrieves candlestick data for a symbol and interval
func (c *BinanceClient) GetKlines(ctx context.Context, symbol, interval string, startTime, endTime *time.Time, limit int) ([]model.BinanceKline, error) {
	if limit > MaxKlinesLimit {
		limit = MaxKlinesLimit
	}

	reqURL := c.baseURL + BinanceAPIKlines

	// Build query parameters
	params := url.Values{}
	params.Add("symbol", symbol)
	params.Add("interval", interval)

	if limit > 0 {
		params.Add("limit", strconv.Itoa(limit))
	}

	if startTime != nil {
		params.Add("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}

	if endTime != nil {
		params.Add("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}

	reqURL = reqURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to fetch klines from Binance",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.String("interval", interval))
		return nil, fmt.Errorf("failed to fetch klines: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Binance API returned status code %d", resp.StatusCode)
	}

	var rawKlines [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawKlines); err != nil {
		c.logger.Error("Failed to decode Binance klines", zap.Error(err))
		return nil, fmt.Errorf("failed to decode klines: %w", err)
	}

	// Convert the raw klines to our model
	klines := make([]model.BinanceKline, len(rawKlines))
	for i, raw := range rawKlines {
		openTime := int64(raw[0].(float64))
		closeTime := int64(raw[6].(float64))

		// Parse numeric values
		open, _ := strconv.ParseFloat(raw[1].(string), 64)
		high, _ := strconv.ParseFloat(raw[2].(string), 64)
		low, _ := strconv.ParseFloat(raw[3].(string), 64)
		close, _ := strconv.ParseFloat(raw[4].(string), 64)
		volume, _ := strconv.ParseFloat(raw[5].(string), 64)

		klines[i] = model.BinanceKline{
			OpenTime:  time.Unix(openTime/1000, (openTime%1000)*1000000),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			CloseTime: time.Unix(closeTime/1000, (closeTime%1000)*1000000),
		}
	}

	return klines, nil
}

// MapBinanceIntervalToTimeframe maps Binance interval strings to our timeframe enum
func MapBinanceIntervalToTimeframe(interval string) string {
	switch interval {
	case "1m":
		return "1m"
	case "5m":
		return "5m"
	case "15m":
		return "15m"
	case "30m":
		return "30m"
	case "1h":
		return "1h"
	case "4h":
		return "4h"
	case "1d":
		return "1d"
	case "1w":
		return "1w"
	default:
		return ""
	}
}

// MapTimeframeToBinanceInterval maps our timeframe enum to Binance interval strings
func MapTimeframeToBinanceInterval(timeframe string) string {
	switch timeframe {
	case "1m":
		return "1m"
	case "5m":
		return "5m"
	case "15m":
		return "15m"
	case "30m":
		return "30m"
	case "1h":
		return "1h"
	case "4h":
		return "4h"
	case "1d":
		return "1d"
	case "1w":
		return "1w"
	default:
		return ""
	}
}
