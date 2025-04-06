package repository

import (
	"context"
	"database/sql"
	"time"

	"services/historical-data-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// BinanceRepository handles database operations for Binance data downloads
type BinanceRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewBinanceRepository creates a new Binance repository
func NewBinanceRepository(db *sqlx.DB, logger *zap.Logger) *BinanceRepository {
	return &BinanceRepository{
		db:     db,
		logger: logger,
	}
}

// CreateDownloadJob creates a new job for downloading data from Binance
func (r *BinanceRepository) CreateDownloadJob(
	ctx context.Context,
	symbolID int,
	symbol string,
	timeframe string,
	startDate time.Time,
	endDate time.Time,
) (int, error) {
	query := `SELECT create_binance_download_job($1, $2, $3, $4, $5)`

	var jobID int
	err := r.db.GetContext(
		ctx,
		&jobID,
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
			zap.String("symbol", symbol))
		return 0, err
	}

	return jobID, nil
}

// GetDownloadJob gets a Binance download job by ID
func (r *BinanceRepository) GetDownloadJob(ctx context.Context, jobID int) (*model.BinanceDownloadJob, error) {
	query := `SELECT * FROM binance_download_jobs WHERE id = $1`

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

// UpdateDownloadJobStatus updates the status of a Binance download job
func (r *BinanceRepository) UpdateDownloadJobStatus(
	ctx context.Context,
	jobID int,
	status string,
	progress float64,
	errorMsg string,
) (bool, error) {
	query := `SELECT update_binance_download_job_status($1, $2, $3, $4)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, jobID, status, progress, errorMsg)
	if err != nil {
		r.logger.Error("Failed to update Binance download job status",
			zap.Error(err),
			zap.Int("jobID", jobID),
			zap.String("status", status))
		return false, err
	}

	return success, nil
}

// GetActiveDownloadJobs gets all active Binance download jobs
func (r *BinanceRepository) GetActiveDownloadJobs(ctx context.Context) ([]model.BinanceDownloadJob, error) {
	query := `SELECT * FROM get_active_binance_download_jobs()`

	var jobs []model.BinanceDownloadJob
	err := r.db.SelectContext(ctx, &jobs, query)
	if err != nil {
		r.logger.Error("Failed to get active Binance download jobs", zap.Error(err))
		return nil, err
	}

	return jobs, nil
}
