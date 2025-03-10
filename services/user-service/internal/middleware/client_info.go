package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
)

// ClientInfoMiddleware extracts client information and adds it to context
func ClientInfoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Add client IP address
		ctx = context.WithValue(ctx, "client_ip", c.ClientIP())

		// Add user agent
		ctx = context.WithValue(ctx, "user_agent", c.Request.UserAgent())

		// Update the request context
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
