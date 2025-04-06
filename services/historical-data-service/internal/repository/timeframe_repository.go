package repository

import (
	"context"
	"time"

	"services/historical-data-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// TimeframeRepository handles database operations for timeframes
type TimeframeRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewTimeframeRepository creates a new timeframe repository
func NewTimeframeRepository(db *sqlx.DB, logger *zap.Logger) *TimeframeRepository {
	return &TimeframeRepository{
		db:     db,
		logger: logger,
	}
}

// GetAllTimeframes retrieves all available timeframes
func (r *TimeframeRepository) GetAllTimeframes(ctx context.Context) ([]model.Timeframe, error) {
	// Direct query to get timeframes from the timeframe_type enum
	query := `
		SELECT 
			t.typname AS name,
			e.enumlabel AS display_name,
			CASE
				WHEN e.enumlabel = '1m' THEN 1
				WHEN e.enumlabel = '5m' THEN 5
				WHEN e.enumlabel = '15m' THEN 15
				WHEN e.enumlabel = '30m' THEN 30
				WHEN e.enumlabel = '1h' THEN 60
				WHEN e.enumlabel = '4h' THEN 240
				WHEN e.enumlabel = '1d' THEN 1440
				WHEN e.enumlabel = '1w' THEN 10080
			END AS minutes
		FROM pg_type t
		JOIN pg_enum e ON t.oid = e.enumtypid
		WHERE t.typname = 'timeframe_type'
		ORDER BY minutes
	`

	var timeframes []struct {
		Name        string `db:"name"`
		DisplayName string `db:"display_name"`
		Minutes     int    `db:"minutes"`
	}

	err := r.db.SelectContext(ctx, &timeframes, query)
	if err != nil {
		r.logger.Error("Failed to get timeframes", zap.Error(err))
		return nil, err
	}

	// Convert to the model format
	result := make([]model.Timeframe, len(timeframes))
	for i, tf := range timeframes {
		result[i] = model.Timeframe{
			ID:          i + 1, // Assign an auto-incrementing ID
			Name:        tf.Name,
			DisplayName: tf.DisplayName,
			Minutes:     tf.Minutes,
			CreatedAt:   time.Now(),
		}
	}

	return result, nil
}

// ValidateTimeframe checks if a timeframe is valid
func (r *TimeframeRepository) ValidateTimeframe(ctx context.Context, timeframe string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM pg_enum e
			JOIN pg_type t ON e.enumtypid = t.oid
			WHERE t.typname = 'timeframe_type' AND e.enumlabel = $1
		)
	`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, timeframe)
	if err != nil {
		r.logger.Error("Failed to validate timeframe", zap.Error(err), zap.String("timeframe", timeframe))
		return false, err
	}

	return exists, nil
}
