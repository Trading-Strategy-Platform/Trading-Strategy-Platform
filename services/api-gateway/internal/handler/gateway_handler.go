package handler

import (
	"services/api-gateway/internal/proxy"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GatewayHandler handles routing requests to the appropriate service
type GatewayHandler struct {
	userServiceProxy       *proxy.ServiceProxy
	strategyServiceProxy   *proxy.ServiceProxy
	historicalServiceProxy *proxy.ServiceProxy
	logger                 *zap.Logger
}

// NewGatewayHandler creates a new gateway handler
func NewGatewayHandler(
	userServiceProxy *proxy.ServiceProxy,
	strategyServiceProxy *proxy.ServiceProxy,
	historicalServiceProxy *proxy.ServiceProxy,
	logger *zap.Logger,
) *GatewayHandler {
	return &GatewayHandler{
		userServiceProxy:       userServiceProxy,
		strategyServiceProxy:   strategyServiceProxy,
		historicalServiceProxy: historicalServiceProxy,
		logger:                 logger,
	}
}

// ProxyUserService proxies requests to the user service
func (h *GatewayHandler) ProxyUserService(c *gin.Context) {
	// Extract path to proxy, preserving any path parameters
	path := getProxyPath(c.Request.URL.Path, "/api/v1")

	// Log the request
	h.logger.Debug("Proxying to user service",
		zap.String("method", c.Request.Method),
		zap.String("path", path),
		zap.String("client_ip", c.ClientIP()))

	// Proxy the request
	h.userServiceProxy.ProxyRequest(c, path)
}

// ProxyStrategyService proxies requests to the strategy service
func (h *GatewayHandler) ProxyStrategyService(c *gin.Context) {
	// Extract path to proxy, preserving any path parameters
	path := getProxyPath(c.Request.URL.Path, "/api/v1")

	// Log the request
	h.logger.Debug("Proxying to strategy service",
		zap.String("method", c.Request.Method),
		zap.String("path", path),
		zap.String("client_ip", c.ClientIP()))

	// Proxy the request
	h.strategyServiceProxy.ProxyRequest(c, path)
}

// ProxyHistoricalService proxies requests to the historical data service
func (h *GatewayHandler) ProxyHistoricalService(c *gin.Context) {
	// Extract path to proxy, preserving any path parameters
	path := getProxyPath(c.Request.URL.Path, "/api/v1")

	// Log the request
	h.logger.Debug("Proxying to historical data service",
		zap.String("method", c.Request.Method),
		zap.String("path", path),
		zap.String("client_ip", c.ClientIP()))

	// Proxy the request
	h.historicalServiceProxy.ProxyRequest(c, path)
}

// getProxyPath extracts the path to proxy from the original request path
// It removes the API prefix (e.g. "/api/v1") from the path
func getProxyPath(originalPath, prefix string) string {
	// Check if the path starts with the prefix
	if strings.HasPrefix(originalPath, prefix) {
		// Remove the prefix
		path := originalPath[len(prefix):]

		// Ensure the path starts with "/" if it's not empty
		if path != "" && !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		return path
	}

	// If the path doesn't start with the prefix, return the original path
	return originalPath
}

// Add RegisterRoutes method to GatewayHandler
func (h *GatewayHandler) RegisterRoutes(router *gin.Engine) {
	// API v1 routes group
	v1 := router.Group("/api/v1")

	// Auth service routes
	auth := v1.Group("/auth")
	{
		auth.POST("/login", h.ProxyUserService)
		auth.POST("/register", h.ProxyUserService)
		auth.POST("/refresh-token", h.ProxyUserService)
		auth.POST("/validate", h.ProxyUserService)
	}

	// User management routes
	users := v1.Group("/users")
	{
		users.GET("/me", h.ProxyUserService)
		users.PUT("/me", h.ProxyUserService)
		users.PUT("/me/password", h.ProxyUserService)
	}

	// Admin routes
	admin := v1.Group("/admin")
	{
		admin.GET("/users", h.ProxyUserService)
		admin.GET("/users/:id", h.ProxyUserService)
		admin.PUT("/users/:id", h.ProxyUserService)
	}

	// Strategy routes
	strategy := v1.Group("/strategies")
	{
		strategy.GET("", h.ProxyStrategyService)
		strategy.POST("", h.ProxyStrategyService)
		strategy.GET("/public", h.ProxyStrategyService)
		strategy.GET("/:id", h.ProxyStrategyService)
		strategy.PUT("/:id", h.ProxyStrategyService)
		strategy.DELETE("/:id", h.ProxyStrategyService)
		strategy.POST("/:id/versions", h.ProxyStrategyService)
		strategy.GET("/:id/versions", h.ProxyStrategyService)
		strategy.GET("/:id/versions/:version", h.ProxyStrategyService)
		strategy.POST("/:id/versions/:version/restore", h.ProxyStrategyService)
		strategy.POST("/:id/clone", h.ProxyStrategyService)
		strategy.POST("/:id/backtest", h.ProxyStrategyService)
	}

	// Market data routes
	market := v1.Group("/market")
	{
		market.GET("/symbols", h.ProxyHistoricalService)
		market.GET("/timeframes", h.ProxyHistoricalService)
		market.GET("/data/:symbol_id/:timeframe_id", h.ProxyHistoricalService)
		market.POST("/data/import", h.ProxyHistoricalService)
	}

	// Backtest routes
	backtests := v1.Group("/backtests")
	{
		backtests.POST("", h.ProxyHistoricalService)
		backtests.GET("", h.ProxyHistoricalService)
		backtests.GET("/:id", h.ProxyHistoricalService)
		backtests.DELETE("/:id", h.ProxyHistoricalService)
	}

	// Marketplace routes
	marketplace := v1.Group("/marketplace")
	{
		marketplace.GET("", h.ProxyStrategyService)
		marketplace.POST("", h.ProxyStrategyService)
		marketplace.GET("/:id", h.ProxyStrategyService)
		marketplace.PUT("/:id", h.ProxyStrategyService)
		marketplace.DELETE("/:id", h.ProxyStrategyService)
		marketplace.POST("/:id/purchase", h.ProxyStrategyService)
		marketplace.GET("/:id/reviews", h.ProxyStrategyService)
		marketplace.POST("/:id/reviews", h.ProxyStrategyService)
	}

	// Purchases routes
	purchases := v1.Group("/purchases")
	{
		purchases.GET("", h.ProxyStrategyService)
	}

	// Tags routes
	tags := v1.Group("/tags")
	{
		tags.GET("", h.ProxyStrategyService)
		tags.POST("", h.ProxyStrategyService)
	}

	// Indicators routes
	indicators := v1.Group("/indicators")
	{
		indicators.GET("", h.ProxyStrategyService)
		indicators.GET("/:id", h.ProxyStrategyService)
	}
}
