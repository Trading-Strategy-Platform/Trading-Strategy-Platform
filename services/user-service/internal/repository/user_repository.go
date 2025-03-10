package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"services/user-service/internal/model"

	"github.com/jmoiron/sqlx"
	sharedErrors "github.com/yourorg/trading-platform/shared/go/errors"
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

// Create adds a new user to the database
func (r *UserRepository) Create(ctx context.Context, user *model.UserCreate, passwordHash string) (int, error) {
	query := `
		INSERT INTO users (username, email, password_hash, role, is_active, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Email,
		passwordHash,
		user.Role,
		true,
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("failed to create user", zap.Error(err))
		return 0, sharedErrors.NewDatabaseError("creating user", err)
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
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found, but not an error
		}
		r.logger.Error("failed to get user by ID", zap.Error(err), zap.Int("id", id))
		return nil, sharedErrors.NewDatabaseError("getting user by ID", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, is_active, last_login, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user model.User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found, but not an error
		}
		r.logger.Error("failed to get user by email", zap.Error(err), zap.String("email", email))
		return nil, sharedErrors.NewDatabaseError("getting user by email", err)
	}

	return &user, nil
}

// Update updates a user's details
func (r *UserRepository) Update(ctx context.Context, id int, update *model.UserUpdate) error {
	// Build dynamic SQL for updates
	query := "UPDATE users SET updated_at = NOW()"
	params := []interface{}{}
	paramCount := 1

	// Add fields only if they're provided
	if update.Username != nil {
		query += fmt.Sprintf(", username = $%d", paramCount)
		params = append(params, *update.Username)
		paramCount++
	}

	if update.Email != nil {
		query += fmt.Sprintf(", email = $%d", paramCount)
		params = append(params, *update.Email)
		paramCount++
	}

	if update.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", paramCount)
		params = append(params, *update.IsActive)
		paramCount++
	}

	if update.Role != nil {
		query += fmt.Sprintf(", role = $%d", paramCount)
		params = append(params, *update.Role)
		paramCount++
	}

	// Add WHERE clause
	query += fmt.Sprintf(" WHERE id = $%d", paramCount)
	params = append(params, id)

	// Execute update
	_, err := r.db.ExecContext(ctx, query, params...)
	if err != nil {
		r.logger.Error("failed to update user", zap.Error(err), zap.Int("id", id))
		return sharedErrors.NewDatabaseError("updating user", err)
	}

	return nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, id int, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, passwordHash, time.Now(), id)
	if err != nil {
		r.logger.Error("failed to update password", zap.Error(err), zap.Int("id", id))
		return sharedErrors.NewDatabaseError("updating password", err)
	}

	return nil
}

// UpdateLastLogin updates a user's last login time
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id int) error {
	query := `
		UPDATE users
		SET last_login = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		r.logger.Error("failed to update last login", zap.Error(err), zap.Int("id", id))
		return sharedErrors.NewDatabaseError("updating last login", err)
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
		return nil, sharedErrors.NewDatabaseError("listing users", err)
	}

	return users, nil
}

// Count returns the total number of users
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users`

	var count int
	if err := r.db.GetContext(ctx, &count, query); err != nil {
		r.logger.Error("failed to count users", zap.Error(err))
		return 0, sharedErrors.NewDatabaseError("counting users", err)
	}

	return count, nil
}
