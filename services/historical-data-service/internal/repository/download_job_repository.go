package repository

import (
	"context"
	"database/sql"
	"time"

	"services/historical-data-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// DownloadJobRepository handles database operations for market data downloads
type DownloadJobRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewDownloadJobRepository creates a new download job repository
func NewDownloadJobRepository(db *sqlx.DB, logger *zap.Logger) *DownloadJobRepository {
	return &DownloadJobRepository{
		db:     db,
		logger: logger,
	}
}

// CreateDownloadJob creates a new job for downloading market data
func (r *DownloadJobRepository) CreateDownloadJob(
	ctx context.Context,
	symbolID int,
	symbol string,
	source string,
	timeframe string,
	startDate time.Time,
	endDate time.Time,
) (int, error) {
	query := `SELECT create_market_data_download_job($1, $2, $3, $4, $5, $6)`

	var jobID int
	err := r.db.GetContext(
		ctx,
		&jobID,
		query,
		symbolID,
		symbol,
		source,
		timeframe,
		startDate,
		endDate,
	)

	if err != nil {
		r.logger.Error("Failed to create market data download job",
			zap.Error(err),
			zap.Int("symbolID", symbolID),
			zap.String("symbol", symbol),
			zap.String("source", source))
		return 0, err
	}

	return jobID, nil
}

// GetDownloadJob gets a market data download job by ID
func (r *DownloadJobRepository) GetDownloadJob(ctx context.Context, jobID int) (*model.MarketDataDownloadJob, error) {
	query := `SELECT * FROM market_data_download_jobs WHERE id = $1`

	var job model.MarketDataDownloadJob
	err := r.db.GetContext(ctx, &job, query, jobID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get market data download job",
			zap.Error(err),
			zap.Int("jobID", jobID))
		return nil, err
	}

	return &job, nil
}

// UpdateDownloadJobStatus updates the status of a market data download job
func (r *DownloadJobRepository) UpdateDownloadJobStatus(
	ctx context.Context,
	jobID int,
	status string,
	progress float64,
	processedCandles int,
	totalCandles int,
	retries int,
	errorMsg string,
) (bool, error) {
	query := `SELECT update_market_data_download_job_status($1, $2, $3, $4, $5, $6, $7)`

	var success bool
	err := r.db.GetContext(
		ctx,
		&success,
		query,
		jobID,
		status,
		progress,
		processedCandles,
		totalCandles,
		retries,
		errorMsg,
	)

	if err != nil {
		r.logger.Error("Failed to update market data download job status",
			zap.Error(err),
			zap.Int("jobID", jobID),
			zap.String("status", status))
		return false, err
	}

	return success, nil
}

// GetActiveDownloadJobs gets all active market data download jobs
func (r *DownloadJobRepository) GetActiveDownloadJobs(ctx context.Context, source string) ([]model.MarketDataDownloadJob, error) {
	query := `SELECT * FROM get_active_market_data_download_jobs($1)`

	var jobs []model.MarketDataDownloadJob
	err := r.db.SelectContext(ctx, &jobs, query, source)
	if err != nil {
		r.logger.Error("Failed to get active market data download jobs",
			zap.Error(err),
			zap.String("source", source))
		return nil, err
	}

	return jobs, nil
}

// GetJobsSummary gets a summary of download jobs by status
func (r *DownloadJobRepository) GetJobsSummary(ctx context.Context) (map[string]interface{}, error) {
	// Get counts of jobs by status
	statusQuery := `
		SELECT 
			status, 
			COUNT(*) as count,
			SUM(CASE WHEN created_at > NOW() - INTERVAL '24 hours' THEN 1 ELSE 0 END) as last_24h
		FROM market_data_download_jobs
		GROUP BY status
	`

	type statusCount struct {
		Status  string `db:"status"`
		Count   int    `db:"count"`
		Last24h int    `db:"last_24h"`
	}

	var counts []statusCount
	err := r.db.SelectContext(ctx, &counts, statusQuery)
	if err != nil {
		r.logger.Error("Failed to get job status counts", zap.Error(err))
		return nil, err
	}

	// Get recent jobs
	recentQuery := `
		SELECT * FROM market_data_download_jobs
		ORDER BY created_at DESC
		LIMIT 10
	`

	var recentJobs []model.MarketDataDownloadJob
	err = r.db.SelectContext(ctx, &recentJobs, recentQuery)
	if err != nil {
		r.logger.Error("Failed to get recent jobs", zap.Error(err))
		return nil, err
	}

	// Get counts by source
	sourceQuery := `
		SELECT 
			source, 
			COUNT(*) as count
		FROM market_data_download_jobs
		GROUP BY source
	`

	type sourceCount struct {
		Source string `db:"source"`
		Count  int    `db:"count"`
	}

	var sourceCounts []sourceCount
	err = r.db.SelectContext(ctx, &sourceCounts, sourceQuery)
	if err != nil {
		r.logger.Error("Failed to get job source counts", zap.Error(err))
		return nil, err
	}

	// Format the data for the response
	statusCounts := make(map[string]map[string]int)
	for _, c := range counts {
		statusCounts[c.Status] = map[string]int{
			"total":    c.Count,
			"last_24h": c.Last24h,
		}
	}

	sourcesMap := make(map[string]int)
	for _, c := range sourceCounts {
		sourcesMap[c.Source] = c.Count
	}

	return map[string]interface{}{
		"status_counts": statusCounts,
		"sources":       sourcesMap,
		"recent_jobs":   recentJobs,
	}, nil
}

// GetAvailableTimeframes gets all timeframes that have data for a symbol
func (r *DownloadJobRepository) GetAvailableTimeframes(ctx context.Context, symbolID int) ([]string, error) {
	query := `
		SELECT DISTINCT timeframe::text 
		FROM (
			SELECT '1m'::timeframe_type as timeframe 
			WHERE EXISTS (SELECT 1 FROM candles WHERE symbol_id = $1)
			UNION SELECT '5m'::timeframe_type WHERE true
			UNION SELECT '15m'::timeframe_type WHERE true
			UNION SELECT '30m'::timeframe_type WHERE true
			UNION SELECT '1h'::timeframe_type WHERE true
			UNION SELECT '4h'::timeframe_type WHERE true
			UNION SELECT '1d'::timeframe_type WHERE true
			UNION SELECT '1w'::timeframe_type WHERE true
		) tf
	`

	var timeframes []string
	err := r.db.SelectContext(ctx, &timeframes, query, symbolID)
	return timeframes, err
}

// GetCandleCount gets the number of candles for a symbol and timeframe
func (r *DownloadJobRepository) GetCandleCount(ctx context.Context, symbolID int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM candles WHERE symbol_id = $1`
	err := r.db.GetContext(ctx, &count, query, symbolID)
	if err != nil {
		r.logger.Error("Failed to get candle count",
			zap.Error(err),
			zap.Int("symbolID", symbolID))
		return 0, err
	}
	return count, nil
}
