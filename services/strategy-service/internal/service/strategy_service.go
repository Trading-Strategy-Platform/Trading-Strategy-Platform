package service

import (
	"context"
	"errors"
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
	// Validate strategy structure
	if err := validator.ValidateStrategyStructure(&strategy.Structure); err != nil {
		return nil, err
	}

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

	// Validate strategy structure if provided
	if update.Structure != nil {
		if err := validator.ValidateStrategyStructure(update.Structure); err != nil {
			return nil, err
		}
	}

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

// GetAllListings retrieves marketplace listings using get_marketplace_strategies function
func (s *MarketplaceService) GetAllListings(ctx context.Context, searchTerm string, minPrice *float64, maxPrice *float64, isFree *bool, tags []int, minRating *float64, sortBy string, page, limit int) ([]model.MarketplaceItem, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Get listings using the get_marketplace_strategies function
	return s.marketplaceRepo.GetAll(ctx, searchTerm, minPrice, maxPrice, isFree, tags, minRating, sortBy, page, limit)
}

// CreateListing creates a new marketplace listing using add_to_marketplace function
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

	// Create listing using add_to_marketplace function
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

// DeleteListing deletes a marketplace listing using remove_from_marketplace function
func (s *MarketplaceService) DeleteListing(ctx context.Context, id int, userID int) error {
	// Check if listing exists
	listing, err := s.marketplaceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if listing == nil {
		return errors.New("listing not found")
	}

	// Check if strategy belongs to the user
	strategy, err := s.strategyRepo.GetByID(ctx, listing.StrategyID)
	if err != nil {
		return err
	}

	if strategy.UserID != userID {
		return errors.New("access denied")
	}

	// Delete listing using remove_from_marketplace function
	return s.marketplaceRepo.Delete(ctx, id, userID)
}

// PurchaseStrategy purchases a strategy from the marketplace using purchase_strategy function
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
	strategy, err := s.strategyRepo.GetByID(ctx, listing.StrategyID)
	if err != nil {
		return nil, err
	}

	if strategy.UserID == userID {
		return nil, errors.New("cannot purchase own strategy")
	}

	// Create purchase record using purchase_strategy function
	purchaseID, err := s.purchaseRepo.Purchase(ctx, marketplaceID, userID)
	if err != nil {
		return nil, err
	}

	// Return purchase info
	return &model.StrategyPurchase{
		ID:              purchaseID,
		MarketplaceID:   marketplaceID,
		BuyerID:         userID,
		PurchasePrice:   listing.Price,
		SubscriptionEnd: nil, // This would be set by the purchase_strategy function
		CreatedAt:       time.Now(),
	}, nil
}

// CancelSubscription cancels a subscription using cancel_subscription function
func (s *MarketplaceService) CancelSubscription(ctx context.Context, purchaseID int, userID int) error {
	return s.purchaseRepo.CancelSubscription(ctx, purchaseID, userID)
}

// GetReviews retrieves reviews for a marketplace listing using get_strategy_reviews function
func (s *MarketplaceService) GetReviews(ctx context.Context, marketplaceID int, page, limit int) ([]model.StrategyReview, int, error) {
	// Check if listing exists
	listing, err := s.marketplaceRepo.GetByID(ctx, marketplaceID)
	if err != nil {
		return nil, 0, err
	}

	if listing == nil {
		return nil, 0, errors.New("listing not found")
	}

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	// Get reviews with pagination
	return s.reviewRepo.GetByMarketplaceID(ctx, marketplaceID, page, limit)
}

// CreateReview creates a review for a purchased strategy using add_review function
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
	hasAccess, err := s.checkUserHasAccess(ctx, listing.StrategyID, userID)
	if err != nil || !hasAccess {
		return nil, errors.New("must purchase strategy before reviewing")
	}

	// Create review using add_review function
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

// checkUserHasAccess checks if a user has purchased a strategy
func (s *MarketplaceService) checkUserHasAccess(ctx context.Context, strategyID int, userID int) (bool, error) {
	strategies, _, err := s.strategyRepo.GetUserStrategies(ctx, userID, "", true, nil, 1, 100)
	if err != nil {
		return false, err
	}

	for _, strategy := range strategies {
		if strategy.ID == strategyID && strategy.AccessType == "purchased" {
			return true, nil
		}
	}

	return false, nil
}

// UpdateReview updates a review using edit_review function
func (s *MarketplaceService) UpdateReview(ctx context.Context, reviewID int, userID int, rating int, comment string) error {
	return s.reviewRepo.Update(ctx, reviewID, userID, rating, comment)
}

// DeleteReview deletes a review using delete_review function
func (s *MarketplaceService) DeleteReview(ctx context.Context, reviewID int, userID int) error {
	return s.reviewRepo.Delete(ctx, reviewID, userID)
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

// GetAllIndicators retrieves all technical indicators using get_indicators function
func (s *IndicatorService) GetAllIndicators(ctx context.Context, category *string, page, limit int) ([]model.TechnicalIndicator, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	return s.indicatorRepo.GetAll(ctx, category, page, limit)
}

// CreateIndicator creates a new technical indicator
func (s *IndicatorService) CreateIndicator(ctx context.Context, name, description, category, formula string) (*model.TechnicalIndicator, error) {
	// Here you would implement the logic to create a new indicator in the database
	// This is a placeholder implementation
	indicator := &model.TechnicalIndicator{
		Name:        name,
		Description: description,
		Category:    category,
		Formula:     formula,
		CreatedAt:   time.Now(),
	}

	// TODO: Implement actual database operation
	// For now, we'll return a mock response
	return indicator, nil
}

// AddParameter adds a parameter to an indicator
func (s *IndicatorService) AddParameter(
	ctx context.Context,
	indicatorID int,
	paramName string,
	paramType string,
	isRequired bool,
	minValue *float64,
	maxValue *float64,
	defaultValue string,
	description string,
) (*model.IndicatorParameter, error) {
	// Here you would implement the logic to add a parameter to an indicator
	// This is a placeholder implementation
	parameter := &model.IndicatorParameter{
		IndicatorID:   indicatorID,
		ParameterName: paramName,
		ParameterType: paramType,
		IsRequired:    isRequired,
		MinValue:      minValue,
		MaxValue:      maxValue,
		DefaultValue:  defaultValue,
		Description:   description,
	}

	// TODO: Implement actual database operation
	// For now, we'll return a mock response
	return parameter, nil
}

// AddEnumValue adds an enum value to a parameter
func (s *IndicatorService) AddEnumValue(
	ctx context.Context,
	parameterID int,
	enumValue string,
	displayName string,
) (*model.ParameterEnumValue, error) {
	// Here you would implement the logic to add an enum value to a parameter
	// This is a placeholder implementation
	paramEnum := &model.ParameterEnumValue{
		ParameterID: parameterID,
		EnumValue:   enumValue,
		DisplayName: displayName,
	}

	// TODO: Implement actual database operation
	// For now, we'll return a mock response
	return paramEnum, nil
}

// GetIndicator retrieves a specific indicator by ID using get_indicator_by_id function
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

// GetCategories retrieves indicator categories using get_indicator_categories function
func (s *IndicatorService) GetCategories(ctx context.Context) ([]struct {
	Category string `db:"category"`
	Count    int    `db:"count"`
}, error) {
	return s.indicatorRepo.GetCategories(ctx)
}
