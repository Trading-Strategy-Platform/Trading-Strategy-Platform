package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"services/strategy-service/internal/client"
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

// GetAllListings retrieves marketplace listings with enhanced filtering, sorting, and pagination
func (s *MarketplaceService) GetAllListings(
	ctx context.Context,
	searchTerm string,
	minPrice *float64,
	maxPrice *float64,
	isFree *bool,
	tags []int,
	minRating *float64,
	sortBy string,
	sortDirection string,
	page, limit int,
) ([]model.MarketplaceItem, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Get listings from repository with all filtering and sorting parameters
	items, total, err := s.marketplaceRepo.GetAllListings(
		ctx,
		searchTerm,
		minPrice,
		maxPrice,
		isFree,
		tags,
		minRating,
		sortBy,
		sortDirection,
		page,
		limit,
	)
	if err != nil {
		return nil, 0, err
	}

	// If no items, return early
	if len(items) == 0 {
		return items, total, nil
	}

	// Extract unique user IDs from listings
	userIDs := make([]int, 0)
	userIDSet := make(map[int]bool)

	for _, item := range items {
		if !userIDSet[item.UserID] {
			userIDSet[item.UserID] = true
			userIDs = append(userIDs, item.UserID)
		}
	}

	// Try to batch fetch user details - use a separate context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	userDetails := make(map[int]client.UserDetails)
	userDetailsErr := ""

	// Try to get user details, but don't fail if this doesn't work
	if fetchedDetails, err := s.userClient.BatchGetUsersByIDs(timeoutCtx, userIDs); err != nil {
		userDetailsErr = err.Error()
		s.logger.Warn("Failed to get user details from user service",
			zap.Error(err),
			zap.Ints("user_ids", userIDs))
	} else {
		userDetails = fetchedDetails
	}

	// Enhance listings with user details (or fallbacks)
	for i := range items {
		userDetail, exists := userDetails[items[i].UserID]
		if exists {
			items[i].CreatorName = userDetail.Username
			items[i].CreatorPhotoURL = userDetail.ProfilePhotoURL
		} else {
			// Fallback if user details not found
			items[i].CreatorName = fmt.Sprintf("User %d", items[i].UserID)
			items[i].CreatorPhotoURL = ""
		}
	}

	// Add debug info to the first item if there was an error
	if len(items) > 0 && userDetailsErr != "" {
		s.logger.Debug("Including user service error in debug_info",
			zap.String("error", userDetailsErr))
	}

	return items, total, nil
}

// CreateListing creates a new marketplace listing
func (s *MarketplaceService) CreateListing(ctx context.Context, listing *model.MarketplaceCreate, userID int) (*model.MarketplaceItem, error) {
	// Check if strategy exists and belongs to the user
	strategy, err := s.strategyRepo.GetStrategyByID(ctx, listing.StrategyID)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return nil, errors.New("access denied: you can only list strategies you own")
	}

	// Create listing using create_marketplace_listing function
	id, err := s.marketplaceRepo.CreateListing(ctx, listing, userID)
	if err != nil {
		return nil, err
	}

	// Get created listing
	createdListing, err := s.marketplaceRepo.GetListingByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Enhance with strategy details
	createdListing.Name = strategy.Name
	createdListing.ThumbnailURL = strategy.ThumbnailURL

	// Try to get creator username
	username, err := s.userClient.GetUserByID(ctx, userID)
	if err == nil {
		createdListing.CreatorName = username
	} else {
		createdListing.CreatorName = fmt.Sprintf("User %d", userID)
	}

	return createdListing, nil
}

// GetListingByID retrieves a marketplace listing by ID with detailed information
func (s *MarketplaceService) GetListingByID(ctx context.Context, id int) (*model.MarketplaceItem, error) {
	// Get listing
	listing, err := s.marketplaceRepo.GetListingByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if listing == nil {
		return nil, errors.New("listing not found")
	}

	// Get strategy details to enhance the listing
	strategy, err := s.strategyRepo.GetStrategyByID(ctx, listing.StrategyID)
	if err != nil {
		s.logger.Warn("Failed to get strategy details for listing",
			zap.Error(err),
			zap.Int("strategyID", listing.StrategyID))
	} else if strategy != nil {
		listing.Name = strategy.Name
		listing.ThumbnailURL = strategy.ThumbnailURL
	}

	// Try to get creator name
	username, err := s.userClient.GetUserByID(ctx, listing.UserID)
	if err == nil {
		listing.CreatorName = username
	} else {
		listing.CreatorName = fmt.Sprintf("User %d", listing.UserID)
	}

	// Get rating information
	reviews, _, err := s.reviewRepo.GetByMarketplaceID(ctx, id, nil, 1, 1)
	if err == nil {
		reviewCount := len(reviews)
		avgRating := 0.0
		listing.ReviewsCount = reviewCount

		if reviewCount > 0 {
			totalRating := 0
			for _, r := range reviews {
				totalRating += r.Rating
			}
			avgRating = float64(totalRating) / float64(reviewCount)
			listing.AverageRating = avgRating
		}
	}

	return listing, nil
}

// DeleteListing deletes a marketplace listing
func (s *MarketplaceService) DeleteListing(ctx context.Context, id int, userID int) error {
	// Check if listing exists
	listing, err := s.marketplaceRepo.GetListingByID(ctx, id)
	if err != nil {
		return err
	}

	if listing == nil {
		return errors.New("listing not found")
	}

	// Check if strategy belongs to the user
	strategy, err := s.strategyRepo.GetStrategyByID(ctx, listing.StrategyID)
	if err != nil {
		return err
	}

	if strategy == nil {
		return errors.New("strategy not found for this listing")
	}

	if strategy.UserID != userID {
		return errors.New("access denied: you can only delete your own listings")
	}

	// Delete listing
	return s.marketplaceRepo.DeleteListing(ctx, id, userID)
}

// PurchaseStrategy purchases a strategy from the marketplace
func (s *MarketplaceService) PurchaseStrategy(ctx context.Context, marketplaceID int, userID int) (*model.StrategyPurchase, error) {
	// Get listing
	listing, err := s.marketplaceRepo.GetListingByID(ctx, marketplaceID)
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
	strategy, err := s.strategyRepo.GetStrategyByID(ctx, listing.StrategyID)
	if err != nil {
		return nil, err
	}

	if strategy == nil {
		return nil, errors.New("strategy not found")
	}

	if strategy.UserID == userID {
		return nil, errors.New("cannot purchase your own strategy")
	}

	// Create purchase record
	purchaseID, err := s.purchaseRepo.Purchase(ctx, marketplaceID, userID)
	if err != nil {
		return nil, err
	}

	// Calculate subscription end date if applicable
	var subscriptionEnd *time.Time
	if listing.IsSubscription {
		endDate := time.Now()
		switch listing.SubscriptionPeriod {
		case "monthly":
			endDate = endDate.AddDate(0, 1, 0)
		case "quarterly":
			endDate = endDate.AddDate(0, 3, 0)
		case "yearly":
			endDate = endDate.AddDate(1, 0, 0)
		}
		subscriptionEnd = &endDate
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

// CancelSubscription cancels a subscription
func (s *MarketplaceService) CancelSubscription(ctx context.Context, purchaseID int, userID int) error {
	return s.purchaseRepo.CancelSubscription(ctx, purchaseID, userID)
}

// GetReviews retrieves reviews for a marketplace listing
func (s *MarketplaceService) GetReviews(
	ctx context.Context,
	marketplaceID int,
	minRating *float64,
	page,
	limit int,
) ([]model.StrategyReview, int, error) {
	// Check if listing exists
	listing, err := s.marketplaceRepo.GetListingByID(ctx, marketplaceID)
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

	// Get reviews with pagination and minimum rating filter
	reviews, total, err := s.reviewRepo.GetByMarketplaceID(
		ctx,
		marketplaceID,
		minRating,
		page,
		limit,
	)
	if err != nil {
		return nil, 0, err
	}

	// Enhance reviews with usernames
	userIDs := make([]int, 0, len(reviews))
	for _, r := range reviews {
		userIDs = append(userIDs, r.UserID)
	}

	// Try batch fetch of usernames if there are reviews
	if len(userIDs) > 0 {
		userDetails, err := s.userClient.BatchGetUsersByIDs(ctx, userIDs)
		if err != nil {
			s.logger.Warn("Failed to batch get user details", zap.Error(err))
			// Continue, we'll use fallback names
		} else {
			// Set usernames in reviews
			for i, review := range reviews {
				if details, ok := userDetails[review.UserID]; ok {
					reviews[i].UserName = details.Username
				} else {
					reviews[i].UserName = fmt.Sprintf("User %d", review.UserID)
				}
			}
		}
	}

	return reviews, total, nil
}

// CreateReview creates a review for a purchased strategy
func (s *MarketplaceService) CreateReview(ctx context.Context, review *model.ReviewCreate, userID int) (*model.StrategyReview, error) {
	// Check if listing exists
	listing, err := s.marketplaceRepo.GetListingByID(ctx, review.MarketplaceID)
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

	// Create review
	reviewID, err := s.reviewRepo.Create(ctx, review, userID)
	if err != nil {
		return nil, err
	}

	// Get user name
	userName, err := s.userClient.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get user name", zap.Error(err))
		userName = fmt.Sprintf("User %d", userID)
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

// updateReview updates a review
func (s *MarketplaceService) UpdateReview(ctx context.Context, reviewID int, userID int, rating int, comment string) error {
	return s.reviewRepo.Update(ctx, reviewID, userID, rating, comment)
}

// DeleteReview deletes a review
func (s *MarketplaceService) DeleteReview(ctx context.Context, reviewID int, userID int) error {
	return s.reviewRepo.Delete(ctx, reviewID, userID)
}

// Helper: checkUserHasAccess checks if a user has purchased a strategy
func (s *MarketplaceService) checkUserHasAccess(ctx context.Context, strategyID int, userID int) (bool, error) {
	strategies, _, err := s.strategyRepo.GetAllStrategies(ctx, userID, "", true, nil, "created_at", "DESC", 1, 100)
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
