package repository

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

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

// Purchase adds a new purchase record using purchase_strategy function
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
