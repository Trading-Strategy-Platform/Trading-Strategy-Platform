// internal/repository/user_repository.go
package repository

import (
	"context"
	"database/sql"
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

// GetByID retrieves a user by ID using get_user_by_id function
func (r *UserRepository) GetByID(ctx context.Context, id int) (*model.User, error) {
	query := `SELECT * FROM get_user_by_id($1)`

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

// GetUserDetails retrieves detailed user information using get_user_details function
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

// GetByEmail retrieves a user by email using get_user_by_email function
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT * FROM get_user_by_email($1)`

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
	isActive *bool,
) (bool, error) {
	query := `SELECT update_user($1, $2, $3, $4, $5)`

	var success bool
	err := r.db.GetContext(
		ctx,
		&success,
		query,
		userID,
		username,
		email,
		profilePhotoURL,
		isActive,
	)

	if err != nil {
		r.logger.Error("failed to update user", zap.Error(err), zap.Int("id", userID))
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

// List retrieves a paginated list of users using list_users function
func (r *UserRepository) List(ctx context.Context, offset, limit int) ([]model.User, error) {
	query := `SELECT * FROM list_users($1, $2)`

	var users []model.User
	if err := r.db.SelectContext(ctx, &users, query, limit, offset); err != nil {
		r.logger.Error("failed to list users", zap.Error(err))
		return nil, err
	}

	return users, nil
}

// Count returns the total number of users using get_user_count function
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT get_user_count()`

	var count int
	if err := r.db.GetContext(ctx, &count, query); err != nil {
		r.logger.Error("failed to count users", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// GetRole returns the role of a user using get_user_role function
func (r *UserRepository) GetRole(ctx context.Context, id int) (string, error) {
	query := `SELECT get_user_role($1)`

	var role string
	if err := r.db.GetContext(ctx, &role, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		r.logger.Error("failed to get user role", zap.Error(err), zap.Int("id", id))
		return "", err
	}

	return role, nil
}
