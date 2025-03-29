package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// PreferencesRepository handles database operations for user preferences
type PreferencesRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewPreferencesRepository creates a new preferences repository
func NewPreferencesRepository(db *sqlx.DB, logger *zap.Logger) *PreferencesRepository {
	return &PreferencesRepository{
		db:     db,
		logger: logger,
	}
}

// GetPreferences retrieves user preferences
// Note: This is primarily handled through the user_details view and get_user_details function
// But we keep this separate for organization and potential future extensions
func (r *PreferencesRepository) GetPreferences(ctx context.Context, userID int) (map[string]interface{}, error) {
	query := `
		SELECT theme, default_timeframe, chart_preferences, notification_settings
		FROM user_preferences
		WHERE user_id = $1
	`

	var preferences map[string]interface{}
	if err := r.db.GetContext(ctx, &preferences, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("failed to get user preferences", zap.Error(err), zap.Int("user_id", userID))
		return nil, err
	}

	return preferences, nil
}
