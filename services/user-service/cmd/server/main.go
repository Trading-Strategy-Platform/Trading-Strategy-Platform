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
	"github.com/go-redis/redis/v8"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/segmentio/kafka-go"
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

	// Initialize Redis client (if enabled)
	var redisClient *redis.Client
	if cfg.Redis.Enabled {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.URL,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})

		// Test Redis connection
		_, err = redisClient.Ping(context.Background()).Result()
		if err != nil {
			logger.Warn("Failed to connect to Redis, running without cache", zap.Error(err))
			redisClient = nil
		} else {
			logger.Info("Connected to Redis", zap.String("address", cfg.Redis.URL))
		}
	}

	// Initialize Kafka writer (if enabled)
	var kafkaWriter *kafka.Writer
	if cfg.Kafka.Enabled && len(cfg.Kafka.Brokers) > 0 {
		kafkaWriter = &kafka.Writer{
			Addr:     kafka.TCP(cfg.Kafka.Brokers...),
			Topic:    "user-events",
			Balancer: &kafka.LeastBytes{},
		}
		logger.Info("Initialized Kafka writer", zap.Strings("brokers", cfg.Kafka.Brokers))
	}

	// Create repositories
	userRepo := repository.NewUserRepository(db, logger)
	authRepo := repository.NewAuthRepository(db, logger)
	notificationRepo := repository.NewNotificationRepository(db, logger)
	preferenceRepo := repository.NewPreferenceRepository(db, logger)
	profileRepo := repository.NewProfileRepository(db, logger)

	// Create clients
	mediaClient := client.NewMediaClient(cfg.Media.URL, cfg.Media.ServiceKey, logger)

	// Create services with Redis and Kafka integration
	authService := service.NewAuthService(userRepo, authRepo, cfg, logger)
	userService := service.NewUserService(
		userRepo,
		logger,
		redisClient, // Add Redis client
		kafkaWriter, // Add Kafka writer
	)
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

	// Close Kafka writer if initialized
	if kafkaWriter != nil {
		kafkaWriter.Close()
	}

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
		// ==================== AUTH ROUTES ====================
		auth := v1.Group("/auth")
		{
			authHandler := handler.NewAuthHandler(authService, logger)

			// Public auth routes - these are critical
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh-token", authHandler.RefreshToken)

			// Protected auth routes
			authProtected := auth.Group("")
			authProtected.Use(middleware.AuthMiddleware(authService, logger))
			authProtected.POST("/logout", authHandler.Logout)
			authProtected.POST("/logout-all", authHandler.LogoutAll)

			// Only validation endpoint needed - for Nginx auth_request
			// Even this could be eliminated if Nginx used JWT libraries directly
			authProtected.GET("/validate", authHandler.Validate)
		}

		// ==================== USER ROUTES ====================
		users := v1.Group("/users")
		{
			// All require auth middleware - role checking happens inside handlers based on token
			users.Use(middleware.AuthMiddleware(authService, logger))

			// User handlers
			userHandler := handler.NewUserHandler(userService, logger)
			passwordHandler := handler.NewPasswordHandler(authService, logger)
			prefHandler := handler.NewPreferenceHandler(preferenceService, logger)
			notifHandler := handler.NewNotificationHandler(notificationService, logger)
			profileHandler := handler.NewProfileHandler(profileService, logger)

			// User profile routes
			users.GET("/me", userHandler.GetCurrentUser)
			users.PUT("/me", userHandler.UpdateCurrentUser)
			users.DELETE("/me", userHandler.DeleteCurrentUser)

			// Password management
			users.PUT("/me/password", passwordHandler.ChangePassword)

			// User preferences routes
			users.GET("/me/preferences", prefHandler.GetUserPreferences)
			users.PUT("/me/preferences", prefHandler.UpdateUserPreferences)
			users.POST("/me/preferences/reset", prefHandler.ResetUserPreferences)

			// User notifications routes
			users.GET("/me/notifications", notifHandler.GetNotifications)
			users.GET("/me/notifications/count", notifHandler.GetUnreadCount)
			users.PUT("/me/notifications/:id/read", notifHandler.MarkNotificationAsRead)
			users.PUT("/me/notifications/read-all", notifHandler.MarkAllAsRead)

			// Profile photo routes
			users.GET("/me/profile-photo", profileHandler.GetProfilePhoto)
			users.POST("/me/profile-photo", profileHandler.UploadProfilePhoto)
			users.DELETE("/me/profile-photo", profileHandler.DeleteProfilePhoto)
		}

		// ==================== ADMIN ROUTES ====================
		admin := v1.Group("/admin")
		{
			// Admin routes require auth middleware WITH role check
			admin.Use(middleware.AuthMiddleware(authService, logger))
			admin.Use(middleware.RequireRole("admin")) // Check role from token

			userHandler := handler.NewUserHandler(userService, logger)
			notifHandler := handler.NewNotificationHandler(notificationService, logger)

			// User management (admin only)
			admin.GET("/users", userHandler.ListUsers)
			admin.GET("/users/:id", userHandler.GetUserByID)
			admin.PUT("/users/:id", userHandler.UpdateUser)

			// Notification management (admin)
			admin.POST("/notifications", notifHandler.CreateNotification)
		}

		// ==================== SERVICE API ====================
		// Only for data not available in tokens
		service := v1.Group("/service")
		{
			// Protected with service key
			service.Use(middleware.ServiceAuthMiddleware(cfg.ServiceKey, logger))

			serviceHandler := handler.NewServiceHandler(userService, logger)

			// User profile data (only for getting data NOT in the token)
			service.GET("/users/batch", serviceHandler.BatchGetUsers)
			service.GET("/users/:id", serviceHandler.GetUserByID)

		}
	}

	return router
}
