package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"services/historical-data-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// BacktestRepository handles database operations for backtests
type BacktestRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewBacktestRepository creates a new backtest repository
func NewBacktestRepository(db *sqlx.DB, logger *zap.Logger) *BacktestRepository {
	return &BacktestRepository{
		db:     db,
		logger: logger,
	}
}

// CreateBacktest creates a new backtest
func (r *BacktestRepository) CreateBacktest(
	ctx context.Context,
	backtest *model.Backtest,
) (int, error) {
	query := `
		INSERT INTO backtests (
			user_id, strategy_id, strategy_name, strategy_version, 
			symbol_id, timeframe_id, start_date, end_date, 
			initial_capital, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		backtest.UserID,
		backtest.StrategyID,
		backtest.StrategyName,
		backtest.StrategyVersion,
		backtest.SymbolID,
		backtest.TimeframeID,
		backtest.StartDate,
		backtest.EndDate,
		backtest.InitialCapital,
		backtest.Status,
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create backtest", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// GetBacktest retrieves a backtest by ID
func (r *BacktestRepository) GetBacktest(
	ctx context.Context,
	id int,
) (*model.Backtest, error) {
	query := `
		SELECT 
			id, user_id, strategy_id, strategy_name, strategy_version,
			symbol_id, timeframe_id, start_date, end_date, initial_capital,
			status, results, error_message, created_at, updated_at, completed_at
		FROM backtests
		WHERE id = $1
	`

	var backtest model.Backtest
	err := r.db.GetContext(ctx, &backtest, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get backtest", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &backtest, nil
}

// GetBacktestsByUser retrieves backtests for a user
func (r *BacktestRepository) GetBacktestsByUser(
	ctx context.Context,
	userID int,
	page, limit int,
) ([]model.Backtest, int, error) {
	// Calculate offset
	offset := (page - 1) * limit

	// Query to get total count
	countQuery := `
		SELECT COUNT(*) 
		FROM backtests
		WHERE user_id = $1
	`

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, userID)
	if err != nil {
		r.logger.Error("Failed to count backtests", zap.Error(err), zap.Int("userID", userID))
		return nil, 0, err
	}

	// Query to get paginated backtests
	query := `
		SELECT 
			id, user_id, strategy_id, strategy_name, strategy_version,
			symbol_id, timeframe_id, start_date, end_date, initial_capital,
			status, results, error_message, created_at, updated_at, completed_at
		FROM backtests
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var backtests []model.Backtest
	err = r.db.SelectContext(ctx, &backtests, query, userID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get backtests",
			zap.Error(err),
			zap.Int("userID", userID),
			zap.Int("page", page),
			zap.Int("limit", limit))
		return nil, 0, err
	}

	return backtests, total, nil
}

// UpdateBacktestStatus updates the status of a backtest
func (r *BacktestRepository) UpdateBacktestStatus(
	ctx context.Context,
	id int,
	status model.BacktestStatus,
) error {
	query := `
		UPDATE backtests
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		r.logger.Error("Failed to update backtest status",
			zap.Error(err),
			zap.Int("id", id),
			zap.String("status", string(status)))
		return err
	}

	return nil
}

// CompleteBacktest completes a backtest with results
func (r *BacktestRepository) CompleteBacktest(
	ctx context.Context,
	id int,
	results model.BacktestResults,
) error {
	// Convert results to JSON
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		r.logger.Error("Failed to marshal backtest results", zap.Error(err))
		return err
	}

	query := `
		UPDATE backtests
		SET 
			status = $1, 
			results = $2, 
			updated_at = CURRENT_TIMESTAMP,
			completed_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		model.BacktestStatusCompleted,
		resultsJSON,
		id,
	)
	if err != nil {
		r.logger.Error("Failed to complete backtest", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// FailBacktest marks a backtest as failed with an error message
func (r *BacktestRepository) FailBacktest(
	ctx context.Context,
	id int,
	errorMessage string,
) error {
	query := `
		UPDATE backtests
		SET 
			status = $1, 
			error_message = $2, 
			updated_at = CURRENT_TIMESTAMP,
			completed_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		model.BacktestStatusFailed,
		errorMessage,
		id,
	)
	if err != nil {
		r.logger.Error("Failed to set backtest as failed",
			zap.Error(err),
			zap.Int("id", id),
			zap.String("errorMessage", errorMessage))
		return err
	}

	return nil
}

// DeleteBacktest deletes a backtest
func (r *BacktestRepository) DeleteBacktest(
	ctx context.Context,
	id int,
	userID int,
) error {
	query := `DELETE FROM backtests WHERE id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		r.logger.Error("Failed to delete backtest",
			zap.Error(err),
			zap.Int("id", id),
			zap.Int("userID", userID))
		return err
	}

	// Check if any row was affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("backtest not found or not owned by user")
	}

	return nil
}

// GetQueuedBacktests retrieves backtests in queued status
func (r *BacktestRepository) GetQueuedBacktests(
	ctx context.Context,
	limit int,
) ([]model.Backtest, error) {
	query := `
		SELECT 
			id, user_id, strategy_id, strategy_name, strategy_version,
			symbol_id, timeframe_id, start_date, end_date, initial_capital,
			status, results, error_message, created_at, updated_at, completed_at
		FROM backtests
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	var backtests []model.Backtest
	err := r.db.SelectContext(ctx, &backtests, query, model.BacktestStatusQueued, limit)
	if err != nil {
		r.logger.Error("Failed to get queued backtests", zap.Error(err))
		return nil, err
	}

	return backtests, nil
}
