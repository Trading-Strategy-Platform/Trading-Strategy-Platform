// internal/repository/preference_repository.go
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

// PreferenceRepository handles database operations for user preferences
type PreferenceRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewPreferenceRepository creates a new preference repository
func NewPreferenceRepository(db *sqlx.DB, logger *zap.Logger) *PreferenceRepository {
	return &PreferenceRepository{
		db:     db,
		logger: logger,
	}
}

// GetPreferences retrieves user preferences using get_user_preferences function
func (r *PreferenceRepository) GetPreferences(ctx context.Context, userID int) (*model.UserPreferences, error) {
	query := `SELECT * FROM get_user_preferences($1)`

	var prefs model.UserPreferences
	if err := r.db.GetContext(ctx, &prefs, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("Failed to get user preferences", zap.Error(err), zap.Int("user_id", userID))
		return nil, err
	}

	return &prefs, nil
}

// UpdatePreferences updates user preferences using update_user_preferences function
func (r *PreferenceRepository) UpdatePreferences(
	ctx context.Context,
	userID int,
	theme *string,
	defaultTimeframe *string,
	chartPreferences json.RawMessage,
	notificationSettings json.RawMessage,
) (bool, error) {
	query := `SELECT update_user_preferences($1, $2, $3, $4, $5)`

	var success bool
	err := r.db.GetContext(
		ctx,
		&success,
		query,
		userID,
		theme,
		defaultTimeframe,
		chartPreferences,
		notificationSettings,
	)

	if err != nil {
		r.logger.Error("Failed to update user preferences", zap.Error(err), zap.Int("user_id", userID))
		return false, err
	}

	return success, nil
}

// CheckPreferencesExist checks if user preferences exist using check_user_preferences_exist function
func (r *PreferenceRepository) CheckPreferencesExist(ctx context.Context, userID int) (bool, error) {
	query := `SELECT check_user_preferences_exist($1)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, userID)
	if err != nil {
		r.logger.Error("Failed to check if preferences exist", zap.Error(err), zap.Int("user_id", userID))
		return false, err
	}

	return exists, nil
}

// DeletePreferences deletes user preferences using delete_user_preferences function
func (r *PreferenceRepository) DeletePreferences(ctx context.Context, userID int) (bool, error) {
	query := `SELECT delete_user_preferences($1)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, userID)
	if err != nil {
		r.logger.Error("Failed to delete user preferences", zap.Error(err), zap.Int("user_id", userID))
		return false, err
	}

	return success, nil
}
