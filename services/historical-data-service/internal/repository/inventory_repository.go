package repository

import (
	"context"
	"services/historical-data-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// InventoryRepository handles database operations for market data inventory
type InventoryRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewInventoryRepository creates a new inventory repository
func NewInventoryRepository(db *sqlx.DB, logger *zap.Logger) *InventoryRepository {
	return &InventoryRepository{
		db:     db,
		logger: logger,
	}
}

// CountDataInventory counts the number of symbols with available data
func (r *InventoryRepository) CountDataInventory(
	ctx context.Context,
	assetType, exchange string,
) (int, error) {
	query := `SELECT count_data_inventory($1, $2)`

	var count int
	err := r.db.GetContext(ctx, &count, query, assetType, exchange)
	if err != nil {
		r.logger.Error("Failed to count data inventory",
			zap.Error(err),
			zap.String("assetType", assetType),
			zap.String("exchange", exchange))
		return 0, err
	}

	return count, nil
}

// GetDataInventory retrieves the data inventory with filtering and pagination
func (r *InventoryRepository) GetDataInventory(
	ctx context.Context,
	assetType, exchange string,
	limit, offset int,
) ([]model.DataInventoryItem, error) {
	query := `SELECT * FROM get_data_inventory($1, $2, $3, $4)`

	var inventory []model.DataInventoryItem
	err := r.db.SelectContext(ctx, &inventory, query, assetType, exchange, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get data inventory",
			zap.Error(err),
			zap.String("assetType", assetType),
			zap.String("exchange", exchange))
		return nil, err
	}

	return inventory, nil
}

// GetSymbolAvailableTimeframes gets all timeframes that have data for a symbol
func (r *InventoryRepository) GetSymbolAvailableTimeframes(
	ctx context.Context,
	symbolID int,
) ([]string, error) {
	query := `SELECT timeframe FROM get_symbol_available_timeframes($1)`

	var timeframes []string
	err := r.db.SelectContext(ctx, &timeframes, query, symbolID)
	if err != nil {
		r.logger.Error("Failed to get available timeframes",
			zap.Error(err),
			zap.Int("symbolID", symbolID))
		return nil, err
	}

	return timeframes, nil
}

// GetSymbolCandleCount gets the number of candles for a symbol and timeframe
func (r *InventoryRepository) GetSymbolCandleCount(
	ctx context.Context,
	symbolID int,
	timeframe string,
) (int, error) {
	var query string
	var args []interface{}

	if timeframe == "" {
		query = `SELECT get_symbol_candle_count($1)`
		args = []interface{}{symbolID}
	} else {
		query = `SELECT get_symbol_candle_count($1, $2)`
		args = []interface{}{symbolID, timeframe}
	}

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		r.logger.Error("Failed to get candle count",
			zap.Error(err),
			zap.Int("symbolID", symbolID),
			zap.String("timeframe", timeframe))
		return 0, err
	}

	return count, nil
}
