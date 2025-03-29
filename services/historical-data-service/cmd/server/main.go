package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"services/historical-data-service/internal/client"
	"services/historical-data-service/internal/config"
	"services/historical-data-service/internal/handler"
	"services/historical-data-service/internal/middleware"
	"services/historical-data-service/internal/repository"
	"services/historical-data-service/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
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

	// Connect to database
	db, err := connectToDB(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize repositories
	marketDataRepo := repository.NewMarketDataRepository(db, logger)
	backtestRepo := repository.NewBacktestRepository(db, logger)
	symbolRepo := repository.NewSymbolRepository(db, logger)
	timeframeRepo := repository.NewTimeframeRepository(db, logger)

	// Initialize clients
	userClient := client.NewUserClient(cfg.UserService.URL, logger)
	strategyClient := client.NewStrategyClient(cfg.StrategyService.URL, logger)

	// Initialize services
	marketDataService := service.NewMarketDataService(marketDataRepo, symbolRepo, logger)
	backtestService := service.NewBacktestService(
		backtestRepo,
		marketDataRepo,
		strategyClient,
		logger,
	)
	symbolService := service.NewSymbolService(symbolRepo, logger)
	timeframeService := service.NewTimeframeService(timeframeRepo, logger)

	// Initialize handlers
	marketDataHandler := handler.NewMarketDataHandler(marketDataService, logger)
	backtestHandler := handler.NewBacktestHandler(backtestService, logger)
	symbolHandler := handler.NewSymbolHandler(symbolService, logger)
	timeframeHandler := handler.NewTimeframeHandler(timeframeService, logger)

	// Set up HTTP server with Gin
	router := setupRouter(
		marketDataHandler,
		backtestHandler,
		symbolHandler,
		timeframeHandler,
		userClient,
		logger,
		cfg,
	)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start the server in a goroutine
	go func() {
		logger.Info("Starting server", zap.String("port", cfg.Server.Port))
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

func connectToDB(dbConfig config.DatabaseConfig) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.DBName,
		dbConfig.SSLMode,
	)

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(dbConfig.ConnMaxLifetime)

	return db, nil
}

func setupRouter(
	marketDataHandler *handler.MarketDataHandler,
	backtestHandler *handler.BacktestHandler,
	symbolHandler *handler.SymbolHandler,
	timeframeHandler *handler.TimeframeHandler,
	userClient *client.UserClient,
	logger *zap.Logger,
	cfg *config.Config,
) *gin.Engine {
	router := gin.New()

	// Use middlewares
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Symbol routes
		symbols := v1.Group("/symbols")
		{
			symbols.GET("", symbolHandler.GetAllSymbols) // Use get_symbols

			// Protected symbols management
			symbolsAuth := symbols.Group("")
			symbolsAuth.Use(middleware.AuthMiddleware(userClient, logger))
			symbolsAuth.POST("", symbolHandler.CreateSymbol)       // Use add_symbol
			symbolsAuth.PUT("/:id", symbolHandler.UpdateSymbol)    // Use update_symbol
			symbolsAuth.DELETE("/:id", symbolHandler.DeleteSymbol) // Use delete_symbol
		}

		// Timeframes routes
		timeframes := v1.Group("/timeframes")
		{
			timeframes.GET("", timeframeHandler.GetAllTimeframes) // Direct DB query
		}

		// Market data routes
		marketData := v1.Group("/market-data")
		{
			// Only authenticated users can access market data
			marketData.Use(middleware.AuthMiddleware(userClient, logger))

			marketData.GET("/candles", marketDataHandler.GetCandles)                // Use get_candles
			marketData.POST("/candles/batch", marketDataHandler.BatchImportCandles) // Use insert_candles

			// Asset types and exchanges
			marketData.GET("/asset-types", marketDataHandler.GetAssetTypes) // Use get_asset_types
			marketData.GET("/exchanges", marketDataHandler.GetExchanges)    // Use get_exchanges
		}

		// Backtest routes
		backtests := v1.Group("/backtests")
		{
			backtests.Use(middleware.AuthMiddleware(userClient, logger))

			backtests.GET("", backtestHandler.ListBacktests)         // Use get_backtest_summary
			backtests.POST("", backtestHandler.CreateBacktest)       // Use create_backtest
			backtests.GET("/:id", backtestHandler.GetBacktest)       // Use get_backtest_details
			backtests.DELETE("/:id", backtestHandler.DeleteBacktest) // Use delete_backtest
		}

		// Backtest run management
		backtestRuns := v1.Group("/backtest-runs")
		{
			backtestRuns.Use(middleware.AuthMiddleware(userClient, logger))

			backtestRuns.PUT("/:id/status", backtestHandler.UpdateBacktestRunStatus) // Use update_backtest_run_status
			backtestRuns.POST("/:id/results", backtestHandler.SaveBacktestResults)   // Use save_backtest_result
			backtestRuns.POST("/:id/trades", backtestHandler.AddBacktestTrade)       // Use add_backtest_trade
			backtestRuns.GET("/:id/trades", backtestHandler.GetBacktestTrades)       // Use get_backtest_trades
		}

		// Service-to-service routes (requires service key)
		service := v1.Group("/service")
		service.Use(middleware.ServiceAuthMiddleware(cfg.ServiceKey, logger))
		{
			// Internal routes for other services
			service.POST("/market-data/batch", marketDataHandler.BatchImportMarketData)
			service.POST("/backtests/notify", backtestHandler.NotifyBacktestComplete)
		}
	}
	return router
}
