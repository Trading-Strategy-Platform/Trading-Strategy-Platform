// services/strategy-service/internal/service/strategy_service.go
package service

import (
	"context"
	"errors"
	"fmt"

	"services/strategy-service/internal/client"
	"services/strategy-service/internal/model"
	"services/strategy-service/internal/repository"
	"services/strategy-service/internal/validator"

	sharedErrors "github.com/yourorg/trading-platform/shared/go/errors"
	sharedModel "github.com/yourorg/trading-platform/shared/go/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// StrategyService handles strategy operations
type StrategyService struct {
	db             *sqlx.DB
	strategyRepo   *repository.StrategyRepository
	versionRepo    *repository.VersionRepository
	tagRepo        *repository.TagRepository
	userClient     UserClient
	backtestClient BacktestClient
	validator      *client.ValidatorClient
	kafkaService   *KafkaService
	logger         *zap.Logger
}

// UserClient defines methods for interacting with the User Service
type UserClient interface {
	GetUserByID(ctx context.Context, userID int) (string, error) // Returns username
	ValidateUserAccess(ctx context.Context, userID int, token string) (bool, error)
}

// BacktestClient defines methods for interacting with the Historical Data Service
type BacktestClient interface {
	CreateBacktest(ctx context.Context, request *model.BacktestRequest, userID int) (int, error)
}

// NewStrategyService creates a new strategy service
func NewStrategyService(
	db *sqlx.DB,
	strategyRepo *repository.StrategyRepository,
	versionRepo *repository.VersionRepository,
	tagRepo *repository.TagRepository,
	userClient UserClient,
	backtestClient BacktestClient,
	validator *client.ValidatorClient,
	kafkaService *KafkaService,
	logger *zap.Logger,
) *StrategyService {
	return &StrategyService{
		db:             db,
		strategyRepo:   strategyRepo,
		versionRepo:    versionRepo,
		tagRepo:        tagRepo,
		userClient:     userClient,
		backtestClient: backtestClient,
		validator:      validator,
		kafkaService:   kafkaService,
		logger:         logger,
	}
}

// CreateStrategy creates a new strategy
func (s *StrategyService) CreateStrategy(ctx context.Context, strategy *model.StrategyCreate, userID int) (*model.Strategy, error) {
	// Validate the strategy using shared validator
	if err := s.validator.Validate(strategy); err != nil {
		return nil, err
	}

	// Specific validation for strategy structure
	if err := validator.ValidateStrategyStructure(&strategy.Structure); err != nil {
		return nil, err
	}

	// Start transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, sharedErrors.NewDatabaseError("beginning transaction", err)
	}
	defer tx.Rollback()

	// Create strategy
	strategyID, err := s.strategyRepo.Create(ctx, tx, strategy, userID)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("creating strategy", err)
	}

	// Create initial version
	versionCreate := &model.VersionCreate{
		Structure:   strategy.Structure,
		ChangeNotes: "Initial version",
	}
	_, err = s.versionRepo.Create(ctx, tx, versionCreate, strategyID, 1)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("creating strategy version", err)
	}

	// Update tags if provided
	if len(strategy.Tags) > 0 {
		err = s.strategyRepo.UpdateTags(ctx, tx, strategyID, strategy.Tags)
		if err != nil {
			return nil, sharedErrors.NewDatabaseError("updating strategy tags", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return nil, sharedErrors.NewDatabaseError("committing transaction", err)
	}

	// Get the newly created strategy
	createdStrategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("retrieving created strategy", err)
	}

	// Publish strategy created event
	if s.kafkaService != nil {
		if err := s.kafkaService.PublishStrategyCreated(ctx, createdStrategy); err != nil {
			s.logger.Warn("Failed to publish strategy created event",
				zap.Error(err),
				zap.Int("strategy_id", createdStrategy.ID))
		}
	}

	// Get username for author_name field - we don't use it directly but log if there's an error
	_, err = s.userClient.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get author username", zap.Error(err))
		// Continue without author name, it's not critical
	}

	return createdStrategy, nil
}

// GetStrategy retrieves a strategy by ID
func (s *StrategyService) GetStrategy(ctx context.Context, id int, userID int) (*model.StrategyResponse, error) {
	// Get strategy
	strategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("retrieving strategy", err)
	}
	if strategy == nil {
		return nil, sharedErrors.NewNotFoundError("Strategy", string(id))
	}

	// Check if user has access to this strategy
	if strategy.UserID != userID && !strategy.IsPublic {
		return nil, sharedErrors.NewPermissionError("You don't have permission to view this strategy")
	}

	// Get tags
	tags, err := s.tagRepo.GetForStrategy(ctx, id)
	if err != nil {
		s.logger.Warn("Failed to get strategy tags", zap.Error(err), zap.Int("strategy_id", id))
	}
	strategy.Tags = tags

	// Get author name
	userName, err := s.userClient.GetUserByID(ctx, strategy.UserID)
	if err != nil {
		s.logger.Warn("Failed to get author username", zap.Error(err), zap.Int("user_id", strategy.UserID))
		userName = "Unknown"
	}

	return &model.StrategyResponse{
		Strategy:       *strategy,
		AuthorUsername: userName,
	}, nil
}

// GetUserStrategies retrieves strategies for a user
func (s *StrategyService) GetUserStrategies(ctx context.Context, userID int, requestingUserID int, isPublic *bool, tagID *int, page, limit int) ([]model.Strategy, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Check if requesting user has access to view non-public strategies
	if isPublic == nil && userID != requestingUserID {
		isPublicValue := true
		isPublic = &isPublicValue
	}

	// Get strategies
	strategies, total, err := s.strategyRepo.GetByUserID(ctx, userID, isPublic, tagID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	return strategies, total, nil
}

// GetPublicStrategies retrieves public strategies
func (s *StrategyService) GetPublicStrategies(ctx context.Context, tagID *int, page, limit int) ([]model.Strategy, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Get public strategies
	strategies, total, err := s.strategyRepo.GetPublicStrategies(ctx, tagID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	return strategies, total, nil
}

// UpdateStrategy updates a strategy
func (s *StrategyService) UpdateStrategy(ctx context.Context, id int, update *model.StrategyUpdate, userID int) (*model.Strategy, error) {
	// Check if strategy exists and belongs to the user
	strategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("checking strategy existence", err)
	}
	if strategy == nil {
		return nil, sharedErrors.NewNotFoundError("Strategy", fmt.Sprintf("%d", id))
	}
	if strategy.UserID != userID {
		return nil, sharedErrors.NewPermissionError("You do not have permission to update this strategy")
	}

	// Validate structure if provided
	if update.Structure != nil {
		if err := validator.ValidateStrategyStructure(update.Structure); err != nil {
			return nil, err
		}
	}

	// Start transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, sharedErrors.NewDatabaseError("beginning transaction", err)
	}
	defer tx.Rollback()

	// Update strategy
	err = s.strategyRepo.Update(ctx, tx, id, update)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("updating strategy", err)
	}

	// Create new version if structure changed
	if update.Structure != nil {
		// Increment version number
		newVersion := strategy.Version + 1

		// Create new version record
		versionCreate := &model.VersionCreate{
			Structure:   *update.Structure,
			ChangeNotes: update.Notes, // Use Notes field instead of ChangeNotes
		}
		_, err = s.versionRepo.Create(ctx, tx, versionCreate, id, newVersion)
		if err != nil {
			return nil, sharedErrors.NewDatabaseError("creating strategy version", err)
		}

		// Update strategy version number
		err = s.strategyRepo.UpdateStrategyVersion(ctx, tx, id, newVersion) // Corrected method name
		if err != nil {
			return nil, sharedErrors.NewDatabaseError("updating strategy version", err)
		}
	}

	// Update tags if provided
	if update.Tags != nil {
		err = s.strategyRepo.UpdateTags(ctx, tx, id, update.Tags) // Tags is already the right type
		if err != nil {
			return nil, sharedErrors.NewDatabaseError("updating strategy tags", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return nil, sharedErrors.NewDatabaseError("committing transaction", err)
	}

	// Get updated strategy
	updatedStrategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("retrieving updated strategy", err)
	}

	return updatedStrategy, nil
}

// DeleteStrategy deletes a strategy
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

	// Delete strategy
	return s.strategyRepo.Delete(ctx, id)
}

// CreateVersion creates a new version of a strategy
func (s *StrategyService) CreateVersion(ctx context.Context, strategyID int, version *model.VersionCreate, userID int) (*model.StrategyVersion, error) {
	// Check if strategy exists and belongs to the user
	strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return nil, errors.New("access denied")
	}

	// Validate version structure
	if err := validator.ValidateStrategyStructure(&version.Structure); err != nil {
		return nil, err
	}

	// Start transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, err
	}
	defer tx.Rollback()

	// Get latest version
	latestVersion, err := s.versionRepo.GetLatestVersion(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	// Create new version
	versionID, err := s.versionRepo.Create(ctx, tx, version, strategyID, latestVersion+1)
	if err != nil {
		return nil, err
	}

	// Update strategy with new structure and version
	update := &model.StrategyUpdate{
		Structure: &version.Structure,
	}
	err = s.strategyRepo.Update(ctx, tx, strategyID, update)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return nil, err
	}

	// Get versions
	versions, err := s.versionRepo.GetVersions(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	// Find the created version
	for _, v := range versions {
		if v.ID == versionID {
			return &v, nil
		}
	}

	return nil, errors.New("failed to retrieve created version")
}

// GetVersions retrieves versions of a strategy
func (s *StrategyService) GetVersions(ctx context.Context, strategyID int, userID int) ([]model.StrategyVersion, error) {
	// Check if strategy exists and user has access
	strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	if !strategy.IsPublic && strategy.UserID != userID {
		return nil, errors.New("access denied")
	}

	// Get versions
	versions, err := s.versionRepo.GetVersions(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	return versions, nil
}

// GetVersion retrieves a specific version of a strategy
func (s *StrategyService) GetVersion(ctx context.Context, strategyID int, versionNumber int, userID int) (*model.StrategyVersion, error) {
	// Check if strategy exists and user has access
	strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	if !strategy.IsPublic && strategy.UserID != userID {
		return nil, errors.New("access denied")
	}

	// Get version
	version, err := s.versionRepo.GetVersion(ctx, strategyID, versionNumber)
	if err != nil {
		return nil, err
	}

	if version == nil {
		return nil, errors.New("version not found")
	}

	return version, nil
}

// RestoreVersion restores a strategy to a previous version
func (s *StrategyService) RestoreVersion(ctx context.Context, strategyID int, versionNumber int, userID int) (*model.Strategy, error) {
	// Check if strategy exists and belongs to the user
	strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return nil, errors.New("access denied")
	}

	// Get version to restore
	version, err := s.versionRepo.GetVersion(ctx, strategyID, versionNumber)
	if err != nil {
		return nil, err
	}

	if version == nil {
		return nil, errors.New("version not found")
	}

	// Start transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, err
	}
	defer tx.Rollback()

	// Get latest version
	latestVersion, err := s.versionRepo.GetLatestVersion(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	// Update strategy with restored structure
	update := &model.StrategyUpdate{
		Structure: &version.Structure,
	}
	err = s.strategyRepo.Update(ctx, tx, strategyID, update)
	if err != nil {
		return nil, err
	}

	// Create new version based on the restored one
	versionCreate := &model.VersionCreate{
		Structure:   version.Structure,
		ChangeNotes: fmt.Sprintf("Restored from version %d", versionNumber),
	}
	_, err = s.versionRepo.Create(ctx, tx, versionCreate, strategyID, latestVersion+1)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return nil, err
	}

	// Get the updated strategy
	updatedStrategy, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return nil, err
	}

	return updatedStrategy, nil
}

// CloneStrategy creates a copy of a strategy for the current user
func (s *StrategyService) CloneStrategy(ctx context.Context, sourceID int, userID int, name string) (*model.Strategy, error) {
	// Check if source strategy exists and user has access
	sourceStrategy, err := s.strategyRepo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, err
	}

	if sourceStrategy == nil {
		return nil, errors.New("strategy not found")
	}

	if !sourceStrategy.IsPublic && sourceStrategy.UserID != userID {
		return nil, errors.New("access denied to source strategy")
	}

	// Create new strategy based on source
	newStrategy := &model.StrategyCreate{
		Name:        name,
		Description: "Cloned from: " + sourceStrategy.Name,
		Structure:   sourceStrategy.Structure,
		IsPublic:    false, // Default to private for cloned strategies
	}

	// Clone tags
	for _, tag := range sourceStrategy.Tags {
		newStrategy.Tags = append(newStrategy.Tags, tag.ID)
	}

	// Create the cloned strategy
	clonedStrategy, err := s.CreateStrategy(ctx, newStrategy, userID)
	if err != nil {
		return nil, err
	}

	return clonedStrategy, nil
}

// StartBacktest initiates a backtest for a strategy
func (s *StrategyService) StartBacktest(ctx context.Context, request *model.BacktestRequest, userID int) (int, error) {
	// Check if strategy exists and user has access
	strategy, err := s.strategyRepo.GetByID(ctx, request.StrategyID)
	if err != nil {
		return 0, err
	}

	if strategy == nil {
		return 0, errors.New("strategy not found")
	}

	if !strategy.IsPublic && strategy.UserID != userID {
		return 0, errors.New("access denied")
	}

	// Send backtest request to Historical Data Service
	backtestID, err := s.backtestClient.CreateBacktest(ctx, request, userID)
	if err != nil {
		return 0, err
	}

	// Publish backtest created event
	if s.kafkaService != nil {
		if err := s.kafkaService.PublishBacktestCreated(ctx, request, backtestID, userID); err != nil {
			s.logger.Warn("Failed to publish backtest created event",
				zap.Error(err),
				zap.Int("backtest_id", backtestID))
		}
	}

	return backtestID, nil
}

// VersionService handles strategy version operations
type VersionService struct {
	versionRepo *repository.VersionRepository
	logger      *zap.Logger
}

// NewVersionService creates a new version service
func NewVersionService(versionRepo *repository.VersionRepository, logger *zap.Logger) *VersionService {
	return &VersionService{
		versionRepo: versionRepo,
		logger:      logger,
	}
}

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

// GetAllTags retrieves all tags
func (s *TagService) GetAllTags(ctx context.Context) ([]model.Tag, error) {
	return s.tagRepo.GetAll(ctx)
}

// CreateTag creates a new tag
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

// IndicatorService handles technical indicator operations
type IndicatorService struct {
	indicatorRepo *repository.IndicatorRepository
	logger        *zap.Logger
}

// NewIndicatorService creates a new indicator service
func NewIndicatorService(indicatorRepo *repository.IndicatorRepository, logger *zap.Logger) *IndicatorService {
	return &IndicatorService{
		indicatorRepo: indicatorRepo,
		logger:        logger,
	}
}

// GetAllIndicators retrieves all technical indicators
func (s *IndicatorService) GetAllIndicators(ctx context.Context, category *string) ([]model.TechnicalIndicator, error) {
	return s.indicatorRepo.GetAll(ctx, category)
}

// GetIndicator retrieves a specific indicator by ID
func (s *IndicatorService) GetIndicator(ctx context.Context, id int) (*model.TechnicalIndicator, error) {
	indicator, err := s.indicatorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if indicator == nil {
		return nil, errors.New("indicator not found")
	}

	return indicator, nil
}

// ListStrategies lists strategies for a user with filtering and pagination
func (s *StrategyService) ListStrategies(ctx context.Context, userID int, nameFilter string, pagination *sharedModel.Pagination) ([]model.Strategy, *sharedModel.PaginationMeta, error) {
	offset := pagination.GetOffset()
	limit := pagination.GetPerPage()

	strategies, total, err := s.strategyRepo.ListByUserID(ctx, userID, nameFilter, limit, offset)
	if err != nil {
		return nil, nil, err
	}

	// Get tags for each strategy
	for i, strategy := range strategies {
		tags, err := s.tagRepo.GetForStrategy(ctx, strategy.ID)
		if err != nil {
			s.logger.Warn("Failed to get tags for strategy", zap.Error(err))
			continue
		}
		strategies[i].Tags = tags
	}

	meta := sharedModel.NewPaginationMeta(pagination, total)
	return strategies, &meta, nil
}

// RunBacktest updates the RunBacktest method to use kafkaService
func (s *StrategyService) RunBacktest(ctx context.Context, backtestRequest *model.BacktestRequest, userID int) (int, error) {
	// Existing validation code...

	backtestID, err := s.backtestClient.CreateBacktest(ctx, backtestRequest, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to create backtest: %w", err)
	}

	// Publish backtest created event
	if s.kafkaService != nil {
		if err := s.kafkaService.PublishBacktestCreated(ctx, backtestRequest, backtestID, userID); err != nil {
			s.logger.Warn("Failed to publish backtest created event",
				zap.Error(err),
				zap.Int("backtest_id", backtestID))
		}
	}

	return backtestID, nil
}
