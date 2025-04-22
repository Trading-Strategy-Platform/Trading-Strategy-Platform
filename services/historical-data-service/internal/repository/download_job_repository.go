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
	query := `SELECT * FROM get_download_job_by_id($1)`

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

// CountActiveDownloadJobs counts active market data download jobs for pagination
func (r *DownloadJobRepository) CountActiveDownloadJobs(ctx context.Context, source string) (int, error) {
	query := `SELECT count_active_download_jobs($1)`

	var count int
	err := r.db.GetContext(ctx, &count, query, source)
	if err != nil {
		r.logger.Error("Failed to count active market data download jobs",
			zap.Error(err),
			zap.String("source", source))
		return 0, err
	}

	return count, nil
}

// GetActiveDownloadJobs gets all active market data download jobs with pagination and sorting
func (r *DownloadJobRepository) GetActiveDownloadJobs(
	ctx context.Context,
	source string,
	sortBy string,
	sortDirection string,
	limit int,
	offset int,
) ([]model.MarketDataDownloadJob, error) {
	query := `SELECT * FROM get_active_download_jobs($1, $2, $3, $4, $5)`

	var jobs []model.MarketDataDownloadJob
	err := r.db.SelectContext(
		ctx,
		&jobs,
		query,
		source,
		sortBy,
		sortDirection,
		limit,
		offset,
	)

	if err != nil {
		r.logger.Error("Failed to get active market data download jobs",
			zap.Error(err),
			zap.String("source", source),
			zap.String("sortBy", sortBy),
			zap.String("sortDirection", sortDirection))
		return nil, err
	}

	return jobs, nil
}

// CountDownloadJobsByStatus counts download jobs by status for pagination
func (r *DownloadJobRepository) CountDownloadJobsByStatus(
	ctx context.Context,
	status string,
	source string,
	symbol string,
) (int, error) {
	query := `SELECT count_download_jobs_by_status($1, $2, $3)`

	var count int
	err := r.db.GetContext(ctx, &count, query, status, source, symbol)
	if err != nil {
		r.logger.Error("Failed to count download jobs by status",
			zap.Error(err),
			zap.String("status", status),
			zap.String("source", source),
			zap.String("symbol", symbol))
		return 0, err
	}

	return count, nil
}

// GetDownloadJobsByStatus gets download jobs by status with pagination and sorting
func (r *DownloadJobRepository) GetDownloadJobsByStatus(
	ctx context.Context,
	status string,
	source string,
	symbol string,
	sortBy string,
	sortDirection string,
	limit int,
	offset int,
) ([]model.MarketDataDownloadJob, error) {
	query := `SELECT * FROM get_download_jobs_by_status($1, $2, $3, $4, $5, $6, $7)`

	var jobs []model.MarketDataDownloadJob
	err := r.db.SelectContext(
		ctx,
		&jobs,
		query,
		status,
		source,
		symbol,
		sortBy,
		sortDirection,
		limit,
		offset,
	)

	if err != nil {
		r.logger.Error("Failed to get download jobs by status",
			zap.Error(err),
			zap.String("status", status),
			zap.String("source", source),
			zap.String("symbol", symbol),
			zap.String("sortBy", sortBy),
			zap.String("sortDirection", sortDirection))
		return nil, err
	}

	return jobs, nil
}

// GetJobsSummary gets a summary of download jobs by status
func (r *DownloadJobRepository) GetJobsSummary(ctx context.Context) ([]struct {
	Status  string `db:"status"`
	Count   int64  `db:"count"`
	Last24h int64  `db:"last_24h"`
}, error) {
	query := `SELECT * FROM get_download_jobs_summary()`

	var summary []struct {
		Status  string `db:"status"`
		Count   int64  `db:"count"`
		Last24h int64  `db:"last_24h"`
	}
	err := r.db.SelectContext(ctx, &summary, query)
	if err != nil {
		r.logger.Error("Failed to get jobs summary", zap.Error(err))
		return nil, err
	}

	return summary, nil
}

// CancelDownload cancels a download job
func (r *DownloadJobRepository) CancelDownload(
	ctx context.Context,
	jobID int,
	force bool,
) (bool, error) {
	query := `SELECT cancel_download_job($1, $2)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, jobID, force)
	if err != nil {
		r.logger.Error("Failed to cancel download job",
			zap.Error(err),
			zap.Int("jobID", jobID),
			zap.Bool("force", force))
		return false, err
	}

	return success, nil
}

// GetCandleCount gets the number of candles for a symbol
func (r *DownloadJobRepository) GetCandleCount(ctx context.Context, symbolID int) (int, error) {
	query := `SELECT get_symbol_candle_count($1)`

	var count int
	err := r.db.GetContext(ctx, &count, query, symbolID)
	if err != nil {
		r.logger.Error("Failed to get candle count",
			zap.Error(err),
			zap.Int("symbolID", symbolID))
		return 0, err
	}
	return count, nil
}
