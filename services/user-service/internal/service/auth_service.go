package service

import (
	"context"
	"errors"
	"time"

	"services/user-service/internal/config"
	"services/user-service/internal/model"
	"services/user-service/internal/repository"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication and token generation
type AuthService struct {
	userRepo *repository.UserRepository
	authRepo *repository.AuthRepository
	cfg      *config.Config
	logger   *zap.Logger
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo *repository.UserRepository,
	authRepo *repository.AuthRepository,
	cfg *config.Config,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		authRepo: authRepo,
		cfg:      cfg,
		logger:   logger,
	}
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, userCreate *model.UserCreate) (*model.TokenResponse, error) {
	// Check if email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, userCreate.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("email already in use")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userCreate.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return nil, err
	}

	// Set default role if not provided
	if userCreate.Role == "" {
		userCreate.Role = "user"
	}

	// Create the user using the database function
	userID, err := s.userRepo.Create(ctx, userCreate, string(hashedPassword))
	if err != nil {
		return nil, err
	}

	// Get the created user to access the role
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Generate tokens with role information
	accessToken, refreshToken, expiresAt, err := s.generateTokens(userID, user.Role)
	if err != nil {
		return nil, err
	}

	// Update last login time
	if err := s.authRepo.UpdateLastLogin(ctx, userID); err != nil {
		s.logger.Warn("failed to update last login", zap.Error(err), zap.Int("userID", userID))
	}

	// Store refresh token in the database
	ip := ""
	ua := ""
	_, err = s.authRepo.CreateUserSession(ctx, userID, refreshToken, expiresAt, ip, ua)
	if err != nil {
		s.logger.Warn("failed to create user session", zap.Error(err))
	}

	return &model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         *user,
	}, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, login *model.UserLogin) (*model.TokenResponse, error) {
	// Find user by email
	user, err := s.userRepo.GetByEmail(ctx, login.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid email or password")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("account is disabled")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(login.Password)); err != nil {
		s.logger.Debug("password verification failed", zap.Error(err))
		return nil, errors.New("invalid email or password")
	}

	// Generate tokens with user role
	accessToken, refreshToken, expiresAt, err := s.generateTokens(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	// Store session info
	ip := ""
	ua := ""
	_, err = s.authRepo.CreateUserSession(ctx, user.ID, refreshToken, expiresAt, ip, ua)
	if err != nil {
		s.logger.Warn("failed to create user session", zap.Error(err))
	}

	// Update last login time
	if err := s.authRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("failed to update last login", zap.Error(err), zap.Int("userID", user.ID))
	}

	return &model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         *user,
	}, nil
}

// RefreshToken refreshes the access token using a valid refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*model.TokenResponse, error) {
	// Parse and validate the refresh token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.Auth.JWTSecret), nil
	})

	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	// Verify token type
	if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh" {
		return nil, errors.New("invalid token type")
	}

	// Extract user ID
	userIDFloat, ok := claims["sub"].(float64)
	if !ok {
		return nil, errors.New("invalid user ID in token")
	}
	userID := int(userIDFloat)

	// Check if session exists
	session, err := s.authRepo.GetUserSession(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, errors.New("session not found or expired")
	}

	// Get user with role information
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, errors.New("user not found or inactive")
	}

	// Generate new tokens with role
	accessToken, newRefreshToken, expiresAt, err := s.generateTokens(userID, user.Role)
	if err != nil {
		return nil, err
	}

	// Delete old session and create new one
	_, err = s.authRepo.DeleteUserSession(ctx, refreshToken)
	if err != nil {
		s.logger.Warn("failed to delete old session", zap.Error(err))
	}

	// Store new session
	ip := ""
	ua := ""
	_, err = s.authRepo.CreateUserSession(ctx, user.ID, newRefreshToken, expiresAt, ip, ua)
	if err != nil {
		s.logger.Warn("failed to create new session", zap.Error(err))
	}

	return &model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
		User:         *user,
	}, nil
}

// Logout invalidates a user's session
func (s *AuthService) Logout(ctx context.Context, token string) error {
	success, err := s.authRepo.DeleteUserSession(ctx, token)
	if err != nil {
		return err
	}
	if !success {
		return errors.New("session not found")
	}
	return nil
}

// LogoutAll invalidates all of a user's sessions
func (s *AuthService) LogoutAll(ctx context.Context, userID int) (int, error) {
	return s.authRepo.DeleteUserSessions(ctx, userID)
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, id int, change *model.UserChangePassword) error {
	// Get password hash to verify current password
	passwordHash, err := s.authRepo.GetUserPasswordByID(ctx, id)
	if err != nil {
		return err
	}
	if passwordHash == "" {
		return errors.New("user not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(change.CurrentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(change.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return errors.New("failed to process new password")
	}

	// Update password in database
	success, err := s.authRepo.UpdateUserPassword(ctx, id, string(newHash))
	if err != nil {
		return err
	}

	if !success {
		return errors.New("failed to update password")
	}

	// Invalidate all sessions to force re-login with new password
	_, err = s.authRepo.DeleteUserSessions(ctx, id)
	if err != nil {
		s.logger.Warn("failed to delete user sessions after password change", zap.Error(err))
	}

	return nil
}

// generateTokens creates a new pair of access and refresh tokens with role information
func (s *AuthService) generateTokens(userID int, role string) (accessToken, refreshToken string, expiresAt time.Time, err error) {
	// Access token expiry
	accessExpiry := time.Now().Add(s.cfg.Auth.AccessTokenDuration)

	// Create access token with role information
	accessClaims := jwt.MapClaims{
		"sub":  userID,
		"exp":  accessExpiry.Unix(),
		"iat":  time.Now().Unix(),
		"type": "access",
		"role": role, // Include role in the token
	}

	access := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = access.SignedString([]byte(s.cfg.Auth.JWTSecret))
	if err != nil {
		s.logger.Error("failed to sign access token", zap.Error(err))
		return "", "", time.Time{}, err
	}

	// Create refresh token with longer expiry
	refreshExpiry := time.Now().Add(s.cfg.Auth.RefreshTokenDuration)
	refreshClaims := jwt.MapClaims{
		"sub":  userID,
		"exp":  refreshExpiry.Unix(),
		"iat":  time.Now().Unix(),
		"type": "refresh",
	}

	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refresh.SignedString([]byte(s.cfg.Auth.JWTSecret))
	if err != nil {
		s.logger.Error("failed to sign refresh token", zap.Error(err))
		return "", "", time.Time{}, err
	}

	return accessToken, refreshToken, accessExpiry, nil
}

// ValidateToken validates a JWT token and returns the user ID and role if valid
func (s *AuthService) ValidateToken(tokenString string) (int, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.Auth.JWTSecret), nil
	})

	if err != nil {
		return 0, "", err
	}

	if !token.Valid {
		return 0, "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", errors.New("invalid claims")
	}

	// Check token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "access" {
		return 0, "", errors.New("invalid token type")
	}

	// Extract user ID
	userIDFloat, ok := claims["sub"].(float64)
	if !ok {
		return 0, "", errors.New("invalid user ID in token")
	}

	// Extract role
	role, ok := claims["role"].(string)
	if !ok {
		// Default to 'user' if role not found (for backward compatibility)
		role = "user"
	}

	return int(userIDFloat), role, nil
}

// GetJWTSecret returns the JWT secret for service-to-service validation
func (s *AuthService) GetJWTSecret() string {
	return s.cfg.Auth.JWTSecret
}
