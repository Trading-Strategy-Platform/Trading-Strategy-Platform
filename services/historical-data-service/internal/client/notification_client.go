package client

import (
	"context"
	"fmt"
	"time"

	"github.com/yourorg/trading-platform/shared/go/httpclient"
	"go.uber.org/zap"
)

// NotificationClient handles communication with the notification service
type NotificationClient struct {
	client *httpclient.Client
	logger *zap.Logger
}

// NewNotificationClient creates a new notification client
func NewNotificationClient(baseURL string, logger *zap.Logger) *NotificationClient {
	client := httpclient.New(httpclient.Config{
		BaseURL:       baseURL,
		Timeout:       10 * time.Second,
		ServiceKey:    "historical-service-key",
		RetryAttempts: 3,
	}, logger)

	return &NotificationClient{
		client: client,
		logger: logger,
	}
}

// SendBacktestCompleteNotification notifies a user that their backtest is complete
func (c *NotificationClient) SendBacktestCompleteNotification(
	ctx context.Context,
	userID int,
	backtestID int,
	strategyName string,
	status string,
) error {
	path := "/api/v1/notifications"

	payload := struct {
		UserID       int    `json:"user_id"`
		Type         string `json:"type"`
		Title        string `json:"title"`
		Message      string `json:"message"`
		ResourceType string `json:"resource_type"`
		ResourceID   int    `json:"resource_id"`
	}{
		UserID:       userID,
		Type:         "backtest_complete",
		Title:        "Backtest Complete",
		Message:      fmt.Sprintf("Your backtest for strategy '%s' is now %s", strategyName, status),
		ResourceType: "backtest",
		ResourceID:   backtestID,
	}

	// Add service key to context
	serviceCtx := context.WithValue(ctx, "service_key", "historical-service-key")

	err := c.client.Post(serviceCtx, path, payload, nil)
	if err != nil {
		c.logger.Error("Failed to send backtest complete notification",
			zap.Int("userID", userID),
			zap.Int("backtestID", backtestID),
			zap.Error(err))
		return err
	}

	return nil
}
