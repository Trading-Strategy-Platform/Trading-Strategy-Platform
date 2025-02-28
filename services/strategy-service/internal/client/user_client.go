// services/strategy-service/internal/client/user_client.go
package client

import (
	"context"
	"encoding/json"
	"errors"
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

// GetUserByID retrieves a user's username by ID
func (c *UserClient) GetUserByID(ctx context.Context, userID int) (string, error) {
	url := fmt.Sprintf("%s/api/v1/admin/users/%d", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	// Add service authentication header (this would be replaced with actual service auth)
	req.Header.Set("X-Service-Key", "strategy-service-key")

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

// ValidateUserAccess validates a user's access token
func (c *UserClient) ValidateUserAccess(ctx context.Context, userID int, token string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/auth/validate", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return false, err
	}

	// Add the token to be validated
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to validate token with User Service", zap.Error(err))
		return false, err
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
			return false, err
		}

		// Verify the token belongs to the expected user
		return response.UserID == userID, nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}

	return false, errors.New("unexpected response from user service")
}
