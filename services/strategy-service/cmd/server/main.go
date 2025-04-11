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
	mediaClient := client.NewMediaClient(cfg.MediaService.URL, cfg.MediaService.ServiceKey, logger)

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
	indicatorService := service.NewIndicatorService(db, indicatorRepo, logger)
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
	strategyHandler := handler.NewStrategyHandler(strategyService, userClient, logger)
	tagHandler := handler.NewTagHandler(tagService, logger)
	indicatorHandler := handler.NewIndicatorHandler(indicatorService, logger)
	marketplaceHandler := handler.NewMarketplaceHandler(marketplaceService, logger)
	thumbnailHandler := handler.NewThumbnailHandler(strategyService, mediaClient, logger)

	// Set up HTTP server with Gin
	router := setupRouter(
		strategyHandler,
		tagHandler,
		indicatorHandler,
		marketplaceHandler,
		thumbnailHandler,
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
	thumbnailHandler *handler.ThumbnailHandler,
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
		// ==================== INDICATOR ROUTES ====================
		// IMPORTANT: Order matters - specific routes must come before parameter routes
		indicators := v1.Group("/indicators")
		{
			// 1. Base endpoint
			indicators.GET("", indicatorHandler.GetAllIndicators) // GET /api/v1/indicators

			// 2. Static/enum routes - must come before parameter routes!
			indicators.GET("/categories", indicatorHandler.GetCategories) // GET /api/v1/indicators/categories

			// 3. Parameter routes - these come last!
			indicators.GET("/:id", indicatorHandler.GetIndicator) // GET /api/v1/indicators/{id}

			// Admin-only routes for managing indicators
			adminIndicators := indicators.Group("")
			adminIndicators.Use(middleware.AuthMiddleware(userClient, logger))
			adminIndicators.Use(middleware.RequireRole(userClient, "admin"))

			adminIndicators.POST("", indicatorHandler.CreateIndicator)             // POST /api/v1/indicators
			adminIndicators.POST("/:id/parameters", indicatorHandler.AddParameter) // POST /api/v1/indicators/{id}/parameters
		}

		// Parameter enum values route - separate from indicators to avoid conflicts
		v1.POST("/parameters/:id/enum-values", indicatorHandler.AddEnumValue) // POST /api/v1/parameters/{id}/enum-values

		// ==================== STRATEGY ROUTES ====================
		strategies := v1.Group("/strategies")
		{
			strategies.Use(middleware.AuthMiddleware(userClient, logger))

			// Base route
			strategies.GET("", strategyHandler.ListUserStrategies) // GET /api/v1/strategies
			strategies.POST("", strategyHandler.CreateStrategy)    // POST /api/v1/strategies

			// Parameter routes
			strategies.GET("/:id", strategyHandler.GetStrategy)                        // GET /api/v1/strategies/{id}
			strategies.PUT("/:id", strategyHandler.UpdateStrategy)                     // PUT /api/v1/strategies/{id}
			strategies.DELETE("/:id", strategyHandler.DeleteStrategy)                  // DELETE /api/v1/strategies/{id}
			strategies.GET("/:id/versions", strategyHandler.GetVersions)               // GET /api/v1/strategies/{id}/versions
			strategies.PUT("/:id/active-version", strategyHandler.UpdateActiveVersion) // PUT /api/v1/strategies/{id}/active-version
			strategies.POST("/:id/thumbnail", thumbnailHandler.UploadThumbnail)        // POST /api/v1/strategies/{id}/thumbnail
		}

		// ==================== TAG ROUTES ====================
		tags := v1.Group("/strategy-tags")
		{
			// Public route
			tags.GET("", tagHandler.GetAllTags) // GET /api/v1/strategy-tags

			// Protected routes
			tags.Use(middleware.AuthMiddleware(userClient, logger))
			tags.POST("", tagHandler.CreateTag) // POST /api/v1/strategy-tags
		}

		// ==================== MARKETPLACE ROUTES ====================
		marketplace := v1.Group("/marketplace")
		{
			// Public routes
			marketplace.GET("", marketplaceHandler.ListListings)           // GET /api/v1/marketplace
			marketplace.GET("/:id/reviews", marketplaceHandler.GetReviews) // GET /api/v1/marketplace/{id}/reviews

			// Protected marketplace endpoints
			marketplaceAuth := marketplace.Group("")
			marketplaceAuth.Use(middleware.AuthMiddleware(userClient, logger))

			marketplaceAuth.POST("", marketplaceHandler.CreateListing)                 // POST /api/v1/marketplace
			marketplaceAuth.DELETE("/:id", marketplaceHandler.DeleteListing)           // DELETE /api/v1/marketplace/{id}
			marketplaceAuth.POST("/:id/purchase", marketplaceHandler.PurchaseStrategy) // POST /api/v1/marketplace/{id}/purchase

			// Reviews management
			marketplaceAuth.POST("/:id/reviews", marketplaceHandler.CreateReview) // POST /api/v1/marketplace/{id}/reviews

			// Purchases management
			marketplaceAuth.PUT("/purchases/:id/cancel", marketplaceHandler.CancelSubscription) // PUT /api/v1/marketplace/purchases/{id}/cancel
		}

		// Reviews management (separate from marketplace for edit/delete)
		reviews := v1.Group("/reviews")
		{
			reviews.Use(middleware.AuthMiddleware(userClient, logger))
			reviews.PUT("/:id", marketplaceHandler.UpdateReview)    // PUT /api/v1/reviews/{id}
			reviews.DELETE("/:id", marketplaceHandler.DeleteReview) // DELETE /api/v1/reviews/{id}
		}
	}

	return router
}
