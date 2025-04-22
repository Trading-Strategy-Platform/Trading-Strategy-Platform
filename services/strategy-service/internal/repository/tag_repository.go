package repository

import (
	"context"
	"errors"

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

// GetAll retrieves all tags with proper database-level pagination
func (r *TagRepository) GetAll(ctx context.Context, searchTerm string, page, limit int) ([]model.Tag, int, error) {
	// Calculate offset
	offset := (page - 1) * limit

	// First, get total count using the count function
	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, `SELECT count_strategy_tags($1)`, searchTerm)
	if err != nil {
		r.logger.Error("Failed to count tags", zap.Error(err), zap.String("search", searchTerm))
		return nil, 0, err
	}

	// Now, get paginated tags using the function with pagination parameters
	var tags []model.Tag
	err = r.db.SelectContext(ctx, &tags, `
		SELECT * FROM get_strategy_tags($1, $2, $3)
	`, searchTerm, limit, offset)

	if err != nil {
		r.logger.Error("Failed to get tags", zap.Error(err), zap.String("search", searchTerm))
		return nil, 0, err
	}

	return tags, totalCount, nil
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

// Update updates a tag's name using update_strategy_tag function
func (r *TagRepository) Update(ctx context.Context, id int, name string) error {
	query := `SELECT update_strategy_tag($1, $2)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, id, name).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to update tag", zap.Error(err), zap.Int("id", id))
		return err
	}

	if !success {
		return errors.New("tag not found")
	}

	return nil
}

// Delete removes a tag using delete_strategy_tag function
func (r *TagRepository) Delete(ctx context.Context, id int) error {
	query := `SELECT delete_strategy_tag($1)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to delete tag", zap.Error(err), zap.Int("id", id))
		return err
	}

	if !success {
		return errors.New("tag not found or is in use")
	}

	return nil
}
