// internal/repository/user_repository.go
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"services/user-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// UserRepository handles database operations for users
type UserRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sqlx.DB, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

// Create adds a new user to the database using create_user function
func (r *UserRepository) Create(ctx context.Context, user *model.UserCreate, passwordHash string) (int, error) {
	query := `SELECT create_user($1, $2, $3, $4, $5)`

	var id int
	err := r.db.GetContext(
		ctx,
		&id,
		query,
		user.Username,
		user.Email,
		passwordHash,
		user.Role,
		user.ProfilePhotoURL,
	)

	if err != nil {
		r.logger.Error("failed to create user", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id int) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, is_active, last_login, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user model.User
	if err := r.db.GetContext(ctx, &user, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("failed to get user by ID", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &user, nil
}

// GetUserDetails retrieves detailed user information including preferences using get_user_details
func (r *UserRepository) GetUserDetails(ctx context.Context, id int) (*model.UserDetails, error) {
	query := `SELECT * FROM get_user_details($1)`

	var details model.UserDetails
	if err := r.db.GetContext(ctx, &details, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("failed to get user details", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &details, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, is_active, last_login, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user model.User
	if err := r.db.GetContext(ctx, &user, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("failed to get user by email", zap.Error(err), zap.String("email", email))
		return nil, err
	}

	return &user, nil
}

// UpdateUser updates a user using the update_user function
func (r *UserRepository) UpdateUser(
	ctx context.Context,
	userID int,
	username *string,
	email *string,
	profilePhotoURL *string,
	theme *string,
	defaultTimeframe *string,
	chartPreferences json.RawMessage,
	notificationSettings json.RawMessage,
) (bool, error) {
	query := `SELECT update_user($1, $2, $3, $4, $5, $6, $7, $8)`

	var success bool
	err := r.db.GetContext(
		ctx,
		&success,
		query,
		userID,
		username,
		email,
		profilePhotoURL,
		theme,
		defaultTimeframe,
		chartPreferences,
		notificationSettings,
	)

	if err != nil {
		r.logger.Error("failed to update user", zap.Error(err), zap.Int("id", userID))
		return false, err
	}

	return success, nil
}

// UpdateUserPassword updates a user's password using update_user_password function
func (r *UserRepository) UpdateUserPassword(ctx context.Context, id int, passwordHash string) (bool, error) {
	query := `SELECT update_user_password($1, $2)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, id, passwordHash)
	if err != nil {
		r.logger.Error("failed to update password", zap.Error(err), zap.Int("id", id))
		return false, err
	}

	return success, nil
}

// DeleteUser marks a user as inactive using delete_user function
func (r *UserRepository) DeleteUser(ctx context.Context, id int) (bool, error) {
	query := `SELECT delete_user($1)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, id)
	if err != nil {
		r.logger.Error("failed to delete user", zap.Error(err), zap.Int("id", id))
		return false, err
	}

	return success, nil
}

// UpdateLastLogin updates a user's last login timestamp using update_last_login function
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id int) error {
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

// List retrieves a paginated list of users
func (r *UserRepository) List(ctx context.Context, offset, limit int) ([]model.User, error) {
	query := `
		SELECT id, username, email, role, is_active, last_login, created_at, updated_at
		FROM users
		ORDER BY id
		LIMIT $1 OFFSET $2
	`

	var users []model.User
	if err := r.db.SelectContext(ctx, &users, query, limit, offset); err != nil {
		r.logger.Error("failed to list users", zap.Error(err))
		return nil, err
	}

	return users, nil
}

// Count returns the total number of users
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users`

	var count int
	if err := r.db.GetContext(ctx, &count, query); err != nil {
		r.logger.Error("failed to count users", zap.Error(err))
		return 0, err
	}

	return count, nil
}
