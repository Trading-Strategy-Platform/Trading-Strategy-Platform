package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"services/api-gateway/internal/config"
	"services/api-gateway/internal/handler"
	"services/api-gateway/internal/middleware"
	"services/api-gateway/internal/proxy"

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

	// Create service proxies
	userServiceProxy := proxy.NewServiceProxy(cfg.UserService.URL, logger)
	strategyServiceProxy := proxy.NewServiceProxy(cfg.StrategyService.URL, logger)
	historicalServiceProxy := proxy.NewServiceProxy(cfg.HistoricalService.URL, logger)

	// Create the API gateway handler
	gatewayHandler := handler.NewGatewayHandler(
		userServiceProxy,
		strategyServiceProxy,
		historicalServiceProxy,
		logger,
	)

	// Set up HTTP server with Gin
	router := setupRouter(gatewayHandler, cfg, logger)

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
	router.Use(middleware.DuplicatePathLogger(logger))

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

	// API routes
	api := router.Group("/api")
	{
		// ==================== USER SERVICE ROUTES ====================
		// Auth routes
		api.Any("/v1/auth/login", gatewayHandler.ProxyUserService)
		api.Any("/v1/auth/register", gatewayHandler.ProxyUserService)
		api.Any("/v1/auth/refresh", gatewayHandler.ProxyUserService)
		api.Any("/v1/auth/validate", gatewayHandler.ProxyUserService)

		// User routes
		api.Any("/v1/users/me", gatewayHandler.ProxyUserService)  // Specific route first
		api.Any("/v1/users", gatewayHandler.ProxyUserService)     // Base route next
		api.Any("/v1/users/:id", gatewayHandler.ProxyUserService) // Parameter routes last

		// Admin routes
		api.Any("/v1/admin/users", gatewayHandler.ProxyUserService)
		api.Any("/v1/admin/users/:id", gatewayHandler.ProxyUserService)
		api.Any("/v1/admin/users/:id/roles", gatewayHandler.ProxyUserService)

		// Notifications routes
		api.Any("/v1/notifications", gatewayHandler.ProxyUserService)
		api.Any("/v1/notifications/:id", gatewayHandler.ProxyUserService)

		// ==================== STRATEGY SERVICE ROUTES ====================
		// Indicator routes - ORDER MATTERS!
		api.Any("/v1/indicators", gatewayHandler.ProxyStrategyService)            // Base route
		api.Any("/v1/indicators/categories", gatewayHandler.ProxyStrategyService) // Static/enum route first
		api.Any("/v1/indicators/:id", gatewayHandler.ProxyStrategyService)        // Parameter route last
		api.Any("/v1/indicators/:id/parameters", gatewayHandler.ProxyStrategyService)

		// Parameter routes
		api.Any("/v1/parameters/:id", gatewayHandler.ProxyStrategyService) // For PUT and DELETE operations
		api.Any("/v1/parameters/:id/enum-values", gatewayHandler.ProxyStrategyService)

		// Enum values routes
		api.Any("/v1/enum-values/:id", gatewayHandler.ProxyStrategyService) // For PUT and DELETE operations

		// Strategy routes
		api.Any("/v1/strategies", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategies/:id", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategies/:id/versions", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategies/:id/active-version", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategies/:id/thumbnail", gatewayHandler.ProxyStrategyService)

		// Strategy tags routes
		api.Any("/v1/strategy-tags", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/strategy-tags/:id", gatewayHandler.ProxyStrategyService)

		// Marketplace routes
		api.Any("/v1/marketplace", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/marketplace/:id", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/marketplace/:id/reviews", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/marketplace/:id/purchase", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/marketplace/purchases/:id/cancel", gatewayHandler.ProxyStrategyService)

		// Reviews routes
		api.Any("/v1/reviews", gatewayHandler.ProxyStrategyService)
		api.Any("/v1/reviews/:id", gatewayHandler.ProxyStrategyService)

		// ==================== HISTORICAL SERVICE ROUTES ====================
		// Market data routes
		api.Any("/v1/market-data", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/market-data/:id", gatewayHandler.ProxyHistoricalService)

		// Backtest routes
		api.Any("/v1/backtests", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/backtests/:id", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/backtest-runs", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/backtest-runs/:id", gatewayHandler.ProxyHistoricalService)

		// Symbol routes
		api.Any("/v1/symbols", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/symbols/:id", gatewayHandler.ProxyHistoricalService)

		// Timeframe routes
		api.Any("/v1/timeframes", gatewayHandler.ProxyHistoricalService)
		api.Any("/v1/timeframes/:id", gatewayHandler.ProxyHistoricalService)
	}

	return router
}
