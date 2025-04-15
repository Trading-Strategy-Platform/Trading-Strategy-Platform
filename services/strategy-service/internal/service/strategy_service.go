package service

import (
	"context"
	"errors"

	"services/strategy-service/internal/client"
	"services/strategy-service/internal/model"
	"services/strategy-service/internal/repository"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// UserClient defines methods for interacting with the User Service
type UserClient interface {
	GetUserByID(ctx context.Context, userID int) (string, error) // Returns username
	ValidateUserAccess(ctx context.Context, userID int, token string) (bool, error)
	BatchGetUsersByIDs(ctx context.Context, userIDs []int) (map[int]client.UserDetails, error) // Add this new method
}

// BacktestClient defines methods for interacting with the Historical Data Service
type BacktestClient interface {
	CreateBacktest(ctx context.Context, request *model.BacktestRequest, userID int) (int, error)
}

// StrategyService handles strategy operations
type StrategyService struct {
	db             *sqlx.DB
	strategyRepo   *repository.StrategyRepository
	versionRepo    *repository.VersionRepository
	tagRepo        *repository.TagRepository
	userClient     UserClient
	backtestClient BacktestClient
	logger         *zap.Logger
}

// NewStrategyService creates a new strategy service
func NewStrategyService(
	db *sqlx.DB,
	strategyRepo *repository.StrategyRepository,
	versionRepo *repository.VersionRepository,
	tagRepo *repository.TagRepository,
	userClient UserClient,
	backtestClient BacktestClient,
	logger *zap.Logger,
) *StrategyService {
	return &StrategyService{
		db:             db,
		strategyRepo:   strategyRepo,
		versionRepo:    versionRepo,
		tagRepo:        tagRepo,
		userClient:     userClient,
		backtestClient: backtestClient,
		logger:         logger,
	}
}

// GetUserStrategies retrieves strategies for a user
func (s *StrategyService) GetUserStrategies(ctx context.Context, userID int, searchTerm string, purchasedOnly bool, tagIDs []int, page, limit int) ([]model.ExtendedStrategy, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Get strategies with pagination parameters passed to repository
	return s.strategyRepo.GetUserStrategies(ctx, userID, searchTerm, purchasedOnly, tagIDs, page, limit)
}

// CreateStrategy creates a new strategy
func (s *StrategyService) CreateStrategy(ctx context.Context, strategy *model.StrategyCreate, userID int) (*model.Strategy, error) {

	// Create strategy using add_strategy function
	strategyID, err := s.strategyRepo.Create(ctx, strategy, userID)
	if err != nil {
		return nil, err
	}

	// Get the created strategy
	createdStrategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	return createdStrategy, nil
}

// GetStrategy retrieves a strategy by ID
func (s *StrategyService) GetStrategy(ctx context.Context, id int, userID int) (*model.Strategy, error) {
	// Get strategy
	strategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	// Check if strategy is active
	if !strategy.IsActive {
		return nil, errors.New("strategy is not active")
	}

	return strategy, nil
}

// UpdateStrategy updates a strategy
func (s *StrategyService) UpdateStrategy(ctx context.Context, id int, update *model.StrategyUpdate, userID int) (*model.Strategy, error) {
	// Check if strategy exists and belongs to the user
	strategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return nil, errors.New("access denied")
	}

	// Comment out or remove this validation block:

	// Update strategy using update_strategy function
	err = s.strategyRepo.Update(ctx, id, update, userID)
	if err != nil {
		return nil, err
	}

	// Get the updated strategy
	updatedStrategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return updatedStrategy, nil
}

// DeleteStrategy deletes a strategy (marks it as inactive)
func (s *StrategyService) DeleteStrategy(ctx context.Context, id int, userID int) error {
	// Check if strategy exists and belongs to the user
	strategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if strategy == nil {
		return errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return errors.New("access denied")
	}

	// The delete_strategy function just marks the strategy as inactive
	return s.strategyRepo.Delete(ctx, id, userID)
}

// UpdateUserStrategyVersion updates the active version of a strategy for a user
func (s *StrategyService) UpdateUserStrategyVersion(ctx context.Context, userID, strategyID, version int) error {
	// First check if strategy exists
	strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return err
	}

	if strategy == nil {
		return errors.New("strategy not found")
	}

	// Update active version using update_user_strategy_version function
	return s.versionRepo.UpdateUserVersion(ctx, userID, strategyID, version)
}

// GetVersions retrieves versions of a strategy
func (s *StrategyService) GetVersions(ctx context.Context, strategyID int, userID int, page, limit int) ([]model.StrategyVersion, int, error) {
	// Check if strategy exists
	strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return nil, 0, err
	}

	if strategy == nil {
		return nil, 0, errors.New("strategy not found")
	}

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20 // Default for versions is 20
	}

	// Get versions with pagination passed to repository
	return s.versionRepo.GetVersions(ctx, strategyID, userID, page, limit)
}

// UpdateThumbnail updates a strategy's thumbnail URL
func (s *StrategyService) UpdateThumbnail(ctx context.Context, id int, userID int, thumbnailURL string) error {
	// Check if strategy exists and belongs to the user
	strategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if strategy == nil {
		return errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return errors.New("access denied")
	}

	// Create an update object with all necessary fields to preserve them
	thumbnailUpdate := &model.StrategyUpdate{
		ThumbnailURL: &thumbnailURL,
		Structure:    &strategy.Structure,   // Keep existing structure
		Name:         &strategy.Name,        // Keep existing name
		Description:  &strategy.Description, // Keep existing description
		IsPublic:     &strategy.IsPublic,    // Keep existing public status
	}

	// Update the strategy
	err = s.strategyRepo.Update(ctx, id, thumbnailUpdate, userID)
	if err != nil {
		s.logger.Error("Failed to update strategy thumbnail",
			zap.Error(err),
			zap.Int("id", id),
			zap.Int("userID", userID),
			zap.String("thumbnailURL", thumbnailURL))
		return errors.New("failed to update strategy thumbnail")
	}

	s.logger.Info("Strategy thumbnail updated successfully",
		zap.Int("id", id),
		zap.Int("userID", userID),
		zap.String("thumbnailURL", thumbnailURL))

	return nil
}
