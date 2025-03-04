package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

// Client is a wrapper for http.Client with additional functionality
type Client struct {
	baseURL     string
	httpClient  *http.Client
	logger      *zap.Logger
	serviceName string
	headers     map[string]string
	retryConfig RetryConfig
}

// ClientConfig holds configuration for the HTTP client
type ClientConfig struct {
	BaseURL     string
	Timeout     time.Duration
	ServiceName string
	Headers     map[string]string
	RetryConfig RetryConfig
	Logger      *zap.Logger
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	MaxElapsedTime  time.Duration
}

// DefaultRetryConfig provides sensible defaults for retry configuration
var DefaultRetryConfig = RetryConfig{
	MaxRetries:      3,
	InitialInterval: 100 * time.Millisecond,
	MaxInterval:     10 * time.Second,
	Multiplier:      2.0,
	MaxElapsedTime:  30 * time.Second,
}

// NewClient creates a new HTTP client
func NewClient(config ClientConfig) *Client {
	// Set default timeout
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	// Set default retry config
	if config.RetryConfig.MaxRetries == 0 {
		config.RetryConfig = DefaultRetryConfig
	}

	client := &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger:      config.Logger,
		serviceName: config.ServiceName,
		headers:     config.Headers,
		retryConfig: config.RetryConfig,
	}

	return client
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	return c.Request(ctx, http.MethodGet, path, nil, result)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body, result interface{}) error {
	return c.Request(ctx, http.MethodPost, path, body, result)
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body, result interface{}) error {
	return c.Request(ctx, http.MethodPut, path, body, result)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string, result interface{}) error {
	return c.Request(ctx, http.MethodDelete, path, nil, result)
}

// Request performs an HTTP request with the given method, path, and body
func (c *Client) Request(ctx context.Context, method, path string, body, result interface{}) error {
	url := c.baseURL + path

	// Create request body if provided
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Create exponential backoff for retries
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = c.retryConfig.InitialInterval
	b.MaxInterval = c.retryConfig.MaxInterval
	b.Multiplier = c.retryConfig.Multiplier
	b.MaxElapsedTime = c.retryConfig.MaxElapsedTime

	var resp *http.Response
	var respBody []byte

	// Perform request with retries
	operation := func() error {
		var opErr error
		resp, opErr = c.httpClient.Do(req)
		if opErr != nil {
			return fmt.Errorf("request failed: %w", opErr)
		}

		defer resp.Body.Close()

		// Read response body
		respBody, opErr = io.ReadAll(resp.Body)
		if opErr != nil {
			return fmt.Errorf("failed to read response body: %w", opErr)
		}

		// Check if the request was successful
		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d %s", resp.StatusCode, resp.Status)
		}

		return nil
	}

	// Execute with retry
	err = backoff.Retry(operation, backoff.WithMaxRetries(b, uint64(c.retryConfig.MaxRetries)))
	if err != nil {
		c.logger.Error("request failed after retries",
			zap.String("method", method),
			zap.String("url", url),
			zap.Error(err),
		)
		return err
	}

	// Check if the request was successful
	if resp.StatusCode >= 400 {
		// Try to parse error response
		var apiError struct {
			Error struct {
				Type    string `json:"type"`
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}

		if err := json.Unmarshal(respBody, &apiError); err == nil && apiError.Error.Message != "" {
			return fmt.Errorf("request failed: %s (%s)", apiError.Error.Message, apiError.Error.Code)
		}

		return fmt.Errorf("request failed with status: %d %s", resp.StatusCode, resp.Status)
	}

	// If result is expected, unmarshal response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
