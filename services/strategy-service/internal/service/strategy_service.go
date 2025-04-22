package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"services/strategy-service/internal/client"
	"services/strategy-service/internal/model"
	"services/strategy-service/internal/repository"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// StrategyService handles strategy operations
type StrategyService struct {
	db               *sqlx.DB
	strategyRepo     *repository.StrategyRepository
	versionRepo      *repository.VersionRepository
	tagRepo          *repository.TagRepository
	userClient       *client.UserClient
	historicalClient *client.HistoricalClient
	logger           *zap.Logger
}

// NewStrategyService creates a new strategy service
func NewStrategyService(
	db *sqlx.DB,
	strategyRepo *repository.StrategyRepository,
	versionRepo *repository.VersionRepository,
	tagRepo *repository.TagRepository,
	userClient *client.UserClient,
	historicalClient *client.HistoricalClient,
	logger *zap.Logger,
) *StrategyService {
	return &StrategyService{
		db:               db,
		strategyRepo:     strategyRepo,
		versionRepo:      versionRepo,
		tagRepo:          tagRepo,
		userClient:       userClient,
		historicalClient: historicalClient,
		logger:           logger,
	}
}

// GetAllStrategies retrieves all strategies for a user with filtering options
func (s *StrategyService) GetAllStrategies(
	ctx context.Context,
	userID int,
	searchTerm string,
	purchasedOnly bool,
	tagIDs []int,
	sortBy string,
	sortDirection string,
	page, limit int,
) ([]model.Strategy, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Validate sort parameters
	validSortFields := map[string]bool{
		"name":       true,
		"created_at": true,
		"updated_at": true,
		"version":    true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at" // Default sort by creation date
	}

	// Normalize sort direction
	sortDirection = strings.ToUpper(sortDirection)
	if sortDirection != "ASC" && sortDirection != "DESC" {
		sortDirection = "DESC" // Default to descending (newest first)
	}

	// Get strategies using repository
	strategies, totalCount, err := s.strategyRepo.GetAllStrategies(
		ctx,
		userID,
		searchTerm,
		purchasedOnly,
		tagIDs,
		sortBy,
		sortDirection,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, err
	}

	// Enhance with username if possible
	if len(strategies) > 0 {
		// Extract unique user IDs
		userIDs := make([]int, 0)
		userIDMap := make(map[int]bool)

		for _, strategy := range strategies {
			if !userIDMap[strategy.UserID] {
				userIDMap[strategy.UserID] = true
				userIDs = append(userIDs, strategy.UserID)
			}
		}

		// Try to batch fetch user details
		userDetails, err := s.userClient.BatchGetUsersByIDs(ctx, userIDs)
		if err == nil {
			// Update strategies with usernames
			for i := range strategies {
				if details, ok := userDetails[strategies[i].UserID]; ok {
					strategies[i].Username = details.Username
				}
			}
		} else {
			s.logger.Warn("Failed to fetch user details", zap.Error(err))
			// Continue without usernames
		}
	}

	return strategies, totalCount, nil
}

// GetStrategyByID retrieves a strategy by ID with user access check
func (s *StrategyService) GetStrategyByID(ctx context.Context, strategyID int, userID int) (*model.Strategy, error) {
	// Get strategy with access control
	strategy, err := s.strategyRepo.GetStrategyByIDWithAccess(ctx, strategyID, userID)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found or you don't have access to it")
	}

	// Try to get username
	owner, err := s.userClient.GetUserByID(ctx, strategy.UserID)
	if err == nil {
		strategy.Username = owner
	} else {
		strategy.Username = fmt.Sprintf("User %d", strategy.UserID)
	}

	return strategy, nil
}

// validateStrategyData validates strategy data before creating or updating
func (s *StrategyService) validateStrategyData(data json.RawMessage) error {
	if len(data) == 0 {
		return errors.New("strategy structure cannot be empty")
	}

	// Try to parse the JSON to validate it
	var strategyData map[string]interface{}
	if err := json.Unmarshal(data, &strategyData); err != nil {
		return fmt.Errorf("invalid strategy structure JSON: %w", err)
	}

	return nil
}

// CreateStrategy creates a new strategy
func (s *StrategyService) CreateStrategy(ctx context.Context, strategy *model.StrategyCreate, userID int) (*model.Strategy, error) {
	// Validate strategy data
	if err := s.validateStrategyData(strategy.Structure); err != nil {
		return nil, err
	}

	// Validate tag IDs if provided
	if len(strategy.TagIDs) > 0 {
		// Verify all tag IDs exist
		for _, tagID := range strategy.TagIDs {
			tag, err := s.tagRepo.GetTagByID(ctx, tagID)
			if err != nil {
				return nil, fmt.Errorf("error verifying tag ID %d: %w", tagID, err)
			}
			if tag == nil {
				return nil, fmt.Errorf("tag with ID %d not found", tagID)
			}
		}
	}

	// Create strategy
	strategyID, err := s.strategyRepo.CreateStrategy(ctx, strategy, userID)
	if err != nil {
		return nil, err
	}

	// Fetch the created strategy
	createdStrategy, err := s.strategyRepo.GetStrategyByID(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	// Try to get username
	owner, err := s.userClient.GetUserByID(ctx, userID)
	if err == nil {
		createdStrategy.Username = owner
	} else {
		createdStrategy.Username = fmt.Sprintf("User %d", userID)
	}

	return createdStrategy, nil
}

// UpdateStrategy updates a strategy by creating a new version
func (s *StrategyService) UpdateStrategy(ctx context.Context, strategyID int, userID int, update *model.StrategyUpdate) (*model.Strategy, error) {
	// Validate strategy data
	if err := s.validateStrategyData(update.Structure); err != nil {
		return nil, err
	}

	// Verify strategy exists and user has ownership
	strategy, err := s.strategyRepo.GetStrategyByID(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return nil, errors.New("you don't have permission to update this strategy")
	}

	// Validate tag IDs if provided
	if len(update.TagIDs) > 0 {
		// Verify all tag IDs exist
		for _, tagID := range update.TagIDs {
			tag, err := s.tagRepo.GetTagByID(ctx, tagID)
			if err != nil {
				return nil, fmt.Errorf("error verifying tag ID %d: %w", tagID, err)
			}
			if tag == nil {
				return nil, fmt.Errorf("tag with ID %d not found", tagID)
			}
		}
	}

	// Update strategy (creates new version)
	newVersionID, err := s.strategyRepo.UpdateStrategy(ctx, strategyID, userID, update)
	if err != nil {
		return nil, err
	}

	// Fetch the updated strategy (new version)
	updatedStrategy, err := s.strategyRepo.GetStrategyByID(ctx, newVersionID)
	if err != nil {
		return nil, err
	}

	// Try to get username
	owner, err := s.userClient.GetUserByID(ctx, userID)
	if err == nil {
		updatedStrategy.Username = owner
	} else {
		updatedStrategy.Username = fmt.Sprintf("User %d", userID)
	}

	return updatedStrategy, nil
}

// DeleteStrategy marks a strategy as inactive
func (s *StrategyService) DeleteStrategy(ctx context.Context, strategyID int, userID int) error {
	// Verify strategy exists and user has ownership
	strategy, err := s.strategyRepo.GetStrategyByID(ctx, strategyID)
	if err != nil {
		return err
	}

	if strategy == nil {
		return errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return errors.New("you don't have permission to delete this strategy")
	}

	return s.strategyRepo.DeleteStrategy(ctx, strategyID, userID)
}

// GetVersions retrieves all versions of a strategy with pagination
func (s *StrategyService) GetVersions(
	ctx context.Context,
	strategyGroupID int,
	userID int,
	sortBy string,
	sortDirection string,
	page, limit int,
) ([]model.Strategy, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Validate sort parameters
	validSortFields := map[string]bool{
		"version":    true,
		"created_at": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "version" // Default sort by version
	}

	// Normalize sort direction
	sortDirection = strings.ToUpper(sortDirection)
	if sortDirection != "ASC" && sortDirection != "DESC" {
		sortDirection = "DESC" // Default to descending for versions
	}

	return s.strategyRepo.GetStrategyVersions(
		ctx,
		strategyGroupID,
		userID,
		sortBy,
		sortDirection,
		limit,
		offset,
	)
}

// GetVersionByID retrieves a specific version of a strategy
func (s *StrategyService) GetVersionByID(ctx context.Context, strategyID int, versionID int, userID int) (*model.Strategy, error) {
	// First verify the strategy exists
	strategy, err := s.strategyRepo.GetStrategyByID(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	// Get the specific version
	version, err := s.strategyRepo.GetStrategyByIDWithAccess(ctx, versionID, userID)
	if err != nil {
		return nil, err
	}

	if version == nil {
		return nil, errors.New("strategy version not found or you don't have access to it")
	}

	// Verify this version belongs to the strategy
	if version.StrategyGroupID != strategy.StrategyGroupID {
		return nil, errors.New("requested version does not belong to this strategy")
	}

	// Try to get username
	owner, err := s.userClient.GetUserByID(ctx, version.UserID)
	if err == nil {
		version.Username = owner
	} else {
		version.Username = fmt.Sprintf("User %d", version.UserID)
	}

	return version, nil
}

// SetActiveVersion sets a strategy version as the active one for a user
func (s *StrategyService) SetActiveVersion(ctx context.Context, userID, strategyGroupID, versionID int) error {
	// Verify strategy exists
	strategy, err := s.strategyRepo.GetStrategyByID(ctx, strategyGroupID)
	if err != nil {
		return err
	}

	if strategy == nil {
		return errors.New("strategy not found")
	}

	// Verify version exists and belongs to the strategy
	version, err := s.strategyRepo.GetStrategyByID(ctx, versionID)
	if err != nil {
		return err
	}

	if version == nil {
		return errors.New("version not found")
	}

	if version.StrategyGroupID != strategyGroupID {
		return errors.New("requested version does not belong to this strategy")
	}

	return s.strategyRepo.SetUserActiveVersion(ctx, userID, strategyGroupID, versionID)
}

// UpdateThumbnail updates a strategy's thumbnail URL
func (s *StrategyService) UpdateThumbnail(ctx context.Context, strategyID int, userID int, thumbnailURL string) error {
	// Verify strategy exists and user has ownership
	strategy, err := s.strategyRepo.GetStrategyByID(ctx, strategyID)
	if err != nil {
		return err
	}

	if strategy == nil {
		return errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return errors.New("you don't have permission to update this strategy")
	}

	return s.strategyRepo.UpdateThumbnail(ctx, strategyID, userID, thumbnailURL)
}

// CreateBacktest submits a backtest request to the historical data service
func (s *StrategyService) CreateBacktest(ctx context.Context, request *model.BacktestRequest, userID int) (int, error) {
	// Verify strategy exists and user has access to it
	strategy, err := s.strategyRepo.GetStrategyByIDWithAccess(ctx, request.StrategyID, userID)
	if err != nil {
		return 0, err
	}

	if strategy == nil {
		return 0, errors.New("strategy not found or you don't have access to it")
	}

	// Submit backtest request to historical data service
	backtestID, err := s.historicalClient.CreateBacktest(ctx, request, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to submit backtest: %w", err)
	}

	return backtestID, nil
}
