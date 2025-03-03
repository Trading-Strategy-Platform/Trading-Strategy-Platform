package repository

import (
	"context"
	"fmt"
	"time"

	"services/historical-data-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// MarketDataRepository handles database operations for market data
type MarketDataRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewMarketDataRepository creates a new market data repository
func NewMarketDataRepository(db *sqlx.DB, logger *zap.Logger) *MarketDataRepository {
	return &MarketDataRepository{
		db:     db,
		logger: logger,
	}
}

// GetMarketData retrieves market data for a symbol and timeframe
func (r *MarketDataRepository) GetMarketData(
	ctx context.Context,
	symbolID int,
	timeframeID int,
	startDate *time.Time,
	endDate *time.Time,
	limit *int,
) ([]model.MarketData, error) {
	query := `
		SELECT id, symbol_id, timeframe_id, timestamp, open, high, low, close, volume, created_at, updated_at
		FROM market_data
		WHERE symbol_id = $1 AND timeframe_id = $2
	`

	args := []interface{}{symbolID, timeframeID}
	argCount := 3

	if startDate != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argCount)
		args = append(args, *startDate)
		argCount++
	}

	if endDate != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argCount)
		args = append(args, *endDate)
		argCount++
	}

	query += " ORDER BY timestamp"

	if limit != nil {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, *limit)
	}

	var data []model.MarketData
	err := r.db.SelectContext(ctx, &data, query, args...)
	if err != nil {
		r.logger.Error("Failed to get market data",
			zap.Error(err),
			zap.Int("symbol_id", symbolID),
			zap.Int("timeframe_id", timeframeID))
		return nil, err
	}

	return data, nil
}

// InsertMarketData inserts a batch of market data
func (r *MarketDataRepository) InsertMarketData(
	ctx context.Context,
	symbolID int,
	timeframeID int,
	data []model.OHLCV,
) error {
	// Using transaction for batch insert
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction", zap.Error(err))
		return err
	}
	defer tx.Rollback()

	// Prepare the statement for bulk insert
	stmt, err := tx.PreparexContext(ctx, `
		INSERT INTO market_data (symbol_id, timeframe_id, timestamp, open, high, low, close, volume, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (symbol_id, timeframe_id, timestamp) 
		DO UPDATE SET 
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		r.logger.Error("Failed to prepare statement", zap.Error(err))
		return err
	}
	defer stmt.Close()

	// Execute batch insert
	now := time.Now()
	for _, item := range data {
		_, err = stmt.ExecContext(
			ctx,
			symbolID,
			timeframeID,
			item.Timestamp,
			item.Open,
			item.High,
			item.Low,
			item.Close,
			item.Volume,
			now,
		)
		if err != nil {
			r.logger.Error("Failed to insert market data",
				zap.Error(err),
				zap.Time("timestamp", item.Timestamp))
			return err
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction", zap.Error(err))
		return err
	}

	return nil
}

// HasData checks if there is market data for a symbol and timeframe
func (r *MarketDataRepository) HasData(
	ctx context.Context,
	symbolID int,
	timeframeID int,
) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM market_data
			WHERE symbol_id = $1 AND timeframe_id = $2
			LIMIT 1
		)
	`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, symbolID, timeframeID)
	if err != nil {
		r.logger.Error("Failed to check if market data exists",
			zap.Error(err),
			zap.Int("symbol_id", symbolID),
			zap.Int("timeframe_id", timeframeID))
		return false, err
	}

	return exists, nil
}

// GetDataRange returns the date range of available data
func (r *MarketDataRepository) GetDataRange(
	ctx context.Context,
	symbolID int,
	timeframeID int,
) (startDate, endDate time.Time, err error) {
	query := `
		SELECT 
			MIN(timestamp) as start_date,
			MAX(timestamp) as end_date
		FROM market_data
		WHERE symbol_id = $1 AND timeframe_id = $2
	`

	var result struct {
		StartDate time.Time `db:"start_date"`
		EndDate   time.Time `db:"end_date"`
	}

	err = r.db.GetContext(ctx, &result, query, symbolID, timeframeID)
	if err != nil {
		r.logger.Error("Failed to get data range",
			zap.Error(err),
			zap.Int("symbol_id", symbolID),
			zap.Int("timeframe_id", timeframeID))
		return time.Time{}, time.Time{}, err
	}

	return result.StartDate, result.EndDate, nil
}

// UpdateSymbolDataAvailability updates the data_available flag for a symbol
func (r *MarketDataRepository) UpdateSymbolDataAvailability(
	ctx context.Context,
	symbolID int,
	available bool,
) error {
	query := `
		UPDATE symbols
		SET data_available = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, available, symbolID)
	if err != nil {
		r.logger.Error("Failed to update symbol data availability",
			zap.Error(err),
			zap.Int("symbol_id", symbolID))
		return err
	}

	return nil
}
