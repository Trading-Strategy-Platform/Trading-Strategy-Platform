package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/trading-platform/shared/go/httpclient"
	"github.com/yourorg/trading-platform/shared/go/response"
	"go.uber.org/zap"
)

// ServiceProxy handles proxying requests to a specific service
type ServiceProxy struct {
	client      *httpclient.Client
	logger      *zap.Logger
	serviceName string
}

// NewServiceProxy creates a new service proxy
func NewServiceProxy(client *httpclient.Client, serviceName string, logger *zap.Logger) *ServiceProxy {
	return &ServiceProxy{
		client:      client,
		logger:      logger,
		serviceName: serviceName,
	}
}

// ProxyRequest proxies a request to the service using the shared HTTP client
func (p *ServiceProxy) ProxyRequest(c *gin.Context, path string) {
	// Handle path with or without leading slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Create context with any auth token from original request
	requestCtx := context.Background()
	if token := c.GetHeader("Authorization"); token != "" {
		requestCtx = context.WithValue(requestCtx, "auth_token", token)
	}

	// Extract request body if present
	var reqBody io.Reader
	if c.Request.Body != nil {
		reqBody = c.Request.Body
	}

	// Determine method and make appropriate request
	method := strings.ToUpper(c.Request.Method)
	var respBody interface{}
	var err error

	// Add query string if any
	if c.Request.URL.RawQuery != "" {
		path = fmt.Sprintf("%s?%s", path, c.Request.URL.RawQuery)
	}

	// Execute request based on HTTP method
	switch method {
	case http.MethodGet:
		err = p.client.Get(requestCtx, path, &respBody)
	case http.MethodPost:
		err = p.client.Post(requestCtx, path, reqBody, &respBody)
	case http.MethodPut:
		err = p.client.Put(requestCtx, path, reqBody, &respBody)
	case http.MethodDelete:
		err = p.client.Delete(requestCtx, path, &respBody)
	case http.MethodPatch:
		err = p.client.Patch(requestCtx, path, reqBody, &respBody)
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
		return
	}

	// Handle errors
	if err != nil {
		p.logger.Error("Service request failed",
			zap.String("service", p.serviceName),
			zap.String("path", path),
			zap.String("method", method),
			zap.Error(err))

		// Convert httpclient errors to appropriate responses
		if httpErr, ok := err.(*httpclient.Error); ok {
			c.Status(httpErr.StatusCode)
			c.Writer.Write(httpErr.ResponseBody)
			return
		}

		response.InternalError(c, "Service unavailable")
		return
	}

	// Return response
	c.JSON(http.StatusOK, respBody)
}

// ProxyWithReverseProxy uses httputil.ReverseProxy to proxy requests
// This is kept as a backup method for complex proxying needs
func (p *ServiceProxy) ProxyWithReverseProxy(c *gin.Context, path string) {
	// Extract base URL from the client config
	baseURL, _ := url.Parse(p.client.BaseURL())

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(baseURL)

	// Customize the Director function to properly set the URL path
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Set the target URL path
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		req.URL.Path = path

		// Add X-Forwarded headers
		req.Header.Set("X-Forwarded-For", c.ClientIP())
		req.Header.Set("X-Forwarded-Proto", c.Request.Proto)
		req.Header.Set("X-Forwarded-Host", c.Request.Host)

		// Set service key header for internal communication
		req.Header.Set("X-Service-Key", "api-gateway-key")
	}

	// Handle errors
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		p.logger.Error("Reverse proxy error",
			zap.Error(err),
			zap.String("service", p.serviceName),
			zap.String("url", req.URL.String()))

		// Write error response
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusBadGateway)
		_, _ = rw.Write([]byte(`{"error": {"type": "service_error", "code": "service_unavailable", "message": "Service unavailable"}}`))
	}

	// Serve the request
	proxy.ServeHTTP(c.Writer, c.Request)
}
