// internal/repository/market_data_repository.go
package repository

import (
	"context"
	"database/sql"
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
) ([]model.Candle, error) {
	query := `SELECT * FROM get_candles($1, $2, $3, $4, $5)`

	// Use default limit if not provided
	var limitValue *int
	if limit == nil {
		defaultLimit := 1000
		limitValue = &defaultLimit
	} else {
		limitValue = limit
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

// GetDataRanges returns all available data ranges for a symbol and timeframe
func (r *MarketDataRepository) GetDataRanges(ctx context.Context, symbolID int, timeframe string) ([]model.DateRange, error) {
	// Query to find continuous blocks of data
	query := `
		WITH dates AS (
			SELECT 
				time,
				LEAD(time) OVER (ORDER BY time) as next_time
			FROM candles
			WHERE symbol_id = $1
			ORDER BY time
		)
		SELECT 
			MIN(time) as start_date,
			MAX(time) as end_date
		FROM (
			SELECT 
				time,
				next_time,
				CASE WHEN next_time IS NULL OR next_time > time + INTERVAL '1 day' THEN 1 ELSE 0 END as is_gap,
				SUM(CASE WHEN next_time IS NULL OR next_time > time + INTERVAL '1 day' THEN 1 ELSE 0 END) OVER (ORDER BY time) as group_id
			FROM dates
		) t
		GROUP BY group_id
		ORDER BY start_date
	`

	var ranges []struct {
		StartDate time.Time `db:"start_date"`
		EndDate   time.Time `db:"end_date"`
	}

	err := r.db.SelectContext(ctx, &ranges, query, symbolID)
	if err != nil {
		r.logger.Error("Failed to get data ranges",
			zap.Error(err),
			zap.Int("symbolID", symbolID),
			zap.String("timeframe", timeframe))
		return nil, err
	}

	// Convert to the model format
	result := make([]model.DateRange, len(ranges))
	for i, r := range ranges {
		result[i] = model.DateRange{
			Start: r.StartDate,
			End:   r.EndDate,
		}
	}

	return result, nil
}

// CreateBinanceDownloadJob creates a new job for downloading data from Binance
func (r *MarketDataRepository) CreateBinanceDownloadJob(
	ctx context.Context,
	symbolID int,
	symbol string,
	timeframe string,
	startDate time.Time,
	endDate time.Time,
) (int, error) {
	query := `
		INSERT INTO binance_download_jobs (
			symbol_id, 
			symbol, 
			timeframe, 
			start_date, 
			end_date, 
			status, 
			progress, 
			created_at, 
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 'pending', 0, NOW(), NOW())
		RETURNING id
	`

	var id int
	err := r.db.GetContext(
		ctx,
		&id,
		query,
		symbolID,
		symbol,
		timeframe,
		startDate,
		endDate,
	)

	if err != nil {
		r.logger.Error("Failed to create Binance download job",
			zap.Error(err),
			zap.Int("symbolID", symbolID),
			zap.String("symbol", symbol),
			zap.String("timeframe", timeframe))
		return 0, err
	}

	return id, nil
}

// GetBinanceDownloadJob gets a Binance download job by ID
func (r *MarketDataRepository) GetBinanceDownloadJob(ctx context.Context, jobID int) (*model.BinanceDownloadJob, error) {
	query := `
		SELECT * FROM binance_download_jobs WHERE id = $1
	`

	var job model.BinanceDownloadJob
	err := r.db.GetContext(ctx, &job, query, jobID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get Binance download job",
			zap.Error(err),
			zap.Int("jobID", jobID))
		return nil, err
	}

	return &job, nil
}

// UpdateBinanceDownloadJobStatus updates the status of a Binance download job
func (r *MarketDataRepository) UpdateBinanceDownloadJobStatus(
	ctx context.Context,
	jobID int,
	status string,
	progress float64,
	errorMsg string,
) error {
	query := `
		UPDATE binance_download_jobs
		SET 
			status = $2, 
			progress = $3, 
			error = $4, 
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, jobID, status, progress, errorMsg)
	if err != nil {
		r.logger.Error("Failed to update Binance download job status",
			zap.Error(err),
			zap.Int("jobID", jobID),
			zap.String("status", status))
		return err
	}

	return nil
}

// GetActiveBinanceDownloadJobs gets all active Binance download jobs
func (r *MarketDataRepository) GetActiveBinanceDownloadJobs(ctx context.Context) ([]model.BinanceDownloadJob, error) {
	query := `
		SELECT * FROM binance_download_jobs
		WHERE status IN ('pending', 'in_progress')
		ORDER BY created_at DESC
	`

	var jobs []model.BinanceDownloadJob
	err := r.db.SelectContext(ctx, &jobs, query)
	if err != nil {
		r.logger.Error("Failed to get active Binance download jobs", zap.Error(err))
		return nil, err
	}

	return jobs, nil
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
	candles, err := r.GetCandles(ctx, symbolID, timeframe, nil, nil, &[]int{1}[0])
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
			MIN(time) as start_date,
			MAX(time) as end_date
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
			zap.Int("symbolID", symbolID))
		return err
	}

	return nil
}
