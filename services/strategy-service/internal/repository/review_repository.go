package repository

import (
	"context"
	"errors"
	"time"

	"services/strategy-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

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

// GetByMarketplaceID retrieves reviews for a marketplace listing with proper database-level pagination
func (r *ReviewRepository) GetByMarketplaceID(ctx context.Context, marketplaceID int, page, limit int) ([]model.StrategyReview, int, error) {
	// First, count total reviews for this marketplace listing
	countQuery := `
		SELECT COUNT(*)
		FROM strategy_reviews
		WHERE marketplace_id = $1
	`

	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, marketplaceID)
	if err != nil {
		r.logger.Error("Failed to count reviews", zap.Error(err))
		return nil, 0, err
	}

	// Calculate offset for pagination
	offset := (page - 1) * limit

	// Now, use the get_strategy_reviews function with LIMIT and OFFSET directly in the SQL
	query := `SELECT * FROM get_strategy_reviews($1, $2, $3)`

	// Execute query
	var reviews []struct {
		ReviewID  int       `db:"review_id"`
		UserID    int       `db:"user_id"`
		Rating    int       `db:"rating"`
		Comment   string    `db:"comment"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	// Using limit and offset for pagination
	err = r.db.SelectContext(ctx, &reviews, query, marketplaceID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get reviews", zap.Error(err))
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
		}
	}

	return result, totalCount, nil
}
