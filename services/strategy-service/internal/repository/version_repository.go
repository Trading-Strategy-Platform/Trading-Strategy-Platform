package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"services/strategy-service/internal/model"

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

// GetVersions retrieves all versions for a strategy with proper database-level pagination
func (r *VersionRepository) GetVersions(ctx context.Context, strategyID int, userID int, page, limit int) ([]model.StrategyVersion, int, error) {
	// First, count total accessible versions
	countQuery := `
		SELECT COUNT(*) FROM get_accessible_strategy_versions($1, $2)
	`

	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, userID, strategyID)
	if err != nil {
		r.logger.Error("Failed to count strategy versions", zap.Error(err))
		return nil, 0, err
	}

	// Now get paginated accessible versions
	// Calculate offset
	offset := (page - 1) * limit

	versionQuery := `
		SELECT * FROM get_accessible_strategy_versions($1, $2)
		LIMIT $3 OFFSET $4
	`

	var versionResults []struct {
		Version         int       `db:"version"`
		ChangeNotes     string    `db:"change_notes"`
		CreatedAt       time.Time `db:"created_at"`
		IsActiveVersion bool      `db:"is_active_version"`
	}

	err = r.db.SelectContext(ctx, &versionResults, versionQuery, userID, strategyID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to query strategy versions", zap.Error(err))
		return nil, 0, err
	}

	// Get all version details
	allVersions := []model.StrategyVersion{}
	for _, v := range versionResults {
		// Get full version details including structure
		fullVersion, err := r.GetVersion(ctx, strategyID, v.Version)
		if err != nil {
			r.logger.Error("Failed to get version details", zap.Error(err))
			continue
		}
		if fullVersion != nil {
			allVersions = append(allVersions, *fullVersion)
		}
	}

	return allVersions, totalCount, nil
}

// GetVersion retrieves a specific version of a strategy
func (r *VersionRepository) GetVersion(ctx context.Context, strategyID int, versionNumber int) (*model.StrategyVersion, error) {
	query := `
		SELECT id, strategy_id, version, structure, change_notes, created_at
		FROM strategy_versions
		WHERE strategy_id = $1 AND version = $2
	`

	var version model.StrategyVersion
	var structureBytes []byte

	row := r.db.QueryRowContext(ctx, query, strategyID, versionNumber)
	err := row.Scan(
		&version.ID,
		&version.StrategyID,
		&version.Version,
		&structureBytes,
		&version.ChangeNotes,
		&version.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get strategy version", zap.Error(err))
		return nil, err
	}

	// Unmarshal the structure
	if err := json.Unmarshal(structureBytes, &version.Structure); err != nil {
		r.logger.Error("Failed to unmarshal version structure", zap.Error(err))
		return nil, err
	}

	return &version, nil
}

// UpdateUserVersion updates the active version for a user using update_user_strategy_version function
func (r *VersionRepository) UpdateUserVersion(ctx context.Context, userID int, strategyID int, version int) error {
	query := `SELECT update_user_strategy_version($1, $2, $3)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, userID, strategyID, version).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to update user strategy version", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to update active version or not authorized")
	}

	return nil
}
