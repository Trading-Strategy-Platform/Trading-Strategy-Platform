package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

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

// GetUserStrategies retrieves strategies with proper database-level pagination
func (r *StrategyRepository) GetUserStrategies(ctx context.Context, userID int, searchTerm string, purchasedOnly bool, tags []int, page, limit int) ([]model.ExtendedStrategy, int, error) {
	// Calculate offset
	offset := (page - 1) * limit

	// Ensure tags is initialized to empty array if nil
	tagsParam := pq.Array(tags)
	if tags == nil {
		tagsParam = pq.Array([]int{})
	}

	// First, get total count using the count function
	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, `SELECT count_my_strategies($1, $2, $3, $4)`,
		userID,
		searchTerm,
		purchasedOnly,
		tagsParam,
	)

	if err != nil {
		r.logger.Error("Failed to count user strategies", zap.Error(err))
		return nil, 0, err
	}

	// Now get the actual paginated data from the function with pagination parameters
	var strategies []model.ExtendedStrategy
	err = r.db.SelectContext(ctx, &strategies, `
		SELECT * FROM get_my_strategies($1, $2, $3, $4, $5, $6)
	`,
		userID,
		searchTerm,
		purchasedOnly,
		tagsParam,
		limit,
		offset,
	)

	if err != nil {
		r.logger.Error("Failed to get user strategies", zap.Error(err))
		return nil, 0, err
	}

	// Fetch tags for each strategy in the paginated results
	for i, strategy := range strategies {
		if len(strategy.TagIDs) > 0 {
			tags, err := r.getTagsByIDs(ctx, strategy.TagIDs)
			if err != nil {
				r.logger.Warn("Failed to get strategy tags",
					zap.Error(err),
					zap.Int("strategy_id", strategy.ID),
				)
			} else {
				strategies[i].Tags = tags
			}
		}
	}

	return strategies, totalCount, nil
}

// Create adds a new strategy using add_strategy function
func (r *StrategyRepository) Create(ctx context.Context, strategy *model.StrategyCreate, userID int) (int, error) {
	query := `SELECT add_strategy($1, $2, $3, $4, $5, $6, $7)`

	// Convert strategy structure to JSON
	structureBytes, err := json.Marshal(strategy.Structure)
	if err != nil {
		r.logger.Error("Failed to marshal strategy structure", zap.Error(err))
		return 0, err
	}

	var id int
	err = r.db.QueryRowContext(
		ctx,
		query,
		userID,                  // p_user_id
		strategy.Name,           // p_name
		strategy.Description,    // p_description
		strategy.ThumbnailURL,   // p_thumbnail_url
		structureBytes,          // p_structure
		strategy.IsPublic,       // p_is_public
		pq.Array(strategy.Tags), // p_tag_ids
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create strategy", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// GetByID retrieves a strategy by ID
func (r *StrategyRepository) GetByID(ctx context.Context, id int) (*model.Strategy, error) {
	query := `
		SELECT id, name, user_id, description, thumbnail_url, structure, is_public, is_active, version, created_at, updated_at
		FROM strategies
		WHERE id = $1
	`

	var strategy model.Strategy
	var structureBytes []byte

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&strategy.ID,
		&strategy.Name,
		&strategy.UserID,
		&strategy.Description,
		&strategy.ThumbnailURL,
		&structureBytes,
		&strategy.IsPublic,
		&strategy.IsActive,
		&strategy.Version,
		&strategy.CreatedAt,
		&strategy.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get strategy by ID", zap.Error(err), zap.Int("strategy_id", id))
		return nil, err
	}

	// Unmarshal the strategy structure
	if err := json.Unmarshal(structureBytes, &strategy.Structure); err != nil {
		r.logger.Error("Failed to unmarshal strategy structure", zap.Error(err))
		return nil, err
	}

	// Get tags for the strategy
	tags, err := r.getTagsByIDs(ctx, r.getStrategyTagIDs(ctx, id))
	if err != nil {
		r.logger.Warn("Failed to get strategy tags", zap.Error(err), zap.Int("strategy_id", id))
	} else {
		strategy.Tags = tags
	}

	return &strategy, nil
}

// Update updates a strategy using update_strategy function
func (r *StrategyRepository) Update(ctx context.Context, id int, update *model.StrategyUpdate, userID int) error {
	query := `SELECT update_strategy($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	// Handle nil Structure case
	var structureBytes []byte
	if update.Structure != nil {
		var err error
		structureBytes, err = json.Marshal(*update.Structure)
		if err != nil {
			r.logger.Error("Failed to marshal strategy structure", zap.Error(err))
			return err
		}
	}

	// Default values for nullable fields
	name := ""
	if update.Name != nil {
		name = *update.Name
	}

	description := ""
	if update.Description != nil {
		description = *update.Description
	}

	thumbnailURL := ""
	if update.ThumbnailURL != nil {
		thumbnailURL = *update.ThumbnailURL
	}

	isPublic := false
	if update.IsPublic != nil {
		isPublic = *update.IsPublic
	}

	// Generate change notes based on what's being updated
	changeNotes := "Updated strategy"
	if update.ChangeNotes != nil && *update.ChangeNotes != "" {
		changeNotes = *update.ChangeNotes
	} else if update.Structure != nil {
		changeNotes = "Updated strategy structure"
	}

	var success bool
	err := r.db.QueryRowContext(
		ctx,
		query,
		id,                    // p_strategy_id
		userID,                // p_user_id
		name,                  // p_name
		description,           // p_description
		thumbnailURL,          // p_thumbnail_url
		structureBytes,        // p_structure
		isPublic,              // p_is_public
		changeNotes,           // p_change_notes
		pq.Array(update.Tags), // p_tag_ids
	).Scan(&success)

	if err != nil {
		r.logger.Error("Failed to update strategy", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to update strategy or not authorized")
	}

	return nil
}

// Delete marks a strategy as inactive using delete_strategy function
func (r *StrategyRepository) Delete(ctx context.Context, id int, userID int) error {
	query := `SELECT delete_strategy($1, $2)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to delete strategy", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to delete strategy or not authorized")
	}

	return nil
}

// getStrategyTagIDs retrieves tag IDs for a strategy
func (r *StrategyRepository) getStrategyTagIDs(ctx context.Context, strategyID int) []int {
	query := `
		SELECT tag_id
		FROM strategy_tag_mappings
		WHERE strategy_id = $1
	`

	var tagIDs []int
	err := r.db.SelectContext(ctx, &tagIDs, query, strategyID)
	if err != nil {
		r.logger.Warn("Failed to get strategy tag IDs", zap.Error(err), zap.Int("strategy_id", strategyID))
		return []int{}
	}

	return tagIDs
}

// getTagsByIDs retrieves tags by their IDs
func (r *StrategyRepository) getTagsByIDs(ctx context.Context, tagIDs []int) ([]model.Tag, error) {
	if len(tagIDs) == 0 {
		return []model.Tag{}, nil
	}

	query := `
		SELECT id, name
		FROM strategy_tags
		WHERE id = ANY($1)
	`

	var tags []model.Tag
	err := r.db.SelectContext(ctx, &tags, query, pq.Array(tagIDs))
	if err != nil {
		return nil, err
	}

	return tags, nil
}
