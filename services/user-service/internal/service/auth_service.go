package service

import (
	"context"
	"time"

	"services/user-service/internal/config"
	"services/user-service/internal/model"
	"services/user-service/internal/repository"

	"github.com/golang-jwt/jwt/v4"
	sharedErrors "github.com/yourorg/trading-platform/shared/go/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication and token generation
type AuthService struct {
	userRepo     *repository.UserRepository
	cfg          *config.Config
	logger       *zap.Logger
	jwtSecret    string
	kafkaService *KafkaService
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config, kafkaService *KafkaService, logger *zap.Logger) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		cfg:          cfg,
		logger:       logger,
		jwtSecret:    cfg.Auth.JWTSecret,
		kafkaService: kafkaService,
	}
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, userCreate *model.UserCreate) (*model.TokenResponse, error) {
	// Check if email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, userCreate.Email)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("checking email existence", err)
	}
	if existingUser != nil {
		return nil, sharedErrors.NewDuplicateError("User", "email")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userCreate.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return nil, sharedErrors.NewInternalError("Failed to secure password", err)
	}

	// Set default role if not provided
	if userCreate.Role == "" {
		userCreate.Role = "user"
	}

	// Create the user
	userID, err := s.userRepo.Create(ctx, userCreate, string(hashedPassword))
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("creating user", err)
	}

	// Generate tokens
	accessToken, refreshToken, expiry, err := s.generateTokens(userID, userCreate.Email, userCreate.Role)
	if err != nil {
		return nil, sharedErrors.NewInternalError("Failed to generate authentication tokens", err)
	}

	// Retrieve the created user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("retrieving created user", err)
	}

	// Publish user created event
	if s.kafkaService != nil {
		if err := s.kafkaService.PublishUserCreated(ctx, user); err != nil {
			// Log but don't fail if event publishing fails
			s.logger.Warn("Failed to publish user created event", zap.Error(err), zap.Int("user_id", user.ID))
		}
	}

	// Update last login time
	if err := s.userRepo.UpdateLastLogin(ctx, userID); err != nil {
		s.logger.Warn("failed to update last login", zap.Error(err), zap.Int("userID", userID))
	}

	return &model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiry,
		User:         *user,
	}, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, login *model.UserLogin) (*model.TokenResponse, error) {
	// Find user by email
	user, err := s.userRepo.GetByEmail(ctx, login.Email)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("retrieving user by email", err)
	}
	if user == nil {
		return nil, sharedErrors.NewAuthError("Invalid email or password")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, sharedErrors.NewAuthError("Account is inactive")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(login.Password)); err != nil {
		return nil, sharedErrors.NewAuthError("Invalid email or password")
	}

	// Generate tokens
	accessToken, refreshToken, expiry, err := s.generateTokens(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, sharedErrors.NewInternalError("Failed to generate authentication tokens", err)
	}

	// Publish login event
	if s.kafkaService != nil {
		// Extract IP and user agent from context if available
		ipAddress := ""
		userAgent := ""
		if ctx.Value("client_ip") != nil {
			ipAddress = ctx.Value("client_ip").(string)
		}
		if ctx.Value("user_agent") != nil {
			userAgent = ctx.Value("user_agent").(string)
		}

		// Publish login event
		if err := s.kafkaService.PublishLoginEvent(ctx, user.ID, true, ipAddress, userAgent); err != nil {
			s.logger.Warn("Failed to publish login event", zap.Error(err), zap.Int("user_id", user.ID))
		}
	}

	// Update last login time
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("Failed to update last login time", zap.Error(err), zap.Int("user_id", user.ID))
	}

	return &model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiry,
		User:         *user,
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (*model.TokenResponse, error) {
	// Parse and validate refresh token
	token, err := jwt.Parse(refreshTokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, sharedErrors.NewAuthError("Unexpected token signing method")
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, sharedErrors.NewAuthError("Invalid refresh token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, sharedErrors.NewAuthError("Invalid token claims")
	}

	// Check token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return nil, sharedErrors.NewAuthError("Invalid token type")
	}

	// Extract user ID and email
	userIDFloat, ok := claims["sub"].(float64)
	if !ok {
		return nil, sharedErrors.NewAuthError("Invalid user ID in token")
	}
	userID := int(userIDFloat)

	email, ok := claims["email"].(string)
	if !ok {
		return nil, sharedErrors.NewAuthError("Invalid email in token")
	}

	role, ok := claims["role"].(string)
	if !ok {
		return nil, sharedErrors.NewAuthError("Invalid role in token")
	}

	// Get user to verify they still exist and are active
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("retrieving user", err)
	}
	if user == nil {
		return nil, sharedErrors.NewAuthError("User no longer exists")
	}
	if !user.IsActive {
		return nil, sharedErrors.NewAuthError("Account is inactive")
	}

	// Generate new tokens
	accessToken, refreshToken, expiry, err := s.generateTokens(userID, email, role)
	if err != nil {
		return nil, sharedErrors.NewInternalError("Failed to generate tokens", err)
	}

	return &model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiry,
		User:         *user,
	}, nil
}

// generateTokens creates access and refresh tokens for a user
func (s *AuthService) generateTokens(userID int, email, role string) (string, string, time.Time, error) {
	// Create access token
	accessExpiry := time.Now().Add(time.Duration(s.cfg.Auth.AccessTokenTTL) * time.Minute)
	accessClaims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"type":  "access",
		"exp":   accessExpiry.Unix(),
		"iat":   time.Now().Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		s.logger.Error("failed to sign access token", zap.Error(err))
		return "", "", time.Time{}, sharedErrors.NewInternalError("Failed to generate access token", err)
	}

	// Create refresh token with longer expiry
	refreshExpiry := time.Now().Add(time.Duration(s.cfg.Auth.RefreshTokenTTL) * time.Hour)
	refreshClaims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"type":  "refresh",
		"exp":   refreshExpiry.Unix(),
		"iat":   time.Now().Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		s.logger.Error("failed to sign refresh token", zap.Error(err))
		return "", "", time.Time{}, sharedErrors.NewInternalError("Failed to generate refresh token", err)
	}

	return accessTokenString, refreshTokenString, accessExpiry, nil
}

// ValidateToken validates a JWT token and returns the user ID if valid
func (s *AuthService) ValidateToken(tokenString string) (int, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, sharedErrors.NewAuthError("Unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return 0, sharedErrors.NewAuthError("Invalid token: " + err.Error())
	}

	if !token.Valid {
		return 0, sharedErrors.NewAuthError("Invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, sharedErrors.NewAuthError("Invalid claims")
	}

	// Check token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "access" {
		return 0, sharedErrors.NewAuthError("Invalid token type")
	}

	// Extract user ID
	userIDFloat, ok := claims["sub"].(float64)
	if !ok {
		return 0, sharedErrors.NewAuthError("Invalid user ID in token")
	}

	return int(userIDFloat), nil
}

// GetJWTSecret returns the JWT secret
func (s *AuthService) GetJWTSecret() string {
	return s.jwtSecret
}
