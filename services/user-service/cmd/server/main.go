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
	authRepo := repository.NewAuthRepository(db, logger)
	notificationRepo := repository.NewNotificationRepository(db, logger)
	preferenceRepo := repository.NewPreferenceRepository(db, logger)
	profileRepo := repository.NewProfileRepository(db, logger)

	// Create clients
	mediaClient := client.NewMediaClient(cfg.Media.URL, cfg.Media.ServiceKey, logger)

	// Create services
	authService := service.NewAuthService(userRepo, authRepo, cfg, logger)
	userService := service.NewUserService(userRepo, logger)
	notificationService := service.NewNotificationService(notificationRepo, userRepo, logger)
	preferenceService := service.NewPreferenceService(preferenceRepo, userRepo, logger)
	profileService := service.NewProfileService(profileRepo, userRepo, mediaClient, logger)

	// Create HTTP server
	router := setupRouter(
		authService,
		userService,
		notificationService,
		preferenceService,
		profileService,
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
	authService *service.AuthService,
	userService *service.UserService,
	notificationService *service.NotificationService,
	preferenceService *service.PreferenceService,
	profileService *service.ProfileService,
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

			// Protected auth routes
			authProtected := auth.Group("")
			authProtected.Use(middleware.AuthMiddleware(authService, logger))
			authProtected.POST("/logout", authHandler.Logout)
			authProtected.POST("/logout-all", authHandler.LogoutAll)
			authProtected.GET("/validate", authHandler.Validate)
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
			users.DELETE("/me", userHandler.DeleteCurrentUser)

			// Password management
			passwordHandler := handler.NewPasswordHandler(authService, logger)
			users.PUT("/me/password", passwordHandler.ChangePassword)

			// User preferences handlers
			prefHandler := handler.NewPreferenceHandler(preferenceService, logger)
			users.GET("/me/preferences", prefHandler.GetUserPreferences)
			users.PUT("/me/preferences", prefHandler.UpdateUserPreferences)
			users.POST("/me/preferences/reset", prefHandler.ResetUserPreferences)

			// User notifications handlers
			notifHandler := handler.NewNotificationHandler(notificationService, logger)
			users.GET("/me/notifications", notifHandler.GetNotifications)
			users.GET("/me/notifications/count", notifHandler.GetUnreadCount)
			users.PUT("/me/notifications/:id/read", notifHandler.MarkNotificationAsRead)
			users.PUT("/me/notifications/read-all", notifHandler.MarkAllAsRead)

			// Profile photo handlers
			profileHandler := handler.NewProfileHandler(profileService, logger)
			users.GET("/me/profile-photo", profileHandler.GetProfilePhoto)
			users.POST("/me/profile-photo", profileHandler.UploadProfilePhoto)
			users.DELETE("/me/profile-photo", profileHandler.DeleteProfilePhoto)
		}

		// Admin routes (protected with role check)
		admin := v1.Group("/admin")
		{
			admin.Use(middleware.AuthMiddleware(authService, logger))
			admin.Use(middleware.RequireRole(userService, "admin"))

			// User management (admin)
			userHandler := handler.NewUserHandler(userService, logger)
			admin.GET("/users", userHandler.ListUsers)
			admin.GET("/users/:id", userHandler.GetUserByID)
			admin.PUT("/users/:id", userHandler.UpdateUser)
			admin.GET("/users/:id/roles", userHandler.GetUserRoles)

			// Notification management (admin)
			notifHandler := handler.NewNotificationHandler(notificationService, logger)
			admin.POST("/notifications", notifHandler.CreateNotification)
		}

		// Service-to-service API
		service := v1.Group("/service")
		{
			// This is for direct service-to-service communication
			// No auth middleware, but will verify service keys in the handlers

			// Service user endpoints
			serviceUsers := service.Group("/users")
			{
				userHandler := handler.NewUserHandler(userService, logger)
				serviceUsers.GET("/batch", userHandler.BatchGetServiceUsers)
			}

			// Add other service endpoints as needed
		}
	}

	return router
}
