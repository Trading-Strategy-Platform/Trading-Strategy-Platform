package handler

import (
	"services/api-gateway/internal/proxy"

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

func getProxyPath(originalPath, prefix string) string {
	// Just return the original path - this ensures the microservices
	// receive exactly the same path that came to the gateway
	return originalPath
}
