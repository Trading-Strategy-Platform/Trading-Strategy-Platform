package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/trading-platform/services/user-service/internal/model"
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

	row := r.db.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Email,
		passwordHash,
		user.Role,
		true,
		time.Now(),
	)

	var id int
	if err := row.Scan(&id); err != nil {
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

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, id int, user *model.UserUpdate) error {
	query := `
		UPDATE users
		SET
	`

	params := []interface{}{}
	paramCount := 1
	setValues := []string{}

	if user.Username != nil {
		setValues = append(setValues, "username = $"+string(paramCount))
		params = append(params, *user.Username)
		paramCount++
	}

	if user.Email != nil {
		setValues = append(setValues, "email = $"+string(paramCount))
		params = append(params, *user.Email)
		paramCount++
	}

	if user.IsActive != nil {
		setValues = append(setValues, "is_active = $"+string(paramCount))
		params = append(params, *user.IsActive)
		paramCount++
	}

	// Always update the updated_at timestamp
	setValues = append(setValues, "updated_at = $"+string(paramCount))
	params = append(params, time.Now())
	paramCount++

	// Add the WHERE clause
	query += ` SET ` + setValues[0]
	for i := 1; i < len(setValues); i++ {
		query += `, ` + setValues[i]
	}

	query += ` WHERE id = $` + string(paramCount)
	params = append(params, id)

	_, err := r.db.ExecContext(ctx, query, params...)
	if err != nil {
		r.logger.Error("failed to update user", zap.Error(err), zap.Int("id", id))
		return err
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
		return err
	}

	return nil
}

// UpdateLastLogin updates a user's last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id int) error {
	query := `
		UPDATE users
		SET last_login = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		r.logger.Error("failed to update last login", zap.Error(err), zap.Int("id", id))
		return err
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
