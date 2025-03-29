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
		// Strategy routes
		strategies := v1.Group("/strategies")
		{
			// Public strategy endpoints
			strategies.GET("/public", strategyHandler.ListPublicStrategies) // Use get_public_strategies

			// Protected strategy endpoints
			strategies.Use(middleware.AuthMiddleware(userClient, logger))
			strategies.GET("", strategyHandler.ListUserStrategies)          // Use get_my_strategies
			strategies.POST("", strategyHandler.CreateStrategy)             // Use add_strategy
			strategies.GET("/:id", strategyHandler.GetStrategy)             // Use strategy repo + check access
			strategies.PUT("/:id", strategyHandler.UpdateStrategy)          // Use update_strategy
			strategies.DELETE("/:id", strategyHandler.DeleteStrategy)       // Use delete_strategy
			strategies.POST("/:id/clone", strategyHandler.CloneStrategy)    // Custom logic
			strategies.POST("/:id/backtest", strategyHandler.StartBacktest) // Use historical service

			// Version management endpoints
			versions := strategies.Group("/:id/versions")
			versions.GET("", strategyHandler.GetVersions)                      // Use get_accessible_strategy_versions
			versions.POST("", strategyHandler.CreateVersion)                   // New version via update_strategy
			versions.GET("/:version", strategyHandler.GetVersion)              // Custom logic to get specific version
			versions.POST("/:version/restore", strategyHandler.RestoreVersion) // Custom logic

			strategies.PUT("/:id/active-version", strategyHandler.UpdateActiveVersion) // Use update_user_strategy_version
		}

		// Strategy tags routes
		tags := v1.Group("/strategy-tags")
		{
			tags.GET("", tagHandler.GetAllTags) // Use get_strategy_tags

			tags.Use(middleware.AuthMiddleware(userClient, logger))
			tags.POST("", tagHandler.CreateTag) // Use add_strategy_tag
		}

		// Indicator routes
		indicators := v1.Group("/indicators")
		{
			indicators.GET("", indicatorHandler.GetAllIndicators)         // Use get_indicators
			indicators.GET("/:id", indicatorHandler.GetIndicator)         // Use get_indicator_by_id
			indicators.GET("/categories", indicatorHandler.GetCategories) // Use get_indicator_categories

			// Admin-only routes for managing indicators
			adminIndicators := indicators.Group("")
			adminIndicators.Use(middleware.AuthMiddleware(userClient, logger))
			adminIndicators.Use(middleware.RequireRole(userService, "admin"))

			adminIndicators.POST("", indicatorHandler.CreateIndicator)                         // Use add_indicator
			adminIndicators.POST("/:id/parameters", indicatorHandler.AddParameter)             // Use add_indicator_parameter
			adminIndicators.POST("/parameters/:id/enum-values", indicatorHandler.AddEnumValue) // Use add_parameter_enum_value
		}

		// Marketplace routes
		marketplace := v1.Group("/marketplace")
		{
			marketplace.GET("", marketplaceHandler.ListListings)           // Use get_marketplace_strategies
			marketplace.GET("/:id", marketplaceHandler.GetListing)         // Custom logic with get details
			marketplace.GET("/:id/reviews", marketplaceHandler.GetReviews) // Use get_strategy_reviews

			// Protected marketplace endpoints
			marketplaceAuth := marketplace.Group("")
			marketplaceAuth.Use(middleware.AuthMiddleware(userClient, logger))

			marketplaceAuth.POST("", marketplaceHandler.CreateListing)                 // Use add_to_marketplace
			marketplaceAuth.PUT("/:id", marketplaceHandler.UpdateListing)              // Custom implementation
			marketplaceAuth.DELETE("/:id", marketplaceHandler.DeleteListing)           // Use remove_from_marketplace
			marketplaceAuth.POST("/:id/purchase", marketplaceHandler.PurchaseStrategy) // Use purchase_strategy

			// Reviews management
			marketplaceAuth.POST("/:id/reviews", marketplaceHandler.CreateReview) // Use add_review

			// Purchases management
			marketplaceAuth.GET("/purchases", marketplaceHandler.GetPurchases)                  // Custom implementation
			marketplaceAuth.PUT("/purchases/:id/cancel", marketplaceHandler.CancelSubscription) // Use cancel_subscription
		}

		// Reviews management (separate from marketplace for edit/delete)
		reviews := v1.Group("/reviews")
		{
			reviews.Use(middleware.AuthMiddleware(userClient, logger))
			reviews.PUT("/:id", marketplaceHandler.UpdateReview)    // Use edit_review
			reviews.DELETE("/:id", marketplaceHandler.DeleteReview) // Use delete_review
		}
	}

	return router
}
