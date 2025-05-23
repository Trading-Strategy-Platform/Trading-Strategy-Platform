package repository

import (
	"context"
	"encoding/json"
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

// GetCandles retrieves candle data using the get_candles function
func (r *MarketDataRepository) GetCandles(
	ctx context.Context,
	symbolID int,
	timeframe string,
	startTime *time.Time,
	endTime *time.Time,
	limit *int,
	offset *int,
) ([]model.Candle, error) {
	query := `SELECT * FROM get_candles($1, $2, $3, $4, $5, $6)`

	// Use default limit if not provided
	var limitValue *int
	if limit == nil {
		defaultLimit := 1000
		limitValue = &defaultLimit
	} else {
		limitValue = limit
	}

	// Use default offset if not provided
	var offsetValue int
	if offset == nil {
		offsetValue = 0
	} else {
		offsetValue = *offset
	}

	// Default time boundaries if not provided
	var startTimeValue time.Time
	if startTime == nil {
		startTimeValue = time.Now().AddDate(-1, 0, 0) // 1 year ago
	} else {
		startTimeValue = *startTime
	}

	var endTimeValue time.Time
	if endTime == nil {
		endTimeValue = time.Now() // now
	} else {
		endTimeValue = *endTime
	}

	var candles []model.Candle
	err := r.db.SelectContext(
		ctx,
		&candles,
		query,
		symbolID,
		timeframe,
		startTimeValue,
		endTimeValue,
		limitValue,
		offsetValue,
	)

	if err != nil {
		r.logger.Error("Failed to get candles",
			zap.Error(err),
			zap.Int("symbolID", symbolID),
			zap.String("timeframe", timeframe))
		return nil, err
	}

	return candles, nil
}

// CountCandles counts the total number of candles for pagination
func (r *MarketDataRepository) CountCandles(
	ctx context.Context,
	symbolID int,
	timeframe string,
	startTime *time.Time,
	endTime *time.Time,
) (int, error) {
	query := `SELECT count_candles($1, $2, $3, $4)`

	// Default time boundaries if not provided
	var startTimeValue time.Time
	if startTime == nil {
		startTimeValue = time.Now().AddDate(-1, 0, 0) // 1 year ago
	} else {
		startTimeValue = *startTime
	}

	var endTimeValue time.Time
	if endTime == nil {
		endTimeValue = time.Now() // now
	} else {
		endTimeValue = *endTime
	}

	var count int
	err := r.db.GetContext(
		ctx,
		&count,
		query,
		symbolID,
		timeframe,
		startTimeValue,
		endTimeValue,
	)

	if err != nil {
		r.logger.Error("Failed to count candles",
			zap.Error(err),
			zap.Int("symbolID", symbolID),
			zap.String("timeframe", timeframe))
		return 0, err
	}

	return count, nil
}

// GetDataRanges returns all available data ranges for a symbol and timeframe
func (r *MarketDataRepository) GetDataRanges(ctx context.Context, symbolID int, timeframe string) ([]model.DateRange, error) {
	// Add logging for debugging
	r.logger.Debug("Getting data ranges",
		zap.Int("symbolID", symbolID),
		zap.String("timeframe", timeframe))

	// First check if we have any data at all for this symbol and timeframe
	var count int
	checkQuery := `SELECT COUNT(*) FROM candles WHERE symbol_id = $1 LIMIT 1`
	err := r.db.GetContext(ctx, &count, checkQuery, symbolID)
	if err != nil {
		r.logger.Error("Failed to check if candles exist",
			zap.Error(err),
			zap.Int("symbolID", symbolID))
		return nil, err
	}

	r.logger.Debug("Candle count check",
		zap.Int("symbolID", symbolID),
		zap.Int("count", count))

	if count == 0 {
		// No data for this symbol
		return []model.DateRange{}, nil
	}

	// Try the get_symbol_data_ranges function first
	query := `SELECT * FROM get_symbol_data_ranges($1, $2)`

	var ranges []model.DateRange
	err = r.db.SelectContext(ctx, &ranges, query, symbolID, timeframe)

	// If the function-based approach fails, fall back to a direct query
	if err != nil {
		r.logger.Warn("Function-based data range query failed, falling back to direct query",
			zap.Error(err),
			zap.Int("symbolID", symbolID),
			zap.String("timeframe", timeframe))

		// Direct query approach
		fallbackQuery := `
			SELECT 
				MIN(candle_time) as start,
				MAX(candle_time) as end
			FROM candles
			WHERE symbol_id = $1
		`

		var singleRange model.DateRange
		err = r.db.GetContext(ctx, &singleRange, fallbackQuery, symbolID)
		if err != nil {
			r.logger.Error("Failed to get data range with fallback query",
				zap.Error(err),
				zap.Int("symbolID", symbolID))
			return nil, err
		}

		// Only add the range if it has valid dates
		if !singleRange.Start.IsZero() && !singleRange.End.IsZero() {
			ranges = []model.DateRange{singleRange}
			r.logger.Debug("Got data range with fallback query",
				zap.Int("symbolID", symbolID),
				zap.Time("start", singleRange.Start),
				zap.Time("end", singleRange.End))
		}
	} else {
		r.logger.Debug("Got data ranges with function query",
			zap.Int("symbolID", symbolID),
			zap.Int("rangesCount", len(ranges)))
	}

	return ranges, nil
}

// CalculateMissingDataRanges calculates date ranges that are missing from our database
func (r *MarketDataRepository) CalculateMissingDataRanges(
	ctx context.Context,
	symbolID int,
	timeframe string,
	fullRangeStart time.Time,
	fullRangeEnd time.Time,
) ([]model.DateRange, error) {
	// Get existing data ranges
	existingRanges, err := r.GetDataRanges(ctx, symbolID, timeframe)
	if err != nil {
		return nil, err
	}

	// If no existing data, the entire requested range is missing
	if len(existingRanges) == 0 {
		return []model.DateRange{{Start: fullRangeStart, End: fullRangeEnd}}, nil
	}

	// Initialize missing ranges
	var missingRanges []model.DateRange

	// Check if there's a gap at the beginning
	if fullRangeStart.Before(existingRanges[0].Start) {
		missingRanges = append(missingRanges, model.DateRange{
			Start: fullRangeStart,
			End:   existingRanges[0].Start.Add(-time.Second),
		})
	}

	// Check for gaps between existing ranges
	for i := 0; i < len(existingRanges)-1; i++ {
		if existingRanges[i].End.Add(time.Second).Before(existingRanges[i+1].Start) {
			missingRanges = append(missingRanges, model.DateRange{
				Start: existingRanges[i].End.Add(time.Second),
				End:   existingRanges[i+1].Start.Add(-time.Second),
			})
		}
	}

	// Check if there's a gap at the end
	if fullRangeEnd.After(existingRanges[len(existingRanges)-1].End) {
		missingRanges = append(missingRanges, model.DateRange{
			Start: existingRanges[len(existingRanges)-1].End.Add(time.Second),
			End:   fullRangeEnd,
		})
	}

	return missingRanges, nil
}

// BatchImportCandles inserts a batch of candles using the insert_candles function
func (r *MarketDataRepository) BatchImportCandles(
	ctx context.Context,
	candles []model.CandleBatch,
) (int, error) {
	// Convert to JSONB for the database function
	candlesJSON, err := json.Marshal(candles)
	if err != nil {
		r.logger.Error("Failed to marshal candles to JSON", zap.Error(err))
		return 0, err
	}

	query := `SELECT insert_candles($1)`

	var insertedCount int
	err = r.db.GetContext(ctx, &insertedCount, query, candlesJSON)
	if err != nil {
		r.logger.Error("Failed to batch import candles", zap.Error(err))
		return 0, err
	}

	return insertedCount, nil
}

// HasData checks if there is market data for a symbol and timeframe
func (r *MarketDataRepository) HasData(
	ctx context.Context,
	symbolID int,
	timeframe string,
) (bool, error) {
	// Get a single candle to check if data exists
	limit := 1
	candles, err := r.GetCandles(ctx, symbolID, timeframe, nil, nil, &limit, nil)
	if err != nil {
		r.logger.Error("Failed to check if market data exists",
			zap.Error(err),
			zap.Int("symbolID", symbolID),
			zap.String("timeframe", timeframe))
		return false, err
	}

	return len(candles) > 0, nil
}

// GetDataRange returns the date range of available data
func (r *MarketDataRepository) GetDataRange(
	ctx context.Context,
	symbolID int,
	timeframe string,
) (startDate, endDate time.Time, err error) {
	// Using get_candles with extreme dates to find boundaries
	query := `
		SELECT
			MIN(candle_time) as start_date,
			MAX(candle_time) as end_date
		FROM (
			SELECT * FROM get_candles($1, $2, $3, $4, NULL)
		) as candles
	`

	var result struct {
		StartDate time.Time `db:"start_date"`
		EndDate   time.Time `db:"end_date"`
	}

	// Use extreme dates for the range query
	startValue := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	endValue := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

	err = r.db.GetContext(ctx, &result, query, symbolID, timeframe, startValue, endValue)
	if err != nil {
		r.logger.Error("Failed to get data range",
			zap.Error(err),
			zap.Int("symbolID", symbolID),
			zap.String("timeframe", timeframe))
		return time.Time{}, time.Time{}, err
	}

	return result.StartDate, result.EndDate, nil
}

// Add this method to services/historical-data-service/internal/repository/market_data_repository.go

// GetDateRange returns the minimum and maximum dates for a symbol
func (r *MarketDataRepository) GetDateRange(ctx context.Context, symbolID int) (minDate, maxDate time.Time, err error) {
	// Get min date
	minQuery := `SELECT MIN(candle_time) FROM candles WHERE symbol_id = $1`
	err = r.db.GetContext(ctx, &minDate, minQuery, symbolID)
	if err != nil {
		r.logger.Error("Failed to get min date",
			zap.Error(err),
			zap.Int("symbolID", symbolID))
		return time.Time{}, time.Time{}, err
	}

	// Get max date
	maxQuery := `SELECT MAX(candle_time) FROM candles WHERE symbol_id = $1`
	err = r.db.GetContext(ctx, &maxDate, maxQuery, symbolID)
	if err != nil {
		r.logger.Error("Failed to get max date",
			zap.Error(err),
			zap.Int("symbolID", symbolID))
		return time.Time{}, time.Time{}, err
	}

	return minDate, maxDate, nil
}
