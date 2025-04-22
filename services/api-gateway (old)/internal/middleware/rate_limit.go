package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a token bucket rate limiting algorithm
type RateLimiter struct {
	requestsPerMinute int
	burstSize         int
	clients           map[string]*TokenBucket
	mu                sync.Mutex
}

// TokenBucket implements a token bucket for rate limiting
type TokenBucket struct {
	tokens       float64
	lastRefill   time.Time
	tokensPerSec float64
	maxTokens    float64
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute, burstSize int) *RateLimiter {
	return &RateLimiter{
		requestsPerMinute: requestsPerMinute,
		burstSize:         burstSize,
		clients:           make(map[string]*TokenBucket),
	}
}

// Allow checks if a request is allowed based on rate limits
func (r *RateLimiter) Allow(clientIP string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get or create bucket for client IP
	bucket, exists := r.clients[clientIP]
	if !exists {
		tokensPerSec := float64(r.requestsPerMinute) / 60.0
		bucket = &TokenBucket{
			tokens:       float64(r.burstSize),
			lastRefill:   time.Now(),
			tokensPerSec: tokensPerSec,
			maxTokens:    float64(r.burstSize),
		}
		r.clients[clientIP] = bucket
	}

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.lastRefill = now
	bucket.tokens += elapsed * bucket.tokensPerSec
	if bucket.tokens > bucket.maxTokens {
		bucket.tokens = bucket.maxTokens
	}

	// Check if request can be allowed
	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

// RateLimit creates middleware for rate limiting requests
func RateLimit(requestsPerMinute, burstSize int) gin.HandlerFunc {
	limiter := NewRateLimiter(requestsPerMinute, burstSize)

	return func(c *gin.Context) {
		// Get client IP
		clientIP := c.ClientIP()

		// Check if request is allowed
		if !limiter.Allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
