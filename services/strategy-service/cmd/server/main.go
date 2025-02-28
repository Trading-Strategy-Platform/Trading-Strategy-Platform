// services/strategy-service/cmd/server/main.go
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

	"services/strategy-service/internal/client"
	"services/strategy-service/internal/config"
	"services/strategy-service/internal/handler"
	"services/strategy-service/internal/middleware"
	"services/strategy-service/internal/repository"
	"services/strategy-service/internal/service"

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
	strategyRepo := repository.NewStrategyRepository(db, logger)
	versionRepo := repository.NewVersionRepository(db, logger)
	tagRepo := repository.NewTagRepository(db, logger)
	indicatorRepo := repository.NewIndicatorRepository(db, logger)
	marketplaceRepo := repository.NewMarketplaceRepository(db, logger)
	purchaseRepo := repository.NewPurchaseRepository(db, logger)
	reviewRepo := repository.NewReviewRepository(db, logger)

	// Initialize clients
	userClient := client.NewUserClient(cfg.UserService.URL, logger)
	historicalClient := client.NewHistoricalClient(cfg.HistoricalService.URL, logger)

	// Initialize services
	strategyService := service.NewStrategyService(
		db,
		strategyRepo,
		versionRepo,
		tagRepo,
		userClient,
		historicalClient,
		logger,
	)

	tagService := service.NewTagService(tagRepo, logger)
	indicatorService := service.NewIndicatorService(indicatorRepo, logger)
	marketplaceService := service.NewMarketplaceService(
		db,
		marketplaceRepo,
		strategyRepo,
		purchaseRepo,
		reviewRepo,
		userClient,
		logger,
	)

	// Initialize handlers
	strategyHandler := handler.NewStrategyHandler(strategyService, logger)
	tagHandler := handler.NewTagHandler(tagService, logger)
	indicatorHandler := handler.NewIndicatorHandler(indicatorService, logger)
	marketplaceHandler := handler.NewMarketplaceHandler(marketplaceService, logger)

	// Set up HTTP server with Gin
	router := setupRouter(
		strategyHandler,
		tagHandler,
		indicatorHandler,
		marketplaceHandler,
		userClient,
		logger,
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
	strategyHandler *handler.StrategyHandler,
	tagHandler *handler.TagHandler,
	indicatorHandler *handler.IndicatorHandler,
	marketplaceHandler *handler.MarketplaceHandler,
	userClient *client.UserClient,
	logger *zap.Logger,
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
		// Public routes
		v1.GET("/strategies/public", strategyHandler.ListPublicStrategies)
		v1.GET("/indicators", indicatorHandler.GetAllIndicators)
		v1.GET("/indicators/:id", indicatorHandler.GetIndicator)
		v1.GET("/tags", tagHandler.GetAllTags)
		v1.GET("/marketplace", marketplaceHandler.ListListings)
		v1.GET("/marketplace/:id", marketplaceHandler.GetListing)
		v1.GET("/marketplace/:id/reviews", marketplaceHandler.GetReviews)

		// Authentication required routes
		auth := v1.Group("/")
		auth.Use(middleware.AuthMiddleware(userClient, logger))
		{
			// Strategy routes
			auth.POST("/strategies", strategyHandler.CreateStrategy)
			auth.GET("/strategies", strategyHandler.ListUserStrategies)
			auth.GET("/strategies/:id", strategyHandler.GetStrategy)
			auth.PUT("/strategies/:id", strategyHandler.UpdateStrategy)
			auth.DELETE("/strategies/:id", strategyHandler.DeleteStrategy)
			auth.POST("/strategies/:id/versions", strategyHandler.CreateVersion)
			auth.GET("/strategies/:id/versions", strategyHandler.GetVersions)
			auth.GET("/strategies/:id/versions/:version", strategyHandler.GetVersion)
			auth.POST("/strategies/:id/versions/:version/restore", strategyHandler.RestoreVersion)
			auth.POST("/strategies/:id/clone", strategyHandler.CloneStrategy)
			auth.POST("/strategies/:id/backtest", strategyHandler.StartBacktest)

			// Tag routes
			auth.POST("/tags", tagHandler.CreateTag)

			// Marketplace routes
			auth.POST("/marketplace", marketplaceHandler.CreateListing)
			auth.PUT("/marketplace/:id", marketplaceHandler.UpdateListing)
			auth.DELETE("/marketplace/:id", marketplaceHandler.DeleteListing)
			auth.POST("/marketplace/:id/purchase", marketplaceHandler.PurchaseStrategy)
			auth.GET("/purchases", marketplaceHandler.GetPurchases)
			auth.POST("/marketplace/:id/reviews", marketplaceHandler.CreateReview)
		}
	}

	return router
}
