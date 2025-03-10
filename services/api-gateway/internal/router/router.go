package router

import (
	"services/api-gateway/internal/client"
	"services/api-gateway/internal/proxy"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Configure the router to use the new service factory
func ConfigureRouter(
	router *gin.Engine,
	clientManager *client.ServiceClientManager,
	logger *zap.Logger,
) {
	// Create service factory
	serviceFactory := proxy.NewServiceFactory(clientManager, logger)

	// User service routes
	userProxy := serviceFactory.CreateUserServiceProxy()
	userRoutes := router.Group("/api/users")
	{
		userRoutes.GET("", func(c *gin.Context) {
			userProxy.ProxyRequest(c, "/api/users")
		})
		userRoutes.POST("", func(c *gin.Context) {
			userProxy.ProxyRequest(c, "/api/users")
		})
		// Add other user routes as needed
	}

	// Strategy service routes
	strategyProxy := serviceFactory.CreateStrategyServiceProxy()
	strategyRoutes := router.Group("/api/strategies")
	{
		strategyRoutes.GET("", func(c *gin.Context) {
			strategyProxy.ProxyRequest(c, "/api/strategies")
		})
		// Add other strategy routes as needed
	}

	// Add routes for other services similarly
}
