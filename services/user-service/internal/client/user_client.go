package client

import (
	"context"
	"fmt"
	"time"

	"services/user-service/internal/model"

	"github.com/yourorg/trading-platform/shared/go/httpclient"
	sharedModel "github.com/yourorg/trading-platform/shared/go/model"
	"go.uber.org/zap"
)

// UserClient handles communication with external user services
type UserClient struct {
	client *httpclient.Client
	logger *zap.Logger
}

// NewUserClient creates a new client for communication with external user services
func NewUserClient(baseURL string, logger *zap.Logger) *UserClient {
	client := httpclient.New(httpclient.Config{
		BaseURL:    baseURL,
		Timeout:    15 * time.Second,
		ServiceKey: "user-service-key",
	}, logger)

	return &UserClient{
		client: client,
		logger: logger,
	}
}

// GetUserByID retrieves a user from an external service by ID
func (c *UserClient) GetUserByID(ctx context.Context, userID int) (*model.User, error) {
	path := fmt.Sprintf("/api/v1/users/%d", userID)

	var user model.User
	err := c.client.Get(ctx, path, &user)
	if err != nil {
		c.logger.Error("Failed to get user by ID from external service",
			zap.Int("userID", userID),
			zap.Error(err))
		return nil, err
	}

	return &user, nil
}

// ListExternalUsers gets a list of users from an external service
func (c *UserClient) ListExternalUsers(ctx context.Context, pagination *sharedModel.Pagination) ([]model.User, *sharedModel.PaginationMeta, error) {
	path := fmt.Sprintf("/api/v1/users?page=%d&per_page=%d",
		pagination.Page, pagination.PerPage)

	var response struct {
		Data []model.User               `json:"data"`
		Meta sharedModel.PaginationMeta `json:"meta"`
	}

	err := c.client.Get(ctx, path, &response)
	if err != nil {
		c.logger.Error("Failed to list users from external service", zap.Error(err))
		return nil, nil, err
	}

	return response.Data, &response.Meta, nil
}

// SyncUserProfile syncs a local user profile with an external identity provider
func (c *UserClient) SyncUserProfile(ctx context.Context, userID int) error {
	path := fmt.Sprintf("/api/v1/users/%d/sync", userID)

	err := c.client.Post(ctx, path, nil, nil)
	if err != nil {
		c.logger.Error("Failed to sync user profile with external service",
			zap.Int("userID", userID),
			zap.Error(err))
		return err
	}

	return nil
}

// CreateExternalUser creates a user in an external system
func (c *UserClient) CreateExternalUser(ctx context.Context, userCreate *model.UserCreate) (int, error) {
	var response struct {
		ID int `json:"id"`
	}

	err := c.client.Post(ctx, "/api/v1/users", userCreate, &response)
	if err != nil {
		c.logger.Error("Failed to create user in external service", zap.Error(err))
		return 0, err
	}

	return response.ID, nil
}
