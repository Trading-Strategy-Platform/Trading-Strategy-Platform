package proxy

import (
	"services/api-gateway/internal/client"

	"go.uber.org/zap"
)

// ServiceFactory creates service proxies for all backend services
type ServiceFactory struct {
	clientManager *client.ServiceClientManager
	logger        *zap.Logger
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(clientManager *client.ServiceClientManager, logger *zap.Logger) *ServiceFactory {
	return &ServiceFactory{
		clientManager: clientManager,
		logger:        logger,
	}
}

// CreateUserServiceProxy creates a proxy for the User Service
func (f *ServiceFactory) CreateUserServiceProxy() *ServiceProxy {
	return NewServiceProxy(f.clientManager.UserClient(), "user-service", f.logger)
}

// CreateStrategyServiceProxy creates a proxy for the Strategy Service
func (f *ServiceFactory) CreateStrategyServiceProxy() *ServiceProxy {
	return NewServiceProxy(f.clientManager.StrategyClient(), "strategy-service", f.logger)
}

// CreateHistoricalServiceProxy creates a proxy for the Historical Data Service
func (f *ServiceFactory) CreateHistoricalServiceProxy() *ServiceProxy {
	return NewServiceProxy(f.clientManager.HistoricalClient(), "historical-service", f.logger)
}

// CreateExecutionServiceProxy creates a proxy for the Execution Service
func (f *ServiceFactory) CreateExecutionServiceProxy() *ServiceProxy {
	return NewServiceProxy(f.clientManager.ExecutionClient(), "execution-service", f.logger)
}

// CreateNotificationServiceProxy creates a proxy for the Notification Service
func (f *ServiceFactory) CreateNotificationServiceProxy() *ServiceProxy {
	return NewServiceProxy(f.clientManager.NotificationClient(), "notification-service", f.logger)
}
