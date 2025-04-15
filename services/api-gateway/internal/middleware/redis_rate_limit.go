package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// RedisRateLimitConfig holds configuration for the rate limiter
type RedisRateLimitConfig struct {
	Enabled            bool
	RequestsPerMinute  int
	BurstSize          int
	ClientIPHeaderName string
}

// RedisRateLimit creates middleware for rate limiting requests using Redis
func RedisRateLimit(redisClient *redis.Client, config RedisRateLimitConfig, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		// Get client IP
		clientIP := c.ClientIP()

		// Use header if specified
		if config.ClientIPHeaderName != "" {
			if headerIP := c.GetHeader(config.ClientIPHeaderName); headerIP != "" {
				clientIP = headerIP
			}
		}

		// Check rate limit
		allowed, remaining, resetTime, err := checkRateLimit(redisClient, clientIP, config.RequestsPerMinute, config.BurstSize)
		if err != nil {
			logger.Error("Rate limit check failed", zap.Error(err), zap.String("client_ip", clientIP))
			c.Next() // Continue on error
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))

		if !allowed {
			c.Header("Retry-After", strconv.FormatInt(resetTime-time.Now().Unix(), 10))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkRateLimit checks if a request is allowed based on rate limits
func checkRateLimit(redisClient *redis.Client, key string, requestsPerMinute int, burstSize int) (bool, int, int64, error) {
	ctx := context.Background()
	now := time.Now()
	windowKey := fmt.Sprintf("ratelimit:%s:%d", key, now.Unix()/60) // Per minute window
	countKey := fmt.Sprintf("ratelimit:%s:count", key)

	// Execute rate limit script
	script := redis.NewScript(`
		local window_key = KEYS[1]
		local count_key = KEYS[2]
		local limit = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local reset_time = math.ceil(now/60) * 60
		
		local current = tonumber(redis.call('GET', count_key) or "0")
		
		-- Check if already over limit
		if current >= limit then
			return {0, current, reset_time}
		end
		
		-- Increment and check
		redis.call('INCR', count_key)
		redis.call('EXPIRE', count_key, 60)
		
		-- Get updated count
		current = tonumber(redis.call('GET', count_key) or "1")
		
		if current <= limit then
			return {1, limit - current, reset_time}
		else
			return {0, 0, reset_time}
		end
	`)

	// Run the script
	result, err := script.Run(
		ctx,
		redisClient,
		[]string{windowKey, countKey},
		requestsPerMinute,
		burstSize,
		now.Unix(),
	).Result()

	if err != nil {
		return false, 0, 0, err
	}

	// Parse the result
	resultArray := result.([]interface{})
	allowed := resultArray[0].(int64) == 1
	remaining := int(resultArray[1].(int64))
	resetTime := resultArray[2].(int64)

	return allowed, remaining, resetTime, nil
}
