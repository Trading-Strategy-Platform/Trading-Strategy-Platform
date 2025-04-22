package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"services/strategy-service/internal/model"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// StrategyRepository handles database operations for strategies
type StrategyRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewStrategyRepository creates a new strategy repository
func NewStrategyRepository(db *sqlx.DB, logger *zap.Logger) *StrategyRepository {
	return &StrategyRepository{
		db:     db,
		logger: logger,
	}
}

// GetAllStrategies retrieves all strategies for a user with filtering options
func (r *StrategyRepository) GetAllStrategies(
	ctx context.Context,
	userID int,
	searchTerm string,
	purchasedOnly bool,
	tagIDs []int,
	sortBy string,
	sortDirection string,
	limit, offset int,
) ([]model.Strategy, int, error) {
	// First, get total count using count_strategies function
	countQuery := `SELECT count_strategies($1, $2, $3, $4)`

	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, userID, searchTerm, purchasedOnly, pq.Array(tagIDs))
	if err != nil {
		r.logger.Error("Failed to count strategies", zap.Error(err))
		return nil, 0, err
	}

	// Use the get_all_strategies function to fetch strategies with pagination
	query := `SELECT * FROM get_all_strategies($1, $2, $3, $4, $5, $6, $7, $8)`

	rows, err := r.db.QueryxContext(
		ctx,
		query,
		userID,
		searchTerm,
		purchasedOnly,
		pq.Array(tagIDs),
		sortBy,
		sortDirection,
		limit,
		offset,
	)
	if err != nil {
		r.logger.Error("Failed to get strategies", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var strategies []model.Strategy
	for rows.Next() {
		var s model.Strategy
		var updatedAt sql.NullTime
		var purchaseID sql.NullInt64
		var purchaseDate sql.NullTime
		var tagIDsArray []int

		err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.Description,
			&s.ThumbnailURL,
			&s.UserID, // This is owner_id
			&s.UserID, // This is owner_user_id (duplicate in the function)
			&s.IsPublic,
			&s.IsActive,
			&s.Version,
			&s.CreatedAt,
			&updatedAt,
			&s.StrategyGroupID,
			&s.AccessType,
			&purchaseID,
			&purchaseDate,
			pq.Array(&tagIDsArray),
			&s.Structure,
		)

		if err != nil {
			r.logger.Error("Failed to scan strategy row", zap.Error(err))
			return nil, 0, err
		}

		// Set nullable fields
		if updatedAt.Valid {
			s.UpdatedAt = &updatedAt.Time
		}
		if purchaseID.Valid {
			pid := int(purchaseID.Int64)
			s.PurchaseID = &pid
		}
		if purchaseDate.Valid {
			s.PurchaseDate = &purchaseDate.Time
		}

		s.TagIDs = tagIDsArray

		strategies = append(strategies, s)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating strategy rows", zap.Error(err))
		return nil, 0, err
	}

	return strategies, totalCount, nil
}

// GetStrategyByID retrieves a strategy by ID
func (r *StrategyRepository) GetStrategyByID(ctx context.Context, id int) (*model.Strategy, error) {
	query := `
		SELECT 
			id, name, user_id, description, thumbnail_url, structure, 
			is_public, is_active, version, created_at, updated_at, strategy_group_id
		FROM strategies 
		WHERE id = $1 AND is_active = TRUE
	`

	var strategy model.Strategy
	var updatedAt sql.NullTime

	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&strategy.ID,
		&strategy.Name,
		&strategy.UserID,
		&strategy.Description,
		&strategy.ThumbnailURL,
		&strategy.Structure,
		&strategy.IsPublic,
		&strategy.IsActive,
		&strategy.Version,
		&strategy.CreatedAt,
		&updatedAt,
		&strategy.StrategyGroupID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error
		}
		r.logger.Error("Failed to get strategy by ID", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	if updatedAt.Valid {
		strategy.UpdatedAt = &updatedAt.Time
	}

	// Get tags for the strategy
	tagsQuery := `
		SELECT t.id, t.name
		FROM strategy_tags t
		JOIN strategy_tag_mappings m ON t.id = m.tag_id
		WHERE m.strategy_id = $1
	`

	var tags []model.Tag
	err = r.db.SelectContext(ctx, &tags, tagsQuery, strategy.StrategyGroupID)
	if err != nil {
		r.logger.Warn("Failed to get tags for strategy", zap.Error(err), zap.Int("id", id))
		// Continue without tags rather than failing
	} else {
		strategy.Tags = tags

		// Extract tag IDs
		tagIDs := make([]int, len(tags))
		for i, tag := range tags {
			tagIDs[i] = tag.ID
		}
		strategy.TagIDs = tagIDs
	}

	return &strategy, nil
}

// GetStrategyByIDWithAccess retrieves a strategy by ID with user access check
func (r *StrategyRepository) GetStrategyByIDWithAccess(ctx context.Context, id int, userID int) (*model.Strategy, error) {
	query := `SELECT * FROM get_strategy_by_id($1, $2)`

	rows, err := r.db.QueryxContext(ctx, query, id, userID)
	if err != nil {
		r.logger.Error("Failed to execute get_strategy_by_id function", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil // No strategy found
	}

	var strategy model.Strategy
	var updatedAt sql.NullTime

	err = rows.Scan(
		&strategy.ID,
		&strategy.Name,
		&strategy.UserID,
		&strategy.Description,
		&strategy.ThumbnailURL,
		&strategy.Structure,
		&strategy.IsPublic,
		&strategy.IsActive,
		&strategy.Version,
		&strategy.CreatedAt,
		&updatedAt,
		&strategy.StrategyGroupID,
	)

	if err != nil {
		r.logger.Error("Failed to scan strategy row", zap.Error(err))
		return nil, err
	}

	if updatedAt.Valid {
		strategy.UpdatedAt = &updatedAt.Time
	}

	// Get tags for the strategy
	tagsQuery := `
		SELECT t.id, t.name
		FROM strategy_tags t
		JOIN strategy_tag_mappings m ON t.id = m.tag_id
		WHERE m.strategy_id = $1
	`

	var tags []model.Tag
	err = r.db.SelectContext(ctx, &tags, tagsQuery, strategy.StrategyGroupID)
	if err != nil {
		r.logger.Warn("Failed to get tags for strategy", zap.Error(err), zap.Int("id", id))
		// Continue without tags rather than failing
	} else {
		strategy.Tags = tags

		// Extract tag IDs
		tagIDs := make([]int, len(tags))
		for i, tag := range tags {
			tagIDs[i] = tag.ID
		}
		strategy.TagIDs = tagIDs
	}

	return &strategy, nil
}

// CreateStrategy creates a new strategy
func (r *StrategyRepository) CreateStrategy(ctx context.Context, strategy *model.StrategyCreate, userID int) (int, error) {
	query := `SELECT create_strategy($1, $2, $3, $4, $5, $6, $7)`

	var strategyID int
	err := r.db.QueryRowContext(
		ctx,
		query,
		userID,
		strategy.Name,
		strategy.Description,
		strategy.ThumbnailURL,
		strategy.Structure,
		strategy.IsPublic,
		pq.Array(strategy.TagIDs),
	).Scan(&strategyID)

	if err != nil {
		r.logger.Error("Failed to create strategy", zap.Error(err))
		return 0, err
	}

	return strategyID, nil
}

// UpdateStrategy updates a strategy by creating a new version
func (r *StrategyRepository) UpdateStrategy(
	ctx context.Context,
	strategyID int,
	userID int,
	update *model.StrategyUpdate,
) (int, error) {
	query := `SELECT update_strategy($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	var newVersionID int
	err := r.db.QueryRowContext(
		ctx,
		query,
		strategyID,
		userID,
		update.Name,
		update.Description,
		update.ThumbnailURL,
		update.Structure,
		update.IsPublic,
		update.ChangeNotes,
		pq.Array(update.TagIDs),
	).Scan(&newVersionID)

	if err != nil {
		r.logger.Error("Failed to update strategy", zap.Error(err))
		return 0, err
	}

	return newVersionID, nil
}

// DeleteStrategy marks a strategy as inactive
func (r *StrategyRepository) DeleteStrategy(ctx context.Context, strategyID int, userID int) error {
	query := `SELECT delete_strategy($1, $2)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, strategyID, userID).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to delete strategy", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("strategy not found or you don't have permission to delete it")
	}

	return nil
}

// GetStrategyVersions retrieves all versions of a strategy
func (r *StrategyRepository) GetStrategyVersions(
	ctx context.Context,
	strategyGroupID int,
	userID int,
	sortBy string,
	sortDirection string,
	limit, offset int,
) ([]model.Strategy, int, error) {
	// First get the total count of versions
	countQuery := `SELECT count_strategy_versions($1, $2)`

	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, strategyGroupID, userID)
	if err != nil {
		r.logger.Error("Failed to count strategy versions", zap.Error(err))
		return nil, 0, err
	}

	// Get versions with pagination
	query := `SELECT * FROM get_strategy_versions($1, $2, $3, $4, $5, $6)`

	rows, err := r.db.QueryxContext(
		ctx,
		query,
		strategyGroupID,
		userID,
		sortBy,
		sortDirection,
		limit,
		offset,
	)
	if err != nil {
		r.logger.Error("Failed to get strategy versions", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var versions []model.Strategy
	for rows.Next() {
		var version model.Strategy
		var updatedAt sql.NullTime

		err := rows.Scan(
			&version.ID,
			&version.Name,
			&version.UserID,
			&version.Description,
			&version.ThumbnailURL,
			&version.Structure,
			&version.IsPublic,
			&version.IsActive,
			&version.Version,
			&version.CreatedAt,
			&updatedAt,
			&version.StrategyGroupID,
			&version.IsCurrentVersion,
		)

		if err != nil {
			r.logger.Error("Failed to scan version row", zap.Error(err))
			return nil, 0, err
		}

		if updatedAt.Valid {
			version.UpdatedAt = &updatedAt.Time
		}

		versions = append(versions, version)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating version rows", zap.Error(err))
		return nil, 0, err
	}

	return versions, totalCount, nil
}

// SetUserActiveVersion sets the active version for a user
func (r *StrategyRepository) SetUserActiveVersion(ctx context.Context, userID, strategyGroupID, versionID int) error {
	query := `SELECT set_user_active_version($1, $2, $3)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, userID, strategyGroupID, versionID).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to set user active version", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to set active version or user doesn't have access")
	}

	return nil
}

// UpdateThumbnail updates a strategy's thumbnail URL
func (r *StrategyRepository) UpdateThumbnail(ctx context.Context, strategyID int, userID int, thumbnailURL string) error {
	query := `
		UPDATE strategies 
		SET thumbnail_url = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(ctx, query, thumbnailURL, strategyID, userID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("strategy not found or you don't have permission to update it")
		}
		r.logger.Error("Failed to update strategy thumbnail", zap.Error(err))
		return err
	}

	return nil
}
