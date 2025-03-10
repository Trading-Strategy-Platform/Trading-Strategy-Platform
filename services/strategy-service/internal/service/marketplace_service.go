package service

import (
	"context"
	"errors"
	"time"

	"services/strategy-service/internal/model"
	"services/strategy-service/internal/repository"

	sharedErrors "github.com/yourorg/trading-platform/shared/go/errors"
	sharedModel "github.com/yourorg/trading-platform/shared/go/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// MarketplaceService handles marketplace-related operations
type MarketplaceService struct {
	db              *sqlx.DB
	marketplaceRepo *repository.MarketplaceRepository
	strategyRepo    *repository.StrategyRepository
	purchaseRepo    *repository.PurchaseRepository
	reviewRepo      *repository.ReviewRepository
	userClient      UserClient
	kafkaService    *KafkaService
	logger          *zap.Logger
}

// NewMarketplaceService creates a new MarketplaceService
func NewMarketplaceService(
	db *sqlx.DB,
	marketplaceRepo *repository.MarketplaceRepository,
	strategyRepo *repository.StrategyRepository,
	purchaseRepo *repository.PurchaseRepository,
	reviewRepo *repository.ReviewRepository,
	userClient UserClient,
	kafkaService *KafkaService,
	logger *zap.Logger,
) *MarketplaceService {
	return &MarketplaceService{
		db:              db,
		marketplaceRepo: marketplaceRepo,
		strategyRepo:    strategyRepo,
		purchaseRepo:    purchaseRepo,
		reviewRepo:      reviewRepo,
		userClient:      userClient,
		kafkaService:    kafkaService,
		logger:          logger,
	}
}

// Get subscription duration based on subscription period
func getSubscriptionDuration(period string) time.Duration {
	switch period {
	case "monthly":
		return 30 * 24 * time.Hour
	case "quarterly":
		return 90 * 24 * time.Hour
	case "yearly":
		return 365 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour // Default to monthly
	}
}

// CreateListing creates a new marketplace listing
func (s *MarketplaceService) CreateListing(ctx context.Context, listing *model.MarketplaceCreate, userID int) (*model.MarketplaceItem, error) {
	// Check if strategy exists and belongs to user
	strategy, err := s.strategyRepo.GetByID(ctx, listing.StrategyID)
	if err != nil {
		return nil, err
	}

	if strategy.UserID != userID {
		return nil, errors.New("you do not own this strategy")
	}

	// Create the listing
	listingID, err := s.marketplaceRepo.Create(ctx, listing, userID)
	if err != nil {
		return nil, err
	}

	// Get the created listing
	createdListing, err := s.marketplaceRepo.GetByID(ctx, listingID)
	if err != nil {
		return nil, err
	}

	// Publish marketplace listing event
	if s.kafkaService != nil {
		// Convert MarketplaceItem to MarketplaceListing for Kafka
		listingEvent := &model.MarketplaceListing{
			ID:                 createdListing.ID,
			StrategyID:         createdListing.StrategyID,
			CreatorID:          createdListing.UserID, // Map UserID to CreatorID
			Name:               createdListing.Name,
			Description:        createdListing.Description,
			Price:              createdListing.Price,
			IsSubscription:     createdListing.IsSubscription,
			SubscriptionPeriod: createdListing.SubscriptionPeriod,
		}

		if err := s.kafkaService.PublishMarketplaceListing(ctx, listingEvent); err != nil {
			s.logger.Warn("Failed to publish marketplace listing event",
				zap.Error(err),
				zap.Int("marketplace_id", createdListing.ID))
		}
	}

	return createdListing, nil
}

// PurchaseStrategy purchases a strategy
func (s *MarketplaceService) PurchaseStrategy(ctx context.Context, marketplaceID int, userID int) (*model.StrategyPurchase, error) {
	// Get the listing
	listing, err := s.marketplaceRepo.GetByID(ctx, marketplaceID)
	if err != nil {
		return nil, err
	}

	// Check if user is not the creator
	if listing.UserID == userID {
		return nil, errors.New("cannot purchase your own strategy")
	}

	// Check if user already purchased this strategy
	purchased, err := s.purchaseRepo.HasUserPurchased(ctx, marketplaceID, userID)
	if err != nil {
		return nil, err
	}

	if purchased {
		return nil, errors.New("you already purchased this strategy")
	}

	// Transaction to record purchase and grant access
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("starting transaction", err)
	}
	defer tx.Rollback()

	// Create purchase record
	purchaseID, err := s.purchaseRepo.CreateWithTx(ctx, tx, marketplaceID, userID, listing.Price, listing.IsSubscription, listing.SubscriptionPeriod)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("creating purchase", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, sharedErrors.NewDatabaseError("committing transaction", err)
	}

	// Create purchase object to return
	purchase := &model.StrategyPurchase{
		ID:            purchaseID,
		MarketplaceID: marketplaceID,
		BuyerID:       userID,
		PurchasePrice: listing.Price,
		CreatedAt:     time.Now(),
	}

	if listing.IsSubscription {
		endTime := time.Now().Add(getSubscriptionDuration(listing.SubscriptionPeriod))
		purchase.SubscriptionEnd = &endTime
	}

	// Publish purchase event
	if s.kafkaService != nil {
		if err := s.kafkaService.PublishPurchaseCreated(ctx, purchase); err != nil {
			s.logger.Warn("Failed to publish purchase event",
				zap.Error(err),
				zap.Int("purchase_id", purchaseID))
		}
	}

	return purchase, nil
}

// CreateReview creates a new strategy review
func (s *MarketplaceService) CreateReview(ctx context.Context, review *model.ReviewCreate, userID int) (*model.StrategyReview, error) {
	// Check if user purchased the strategy
	purchased, err := s.purchaseRepo.HasUserPurchased(ctx, review.MarketplaceID, userID)
	if err != nil {
		return nil, err
	}

	if !purchased {
		return nil, errors.New("you must purchase this strategy before reviewing it")
	}

	// Get user's name from user service
	userName, err := s.userClient.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get username, using anonymous", zap.Error(err))
		userName = "Anonymous"
	}

	// Create review
	reviewID, err := s.reviewRepo.Create(ctx, review, userID)
	if err != nil {
		return nil, sharedErrors.NewDatabaseError("creating review", err)
	}

	// Create review object to return
	createdReview := &model.StrategyReview{
		ID:            reviewID,
		MarketplaceID: review.MarketplaceID,
		UserID:        userID,
		Rating:        review.Rating,
		Comment:       review.Comment,
		CreatedAt:     time.Now(),
		UserName:      userName,
	}

	// Publish review event
	if s.kafkaService != nil {
		if err := s.kafkaService.PublishReviewCreated(ctx, createdReview); err != nil {
			s.logger.Warn("Failed to publish review event",
				zap.Error(err),
				zap.Int("review_id", reviewID))
		}
	}

	return createdReview, nil
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

// GetReviews retrieves reviews for a marketplace listing
func (s *MarketplaceService) GetReviews(ctx context.Context, marketplaceID int, pagination *sharedModel.Pagination) ([]model.StrategyReview, *sharedModel.PaginationMeta, error) {
	offset := pagination.GetOffset()
	limit := pagination.GetPerPage()

	// Get reviews
	reviews, total, err := s.reviewRepo.GetByMarketplaceID(ctx, marketplaceID, offset, limit)
	if err != nil {
		return nil, nil, err
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

	meta := sharedModel.NewPaginationMeta(pagination, total)
	return reviews, &meta, nil
}
