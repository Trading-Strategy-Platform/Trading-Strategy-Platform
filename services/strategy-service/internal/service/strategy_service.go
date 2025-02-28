// services/strategy-service/internal/service/strategy_service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"services/strategy-service/internal/model"
	"services/strategy-service/internal/repository"
	"services/strategy-service/internal/validator"

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

// CreateStrategy creates a new strategy
func (s *StrategyService) CreateStrategy(ctx context.Context, strategy *model.StrategyCreate, userID int) (*model.Strategy, error) {
	// Validate strategy structure
	if err := validator.ValidateStrategyStructure(&strategy.Structure); err != nil {
		return nil, err
	}

	// Start transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, err
	}
	defer tx.Rollback()

	// Create strategy
	strategyID, err := s.strategyRepo.Create(ctx, tx, strategy, userID)
	if err != nil {
		return nil, err
	}

	// Create initial version
	versionCreate := &model.VersionCreate{
		Structure:   strategy.Structure,
		ChangeNotes: "Initial version",
	}
	_, err = s.versionRepo.Create(ctx, tx, versionCreate, strategyID, 1)
	if err != nil {
		return nil, err
	}

	// Update tags if provided
	if len(strategy.Tags) > 0 {
		err = s.strategyRepo.UpdateTags(ctx, tx, strategyID, strategy.Tags)
		if err != nil {
			return nil, err
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
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
func (s *StrategyService) GetStrategy(ctx context.Context, id int, userID int) (*model.StrategyResponse, error) {
	// Get strategy
	strategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	// Check if user has access to the strategy
	if !strategy.IsPublic && strategy.UserID != userID {
		return nil, errors.New("access denied")
	}

	// Get creator username
	creatorName, err := s.userClient.GetUserByID(ctx, strategy.UserID)
	if err != nil {
		s.logger.Warn("Failed to get creator name", zap.Error(err))
		creatorName = "Unknown"
	}

	// Get versions count
	versions, err := s.versionRepo.GetVersions(ctx, id)
	if err != nil {
		s.logger.Warn("Failed to get versions", zap.Error(err))
	}

	// Create response
	response := &model.StrategyResponse{
		Strategy:      *strategy,
		VersionsCount: len(versions),
		CreatorName:   creatorName,
	}

	return response, nil
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
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return nil, errors.New("access denied")
	}

	// Validate strategy structure if provided
	if update.Structure != nil {
		if err := validator.ValidateStrategyStructure(update.Structure); err != nil {
			return nil, err
		}
	}

	// Start transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, err
	}
	defer tx.Rollback()

	// Update strategy
	err = s.strategyRepo.Update(ctx, tx, id, update)
	if err != nil {
		return nil, err
	}

	// Create new version if structure was updated
	if update.Structure != nil {
		// Get latest version
		latestVersion, err := s.versionRepo.GetLatestVersion(ctx, id)
		if err != nil {
			return nil, err
		}

		// Create new version
		versionCreate := &model.VersionCreate{
			Structure:   *update.Structure,
			ChangeNotes: "Updated strategy structure",
		}
		_, err = s.versionRepo.Create(ctx, tx, versionCreate, id, latestVersion+1)
		if err != nil {
			return nil, err
		}
	}

	// Update tags if provided
	if update.Tags != nil {
		err = s.strategyRepo.UpdateTags(ctx, tx, id, update.Tags)
		if err != nil {
			return nil, err
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return nil, err
	}

	// Get the updated strategy
	updatedStrategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
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

// MarketplaceService handles marketplace operations
type MarketplaceService struct {
	db              *sqlx.DB
	marketplaceRepo *repository.MarketplaceRepository
	strategyRepo    *repository.StrategyRepository
	purchaseRepo    *repository.PurchaseRepository
	reviewRepo      *repository.ReviewRepository
	userClient      UserClient
	logger          *zap.Logger
}

// NewMarketplaceService creates a new marketplace service
func NewMarketplaceService(
	db *sqlx.DB,
	marketplaceRepo *repository.MarketplaceRepository,
	strategyRepo *repository.StrategyRepository,
	purchaseRepo *repository.PurchaseRepository,
	reviewRepo *repository.ReviewRepository,
	userClient UserClient,
	logger *zap.Logger,
) *MarketplaceService {
	return &MarketplaceService{
		db:              db,
		marketplaceRepo: marketplaceRepo,
		strategyRepo:    strategyRepo,
		purchaseRepo:    purchaseRepo,
		reviewRepo:      reviewRepo,
		userClient:      userClient,
		logger:          logger,
	}
}

// CreateListing creates a new marketplace listing
func (s *MarketplaceService) CreateListing(ctx context.Context, listing *model.MarketplaceCreate, userID int) (*model.MarketplaceItem, error) {
	// Check if strategy exists and belongs to the user
	strategy, err := s.strategyRepo.GetByID(ctx, listing.StrategyID)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return nil, errors.New("access denied")
	}

	// Create listing
	id, err := s.marketplaceRepo.Create(ctx, listing, userID)
	if err != nil {
		return nil, err
	}

	// Get created listing
	createdListing, err := s.marketplaceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return createdListing, nil
}

// GetListing retrieves a marketplace listing by ID
func (s *MarketplaceService) GetListing(ctx context.Context, id int) (*model.MarketplaceItem, error) {
	// Get listing
	listing, err := s.marketplaceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if listing == nil {
		return nil, errors.New("listing not found")
	}

	// Get associated strategy
	strategy, err := s.strategyRepo.GetByID(ctx, listing.StrategyID)
	if err != nil {
		s.logger.Warn("Failed to get strategy for listing", zap.Error(err))
	} else {
		listing.Strategy = strategy
	}

	// Get creator name
	creatorName, err := s.userClient.GetUserByID(ctx, listing.UserID)
	if err != nil {
		s.logger.Warn("Failed to get creator name", zap.Error(err))
		creatorName = "Unknown"
	}
	listing.CreatorName = creatorName

	// Get average rating
	avgRating, err := s.reviewRepo.GetAverageRating(ctx, id)
	if err != nil {
		s.logger.Warn("Failed to get average rating", zap.Error(err))
	} else {
		listing.AverageRating = avgRating
	}

	// Get reviews count
	reviews, _, err := s.reviewRepo.GetByMarketplaceID(ctx, id, 1, 1)
	if err != nil {
		s.logger.Warn("Failed to get reviews count", zap.Error(err))
	} else {
		listing.ReviewsCount = len(reviews)
	}

	return listing, nil
}

// GetAllListings retrieves marketplace listings
func (s *MarketplaceService) GetAllListings(ctx context.Context, isActive *bool, userID *int, page, limit int) ([]model.MarketplaceItem, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Get listings
	listings, total, err := s.marketplaceRepo.GetAll(ctx, isActive, userID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	// Enrich listings with additional data
	for i, listing := range listings {
		// Get creator name
		creatorName, err := s.userClient.GetUserByID(ctx, listing.UserID)
		if err != nil {
			s.logger.Warn("Failed to get creator name", zap.Error(err))
			listings[i].CreatorName = "Unknown"
		} else {
			listings[i].CreatorName = creatorName
		}

		// Get average rating
		avgRating, err := s.reviewRepo.GetAverageRating(ctx, listing.ID)
		if err != nil {
			s.logger.Warn("Failed to get average rating", zap.Error(err))
		} else {
			listings[i].AverageRating = avgRating
		}
	}

	return listings, total, nil
}

// UpdateListing updates a marketplace listing
func (s *MarketplaceService) UpdateListing(ctx context.Context, id int, price *float64, isActive *bool, description *string, userID int) (*model.MarketplaceItem, error) {
	// Check if listing exists and belongs to the user
	listing, err := s.marketplaceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if listing == nil {
		return nil, errors.New("listing not found")
	}

	if listing.UserID != userID {
		return nil, errors.New("access denied")
	}

	// Update listing
	err = s.marketplaceRepo.Update(ctx, id, price, isActive, description)
	if err != nil {
		return nil, err
	}

	// Get updated listing
	updatedListing, err := s.marketplaceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return updatedListing, nil
}

// DeleteListing deletes a marketplace listing
func (s *MarketplaceService) DeleteListing(ctx context.Context, id int, userID int) error {
	// Check if listing exists and belongs to the user
	listing, err := s.marketplaceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if listing == nil {
		return errors.New("listing not found")
	}

	if listing.UserID != userID {
		return errors.New("access denied")
	}

	// Delete listing
	return s.marketplaceRepo.Delete(ctx, id)
}

// PurchaseStrategy purchases a strategy from the marketplace
func (s *MarketplaceService) PurchaseStrategy(ctx context.Context, marketplaceID int, userID int) (*model.StrategyPurchase, error) {
	// Get listing
	listing, err := s.marketplaceRepo.GetByID(ctx, marketplaceID)
	if err != nil {
		return nil, err
	}

	if listing == nil {
		return nil, errors.New("listing not found")
	}

	if !listing.IsActive {
		return nil, errors.New("listing is not active")
	}

	// Cannot purchase own strategy
	if listing.UserID == userID {
		return nil, errors.New("cannot purchase own strategy")
	}

	// Check if already purchased
	hasPurchased, err := s.purchaseRepo.HasPurchased(ctx, userID, listing.StrategyID)
	if err != nil {
		return nil, err
	}

	if hasPurchased {
		return nil, errors.New("strategy already purchased")
	}

	// Calculate subscription end date if needed
	var subscriptionEnd *time.Time
	if listing.IsSubscription {
		end := time.Now()
		switch listing.SubscriptionPeriod {
		case "monthly":
			end = end.AddDate(0, 1, 0)
		case "quarterly":
			end = end.AddDate(0, 3, 0)
		case "yearly":
			end = end.AddDate(1, 0, 0)
		default:
			// Default to monthly
			end = end.AddDate(0, 1, 0)
		}
		subscriptionEnd = &end
	}

	// Create purchase record
	purchaseID, err := s.purchaseRepo.Create(ctx, marketplaceID, userID, listing.Price, subscriptionEnd)
	if err != nil {
		return nil, err
	}

	// Return purchase info
	return &model.StrategyPurchase{
		ID:              purchaseID,
		MarketplaceID:   marketplaceID,
		BuyerID:         userID,
		PurchasePrice:   listing.Price,
		SubscriptionEnd: subscriptionEnd,
		CreatedAt:       time.Now(),
	}, nil
}

// GetPurchases retrieves purchases for a user
func (s *MarketplaceService) GetPurchases(ctx context.Context, userID int, page, limit int) ([]model.StrategyPurchase, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Get purchases
	return s.purchaseRepo.GetByUser(ctx, userID, page, limit)
}

// CreateReview creates a review for a purchased strategy
func (s *MarketplaceService) CreateReview(ctx context.Context, review *model.ReviewCreate, userID int) (*model.StrategyReview, error) {
	// Check if listing exists
	listing, err := s.marketplaceRepo.GetByID(ctx, review.MarketplaceID)
	if err != nil {
		return nil, err
	}

	if listing == nil {
		return nil, errors.New("listing not found")
	}

	// Check if user has purchased the strategy
	hasPurchased, err := s.purchaseRepo.HasPurchased(ctx, userID, listing.StrategyID)
	if err != nil {
		return nil, err
	}

	if !hasPurchased {
		return nil, errors.New("must purchase strategy before reviewing")
	}

	// Check if user has already reviewed this listing
	hasReviewed, err := s.reviewRepo.HasReviewed(ctx, userID, review.MarketplaceID)
	if err != nil {
		return nil, err
	}

	if hasReviewed {
		return nil, errors.New("already reviewed this strategy")
	}

	// Create review
	reviewID, err := s.reviewRepo.Create(ctx, review, userID)
	if err != nil {
		return nil, err
	}

	// Get user name
	userName, err := s.userClient.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get user name", zap.Error(err))
		userName = "Anonymous"
	}

	// Return created review
	return &model.StrategyReview{
		ID:            reviewID,
		MarketplaceID: review.MarketplaceID,
		UserID:        userID,
		Rating:        review.Rating,
		Comment:       review.Comment,
		CreatedAt:     time.Now(),
		UserName:      userName,
	}, nil
}

// GetReviews retrieves reviews for a marketplace listing
func (s *MarketplaceService) GetReviews(ctx context.Context, marketplaceID int, page, limit int) ([]model.StrategyReview, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Get reviews
	reviews, total, err := s.reviewRepo.GetByMarketplaceID(ctx, marketplaceID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	// Get user names for each review
	for i, review := range reviews {
		userName, err := s.userClient.GetUserByID(ctx, review.UserID)
		if err != nil {
			s.logger.Warn("Failed to get user name", zap.Error(err))
			reviews[i].UserName = "Anonymous"
		} else {
			reviews[i].UserName = userName
		}
	}

	return reviews, total, nil
}
