package repository

import (
	"context"
	"time"

	"services/strategy-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// ReviewRepository handles operations related to strategy reviews
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

// Create adds a new review
func (r *ReviewRepository) Create(ctx context.Context, review *model.ReviewCreate, userID int) (int, error) {
	query := `
		INSERT INTO strategy_reviews (marketplace_id, user_id, rating, comment, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		review.MarketplaceID,
		userID,
		review.Rating,
		review.Comment,
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create review",
			zap.Error(err),
			zap.Int("marketplace_id", review.MarketplaceID),
			zap.Int("user_id", userID))
		return 0, err
	}

	return id, nil
}

// GetByID retrieves a review by its ID
func (r *ReviewRepository) GetByID(ctx context.Context, id int) (*model.StrategyReview, error) {
	query := `
		SELECT id, marketplace_id, user_id, rating, comment, created_at, updated_at
		FROM strategy_reviews
		WHERE id = $1
	`

	var review model.StrategyReview
	err := r.db.GetContext(ctx, &review, query, id)
	if err != nil {
		r.logger.Error("Failed to get review by ID", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &review, nil
}

// GetByMarketplaceID retrieves reviews for a marketplace listing
func (r *ReviewRepository) GetByMarketplaceID(ctx context.Context, marketplaceID, offset, limit int) ([]model.StrategyReview, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM strategy_reviews WHERE marketplace_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, marketplaceID); err != nil {
		r.logger.Error("Failed to count reviews",
			zap.Error(err),
			zap.Int("marketplace_id", marketplaceID))
		return nil, 0, err
	}

	// Get reviews
	query := `
		SELECT id, marketplace_id, user_id, rating, comment, created_at, updated_at
		FROM strategy_reviews
		WHERE marketplace_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var reviews []model.StrategyReview
	if err := r.db.SelectContext(ctx, &reviews, query, marketplaceID, limit, offset); err != nil {
		r.logger.Error("Failed to get reviews by marketplace ID",
			zap.Error(err),
			zap.Int("marketplace_id", marketplaceID))
		return nil, 0, err
	}

	return reviews, total, nil
}

// GetAverageRating gets the average rating for a marketplace listing
func (r *ReviewRepository) GetAverageRating(ctx context.Context, marketplaceID int) (float64, error) {
	query := `
		SELECT COALESCE(AVG(rating), 0)
		FROM strategy_reviews
		WHERE marketplace_id = $1
	`

	var avgRating float64
	err := r.db.GetContext(ctx, &avgRating, query, marketplaceID)
	if err != nil {
		r.logger.Error("Failed to get average rating",
			zap.Error(err),
			zap.Int("marketplace_id", marketplaceID))
		return 0, err
	}

	return avgRating, nil
}

// Update updates a review
func (r *ReviewRepository) Update(ctx context.Context, id int, rating int, comment string) error {
	query := `
		UPDATE strategy_reviews
		SET rating = $1, comment = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, rating, comment, time.Now(), id)
	if err != nil {
		r.logger.Error("Failed to update review",
			zap.Error(err),
			zap.Int("id", id))
		return err
	}

	return nil
}

// Delete removes a review
func (r *ReviewRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM strategy_reviews WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete review",
			zap.Error(err),
			zap.Int("id", id))
		return err
	}

	return nil
}

// GetByUserID gets reviews created by a user
func (r *ReviewRepository) GetByUserID(ctx context.Context, userID int, offset, limit int) ([]model.StrategyReview, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM strategy_reviews WHERE user_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, userID); err != nil {
		r.logger.Error("Failed to count user reviews",
			zap.Error(err),
			zap.Int("user_id", userID))
		return nil, 0, err
	}

	// Get reviews
	query := `
		SELECT r.id, r.marketplace_id, r.user_id, r.rating, r.comment, r.created_at, r.updated_at,
			   m.name as marketplace_name
		FROM strategy_reviews r
		JOIN marketplace_listings m ON r.marketplace_id = m.id
		WHERE r.user_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get user reviews",
			zap.Error(err),
			zap.Int("user_id", userID))
		return nil, 0, err
	}
	defer rows.Close()

	var reviews []model.StrategyReview
	for rows.Next() {
		var review model.StrategyReview
		var marketplaceName string

		err := rows.Scan(
			&review.ID,
			&review.MarketplaceID,
			&review.UserID,
			&review.Rating,
			&review.Comment,
			&review.CreatedAt,
			&review.UpdatedAt,
			&marketplaceName,
		)
		if err != nil {
			r.logger.Error("Failed to scan review row", zap.Error(err))
			return nil, 0, err
		}

		reviews = append(reviews, review)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating review rows", zap.Error(err))
		return nil, 0, err
	}

	return reviews, total, nil
}
