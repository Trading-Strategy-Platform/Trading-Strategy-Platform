// internal/repository/auth_repository.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"services/user-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// AuthRepository handles database operations for authentication
type AuthRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewAuthRepository creates a new authentication repository
func NewAuthRepository(db *sqlx.DB, logger *zap.Logger) *AuthRepository {
	return &AuthRepository{
		db:     db,
		logger: logger,
	}
}

// GetUserPasswordByID gets a user's password hash using get_user_password_by_id function
func (r *AuthRepository) GetUserPasswordByID(ctx context.Context, id int) (string, error) {
	query := `SELECT get_user_password_by_id($1)`

	var passwordHash string
	if err := r.db.GetContext(ctx, &passwordHash, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		r.logger.Error("failed to get user password", zap.Error(err), zap.Int("id", id))
		return "", err
	}

	return passwordHash, nil
}

// UpdateUserPassword updates a user's password using update_user_password function
func (r *AuthRepository) UpdateUserPassword(ctx context.Context, id int, passwordHash string) (bool, error) {
	query := `SELECT update_user_password($1, $2)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, id, passwordHash)
	if err != nil {
		r.logger.Error("failed to update password", zap.Error(err), zap.Int("id", id))
		return false, err
	}

	return success, nil
}

// UpdateLastLogin updates a user's last login timestamp using update_last_login function
func (r *AuthRepository) UpdateLastLogin(ctx context.Context, id int) error {
	query := `SELECT update_last_login($1)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, id)
	if err != nil {
		r.logger.Error("failed to update last login", zap.Error(err), zap.Int("id", id))
		return err
	}

	if !success {
		return errors.New("failed to update last login")
	}

	return nil
}

// CreateUserSession creates a new user session using create_user_session function
func (r *AuthRepository) CreateUserSession(ctx context.Context, userID int, token string, expiresAt time.Time, ipAddress, userAgent string) (int, error) {
	query := `SELECT create_user_session($1, $2, $3, $4, $5)`

	var sessionID int
	err := r.db.GetContext(ctx, &sessionID, query, userID, token, expiresAt, ipAddress, userAgent)
	if err != nil {
		r.logger.Error("failed to create user session", zap.Error(err))
		return 0, err
	}

	return sessionID, nil
}

// GetUserSession gets a user session by token using get_user_session_by_token function
func (r *AuthRepository) GetUserSession(ctx context.Context, token string) (*model.UserSession, error) {
	query := `SELECT * FROM get_user_session_by_token($1)`

	var session model.UserSession
	if err := r.db.GetContext(ctx, &session, query, token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("failed to get user session", zap.Error(err))
		return nil, err
	}

	return &session, nil
}

// DeleteUserSession deletes a user session by token using delete_user_session function
func (r *AuthRepository) DeleteUserSession(ctx context.Context, token string) (bool, error) {
	query := `SELECT delete_user_session($1)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, token)
	if err != nil {
		r.logger.Error("failed to delete user session", zap.Error(err))
		return false, err
	}

	return success, nil
}

// DeleteUserSessions deletes all sessions for a user using delete_user_sessions function
func (r *AuthRepository) DeleteUserSessions(ctx context.Context, userID int) (int, error) {
	query := `SELECT delete_user_sessions($1)`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		r.logger.Error("failed to delete user sessions", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// ValidateServiceKey validates a service key using validate_service_key function
func (r *AuthRepository) ValidateServiceKey(ctx context.Context, serviceName, keyHash string) (bool, error) {
	query := `SELECT validate_service_key($1, $2)`

	var valid bool
	err := r.db.GetContext(ctx, &valid, query, serviceName, keyHash)
	if err != nil {
		r.logger.Error("failed to validate service key", zap.Error(err))
		return false, err
	}

	return valid, nil
}
