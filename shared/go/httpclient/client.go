package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	sharedErrors "github.com/yourorg/trading-platform/shared/go/errors"
	"go.uber.org/zap"
)

// Client provides a wrapper around http.Client with additional functionality
type Client struct {
	httpClient *http.Client
	baseURL    string
	logger     *zap.Logger
	serviceKey string
}

// Config holds configuration for the HTTP client
type Config struct {
	BaseURL       string
	Timeout       time.Duration
	ServiceKey    string
	RetryAttempts int
}

// New creates a new HTTP client with the provided configuration
func New(cfg Config, logger *zap.Logger) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL:    cfg.BaseURL,
		logger:     logger,
		serviceKey: cfg.ServiceKey,
	}
}

// Get performs a GET request to the specified path
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	return c.Request(ctx, http.MethodGet, path, nil, result)
}

// Post performs a POST request to the specified path with the given body
func (c *Client) Post(ctx context.Context, path string, body, result interface{}) error {
	return c.Request(ctx, http.MethodPost, path, body, result)
}

// Put performs a PUT request to the specified path with the given body
func (c *Client) Put(ctx context.Context, path string, body, result interface{}) error {
	return c.Request(ctx, http.MethodPut, path, body, result)
}

// Delete performs a DELETE request to the specified path
func (c *Client) Delete(ctx context.Context, path string, result interface{}) error {
	return c.Request(ctx, http.MethodDelete, path, nil, result)
}

// Patch performs a PATCH request to the specified path with the given body
func (c *Client) Patch(ctx context.Context, path string, body, result interface{}) error {
	return c.Request(ctx, http.MethodPatch, path, body, result)
}

// Request performs an HTTP request with the given method, path, and body
func (c *Client) Request(ctx context.Context, method, path string, body, result interface{}) error {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return sharedErrors.NewInternalError("Failed to marshal request body", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return sharedErrors.NewInternalError("Failed to create HTTP request", err)
	}

	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set service key if available
	if c.serviceKey != "" {
		req.Header.Set("X-Service-Key", c.serviceKey)
	}

	// Extract and set auth token from context if available
	if token, ok := ctx.Value("auth_token").(string); ok && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("HTTP request failed",
			zap.String("method", method),
			zap.String("url", url),
			zap.Error(err))
		return sharedErrors.NewExternalServiceError(getServiceName(c.baseURL), "Service unavailable", err)
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode >= 400 {
		return c.handleErrorResponse(resp)
	}

	// Parse response if a result container was provided
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			c.logger.Error("Failed to decode response",
				zap.String("method", method),
				zap.String("url", url),
				zap.Error(err))
			return sharedErrors.NewInternalError("Failed to decode response", err)
		}
	}

	return nil
}

// Add this type definition for client errors
// Error represents an HTTP client error with status code and response details
type Error struct {
	StatusCode   int
	ResponseBody []byte
	Message      string
	URL          string
	Method       string
}

// Error implements the error interface
func (e *Error) Error() string {
	return fmt.Sprintf("%s %s failed with status %d: %s", e.Method, e.URL, e.StatusCode, e.Message)
}

// Update the handleErrorResponse method to return this error type
func (c *Client) handleErrorResponse(resp *http.Response) error {
	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return sharedErrors.NewInternalError("Failed to read error response", err)
	}

	// Try to parse as JSON error
	var errorResp struct {
		Error   string      `json:"error"`
		Message string      `json:"message"`
		Details interface{} `json:"details,omitempty"`
	}

	// Attempt to decode but don't fail if not possible
	_ = json.Unmarshal(body, &errorResp)

	errorMsg := errorResp.Message
	if errorMsg == "" {
		errorMsg = errorResp.Error
	}
	if errorMsg == "" {
		errorMsg = http.StatusText(resp.StatusCode)
	}

	// Return error with status code and body for detailed handling
	return &Error{
		StatusCode:   resp.StatusCode,
		ResponseBody: body,
		Message:      errorMsg,
		URL:          resp.Request.URL.String(),
		Method:       resp.Request.Method,
	}
}

// Helper function to extract service name from base URL
func getServiceName(baseURL string) string {
	// Simple implementation - in practice you might want to parse the URL better
	if baseURL == "" {
		return "unknown-service"
	}

	// Try to extract service name from URL
	serviceName := baseURL
	// Remove protocol prefix
	if idx := len("https://"); len(serviceName) > idx {
		if serviceName[:idx] == "https://" || serviceName[:8] == "http://" {
			serviceName = serviceName[idx:]
		}
	}
	// Take only up to the first dot or slash
	for i, c := range serviceName {
		if c == '.' || c == '/' {
			serviceName = serviceName[:i]
			break
		}
	}
	return serviceName
}

// createErrorFromStatusCode creates an appropriate error based on HTTP status code
func createErrorFromStatusCode(statusCode int, serviceName string) error {
	switch statusCode {
	case http.StatusBadRequest:
		return sharedErrors.NewValidationError("Invalid request")
	case http.StatusUnauthorized:
		return sharedErrors.NewAuthError("Authentication required")
	case http.StatusForbidden:
		return sharedErrors.NewPermissionError("Permission denied")
	case http.StatusNotFound:
		return sharedErrors.NewNotFoundError("Resource", "not found")
	case http.StatusTooManyRequests:
		return sharedErrors.NewValidationError("Rate limit exceeded")
	default:
		return sharedErrors.NewExternalServiceError(
			serviceName,
			fmt.Sprintf("Service error (status code: %d)", statusCode),
			fmt.Errorf("unexpected status code: %d", statusCode),
		)
	}
}

// BaseURL returns the base URL configured for this client
func (c *Client) BaseURL() string {
	return c.baseURL
}
