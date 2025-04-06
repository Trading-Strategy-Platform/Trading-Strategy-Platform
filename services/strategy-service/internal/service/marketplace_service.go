package service

import (
	"context"
	"errors"
	"time"

	"services/strategy-service/internal/model"
	"services/strategy-service/internal/repository"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

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
