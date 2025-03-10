package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yourorg/trading-platform/shared/go/httpclient"
	"go.uber.org/zap"
)

// UserClient handles communication with the User Service
type UserClient struct {
	client *httpclient.Client
	logger *zap.Logger
}

// NewUserClient creates a new User Service client
func NewUserClient(baseURL string, logger *zap.Logger) *UserClient {
	client := httpclient.New(httpclient.Config{
		BaseURL:    baseURL,
		Timeout:    10 * time.Second,
		ServiceKey: "historical-service-key",
	}, logger)

	return &UserClient{
		client: client,
		logger: logger,
	}
}

// ValidateToken validates a user's token with the User Service
func (c *UserClient) ValidateToken(ctx context.Context, token string) (int, error) {
	// Create context with auth token
	tokenCtx := context.WithValue(ctx, "auth_token", token)

	var response struct {
		UserID int `json:"user_id"`
	}

	err := c.client.Post(tokenCtx, "/api/v1/auth/validate", nil, &response)
	if err != nil {
		c.logger.Error("Failed to validate token with User Service", zap.Error(err))
		return 0, err
	}

	return response.UserID, nil
}

// GetUsername gets a username by user ID
func (c *UserClient) GetUsername(ctx context.Context, userID int) (string, error) {
	path := fmt.Sprintf("/api/v1/admin/users/%d", userID)

	var user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}

	err := c.client.Get(ctx, path, &user)
	if err != nil {
		c.logger.Error("Failed to get user from User Service", zap.Error(err))
		return "", err
	}

	return user.Username, nil
}

// StrategyClient handles communication with the Strategy Service
type StrategyClient struct {
	client *httpclient.Client
	logger *zap.Logger
}

// NewStrategyClient creates a new Strategy Service client
func NewStrategyClient(baseURL string, logger *zap.Logger) *StrategyClient {
	client := httpclient.New(httpclient.Config{
		BaseURL:    baseURL,
		Timeout:    30 * time.Second,
		ServiceKey: "historical-service-key",
	}, logger)

	return &StrategyClient{
		client: client,
		logger: logger,
	}
}

// GetStrategy retrieves a strategy by ID
func (c *StrategyClient) GetStrategy(ctx context.Context, strategyID int, token string) (*Strategy, error) {
	path := fmt.Sprintf("/api/v1/strategies/%d", strategyID)

	// Add token to context if provided
	var requestCtx context.Context
	if token != "" {
		requestCtx = context.WithValue(ctx, "auth_token", token)
	} else {
		// Use service context for service-to-service communication
		requestCtx = context.WithValue(ctx, "service_key", "historical-service-key")
	}

	var strategy Strategy
	err := c.client.Get(requestCtx, path, &strategy)
	if err != nil {
		c.logger.Error("Failed to get strategy",
			zap.Int("strategyID", strategyID),
			zap.Error(err))
		return nil, err
	}

	return &strategy, nil
}

// Strategy represents a trading strategy from the Strategy Service
type Strategy struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	UserID      int             `json:"user_id"`
	Structure   json.RawMessage `json:"structure"`
	// Add other strategy fields as needed
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
	path := fmt.Sprintf("/api/v1/strategies/%d/versions/%d", strategyID, versionNumber)

	// Add token to context if provided
	var requestCtx context.Context
	if token != "" {
		requestCtx = context.WithValue(ctx, "auth_token", token)
	} else {
		requestCtx = ctx
	}

	var version struct {
		ID          int             `json:"id"`
		StrategyID  int             `json:"strategy_id"`
		Version     int             `json:"version"`
		Structure   json.RawMessage `json:"structure"`
		ChangeNotes string          `json:"change_notes"`
		CreatedAt   time.Time       `json:"created_at"`
	}

	err := c.client.Get(requestCtx, path, &version)
	if err != nil {
		c.logger.Error("Failed to get strategy version", zap.Error(err))
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
	path := "/api/v1/service/backtests/notify"

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

	// Use service context with service key
	serviceCtx := context.WithValue(ctx, "service_key", "historical-service-key")

	err := c.client.Post(serviceCtx, path, payload, nil)
	if err != nil {
		c.logger.Error("Failed to notify strategy service", zap.Error(err))
		return err
	}

	return nil
}
