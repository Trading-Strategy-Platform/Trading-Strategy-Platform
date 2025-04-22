package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// CacheConfig holds configuration for the cache middleware
type CacheConfig struct {
	Enabled         bool
	DefaultDuration time.Duration
	PrefixKey       string
	ExcludedPaths   []string
}

// RedisCache creates middleware for caching responses in Redis
func RedisCache(redisClient *redis.Client, config CacheConfig, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if caching is disabled or request method is not GET
		if !config.Enabled || c.Request.Method != "GET" {
			c.Next()
			return
		}

		// Skip excluded paths
		for _, path := range config.ExcludedPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// Generate cache key
		cacheKey := generateCacheKey(c, config.PrefixKey)

		// Try to get from cache
		ctx := context.Background()
		cachedResponse, err := redisClient.Get(ctx, cacheKey).Bytes()
		if err == nil {
			// Cache hit
			logger.Debug("Cache hit",
				zap.String("path", c.Request.URL.Path),
				zap.String("cache_key", cacheKey))

			// Create a custom ResponseWriter to capture status and headers
			c.Writer.Header().Set("Content-Type", "application/json")
			c.Writer.Header().Set("X-Cache", "HIT")
			c.Writer.WriteHeader(http.StatusOK)
			c.Writer.Write(cachedResponse)
			c.Abort()
			return
		}

		// Create a custom writer to capture the response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Only cache successful responses
		if c.Writer.Status() == http.StatusOK {
			// Store response in cache
			duration := config.DefaultDuration
			responseBody := writer.body.Bytes()

			err := redisClient.Set(ctx, cacheKey, responseBody, duration).Err()
			if err != nil {
				logger.Error("Failed to set cache",
					zap.Error(err),
					zap.String("cache_key", cacheKey))
			} else {
				logger.Debug("Cache set",
					zap.String("path", c.Request.URL.Path),
					zap.String("cache_key", cacheKey),
					zap.Duration("duration", duration))
			}
		}
	}
}

// responseWriter captures the response body for caching
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write captures the response for caching
func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// generateCacheKey creates a unique cache key for a request
func generateCacheKey(c *gin.Context, prefix string) string {
	// Combine path and query parameters for the key
	path := c.Request.URL.Path
	query := c.Request.URL.RawQuery

	// Create a hash of the path and query
	hash := sha256.New()
	if query != "" {
		io.WriteString(hash, fmt.Sprintf("%s?%s", path, query))
	} else {
		io.WriteString(hash, path)
	}
	return prefix + ":" + hex.EncodeToString(hash.Sum(nil))
}

// FlushCache clears the cache for a specific path or all paths
func FlushCache(redisClient *redis.Client, prefix string, path string) error {
	ctx := context.Background()

	if path == "" {
		// Flush all cache with the prefix
		pattern := prefix + ":*"
		keys, err := redisClient.Keys(ctx, pattern).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			return redisClient.Del(ctx, keys...).Err()
		}
		return nil
	}

	// Flush specific path
	hash := sha256.New()
	io.WriteString(hash, path)
	cacheKey := prefix + ":" + hex.EncodeToString(hash.Sum(nil))

	return redisClient.Del(ctx, cacheKey).Err()
}
