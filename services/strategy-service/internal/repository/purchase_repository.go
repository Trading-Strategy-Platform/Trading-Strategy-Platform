package repository

import (
	"context"
	"time"

	"services/strategy-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// PurchaseRepository handles purchase operations
type PurchaseRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewPurchaseRepository creates a new PurchaseRepository
func NewPurchaseRepository(db *sqlx.DB, logger *zap.Logger) *PurchaseRepository {
	return &PurchaseRepository{
		db:     db,
		logger: logger,
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

// HasUserPurchased checks if user has purchased the strategy
func (r *PurchaseRepository) HasUserPurchased(ctx context.Context, marketplaceID int, userID int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM strategy_purchases WHERE marketplace_id = $1 AND user_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, marketplaceID, userID)
	if err != nil {
		r.logger.Error("Failed to check if user purchased strategy",
			zap.Error(err),
			zap.Int("marketplace_id", marketplaceID),
			zap.Int("user_id", userID))
		return false, err
	}
	return exists, nil
}

// CreateWithTx creates a purchase using a transaction
func (r *PurchaseRepository) CreateWithTx(
	ctx context.Context,
	tx *sqlx.Tx,
	marketplaceID int,
	userID int,
	price float64,
	isSubscription bool,
	subscriptionPeriod string,
) (int, error) {
	var id int
	query := `
        INSERT INTO strategy_purchases (
            marketplace_id, user_id, price, is_subscription, subscription_period, 
            expires_at, created_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7
        ) RETURNING id
    `

	var expiresAt *time.Time
	if isSubscription {
		expiry := time.Now().Add(getSubscriptionDuration(subscriptionPeriod))
		expiresAt = &expiry
	}

	err := tx.GetContext(
		ctx,
		&id,
		query,
		marketplaceID,
		userID,
		price,
		isSubscription,
		subscriptionPeriod,
		expiresAt,
		time.Now(),
	)

	if err != nil {
		r.logger.Error("Failed to create purchase",
			zap.Error(err),
			zap.Int("marketplace_id", marketplaceID),
			zap.Int("user_id", userID))
		return 0, err
	}

	return id, nil
}

// GetByUser retrieves purchases for a user
func (r *PurchaseRepository) GetByUser(ctx context.Context, userID int, page, limit int) ([]model.StrategyPurchase, int, error) {
	offset := (page - 1) * limit

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM strategy_purchases WHERE user_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, userID); err != nil {
		r.logger.Error("Failed to count user purchases",
			zap.Error(err),
			zap.Int("user_id", userID))
		return nil, 0, err
	}

	// Get purchases
	query := `
		SELECT 
			p.id, p.marketplace_id, p.user_id, p.price, p.is_subscription, 
			p.subscription_period, p.expires_at, p.created_at,
			m.name as marketplace_name, m.strategy_id
		FROM 
			strategy_purchases p
		JOIN 
			marketplace_listings m ON p.marketplace_id = m.id
		WHERE 
			p.user_id = $1
		ORDER BY 
			p.created_at DESC
		LIMIT $2 OFFSET $3
	`

	var purchases []model.StrategyPurchase
	if err := r.db.SelectContext(ctx, &purchases, query, userID, limit, offset); err != nil {
		r.logger.Error("Failed to get user purchases",
			zap.Error(err),
			zap.Int("user_id", userID))
		return nil, 0, err
	}

	return purchases, total, nil
}
