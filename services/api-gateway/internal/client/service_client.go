package client

import (
	"time"

	"github.com/yourorg/trading-platform/shared/go/httpclient"
	"go.uber.org/zap"
)

// ServiceClientManager manages HTTP clients for different backend services
type ServiceClientManager struct {
	userClient         *httpclient.Client
	strategyClient     *httpclient.Client
	historicalClient   *httpclient.Client
	executionClient    *httpclient.Client
	notificationClient *httpclient.Client
	validatorClient    *ValidatorClient
	logger             *zap.Logger
}

// NewServiceClientManager creates a new service client manager
func NewServiceClientManager(
	userServiceURL string,
	strategyServiceURL string,
	historicalServiceURL string,
	executionServiceURL string,
	notificationServiceURL string,
	logger *zap.Logger,
) *ServiceClientManager {
	// Create HTTP clients for each service with appropriate configuration
	userClient := httpclient.New(httpclient.Config{
		BaseURL:       userServiceURL,
		Timeout:       10 * time.Second,
		ServiceKey:    "api-gateway-key",
		RetryAttempts: 1,
	}, logger)

	strategyClient := httpclient.New(httpclient.Config{
		BaseURL:       strategyServiceURL,
		Timeout:       15 * time.Second,
		ServiceKey:    "api-gateway-key",
		RetryAttempts: 2,
	}, logger)

	historicalClient := httpclient.New(httpclient.Config{
		BaseURL:       historicalServiceURL,
		Timeout:       30 * time.Second,
		ServiceKey:    "api-gateway-key",
		RetryAttempts: 1,
	}, logger)

	executionClient := httpclient.New(httpclient.Config{
		BaseURL:       executionServiceURL,
		Timeout:       20 * time.Second,
		ServiceKey:    "api-gateway-key",
		RetryAttempts: 1,
	}, logger)

	notificationClient := httpclient.New(httpclient.Config{
		BaseURL:       notificationServiceURL,
		Timeout:       5 * time.Second,
		ServiceKey:    "api-gateway-key",
		RetryAttempts: 3,
	}, logger)

	return &ServiceClientManager{
		userClient:         userClient,
		strategyClient:     strategyClient,
		historicalClient:   historicalClient,
		executionClient:    executionClient,
		notificationClient: notificationClient,
		validatorClient:    NewValidatorClient(),
		logger:             logger,
	}
}

// ValidatorClient returns the validator client
func (m *ServiceClientManager) ValidatorClient() *ValidatorClient {
	return m.validatorClient
}

// UserClient returns the HTTP client for the User Service
func (m *ServiceClientManager) UserClient() *httpclient.Client {
	return m.userClient
}

// StrategyClient returns the HTTP client for the Strategy Service
func (m *ServiceClientManager) StrategyClient() *httpclient.Client {
	return m.strategyClient
}

// HistoricalClient returns the HTTP client for the Historical Data Service
func (m *ServiceClientManager) HistoricalClient() *httpclient.Client {
	return m.historicalClient
}

// ExecutionClient returns the HTTP client for the Execution Service
func (m *ServiceClientManager) ExecutionClient() *httpclient.Client {
	return m.executionClient
}

// NotificationClient returns the HTTP client for the Notification Service
func (m *ServiceClientManager) NotificationClient() *httpclient.Client {
	return m.notificationClient
}
