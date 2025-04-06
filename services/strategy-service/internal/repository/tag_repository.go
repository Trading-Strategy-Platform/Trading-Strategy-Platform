package repository

import (
	"context"

	"services/strategy-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// TagRepository handles database operations for strategy tags
type TagRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewTagRepository creates a new tag repository
func NewTagRepository(db *sqlx.DB, logger *zap.Logger) *TagRepository {
	return &TagRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll retrieves all tags using get_strategy_tags function
func (r *TagRepository) GetAll(ctx context.Context, page, limit int) ([]model.Tag, int, error) {
	query := `SELECT * FROM get_strategy_tags()`

	var allTags []model.Tag
	err := r.db.SelectContext(ctx, &allTags, query)
	if err != nil {
		r.logger.Error("Failed to get tags", zap.Error(err))
		return nil, 0, err
	}

	// Get total count before pagination
	total := len(allTags)

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		return []model.Tag{}, total, nil
	}

	if end > total {
		end = total
	}

	return allTags[start:end], total, nil
}

// Create adds a new tag using add_strategy_tag function
func (r *TagRepository) Create(ctx context.Context, name string) (int, error) {
	query := `SELECT add_strategy_tag($1)`

	var id int
	err := r.db.QueryRowContext(ctx, query, name).Scan(&id)
	if err != nil {
		r.logger.Error("Failed to create tag", zap.Error(err))
		return 0, err
	}

	return id, nil
}
