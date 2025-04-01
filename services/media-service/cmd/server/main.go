package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"services/media-service/internal/config"
	"services/media-service/internal/handler"
	"services/media-service/internal/middleware"
	"services/media-service/internal/service"
	"services/media-service/internal/storage"

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

	// Initialize storage
	storageProvider, err := storage.NewStorage(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize storage", zap.Error(err))
	}

	// Initialize service
	mediaService := service.NewMediaService(storageProvider, cfg, logger)

	// Initialize handler
	mediaHandler := handler.NewMediaHandler(mediaService, logger)

	// Set up HTTP server with Gin
	router := setupRouter(cfg, mediaHandler, logger)

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

func setupRouter(cfg *config.Config, mediaHandler *handler.MediaHandler, logger *zap.Logger) *gin.Engine {
	router := gin.New()

	// Use middlewares
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))

	// Auth middleware
	authMiddleware := middleware.AuthMiddleware(cfg, logger)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Media endpoints
		media := v1.Group("/media")
		{
			// Upload endpoint (requires authentication)
			media.POST("/upload", authMiddleware, mediaHandler.Upload)

			// Get endpoint (public)
			media.GET("/:id", middleware.PublicRoute(), mediaHandler.Get)

			// Delete endpoint (requires authentication)
			media.DELETE("/:id", authMiddleware, mediaHandler.Delete)
		}
	}

	return router
}