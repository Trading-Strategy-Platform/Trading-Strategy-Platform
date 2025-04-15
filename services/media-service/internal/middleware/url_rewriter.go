package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// URLRewriter middleware replaces internal domain references with the actual host
func URLRewriter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Save the original writer
		originalWriter := c.Writer

		// Create a custom response writer that will rewrite URLs
		c.Writer = &CustomResponseWriter{
			ResponseWriter: originalWriter,
			request:        c.Request,
		}

		// Process the request
		c.Next()

		// Restore the original writer
		c.Writer = originalWriter
	}
}

// CustomResponseWriter is a custom implementation of ResponseWriter
type CustomResponseWriter struct {
	gin.ResponseWriter
	request *http.Request
}

// Write intercepts the response body to rewrite URLs
func (w *CustomResponseWriter) Write(data []byte) (int, error) {
	// Get the host from the request
	host := w.request.Host

	// Only rewrite if content is JSON or HTML
	contentType := w.Header().Get("Content-Type")
	if strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/html") {
		// Replace internal service URLs with host-based URLs
		modifiedData := strings.ReplaceAll(
			string(data),
			"http://media-service:8085",
			"http://"+host,
		)

		return w.ResponseWriter.Write([]byte(modifiedData))
	}

	// For other content types, pass through unchanged
	return w.ResponseWriter.Write(data)
}

// Implement other required methods
func (w *CustomResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *CustomResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
