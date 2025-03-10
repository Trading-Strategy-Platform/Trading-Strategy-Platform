package client

import (
	"context"
	"fmt"
	"time"

	"services/user-service/internal/model"

	sharedErrors "github.com/yourorg/trading-platform/shared/go/errors"
	"github.com/yourorg/trading-platform/shared/go/httpclient"
	"go.uber.org/zap"
)

// AuthClient handles external authentication-related API calls
type AuthClient struct {
	client *httpclient.Client
	logger *zap.Logger
}

// NewAuthClient creates a new authentication client
func NewAuthClient(baseURL string, logger *zap.Logger) *AuthClient {
	client := httpclient.New(httpclient.Config{
		BaseURL:    baseURL,
		Timeout:    10 * time.Second,
		ServiceKey: "user-service-key",
	}, logger)

	return &AuthClient{
		client: client,
		logger: logger,
	}
}

// ValidateExternalToken validates a token with an external identity provider
func (c *AuthClient) ValidateExternalToken(ctx context.Context, token string) (*model.ExternalUserInfo, error) {
	// Create context with the token
	tokenCtx := context.WithValue(ctx, "auth_token", token)

	var userInfo model.ExternalUserInfo
	err := c.client.Get(tokenCtx, "/api/v1/identity/validate", &userInfo)

	if err != nil {
		c.logger.Error("Failed to validate external token", zap.Error(err))
		return nil, sharedErrors.NewAuthError(fmt.Sprintf("Invalid or expired external token: %v", err))
	}

	return &userInfo, nil
}

// CheckPermissions verifies if a user has specific permissions in the external system
func (c *AuthClient) CheckPermissions(ctx context.Context, userID int, permission string) (bool, error) {
	path := fmt.Sprintf("/api/v1/identity/users/%d/permissions/%s", userID, permission)

	var response struct {
		HasPermission bool `json:"has_permission"`
	}

	err := c.client.Get(ctx, path, &response)
	if err != nil {
		c.logger.Error("Failed to check permissions",
			zap.Int("userID", userID),
			zap.String("permission", permission),
			zap.Error(err))
		return false, err
	}

	return response.HasPermission, nil
}
