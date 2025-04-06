package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"services/strategy-service/internal/model"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// StrategyRepository handles database operations for strategies
type StrategyRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewStrategyRepository creates a new strategy repository
func NewStrategyRepository(db *sqlx.DB, logger *zap.Logger) *StrategyRepository {
	return &StrategyRepository{
		db:     db,
		logger: logger,
	}
}

// GetUserStrategies retrieves strategies using the get_my_strategies function
func (r *StrategyRepository) GetUserStrategies(ctx context.Context, userID int, searchTerm string, purchasedOnly bool, tags []int, page, limit int) ([]model.ExtendedStrategy, int, error) {
	// First, get all strategies using the get_my_strategies function
	query := `SELECT * FROM get_my_strategies($1, $2, $3, $4)`

	var allStrategies []model.ExtendedStrategy

	// Ensure tags is initialized to empty array if nil
	tagsParam := pq.Array(tags)
	if tags == nil {
		tagsParam = pq.Array([]int{})
	}

	err := r.db.SelectContext(ctx, &allStrategies, query,
		userID,
		searchTerm,
		purchasedOnly,
		tagsParam,
	)

	if err != nil {
		r.logger.Error("Failed to get user strategies", zap.Error(err))
		return nil, 0, err
	}

	// Get total count before pagination
	total := len(allStrategies)

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		return []model.ExtendedStrategy{}, total, nil
	}

	if end > total {
		end = total
	}

	paginatedStrategies := allStrategies[start:end]

	// Fetch tags for each strategy in the paginated results
	for i, strategy := range paginatedStrategies {
		if len(strategy.TagIDs) > 0 {
			tags, err := r.getTagsByIDs(ctx, strategy.TagIDs)
			if err != nil {
				r.logger.Warn("Failed to get strategy tags",
					zap.Error(err),
					zap.Int("strategy_id", strategy.ID),
				)
			} else {
				paginatedStrategies[i].Tags = tags
			}
		}
	}

	return paginatedStrategies, total, nil
}

// Create adds a new strategy using add_strategy function
func (r *StrategyRepository) Create(ctx context.Context, strategy *model.StrategyCreate, userID int) (int, error) {
	query := `SELECT add_strategy($1, $2, $3, $4, $5, $6, $7)`

	// Convert strategy structure to JSON
	structureBytes, err := json.Marshal(strategy.Structure)
	if err != nil {
		r.logger.Error("Failed to marshal strategy structure", zap.Error(err))
		return 0, err
	}

	var id int
	err = r.db.QueryRowContext(
		ctx,
		query,
		userID,                  // p_user_id
		strategy.Name,           // p_name
		strategy.Description,    // p_description
		strategy.ThumbnailURL,   // p_thumbnail_url
		structureBytes,          // p_structure
		strategy.IsPublic,       // p_is_public
		pq.Array(strategy.Tags), // p_tag_ids
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create strategy", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// GetByID retrieves a strategy by ID
func (r *StrategyRepository) GetByID(ctx context.Context, id int) (*model.Strategy, error) {
	query := `
		SELECT id, name, user_id, description, thumbnail_url, structure, is_public, is_active, version, created_at, updated_at
		FROM strategies
		WHERE id = $1
	`

	var strategy model.Strategy
	var structureBytes []byte

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&strategy.ID,
		&strategy.Name,
		&strategy.UserID,
		&strategy.Description,
		&strategy.ThumbnailURL,
		&structureBytes,
		&strategy.IsPublic,
		&strategy.IsActive,
		&strategy.Version,
		&strategy.CreatedAt,
		&strategy.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get strategy by ID", zap.Error(err), zap.Int("strategy_id", id))
		return nil, err
	}

	// Unmarshal the strategy structure
	if err := json.Unmarshal(structureBytes, &strategy.Structure); err != nil {
		r.logger.Error("Failed to unmarshal strategy structure", zap.Error(err))
		return nil, err
	}

	// Get tags for the strategy
	tags, err := r.getTagsByIDs(ctx, r.getStrategyTagIDs(ctx, id))
	if err != nil {
		r.logger.Warn("Failed to get strategy tags", zap.Error(err), zap.Int("strategy_id", id))
	} else {
		strategy.Tags = tags
	}

	return &strategy, nil
}

// Update updates a strategy using update_strategy function
func (r *StrategyRepository) Update(ctx context.Context, id int, update *model.StrategyUpdate, userID int) error {
	query := `SELECT update_strategy($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	// Handle nil Structure case
	var structureBytes []byte
	if update.Structure != nil {
		var err error
		structureBytes, err = json.Marshal(*update.Structure)
		if err != nil {
			r.logger.Error("Failed to marshal strategy structure", zap.Error(err))
			return err
		}
	}

	// Default values for nullable fields
	name := ""
	if update.Name != nil {
		name = *update.Name
	}

	description := ""
	if update.Description != nil {
		description = *update.Description
	}

	thumbnailURL := ""
	if update.ThumbnailURL != nil {
		thumbnailURL = *update.ThumbnailURL
	}

	isPublic := false
	if update.IsPublic != nil {
		isPublic = *update.IsPublic
	}

	// Generate change notes based on what's being updated
	changeNotes := "Updated strategy"
	if update.Structure != nil {
		changeNotes = "Updated strategy structure"
	}

	var success bool
	err := r.db.QueryRowContext(
		ctx,
		query,
		id,                    // p_strategy_id
		userID,                // p_user_id
		name,                  // p_name
		description,           // p_description
		thumbnailURL,          // p_thumbnail_url
		structureBytes,        // p_structure
		isPublic,              // p_is_public
		changeNotes,           // p_change_notes
		pq.Array(update.Tags), // p_tag_ids
	).Scan(&success)

	if err != nil {
		r.logger.Error("Failed to update strategy", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to update strategy or not authorized")
	}

	return nil
}

// Delete marks a strategy as inactive using delete_strategy function
func (r *StrategyRepository) Delete(ctx context.Context, id int, userID int) error {
	query := `SELECT delete_strategy($1, $2)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to delete strategy", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to delete strategy or not authorized")
	}

	return nil
}

// getStrategyTagIDs retrieves tag IDs for a strategy
func (r *StrategyRepository) getStrategyTagIDs(ctx context.Context, strategyID int) []int {
	query := `
		SELECT tag_id
		FROM strategy_tag_mappings
		WHERE strategy_id = $1
	`

	var tagIDs []int
	err := r.db.SelectContext(ctx, &tagIDs, query, strategyID)
	if err != nil {
		r.logger.Warn("Failed to get strategy tag IDs", zap.Error(err), zap.Int("strategy_id", strategyID))
		return []int{}
	}

	return tagIDs
}

// getTagsByIDs retrieves tags by their IDs
func (r *StrategyRepository) getTagsByIDs(ctx context.Context, tagIDs []int) ([]model.Tag, error) {
	if len(tagIDs) == 0 {
		return []model.Tag{}, nil
	}

	query := `
		SELECT id, name
		FROM strategy_tags
		WHERE id = ANY($1)
	`

	var tags []model.Tag
	err := r.db.SelectContext(ctx, &tags, query, pq.Array(tagIDs))
	if err != nil {
		return nil, err
	}

	return tags, nil
}

// VersionRepository handles database operations for strategy versions
type VersionRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewVersionRepository creates a new version repository
func NewVersionRepository(db *sqlx.DB, logger *zap.Logger) *VersionRepository {
	return &VersionRepository{
		db:     db,
		logger: logger,
	}
}

// GetVersions retrieves all versions for a strategy using get_accessible_strategy_versions function
func (r *VersionRepository) GetVersions(ctx context.Context, strategyID int, userID int, page, limit int) ([]model.StrategyVersion, int, error) {
	// First get accessible versions without pagination
	query := `SELECT * FROM get_accessible_strategy_versions($1, $2)`

	var versionResults []struct {
		Version         int       `db:"version"`
		ChangeNotes     string    `db:"change_notes"`
		CreatedAt       time.Time `db:"created_at"`
		IsActiveVersion bool      `db:"is_active_version"`
	}

	err := r.db.SelectContext(ctx, &versionResults, query, userID, strategyID)
	if err != nil {
		r.logger.Error("Failed to query strategy versions", zap.Error(err))
		return nil, 0, err
	}

	// Get all version details
	allVersions := []model.StrategyVersion{}
	for _, v := range versionResults {
		// Get full version details including structure
		fullVersion, err := r.GetVersion(ctx, strategyID, v.Version)
		if err != nil {
			r.logger.Error("Failed to get version details", zap.Error(err))
			continue
		}
		if fullVersion != nil {
			allVersions = append(allVersions, *fullVersion)
		}
	}

	// Get total count before pagination
	total := len(allVersions)

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		return []model.StrategyVersion{}, total, nil
	}

	if end > total {
		end = total
	}

	return allVersions[start:end], total, nil
}

// GetVersion retrieves a specific version of a strategy
func (r *VersionRepository) GetVersion(ctx context.Context, strategyID int, versionNumber int) (*model.StrategyVersion, error) {
	query := `
		SELECT id, strategy_id, version, structure, change_notes, created_at
		FROM strategy_versions
		WHERE strategy_id = $1 AND version = $2
	`

	var version model.StrategyVersion
	var structureBytes []byte

	row := r.db.QueryRowContext(ctx, query, strategyID, versionNumber)
	err := row.Scan(
		&version.ID,
		&version.StrategyID,
		&version.Version,
		&structureBytes,
		&version.ChangeNotes,
		&version.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get strategy version", zap.Error(err))
		return nil, err
	}

	// Unmarshal the structure
	if err := json.Unmarshal(structureBytes, &version.Structure); err != nil {
		r.logger.Error("Failed to unmarshal version structure", zap.Error(err))
		return nil, err
	}

	return &version, nil
}

// UpdateUserVersion updates the active version for a user using update_user_strategy_version function
func (r *VersionRepository) UpdateUserVersion(ctx context.Context, userID int, strategyID int, version int) error {
	query := `SELECT update_user_strategy_version($1, $2, $3)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, userID, strategyID, version).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to update user strategy version", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to update active version or not authorized")
	}

	return nil
}

// TagRepository handles database operations for strategy tags
type TagRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewTagRepository creates a new tag repository
func NewTagRepository(db *sqlx.DB, logger *zap.Logger) *TagRepository {
	return &TagRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll retrieves all tags using get_strategy_tags function
func (r *TagRepository) GetAll(ctx context.Context, page, limit int) ([]model.Tag, int, error) {
	query := `SELECT * FROM get_strategy_tags()`

	var allTags []model.Tag
	err := r.db.SelectContext(ctx, &allTags, query)
	if err != nil {
		r.logger.Error("Failed to get tags", zap.Error(err))
		return nil, 0, err
	}

	// Get total count before pagination
	total := len(allTags)

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		return []model.Tag{}, total, nil
	}

	if end > total {
		end = total
	}

	return allTags[start:end], total, nil
}

// Create adds a new tag using add_strategy_tag function
func (r *TagRepository) Create(ctx context.Context, name string) (int, error) {
	query := `SELECT add_strategy_tag($1)`

	var id int
	err := r.db.QueryRowContext(ctx, query, name).Scan(&id)
	if err != nil {
		r.logger.Error("Failed to create tag", zap.Error(err))
		return 0, err
	}

	return id, nil
}

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

// GetAll retrieves marketplace listings using get_marketplace_strategies function
func (r *MarketplaceRepository) GetAll(ctx context.Context, searchTerm string, minPrice *float64, maxPrice *float64, isFree *bool, tags []int, minRating *float64, sortBy string, page, limit int) ([]model.MarketplaceItem, int, error) {
	// Calculate offset from page and limit
	offset := (page - 1) * limit

	// Prepare query to the get_marketplace_strategies function
	query := `SELECT * FROM get_marketplace_strategies($1, $2, $3, $4, $5, $6, $7, $8, $9)`

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

	// Execute query to get listings data
	var listings []struct {
		MarketplaceID      int       `db:"marketplace_id"`
		StrategyID         int       `db:"strategy_id"`
		Name               string    `db:"name"`
		Description        string    `db:"description"`
		ThumbnailURL       string    `db:"thumbnail_url"`
		OwnerID            int       `db:"owner_id"`
		OwnerUsername      string    `db:"owner_username"`
		OwnerPhoto         string    `db:"owner_photo"`
		VersionID          int       `db:"version_id"`
		Price              float64   `db:"price"`
		IsSubscription     bool      `db:"is_subscription"`
		SubscriptionPeriod string    `db:"subscription_period"`
		CreatedAt          time.Time `db:"created_at"`
		UpdatedAt          time.Time `db:"updated_at"`
		AvgRating          float64   `db:"avg_rating"`
		RatingCount        int       `db:"rating_count"`
		Tags               []string  `db:"tags"`
		TagIDs             []int     `db:"tag_ids"`
	}

	err := r.db.SelectContext(ctx, &listings, query,
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
			ID:                 l.MarketplaceID,
			StrategyID:         l.StrategyID,
			UserID:             l.OwnerID,
			Price:              l.Price,
			IsSubscription:     l.IsSubscription,
			SubscriptionPeriod: l.SubscriptionPeriod,
			IsActive:           true, // Assume active since the function only returns active listings
			Description:        l.Description,
			CreatedAt:          l.CreatedAt,
			UpdatedAt:          &l.UpdatedAt,

			// Additional fields from function result
			CreatorName:   l.OwnerUsername,
			AverageRating: l.AvgRating,
			ReviewsCount:  l.RatingCount,
		}
	}

	// Get total count - this could be optimized to return from the function directly
	// For now, we'll use the length of returned items if it's less than the requested limit,
	// otherwise execute a count query
	total := len(items)
	if limit > 0 && total == limit {
		// If we got exactly the number of items requested, there might be more
		countQuery := `
            SELECT COUNT(*) 
            FROM v_marketplace_strategies 
            WHERE 1=1`

		var params []interface{}
		if searchTerm != "" {
			countQuery += ` AND (name ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')`
			params = append(params, searchTerm)
		}

		err = r.db.GetContext(ctx, &total, countQuery, params...)
		if err != nil {
			r.logger.Warn("Failed to get total count, using result length", zap.Error(err))
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
		listing.Description,
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
		SELECT id, strategy_id, user_id, price, is_subscription, subscription_period, is_active, description, created_at, updated_at
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

// PurchaseRepository handles database operations for strategy purchases
type PurchaseRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewPurchaseRepository creates a new purchase repository
func NewPurchaseRepository(db *sqlx.DB, logger *zap.Logger) *PurchaseRepository {
	return &PurchaseRepository{
		db:     db,
		logger: logger,
	}
}

// Create adds a new purchase record using purchase_strategy function
func (r *PurchaseRepository) Purchase(ctx context.Context, marketplaceID int, userID int) (int, error) {
	query := `SELECT purchase_strategy($1, $2)`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		userID,
		marketplaceID,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to purchase strategy", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// CancelSubscription cancels a subscription using cancel_subscription function
func (r *PurchaseRepository) CancelSubscription(ctx context.Context, purchaseID int, userID int) error {
	query := `SELECT cancel_subscription($1, $2)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, userID, purchaseID).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to cancel subscription", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to cancel subscription or not authorized")
	}

	return nil
}

// ReviewRepository handles database operations for strategy reviews
type ReviewRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewReviewRepository creates a new review repository
func NewReviewRepository(db *sqlx.DB, logger *zap.Logger) *ReviewRepository {
	return &ReviewRepository{
		db:     db,
		logger: logger,
	}
}

// Create adds a new review using add_review function
func (r *ReviewRepository) Create(ctx context.Context, review *model.ReviewCreate, userID int) (int, error) {
	query := `SELECT add_review($1, $2, $3, $4)`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		userID,
		review.MarketplaceID,
		review.Rating,
		review.Comment,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create review", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// Update updates a review using edit_review function
func (r *ReviewRepository) Update(ctx context.Context, id int, userID int, rating int, comment string) error {
	query := `SELECT edit_review($1, $2, $3, $4)`

	var success bool
	err := r.db.QueryRowContext(
		ctx,
		query,
		userID,
		id,
		rating,
		comment,
	).Scan(&success)

	if err != nil {
		r.logger.Error("Failed to update review", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to update review or not authorized")
	}

	return nil
}

// Delete deletes a review using delete_review function
func (r *ReviewRepository) Delete(ctx context.Context, id int, userID int) error {
	query := `SELECT delete_review($1, $2)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, userID, id).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to delete review", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to delete review or not authorized")
	}

	return nil
}

// GetByMarketplaceID retrieves reviews for a marketplace listing using get_strategy_reviews function
func (r *ReviewRepository) GetByMarketplaceID(ctx context.Context, marketplaceID int, page, limit int) ([]model.StrategyReview, int, error) {
	query := `SELECT * FROM get_strategy_reviews($1, $2, $3)`

	// Calculate offset
	offset := (page - 1) * limit

	// Execute query
	var reviews []struct {
		ReviewID  int       `db:"review_id"`
		UserID    int       `db:"user_id"`
		Username  string    `db:"username"`
		UserPhoto string    `db:"user_photo"`
		Rating    int       `db:"rating"`
		Comment   string    `db:"comment"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	// Using limit and offset for pagination
	err := r.db.SelectContext(ctx, &reviews, query, marketplaceID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get reviews", zap.Error(err))
		return nil, 0, err
	}

	// Count total reviews separately, because the SQL function doesn't return it
	countQuery := `
		SELECT COUNT(*)
		FROM strategy_reviews
		WHERE marketplace_id = $1
	`
	var total int
	err = r.db.GetContext(ctx, &total, countQuery, marketplaceID)
	if err != nil {
		r.logger.Error("Failed to count reviews", zap.Error(err))
		return nil, 0, err
	}

	// Convert to model.StrategyReview
	result := make([]model.StrategyReview, len(reviews))
	for i, r := range reviews {
		result[i] = model.StrategyReview{
			ID:            r.ReviewID,
			MarketplaceID: marketplaceID,
			UserID:        r.UserID,
			Rating:        r.Rating,
			Comment:       r.Comment,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     &r.UpdatedAt,
			UserName:      r.Username,
		}
	}

	return result, total, nil
}

// IndicatorRepository handles database operations for technical indicators
type IndicatorRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewIndicatorRepository creates a new indicator repository
func NewIndicatorRepository(db *sqlx.DB, logger *zap.Logger) *IndicatorRepository {
	return &IndicatorRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll retrieves all indicators with optional category filter
func (r *IndicatorRepository) GetAll(ctx context.Context, category *string, page, limit int) ([]model.TechnicalIndicator, int, error) {
	var query string
	var args []interface{}

	// Build query with explicit column selection
	if category != nil {
		query = `
			SELECT id, name, description, category, formula, created_at, updated_at 
			FROM get_indicators($1::VARCHAR)
		`
		args = append(args, *category)
	} else {
		query = `
			SELECT id, name, description, category, formula, created_at, updated_at 
			FROM get_indicators(NULL::VARCHAR)
		`
	}

	// Execute query using QueryContext instead of SelectContext
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to execute get indicators query", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	// Manually scan rows into slices
	var allIndicators []model.TechnicalIndicator
	for rows.Next() {
		var indicator model.TechnicalIndicator
		var updatedAt sql.NullTime

		// Scan row data into variables
		err := rows.Scan(
			&indicator.ID,
			&indicator.Name,
			&indicator.Description,
			&indicator.Category,
			&indicator.Formula,
			&indicator.CreatedAt,
			&updatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan indicator row", zap.Error(err))
			return nil, 0, err
		}

		// Convert nullable time
		if updatedAt.Valid {
			indicator.UpdatedAt = &updatedAt.Time
		}

		allIndicators = append(allIndicators, indicator)
	}

	// Check for any errors encountered during iteration
	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating indicator rows", zap.Error(err))
		return nil, 0, err
	}

	// Get total count
	total := len(allIndicators)

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		return []model.TechnicalIndicator{}, total, nil
	}

	if end > total {
		end = total
	}

	return allIndicators[start:end], total, nil
}

// GetByID retrieves an indicator by ID
func (r *IndicatorRepository) GetByID(ctx context.Context, id int) (*model.TechnicalIndicator, error) {
	// Use explicit column selection to avoid the parameters column
	query := `
		SELECT id, name, description, category, formula, created_at, updated_at
		FROM get_indicator_by_id($1)
	`

	// Execute query
	var indicator model.TechnicalIndicator
	var updatedAt sql.NullTime

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&indicator.ID,
		&indicator.Name,
		&indicator.Description,
		&indicator.Category,
		&indicator.Formula,
		&indicator.CreatedAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get indicator by ID", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	// Convert nullable time
	if updatedAt.Valid {
		indicator.UpdatedAt = &updatedAt.Time
	}

	return &indicator, nil
}

// GetCategories retrieves indicator categories using get_indicator_categories function
func (r *IndicatorRepository) GetCategories(ctx context.Context) ([]struct {
	Category string `db:"category"`
	Count    int    `db:"count"`
}, error) {
	query := `SELECT * FROM get_indicator_categories()`

	var categories []struct {
		Category string `db:"category"`
		Count    int    `db:"count"`
	}
	err := r.db.SelectContext(ctx, &categories, query)
	if err != nil {
		r.logger.Error("Failed to get indicator categories", zap.Error(err))
		return nil, err
	}

	return categories, nil
}

// Create adds a new indicator to the database
func (r *IndicatorRepository) Create(ctx context.Context, indicator *model.TechnicalIndicator) (int, error) {
	query := `
		INSERT INTO indicators (name, description, category, formula, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		indicator.Name,
		indicator.Description,
		indicator.Category,
		indicator.Formula,
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create indicator", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// AddParameter adds a parameter to an indicator
func (r *IndicatorRepository) AddParameter(ctx context.Context, parameterCreate *model.IndicatorParameterCreate) (int, error) {
	query := `
		INSERT INTO indicator_parameters (
			indicator_id, parameter_name, parameter_type, is_required, 
			min_value, max_value, default_value, description
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		parameterCreate.IndicatorID,
		parameterCreate.ParameterName,
		parameterCreate.ParameterType,
		parameterCreate.IsRequired,
		parameterCreate.MinValue,
		parameterCreate.MaxValue,
		parameterCreate.DefaultValue,
		parameterCreate.Description,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to add parameter", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// AddEnumValue adds an enum value to a parameter
func (r *IndicatorRepository) AddEnumValue(ctx context.Context, enumCreate *model.ParameterEnumValueCreate) (int, error) {
	query := `
		INSERT INTO parameter_enum_values (parameter_id, enum_value, display_name)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		enumCreate.ParameterID,
		enumCreate.EnumValue,
		enumCreate.DisplayName,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to add enum value", zap.Error(err))
		return 0, err
	}

	return id, nil
}
