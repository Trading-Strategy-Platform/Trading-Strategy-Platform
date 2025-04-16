package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"services/historical-data-service/internal/model"

	"go.uber.org/zap"
)

const (
	BinanceAPIBaseURL = "https://api.binance.com/api/v3"
	MaxKlinesLimit    = 1000
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
	reqURL := fmt.Sprintf("%s/exchangeInfo", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
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
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.logger.Error("Binance API error response",
			zap.Int("statusCode", resp.StatusCode),
			zap.String("response", string(bodyBytes)))
		return nil, fmt.Errorf("Binance API returned status code %d: %s", resp.StatusCode, string(bodyBytes))
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

	// Use the exact same URL format as the frontend
	reqURL := fmt.Sprintf("%s/klines", c.baseURL)

	// Build query parameters
	params := url.Values{}
	params.Add("symbol", symbol)
	params.Add("interval", interval)
	params.Add("limit", strconv.Itoa(limit))

	if startTime != nil {
		// Convert to milliseconds timestamp
		params.Add("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}

	if endTime != nil {
		// Convert to milliseconds timestamp
		params.Add("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}

	reqURL = reqURL + "?" + params.Encode()

	// Log the exact URL we're calling for debugging
	c.logger.Debug("Calling Binance API", zap.String("url", reqURL))

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

	// Log more detailed error info on failed responses
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.logger.Error("Binance API error response",
			zap.Int("statusCode", resp.StatusCode),
			zap.String("response", string(bodyBytes)))
		return nil, fmt.Errorf("Binance API returned status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var rawKlines [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawKlines); err != nil {
		c.logger.Error("Failed to decode Binance klines", zap.Error(err))
		return nil, fmt.Errorf("failed to decode klines: %w", err)
	}

	// Add logging for empty responses
	if len(rawKlines) == 0 {
		c.logger.Warn("Binance returned empty klines array",
			zap.String("symbol", symbol),
			zap.String("interval", interval),
			zap.String("url", reqURL))
	} else {
		c.logger.Debug("Successfully fetched klines",
			zap.Int("count", len(rawKlines)),
			zap.String("symbol", symbol))
	}

	// Convert the raw klines to our model
	klines := make([]model.BinanceKline, 0, len(rawKlines))
	for i, raw := range rawKlines {
		if len(raw) < 7 {
			c.logger.Warn("Skipping malformed kline data",
				zap.Int("index", i),
				zap.Any("raw_data", raw))
			continue
		}

		// Handle the timestamp conversion carefully
		var openTime, closeTime time.Time

		// Extract the timestamp as int64
		openTimeVal, ok := raw[0].(float64)
		if !ok {
			c.logger.Warn("Invalid open time format",
				zap.Int("index", i),
				zap.Any("openTime", raw[0]))
			continue
		}

		// Convert milliseconds to time.Time
		openTimeMs := int64(openTimeVal)
		openTime = time.UnixMilli(openTimeMs)

		closeTimeVal, ok := raw[6].(float64)
		if !ok {
			c.logger.Warn("Invalid close time format",
				zap.Int("index", i),
				zap.Any("closeTime", raw[6]))
			continue
		}

		closeTimeMs := int64(closeTimeVal)
		closeTime = time.UnixMilli(closeTimeMs)

		// Validate that we have valid times
		if openTime.IsZero() || closeTime.IsZero() {
			c.logger.Warn("Zero time value detected, skipping candle",
				zap.Int("index", i),
				zap.Time("openTime", openTime),
				zap.Time("closeTime", closeTime))
			continue
		}

		// Parse numeric values
		open, _ := strconv.ParseFloat(raw[1].(string), 64)
		high, _ := strconv.ParseFloat(raw[2].(string), 64)
		low, _ := strconv.ParseFloat(raw[3].(string), 64)
		close, _ := strconv.ParseFloat(raw[4].(string), 64)
		volume, _ := strconv.ParseFloat(raw[5].(string), 64)

		// Log the timestamp to help debug
		c.logger.Debug("Processing kline",
			zap.Int("index", i),
			zap.Int64("openTimeMs", openTimeMs),
			zap.Time("openTime", openTime))

		klines = append(klines, model.BinanceKline{
			OpenTime:  openTime,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			CloseTime: closeTime,
		})
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
