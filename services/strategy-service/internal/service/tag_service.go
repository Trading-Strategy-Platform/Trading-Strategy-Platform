package service

import (
	"context"

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

// GetAllTags retrieves all tags using get_strategy_tags function
func (s *TagService) GetAllTags(ctx context.Context, page, limit int) ([]model.Tag, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 100 // Default larger for tags since there shouldn't be too many
	}

	return s.tagRepo.GetAll(ctx, page, limit)
}

// CreateTag creates a new tag using add_strategy_tag function
func (s *TagService) CreateTag(ctx context.Context, name string) (*model.Tag, error) {
	id, err := s.tagRepo.Create(ctx, name)
	if err != nil {
		return nil, err
	}

	return &model.Tag{
		ID:   id,
		Name: name,
	}, nil
}
