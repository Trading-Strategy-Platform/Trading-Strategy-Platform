package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"services/api-gateway/internal/config"
	"services/api-gateway/internal/handler"
	"services/api-gateway/internal/kafka"
	"services/api-gateway/internal/middleware"
	"services/api-gateway/internal/proxy"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// Redis and Kafka configurations
type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers  []string
	ClientID string
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set up logger
	logger, err := createLogger(cfg.Logging.Level)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Initialize Redis client
	redisClient, err := setupRedis(cfg, logger)
	if err != nil {
		logger.Error("Failed to set up Redis", zap.Error(err))
		// Continue without Redis
	}

	// Initialize Kafka producer
	kafkaProducer := setupKafka(cfg, logger)

	// Create service proxies
	userServiceProxy := proxy.NewServiceProxy(cfg.UserService.URL, logger)
	strategyServiceProxy := proxy.NewServiceProxy(cfg.StrategyService.URL, logger)
	historicalServiceProxy := proxy.NewServiceProxy(cfg.HistoricalService.URL, logger)
	mediaServiceProxy := proxy.NewServiceProxy(cfg.MediaService.URL, logger)

	// Create the API gateway handler
	gatewayHandler := handler.NewGatewayHandler(
		userServiceProxy,
		strategyServiceProxy,
		historicalServiceProxy,
		mediaServiceProxy,
		logger,
	)

	// Set up HTTP server with Gin
	router := setupRouter(gatewayHandler, cfg, logger, redisClient, kafkaProducer)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start the server in a goroutine
	go func() {
		logger.Info("Starting API Gateway server", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	// Close Kafka producer
	if kafkaProducer != nil {
		kafkaProducer.Close()
	}

	// Close Redis client
	if redisClient != nil {
		redisClient.Close()
	}

	logger.Info("Server exited properly")
}

// setupRedis initializes the Redis client
func setupRedis(cfg *config.Config, logger *zap.Logger) (*redis.Client, error) {
	// Extract Redis URL from environment variable or config
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis:6379" // Default URL
	}

	// Parse Redis URL
	redisOptions, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Error("Failed to parse Redis URL", zap.Error(err))
		// Try to connect with default options
		redisOptions = &redis.Options{
			Addr: redisURL,
			DB:   0,
		}
	}

	// Create Redis client
	client := redis.NewClient(redisOptions)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Ping(ctx).Result()
	if err != nil {
		logger.Error("Failed to connect to Redis", zap.Error(err))
		return nil, err
	}

	logger.Info("Connected to Redis", zap.String("addr", redisOptions.Addr))
	return client, nil
}

// setupKafka initializes the Kafka producer
func setupKafka(cfg *config.Config, logger *zap.Logger) *kafka.Producer {
	// Extract Kafka brokers from environment variable or config
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "kafka:9092" // Default broker
	}

	// Split brokers string into slice
	brokers := strings.Split(kafkaBrokers, ",")

	// Create Kafka producer
	producer := kafka.NewProducer(brokers, "api-gateway", logger)

	logger.Info("Initialized Kafka producer", zap.Strings("brokers", brokers))
	return producer
}

func setupRouter(
	gatewayHandler *handler.GatewayHandler,
	cfg *config.Config,
	logger *zap.Logger,
	redisClient *redis.Client,
	kafkaProducer *kafka.Producer,
) *gin.Engine {
	router := gin.New()

	// Use standard middlewares
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.CORS())
	router.Use(middleware.DuplicatePathLogger(logger))

	// Redis-based rate limiting (if Redis is available)
	if redisClient != nil && cfg.RateLimit.Enabled {
		router.Use(middleware.RedisRateLimit(redisClient, middleware.RedisRateLimitConfig{
			Enabled:            cfg.RateLimit.Enabled,
			RequestsPerMinute:  cfg.RateLimit.RequestsPerMinute,
			BurstSize:          cfg.RateLimit.BurstSize,
			ClientIPHeaderName: cfg.RateLimit.ClientIPHeaderName,
		}, logger))
	} else if cfg.RateLimit.Enabled {
		// Fallback to in-memory rate limiter if Redis is not available
		router.Use(middleware.RateLimit(
			cfg.RateLimit.RequestsPerMinute,
			cfg.RateLimit.BurstSize,
		))
	}

	// Redis-based caching for read endpoints (if Redis is available)
	if redisClient != nil {
		router.Use(middleware.RedisCache(redisClient, middleware.CacheConfig{
			Enabled:         true,
			DefaultDuration: 5 * time.Minute,
			PrefixKey:       "api-cache",
			ExcludedPaths:   []string{"/health", "/api/v1/auth/login", "/api/v1/auth/register"},
		}, logger))
	}

	// Request auditing middleware using Kafka
	router.Use(func(c *gin.Context) {
		// Process the request
		c.Next()

		// After processing, log important requests to Kafka
		if kafkaProducer != nil && isImportantRequest(c) {
			// Audit the request
			auditRequest(c, kafkaProducer, logger)
		}
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		status := "healthy"

		// Check Redis connectivity if available
		if redisClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			_, err := redisClient.Ping(ctx).Result()
			if err != nil {
				status = "degraded"
				logger.Warn("Redis health check failed", zap.Error(err))
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status": status,
			"redis":  redisClient != nil,
			"kafka":  kafkaProducer != nil,
		})
	})

	// Media routes
	router.Any("/media/*path", gatewayHandler.ProxyMediaService)

	// API routes
	api := router.Group("/api")
	{
		// USER SERVICE ROUTES
		api.Any("/v1/auth/login", gatewayHandler.ProxyUserService)
		api.Any("/v1/auth/register", gatewayHandler.ProxyUserService)
		api.Any("/v1/auth/refresh", gatewayHandler.ProxyUserService)
		api.Any("/v1/auth/validate", gatewayHandler.ProxyUserService)
		api.Any("/v1/users/me", gatewayHandler.ProxyUserService)
		api.Any("/v1/users", gatewayHandler.ProxyUserService)
		api.Any("/v1/users/:id", gatewayHandler.ProxyUserService)
		api.Any("/v1/admin/users", gatewayHandler.ProxyUserService)
		api.Any("/v1/admin/users/:id", gatewayHandler.ProxyUserService)
		api.Any("/v1/admin/users/:id/roles", gatewayHandler.ProxyUserService)
		api.Any("/v1/notifications", gatewayHandler.ProxyUserService)
		api.Any("/v1/notifications/:id", gatewayHandler.ProxyUserService)

		// STRATEGY SERVICE ROUTES
		api.Any("/v1/indicators", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/indicators/categories", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/indicators/:id", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/indicators/:id/parameters", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/parameters/:id", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/parameters/:id/enum-values", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/enum-values/:id", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategies", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategies/:id", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategies/:id/versions", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategies/:id/active-version", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategies/:id/thumbnail", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategy-tags", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategy-tags/:id", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/marketplace", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/marketplace/:id", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/marketplace/:id/reviews", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/marketplace/:id/purchase", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/marketplace/purchases/:id/cancel", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/reviews", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/reviews/:id", gatewayHandler.ProxyStrategyService)

		// HISTORICAL SERVICE ROUTES - Use ONE wildcard route for all market-data endpoints
		api.Any("/v1/market-data/*path", gatewayHandler.ProxyHistoricalService)

		// Other historical service routes that don't start with market-data
		api.Any("/v1/backtests", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/backtests/:id", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/backtest-runs", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/backtest-runs/:id", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/backtest-runs/:id/*path", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/symbols", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/symbols/:id", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/timeframes", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/timeframes/:id", gatewayHandler.ProxyHistoricalService)

		// MEDIA SERVICE ROUTES
		api.Any("/v1/media/upload", gatewayHandler.ProxyMediaService)
		api.Any("/v1/media/:id", gatewayHandler.ProxyMediaService)
		api.Any("/v1/media/by-path/*path", gatewayHandler.ProxyMediaService)
	}

	return router
}

// isImportantRequest checks if a request should be audited
func isImportantRequest(c *gin.Context) bool {
	// Audit login/register, admin operations, purchases, and other important operations
	path := c.Request.URL.Path
	method := c.Request.Method

	// Admin actions
	if strings.Contains(path, "/admin/") {
		return true
	}

	// Auth actions
	if strings.Contains(path, "/auth/") && (method == "POST" || method == "PUT") {
		return true
	}

	// Marketplace purchases
	if strings.Contains(path, "/marketplace") &&
		(strings.Contains(path, "/purchase") || strings.Contains(path, "/cancel")) {
		return true
	}

	// Strategy creation or updates
	if strings.Contains(path, "/strategies") && (method == "POST" || method == "PUT") {
		return true
	}

	return false
}

// auditRequest publishes request audit data to Kafka
func auditRequest(c *gin.Context, producer *kafka.Producer, logger *zap.Logger) {
	// Extract user ID from authorization header or context
	userID := "anonymous"
	if id, exists := c.Get("user_id"); exists {
		userID = id.(string)
	}

	// Create audit message
	auditData := map[string]interface{}{
		"user_id":    userID,
		"client_ip":  c.ClientIP(),
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
		"status":     c.Writer.Status(),
		"user_agent": c.Request.UserAgent(),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	// Determine which topic to use based on the path
	topic := "user-events" // Default topic

	if strings.Contains(c.Request.URL.Path, "/marketplace") {
		topic = "marketplace-events"
	} else if strings.Contains(c.Request.URL.Path, "/strategies") {
		topic = "strategy-events"
	} else if strings.Contains(c.Request.URL.Path, "/backtest") {
		topic = "backtest-events"
	}

	// Send to Kafka
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := producer.Publish(ctx, topic, kafka.Message{
		Key:   userID,
		Value: auditData,
	})

	if err != nil {
		logger.Error("Failed to publish audit event",
			zap.Error(err),
			zap.String("topic", topic),
			zap.String("user_id", userID))
	}
}

func createLogger(level string) (*zap.Logger, error) {
	// Parse log level
	var zapLevel zap.AtomicLevel
	switch level {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Create logger config
	config := zap.Config{
		Level:            zapLevel,
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}
