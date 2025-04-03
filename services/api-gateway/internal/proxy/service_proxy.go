package proxy

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ServiceProxy handles proxying requests to a specific service
type ServiceProxy struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewServiceProxy creates a new service proxy
func NewServiceProxy(baseURL string, logger *zap.Logger) *ServiceProxy {
	return &ServiceProxy{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ProxyRequest proxies a request to the service and returns the response
func (p *ServiceProxy) ProxyRequest(c *gin.Context, path string) {
	// Construct the target URL
	targetURL, err := url.Parse(p.baseURL)
	if err != nil {
		p.logger.Error("Failed to parse base URL", zap.Error(err), zap.String("baseURL", p.baseURL))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gateway configuration error"})
		return
	}

	// Handle path with or without leading slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Log the actual target URL being constructed
	p.logger.Debug("Constructing target URL",
		zap.String("baseURL", p.baseURL),
		zap.String("path", path),
		zap.String("fullURL", targetURL.String()+path))

	// Set the target URL path
	targetURL.Path = path

	// Add query string if any
	if c.Request.URL.RawQuery != "" {
		targetURL.RawQuery = c.Request.URL.RawQuery
	}

	// Create a new request
	req, err := http.NewRequest(c.Request.Method, targetURL.String(), c.Request.Body)
	if err != nil {
		p.logger.Error("Failed to create request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Copy headers from the original request
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Add X-Forwarded headers
	req.Header.Set("X-Forwarded-For", c.ClientIP())
	req.Header.Set("X-Forwarded-Proto", c.Request.Proto)
	req.Header.Set("X-Forwarded-Host", c.Request.Host)

	// Make the request to the target service
	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Error("Failed to proxy request",
			zap.Error(err),
			zap.String("url", targetURL.String()))
		c.JSON(http.StatusBadGateway, gin.H{"error": "Service unavailable"})
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Set status code
	c.Status(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		p.logger.Error("Failed to copy response body", zap.Error(err))
		// Response has already started, cannot send an error response
		return
	}
}

// ProxyWithReverseProxy uses httputil.ReverseProxy to proxy requests
// This is an alternative implementation that can be used instead of ProxyRequest
func (p *ServiceProxy) ProxyWithReverseProxy(c *gin.Context, path string) {
	targetURL, err := url.Parse(p.baseURL)
	if err != nil {
		p.logger.Error("Failed to parse base URL", zap.Error(err), zap.String("baseURL", p.baseURL))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gateway configuration error"})
		return
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize the Director function to properly set the URL path
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Handle path with or without leading slash
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		// Set the target URL path
		req.URL.Path = path

		// Add X-Forwarded headers
		req.Header.Set("X-Forwarded-For", c.ClientIP())
		req.Header.Set("X-Forwarded-Proto", c.Request.Proto)
		req.Header.Set("X-Forwarded-Host", c.Request.Host)

		// Log the modified request
		p.logger.Debug("Reverse proxy request",
			zap.String("path", path),
			zap.String("fullURL", req.URL.String()))
	}

	// Handle errors
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		p.logger.Error("Reverse proxy error",
			zap.Error(err),
			zap.String("url", req.URL.String()))

		// Write error response
		rw.WriteHeader(http.StatusBadGateway)
		_, _ = rw.Write([]byte(`{"error": "Service unavailable"}`))
	}

	// Serve the request
	proxy.ServeHTTP(c.Writer, c.Request)
}
