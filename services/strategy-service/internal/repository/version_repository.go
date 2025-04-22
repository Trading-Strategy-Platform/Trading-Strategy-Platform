package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// VersionRepository handles database operations for strategy versions
type VersionRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewVersionRepository creates a new version repository
func NewVersionRepository(db *sqlx.DB, logger *zap.Logger) *VersionRepository {
	return &VersionRepository{
		db:     db,
		logger: logger,
	}
}

// GetUserActiveVersion retrieves the active version for a user
func (r *VersionRepository) GetUserActiveVersion(ctx context.Context, userID, strategyGroupID int) (int, error) {
	query := `
		SELECT active_version_id
		FROM user_strategy_versions
		WHERE user_id = $1 AND strategy_group_id = $2
	`

	var activeVersionID int
	err := r.db.GetContext(ctx, &activeVersionID, query, userID, strategyGroupID)
	if err != nil {
		r.logger.Warn("Failed to get active version",
			zap.Error(err),
			zap.Int("user_id", userID),
			zap.Int("strategy_group_id", strategyGroupID))
		return 0, err
	}

	return activeVersionID, nil
}

// SetUserActiveVersion sets the active version for a user
func (r *VersionRepository) SetUserActiveVersion(ctx context.Context, userID, strategyGroupID, versionID int) error {
	query := `
		INSERT INTO user_strategy_versions (user_id, strategy_group_id, active_version_id, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, strategy_group_id) 
		DO UPDATE SET active_version_id = $3, updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, userID, strategyGroupID, versionID)
	if err != nil {
		r.logger.Error("Failed to set active version",
			zap.Error(err),
			zap.Int("user_id", userID),
			zap.Int("strategy_group_id", strategyGroupID),
			zap.Int("version_id", versionID))
		return err
	}

	return nil
}
