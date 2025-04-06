// internal/repository/profile_repository.go
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// ProfileRepository handles database operations for user profiles
type ProfileRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewProfileRepository creates a new profile repository
func NewProfileRepository(db *sqlx.DB, logger *zap.Logger) *ProfileRepository {
	return &ProfileRepository{
		db:     db,
		logger: logger,
	}
}

// GetProfilePhotoURL gets a user's profile photo URL using get_profile_photo_url function
func (r *ProfileRepository) GetProfilePhotoURL(ctx context.Context, userID int) (string, error) {
	query := `SELECT get_profile_photo_url($1)`

	var photoURL string
	err := r.db.GetContext(ctx, &photoURL, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		r.logger.Error("Failed to get profile photo URL", zap.Error(err), zap.Int("user_id", userID))
		return "", err
	}

	return photoURL, nil
}

// UpdateProfilePhoto updates a user's profile photo URL using update_profile_photo function
func (r *ProfileRepository) UpdateProfilePhoto(ctx context.Context, userID int, photoURL string) (bool, error) {
	query := `SELECT update_profile_photo($1, $2)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, userID, photoURL)
	if err != nil {
		r.logger.Error("Failed to update profile photo", zap.Error(err), zap.Int("user_id", userID))
		return false, err
	}

	return success, nil
}

// ClearProfilePhoto clears a user's profile photo URL using clear_profile_photo function
func (r *ProfileRepository) ClearProfilePhoto(ctx context.Context, userID int) (bool, error) {
	query := `SELECT clear_profile_photo($1)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, userID)
	if err != nil {
		r.logger.Error("Failed to clear profile photo", zap.Error(err), zap.Int("user_id", userID))
		return false, err
	}

	return success, nil
}
