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
	mediaServiceProxy      *proxy.ServiceProxy
	logger                 *zap.Logger
}

// NewGatewayHandler creates a new gateway handler
func NewGatewayHandler(
	userServiceProxy *proxy.ServiceProxy,
	strategyServiceProxy *proxy.ServiceProxy,
	historicalServiceProxy *proxy.ServiceProxy,
	mediaServiceProxy *proxy.ServiceProxy,
	logger *zap.Logger,
) *GatewayHandler {
	return &GatewayHandler{
		userServiceProxy:       userServiceProxy,
		strategyServiceProxy:   strategyServiceProxy,
		historicalServiceProxy: historicalServiceProxy,
		mediaServiceProxy:      mediaServiceProxy,
		logger:                 logger,
	}
}

// ProxyUserService proxies requests to the user service
func (h *GatewayHandler) ProxyUserService(c *gin.Context) {
	// Extract path to proxy, preserving any path parameters
	path := c.Request.URL.Path

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
	path := c.Request.URL.Path

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
	path := c.Request.URL.Path

	// Log the request
	h.logger.Debug("Proxying to historical data service",
		zap.String("method", c.Request.Method),
		zap.String("path", path),
		zap.String("client_ip", c.ClientIP()))

	// Proxy the request
	h.historicalServiceProxy.ProxyRequest(c, path)
}

// ProxyMediaService proxies requests to the media service
func (h *GatewayHandler) ProxyMediaService(c *gin.Context) {
	// Get the original path
	originalPath := c.Request.URL.Path

	// Determine the target path for the media service
	var targetPath string

	// For direct media access through /media/* routes, we need to map to the media service's /api/v1/media/* routes
	if strings.HasPrefix(originalPath, "/media/") {
		// Remove the /media prefix and add /api/v1/media
		relativePath := strings.TrimPrefix(originalPath, "/media")

		// Make sure we have a leading slash
		if !strings.HasPrefix(relativePath, "/") {
			relativePath = "/" + relativePath
		}

		// Determine if this is likely a file path (contains dots or specific folders)
		if strings.Contains(relativePath, ".") || strings.Contains(relativePath, "/strategy/") {
			// This is likely a file path, use the by-path route
			targetPath = "/api/v1/media/by-path" + relativePath
		} else {
			// This is likely an ID or other non-file request
			targetPath = "/api/v1/media" + relativePath
		}
	} else {
		// For API access through /api/v1/media/* paths, keep as is
		targetPath = originalPath
	}

	// Log the request
	h.logger.Debug("Proxying to media service",
		zap.String("method", c.Request.Method),
		zap.String("original_path", originalPath),
		zap.String("target_path", targetPath),
		zap.String("client_ip", c.ClientIP()))

	// Add cache headers for GET requests to media files
	if c.Request.Method == "GET" && (strings.Contains(originalPath, ".jpg") ||
		strings.Contains(originalPath, ".png") ||
		strings.Contains(originalPath, ".jpeg") ||
		strings.Contains(originalPath, ".gif") ||
		strings.Contains(originalPath, ".webp")) {
		c.Header("Cache-Control", "public, max-age=31536000")
		c.Header("Expires", "Thu, 31 Dec 2099 23:59:59 GMT")
	}

	// Proxy the request to the media service with the modified path
	h.mediaServiceProxy.ProxyRequest(c, targetPath)
}
