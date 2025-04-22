package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

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

// GetAllTags retrieves all tags with sorting and pagination
func (r *TagRepository) GetAllTags(
	ctx context.Context,
	searchTerm string,
	sortBy string,
	sortDirection string,
	page, limit int,
) ([]model.TagWithCount, int, error) {
	// Calculate offset
	offset := (page - 1) * limit

	// Validate sort field
	validSortFields := map[string]bool{
		"name":           true,
		"strategy_count": true,
		"id":             true,
	}

	if !validSortFields[sortBy] {
		sortBy = "name" // Default sort by name
	}

	// Normalize sort direction
	sortDirection = strings.ToUpper(sortDirection)
	if sortDirection != "ASC" && sortDirection != "DESC" {
		sortDirection = "ASC" // Default ascending for tags
	}

	// First, get total count using the count function
	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, `SELECT count_tags($1)`, searchTerm)
	if err != nil {
		r.logger.Error("Failed to count tags", zap.Error(err), zap.String("search", searchTerm))
		return nil, 0, err
	}

	// Now, get paginated and sorted tags using the function
	var tags []model.TagWithCount
	err = r.db.SelectContext(ctx, &tags, `
		SELECT * FROM get_all_tags($1, $2, $3, $4, $5)
	`, searchTerm, sortBy, sortDirection, limit, offset)

	if err != nil {
		r.logger.Error("Failed to get tags", zap.Error(err), zap.String("search", searchTerm))
		return nil, 0, err
	}

	return tags, totalCount, nil
}

// GetTagByID retrieves a tag by ID with strategy count
func (r *TagRepository) GetTagByID(ctx context.Context, id int) (*model.TagWithCount, error) {
	query := `SELECT * FROM get_tag_by_id($1)`

	var tag model.TagWithCount
	err := r.db.GetContext(ctx, &tag, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get tag by ID", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &tag, nil
}

// CreateTag adds a new tag
func (r *TagRepository) CreateTag(ctx context.Context, name string) (int, error) {
	query := `SELECT create_tag($1)`

	var id int
	err := r.db.QueryRowContext(ctx, query, name).Scan(&id)
	if err != nil {
		r.logger.Error("Failed to create tag", zap.Error(err), zap.String("name", name))
		return 0, err
	}

	return id, nil
}

// UpdateTag updates a tag's name
func (r *TagRepository) UpdateTag(ctx context.Context, id int, name string) error {
	query := `SELECT update_tag($1, $2)`

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

// DeleteTag removes a tag
func (r *TagRepository) DeleteTag(ctx context.Context, id int) error {
	query := `SELECT delete_tag($1)`

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

// GetPopularTags returns the most frequently used tags
func (r *TagRepository) GetPopularTags(ctx context.Context, limit int) ([]model.TagWithCount, error) {
	query := `SELECT * FROM get_popular_tags($1)`

	var tags []model.TagWithCount
	err := r.db.SelectContext(ctx, &tags, query, limit)
	if err != nil {
		r.logger.Error("Failed to get popular tags", zap.Error(err))
		return nil, err
	}

	return tags, nil
}
