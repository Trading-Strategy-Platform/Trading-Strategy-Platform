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

	"services/user-service/internal/client"
	"services/user-service/internal/config"
	"services/user-service/internal/handler"
	"services/user-service/internal/middleware"
	"services/user-service/internal/repository"
	"services/user-service/internal/service"

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

	// Create repositories
	userRepo := repository.NewUserRepository(db, logger)
	notificationRepo := repository.NewNotificationRepository(db, logger)
	preferencesRepo := repository.NewPreferencesRepository(db, logger)

	// Create clients
	mediaClient := client.NewMediaClient(cfg.Media.URL, cfg.Media.ServiceKey, logger)

	// Create services
	authService := service.NewAuthService(userRepo, cfg, logger)
	userService := service.NewUserService(userRepo, notificationRepo, preferencesRepo, logger)

	// Create HTTP server
	router := setupRouter(authService, userService, mediaClient, logger)

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
	authService *service.AuthService,
	userService *service.UserService,
	mediaClient *client.MediaClient,
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
		// Auth routes
		auth := v1.Group("/auth")
		{
			authHandler := handler.NewAuthHandler(authService, logger)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh-token", authHandler.RefreshToken)

			// Protected logout route
			authProtected := auth.Group("")
			authProtected.Use(middleware.AuthMiddleware(authService, logger))
			authProtected.POST("/logout", authHandler.Logout)
		}

		// User routes (protected)
		users := v1.Group("/users")
		{
			// All user routes require authentication
			users.Use(middleware.AuthMiddleware(authService, logger))

			// User profile handlers
			userHandler := handler.NewUserHandler(userService, logger)
			users.GET("/me", userHandler.GetCurrentUser)
			users.PUT("/me", userHandler.UpdateCurrentUser)
			users.PUT("/me/password", userHandler.ChangePassword)
			users.DELETE("/me", userHandler.DeleteCurrentUser)

			// User preferences handlers
			users.GET("/me/preferences", userHandler.GetUserPreferences)
			users.PUT("/me/preferences", userHandler.UpdateUserPreferences)

			// User notifications handlers
			notificationHandler := handler.NewNotificationHandler(userService, logger)
			users.GET("/me/notifications", notificationHandler.GetNotifications)
			users.GET("/me/notifications/count", notificationHandler.GetUnreadCount)
			users.PUT("/me/notifications/:id/read", notificationHandler.MarkNotificationAsRead)
			users.PUT("/me/notifications/read-all", notificationHandler.MarkAllAsRead)

			// Profile photo handler
			profilePhotoHandler := handler.NewProfilePhotoHandler(userService, mediaClient, logger)
			users.POST("/me/profile-photo", profilePhotoHandler.UploadProfilePhoto)
		}

		// Admin routes (protected with role check)
		admin := v1.Group("/admin")
		{
			admin.Use(middleware.AuthMiddleware(authService, logger))
			admin.Use(middleware.RequireRole(userService, "admin"))

			userHandler := handler.NewUserHandler(userService, logger)
			admin.GET("/users", userHandler.ListUsers)
			admin.GET("/users/:id", userHandler.GetUserByID)
			admin.PUT("/users/:id", userHandler.UpdateUser)
		}
	}

	return router
}
