// services/strategy-service/internal/client/user_client.go
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/yourorg/trading-platform/shared/go/httpclient"
	"go.uber.org/zap"
)

// UserClient implements the service.UserClient interface
type UserClient struct {
	client *httpclient.Client
	logger *zap.Logger
}

// NewUserClient creates a new user service client
func NewUserClient(baseURL string, logger *zap.Logger) *UserClient {
	client := httpclient.New(httpclient.Config{
		BaseURL:    baseURL,
		Timeout:    10 * time.Second,
		ServiceKey: "strategy-service-key",
	}, logger)

	return &UserClient{
		client: client,
		logger: logger,
	}
}

// GetUserByID retrieves a username by user ID
func (c *UserClient) GetUserByID(ctx context.Context, userID int) (string, error) {
	path := fmt.Sprintf("/api/v1/users/%d", userID)

	var response struct {
		Data struct {
			Username string `json:"username"`
		} `json:"data"`
	}

	err := c.client.Get(ctx, path, &response)
	if err != nil {
		c.logger.Error("Failed to retrieve user by ID",
			zap.Int("userID", userID),
			zap.Error(err))
		return "", err
	}

	return response.Data.Username, nil
}

// ValidateUserAccess validates user access token
func (c *UserClient) ValidateUserAccess(ctx context.Context, userID int, token string) (bool, error) {
	// Create a context with the auth token
	tokenCtx := context.WithValue(ctx, "auth_token", token)

	path := fmt.Sprintf("/api/v1/auth/validate/%d", userID)

	var response struct {
		Valid bool `json:"valid"`
	}

	err := c.client.Get(tokenCtx, path, &response)
	if err != nil {
		c.logger.Error("Failed to validate user access",
			zap.Int("userID", userID),
			zap.Error(err))
		return false, err
	}

	return response.Valid, nil
}
