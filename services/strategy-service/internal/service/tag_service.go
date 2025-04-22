package service

import (
	"context"
	"errors"
	"strings"

	"services/strategy-service/internal/model"
	"services/strategy-service/internal/repository"

	"go.uber.org/zap"
)

// TagService handles strategy tag operations
type TagService struct {
	tagRepo *repository.TagRepository
	logger  *zap.Logger
}

// NewTagService creates a new tag service
func NewTagService(tagRepo *repository.TagRepository, logger *zap.Logger) *TagService {
	return &TagService{
		tagRepo: tagRepo,
		logger:  logger,
	}
}

// GetAllTags retrieves all tags with optional filtering and sorting
func (s *TagService) GetAllTags(
	ctx context.Context,
	searchTerm string,
	sortBy string,
	sortDirection string,
	page, limit int,
) ([]model.TagWithCount, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 100 // Default larger for tags since there shouldn't be too many
	}

	// Validate sort parameters
	validSortFields := map[string]bool{
		"name":           true,
		"strategy_count": true,
		"id":             true,
	}

	if !validSortFields[sortBy] {
		sortBy = "name" // Default to name
	}

	if sortDirection != "ASC" && sortDirection != "DESC" {
		sortDirection = "ASC" // Default to ascending for tags
	}

	return s.tagRepo.GetAllTags(ctx, searchTerm, sortBy, sortDirection, page, limit)
}

// GetTagByID retrieves a specific tag by ID
func (s *TagService) GetTagByID(ctx context.Context, id int) (*model.TagWithCount, error) {
	tag, err := s.tagRepo.GetTagByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if tag == nil {
		return nil, errors.New("tag not found")
	}

	return tag, nil
}

// CreateTag creates a new tag
func (s *TagService) CreateTag(ctx context.Context, name string) (*model.TagWithCount, error) {
	// Validate tag name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("tag name cannot be empty")
	}

	if len(name) > 50 {
		return nil, errors.New("tag name cannot exceed 50 characters")
	}

	id, err := s.tagRepo.CreateTag(ctx, name)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, errors.New("tag name already exists")
		}
		return nil, err
	}

	// Return the created tag with count 0
	return &model.TagWithCount{
		ID:            id,
		Name:          name,
		StrategyCount: 0,
	}, nil
}

// UpdateTag updates an existing tag
func (s *TagService) UpdateTag(ctx context.Context, id int, name string) (*model.TagWithCount, error) {
	if id <= 0 {
		return nil, errors.New("invalid tag ID")
	}

	// Validate tag name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("tag name cannot be empty")
	}

	if len(name) > 50 {
		return nil, errors.New("tag name cannot exceed 50 characters")
	}

	// Check if tag exists before updating
	existingTag, err := s.tagRepo.GetTagByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existingTag == nil {
		return nil, errors.New("tag not found")
	}

	err = s.tagRepo.UpdateTag(ctx, id, name)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, errors.New("tag name already exists")
		}
		return nil, err
	}

	// Return the updated tag
	return &model.TagWithCount{
		ID:            id,
		Name:          name,
		StrategyCount: existingTag.StrategyCount, // Preserve the strategy count
	}, nil
}

// DeleteTag deletes an existing tag
func (s *TagService) DeleteTag(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("invalid tag ID")
	}

	return s.tagRepo.DeleteTag(ctx, id)
}

// GetPopularTags returns the most frequently used tags
func (s *TagService) GetPopularTags(ctx context.Context, limit int) ([]model.TagWithCount, error) {
	if limit < 1 {
		limit = 10 // Default to top 10
	} else if limit > 100 {
		limit = 100 // Max 100 tags
	}

	return s.tagRepo.GetPopularTags(ctx, limit)
}
