package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// UserClient handles communication with the User Service
type UserClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewUserClient creates a new User Service client
func NewUserClient(baseURL string, logger *zap.Logger) *UserClient {
	return &UserClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// ValidateToken validates a user's token with the User Service
func (c *UserClient) ValidateToken(ctx context.Context, token string) (int, error) {
	url := fmt.Sprintf("%s/api/v1/auth/validate", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return 0, err
	}

	// Add the token to be validated
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to validate token with User Service", zap.Error(err))
		return 0, err
	}
	defer resp.Body.Close()

	// 200 = valid, 401 = invalid
	if resp.StatusCode == http.StatusOK {
		var response struct {
			UserID int `json:"user_id"`
		}

		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			c.logger.Error("Failed to decode validation response", zap.Error(err))
			return 0, err
		}

		return response.UserID, nil
	}

	return 0, fmt.Errorf("invalid token (status code: %d)", resp.StatusCode)
}

// GetUsername gets a username by user ID
func (c *UserClient) GetUsername(ctx context.Context, userID int) (string, error) {
	url := fmt.Sprintf("%s/api/v1/admin/users/%d", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	// Add service authentication header
	req.Header.Set("X-Service-Key", "historical-service-key")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get user from User Service", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("user service returned status code %d", resp.StatusCode)
	}

	var user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}

	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		c.logger.Error("Failed to decode user response", zap.Error(err))
		return "", err
	}

	return user.Username, nil
}

// StrategyClient handles communication with the Strategy Service
type StrategyClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewStrategyClient creates a new Strategy Service client
func NewStrategyClient(baseURL string, logger *zap.Logger) *StrategyClient {
	return &StrategyClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// GetStrategy retrieves a strategy by ID
func (c *StrategyClient) GetStrategy(ctx context.Context, strategyID int, token string) (
	*struct {
		ID            int             `json:"id"`
		Name          string          `json:"name"`
		UserID        int             `json:"user_id"`
		Structure     json.RawMessage `json:"structure"`
		Version       int             `json:"version"`
		CreatorName   string          `json:"creator_name,omitempty"`
		VersionsCount int             `json:"versions_count,omitempty"`
	}, error) {
	url := fmt.Sprintf("%s/api/v1/strategies/%d", c.baseURL, strategyID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication header
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		// Use service authentication for non-user requests
		req.Header.Set("X-Service-Key", "historical-service-key")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get strategy from Strategy Service", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("strategy service returned status code %d", resp.StatusCode)
	}

	var strategy struct {
		ID            int             `json:"id"`
		Name          string          `json:"name"`
		UserID        int             `json:"user_id"`
		Structure     json.RawMessage `json:"structure"`
		Version       int             `json:"version"`
		CreatorName   string          `json:"creator_name,omitempty"`
		VersionsCount int             `json:"versions_count,omitempty"`
	}

	err = json.NewDecoder(resp.Body).Decode(&strategy)
	if err != nil {
		c.logger.Error("Failed to decode strategy response", zap.Error(err))
		return nil, err
	}

	return &strategy, nil
}

// GetStrategyVersion retrieves a specific version of a strategy
func (c *StrategyClient) GetStrategyVersion(
	ctx context.Context,
	strategyID int,
	versionNumber int,
	token string,
) (*struct {
	ID          int             `json:"id"`
	StrategyID  int             `json:"strategy_id"`
	Version     int             `json:"version"`
	Structure   json.RawMessage `json:"structure"`
	ChangeNotes string          `json:"change_notes"`
	CreatedAt   time.Time       `json:"created_at"`
}, error) {
	url := fmt.Sprintf("%s/api/v1/strategies/%d/versions/%d", c.baseURL, strategyID, versionNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication header
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		// Use service authentication for non-user requests
		req.Header.Set("X-Service-Key", "historical-service-key")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get strategy version", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("strategy service returned status code %d", resp.StatusCode)
	}

	var version struct {
		ID          int             `json:"id"`
		StrategyID  int             `json:"strategy_id"`
		Version     int             `json:"version"`
		Structure   json.RawMessage `json:"structure"`
		ChangeNotes string          `json:"change_notes"`
		CreatedAt   time.Time       `json:"created_at"`
	}

	err = json.NewDecoder(resp.Body).Decode(&version)
	if err != nil {
		c.logger.Error("Failed to decode strategy version response", zap.Error(err))
		return nil, err
	}

	return &version, nil
}

// NotifyBacktestComplete notifies the Strategy Service that a backtest is complete
func (c *StrategyClient) NotifyBacktestComplete(
	ctx context.Context,
	backtestID int,
	strategyID int,
	userID int,
	status string,
) error {
	url := fmt.Sprintf("%s/api/v1/service/backtests/notify", c.baseURL)

	payload := struct {
		BacktestID int    `json:"backtest_id"`
		StrategyID int    `json:"strategy_id"`
		UserID     int    `json:"user_id"`
		Status     string `json:"status"`
	}{
		BacktestID: backtestID,
		StrategyID: strategyID,
		UserID:     userID,
		Status:     status,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		c.logger.Error("Failed to marshal backtest notification", zap.Error(err))
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewBuffer(payloadBytes),
	)
	if err != nil {
		return err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Key", "historical-service-key")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to notify strategy service", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("strategy service returned status code %d", resp.StatusCode)
	}

	return nil
}
