package repository

import (
	"context"
	"database/sql"
	"errors"

	"services/strategy-service/internal/model"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// MarketplaceRepository handles database operations for the strategy marketplace
type MarketplaceRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewMarketplaceRepository creates a new marketplace repository
func NewMarketplaceRepository(db *sqlx.DB, logger *zap.Logger) *MarketplaceRepository {
	return &MarketplaceRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll retrieves marketplace listings with proper database-level pagination
func (r *MarketplaceRepository) GetAll(ctx context.Context, searchTerm string, minPrice *float64, maxPrice *float64, isFree *bool, tags []int, minRating *float64, sortBy string, page, limit int) ([]model.MarketplaceItem, int, error) {
	// Calculate offset from page and limit
	offset := (page - 1) * limit

	// Convert Go nil values to SQL NULL values where needed
	var minPriceSQL interface{} = sql.NullFloat64{Float64: 0, Valid: false}
	if minPrice != nil {
		minPriceSQL = *minPrice
	}

	var maxPriceSQL interface{} = sql.NullFloat64{Float64: 0, Valid: false}
	if maxPrice != nil {
		maxPriceSQL = *maxPrice
	}

	var minRatingSQL interface{} = sql.NullFloat64{Float64: 0, Valid: false}
	if minRating != nil {
		minRatingSQL = *minRating
	}

	// If sortBy is empty, use default 'popularity'
	if sortBy == "" {
		sortBy = "popularity"
	}

	// Use a zero-length array if tags is nil
	tagsParam := pq.Array(tags)
	if tags == nil {
		tagsParam = pq.Array([]int{})
	}

	// First, get total count using count_marketplace_strategies function
	countQuery := `SELECT count_marketplace_strategies($1, $2, $3, $4, $5, $6)`

	var total int
	err := r.db.GetContext(ctx, &total, countQuery,
		searchTerm,
		minPriceSQL,
		maxPriceSQL,
		isFree,
		tagsParam,
		minRatingSQL,
	)

	if err != nil {
		r.logger.Error("Failed to count marketplace listings", zap.Error(err))
		return nil, 0, err
	}

	// Now, get paginated data using get_marketplace_strategies function
	dataQuery := `SELECT * FROM get_marketplace_strategies($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	// Define a struct that matches the SQL function columns exactly
	type listing struct {
		ID                 int          `db:"id"`
		StrategyID         int          `db:"strategy_id"`
		Name               string       `db:"name"`
		DescriptionPublic  string       `db:"description_public"`
		ThumbnailURL       string       `db:"thumbnail_url"`
		UserID             int          `db:"user_id"`
		Price              float64      `db:"price"`
		IsSubscription     bool         `db:"is_subscription"`
		SubscriptionPeriod string       `db:"subscription_period"`
		IsActive           bool         `db:"is_active"`
		CreatedAt          sql.NullTime `db:"created_at"`
		UpdatedAt          sql.NullTime `db:"updated_at"`
		AverageRating      float64      `db:"average_rating"`
		ReviewsCount       int64        `db:"reviews_count"`
	}

	var listings []listing

	err = r.db.SelectContext(ctx, &listings, dataQuery,
		searchTerm,   // p_search_term
		minPriceSQL,  // p_min_price
		maxPriceSQL,  // p_max_price
		isFree,       // p_is_free
		tagsParam,    // p_tags
		minRatingSQL, // p_min_rating
		sortBy,       // p_sort_by
		limit,        // p_limit
		offset,       // p_offset
	)

	if err != nil {
		r.logger.Error("Failed to get marketplace listings", zap.Error(err))
		return nil, 0, err
	}

	// Convert to model.MarketplaceItem
	items := make([]model.MarketplaceItem, len(listings))
	for i, l := range listings {
		items[i] = model.MarketplaceItem{
			ID:                 l.ID,
			StrategyID:         l.StrategyID,
			Name:               l.Name,         // Include strategy name
			ThumbnailURL:       l.ThumbnailURL, // Include thumbnail URL
			UserID:             l.UserID,
			Price:              l.Price,
			IsSubscription:     l.IsSubscription,
			SubscriptionPeriod: l.SubscriptionPeriod,
			IsActive:           l.IsActive,
			DescriptionPublic:  l.DescriptionPublic,
			AverageRating:      l.AverageRating,
			ReviewsCount:       int(l.ReviewsCount),
		}

		// Set CreatedAt if valid
		if l.CreatedAt.Valid {
			items[i].CreatedAt = l.CreatedAt.Time
		}

		// Set UpdatedAt if valid
		if l.UpdatedAt.Valid {
			items[i].UpdatedAt = &l.UpdatedAt.Time
		}
	}

	return items, total, nil
}

// Create adds a new marketplace listing using add_to_marketplace function
func (r *MarketplaceRepository) Create(ctx context.Context, listing *model.MarketplaceCreate, userID int) (int, error) {
	query := `SELECT add_to_marketplace($1, $2, $3, $4, $5, $6, $7)`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		userID,
		listing.StrategyID,
		listing.VersionID,
		listing.Price,
		listing.IsSubscription,
		listing.SubscriptionPeriod,
		listing.DescriptionPublic,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create marketplace listing", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// GetByID retrieves a marketplace listing by ID
func (r *MarketplaceRepository) GetByID(ctx context.Context, id int) (*model.MarketplaceItem, error) {
	query := `
		SELECT id, strategy_id, user_id, price, is_subscription, subscription_period, 
		       is_active, description_public, created_at, updated_at
		FROM strategy_marketplace
		WHERE id = $1
	`

	var item model.MarketplaceItem
	err := r.db.GetContext(ctx, &item, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get marketplace item", zap.Error(err))
		return nil, err
	}

	return &item, nil
}

// Delete removes a marketplace listing using remove_from_marketplace function
func (r *MarketplaceRepository) Delete(ctx context.Context, id int, userID int) error {
	query := `SELECT remove_from_marketplace($1, $2)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, userID, id).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to delete marketplace item", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to delete marketplace item or not authorized")
	}

	return nil
}
