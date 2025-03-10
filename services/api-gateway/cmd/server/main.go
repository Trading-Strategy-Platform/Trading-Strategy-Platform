package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"services/api-gateway/internal/client"
	"services/api-gateway/internal/config"
	"services/api-gateway/internal/handler"
	"services/api-gateway/internal/middleware"
	"services/api-gateway/internal/proxy"
	"services/api-gateway/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

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

	// Create service client manager with temporary values for missing services
	clientManager := client.NewServiceClientManager(
		cfg.UserService.URL,
		cfg.StrategyService.URL,
		cfg.HistoricalService.URL,
		"http://execution-service:8084",    // Temporary hardcoded value
		"http://notification-service:8085", // Temporary hardcoded value
		logger,
	)

	// Create service factory
	serviceFactory := proxy.NewServiceFactory(clientManager, logger)

	// Create service proxies
	userServiceProxy := serviceFactory.CreateUserServiceProxy()
	strategyServiceProxy := serviceFactory.CreateStrategyServiceProxy()
	historicalServiceProxy := serviceFactory.CreateHistoricalServiceProxy()

	// Create the API gateway handler
	gatewayHandler := handler.NewGatewayHandler(
		userServiceProxy,
		strategyServiceProxy,
		historicalServiceProxy,
		logger,
	)

	// Initialize Kafka service
	kafkaService, err := service.NewKafkaService(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topics,
		cfg.Kafka.Enabled,
		logger,
	)
	if err != nil {
		logger.Fatal("Failed to create Kafka service", zap.Error(err))
	}
	defer kafkaService.Close()

	// Set up HTTP server with Gin
	router := setupRouter(gatewayHandler, cfg, logger)

	// When setting up the router, add the metrics middleware
	if cfg.Kafka.Enabled {
		router.Use(middleware.MetricsMiddleware(kafkaService, logger))
	}

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

	logger.Info("Server exited properly")
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

func setupRouter(
	gatewayHandler *handler.GatewayHandler,
	cfg *config.Config,
	logger *zap.Logger,
) *gin.Engine {
	router := gin.New()

	// Use middlewares
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.CORS())
	router.Use(middleware.ErrorHandler())

	// Apply authentication middleware if enabled
	if cfg.Auth.Enabled {
		// Default paths that don't require authentication if not specified
		excludedPaths := cfg.Auth.ExcludedPaths
		if len(excludedPaths) == 0 {
			excludedPaths = []string{
				"/api/v1/auth/register",
				"/api/v1/auth/login",
				"/api/v1/health",
				"/metrics",
			}
		}

		// Default public paths (authenticated but no specific role required)
		publicPaths := cfg.Auth.PublicPaths
		if len(publicPaths) == 0 {
			publicPaths = []string{
				"/api/v1/user/profile",
				"/api/v1/symbols",
				"/api/v1/timeframes",
			}
		}

		// Default paths that require admin role
		adminRequiredPaths := cfg.Auth.AdminRequiredPaths
		if len(adminRequiredPaths) == 0 {
			adminRequiredPaths = []string{
				"/api/v1/admin/*",
				"/api/v1/users",
			}
		}

		authConfig := middleware.AuthConfig{
			JWTSecret:            cfg.Auth.JWTSecret,
			ExcludedPaths:        excludedPaths,
			PublicPaths:          publicPaths,
			AdminRequiredPaths:   adminRequiredPaths,
			EnableAuthentication: cfg.Auth.Enabled,
		}
		router.Use(middleware.AuthMiddleware(authConfig, logger))
	}

	// Rate limiting middleware (optional)
	if cfg.RateLimit.Enabled {
		router.Use(middleware.RateLimit(
			cfg.RateLimit.RequestsPerMinute,
			cfg.RateLimit.BurstSize,
		))
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Register routes
	gatewayHandler.RegisterRoutes(router)

	return router
}
