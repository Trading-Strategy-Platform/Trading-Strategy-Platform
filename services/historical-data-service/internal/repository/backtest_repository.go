package repository

import (
	"context"
	"database/sql"
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

// CreateBacktest creates a new backtest using create_backtest function
func (r *BacktestRepository) CreateBacktest(
	ctx context.Context,
	userID int,
	strategyID int,
	strategyVersion int,
	name string,
	description string,
	timeframe string,
	startDate time.Time,
	endDate time.Time,
	initialCapital float64,
	symbolIDs []int,
) (int, error) {
	query := `SELECT create_backtest($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	var backtestID int
	err := r.db.GetContext(
		ctx,
		&backtestID,
		query,
		userID,
		strategyID,
		strategyVersion,
		name,
		description,
		timeframe,
		startDate,
		endDate,
		initialCapital,
		symbolIDs,
	)

	if err != nil {
		r.logger.Error("Failed to create backtest", zap.Error(err))
		return 0, err
	}

	return backtestID, nil
}

// GetBacktest retrieves a backtest by ID using get_backtest_by_id function
func (r *BacktestRepository) GetBacktest(
	ctx context.Context,
	backtestID int,
) (*model.BacktestDetails, error) {
	query := `SELECT * FROM get_backtest_by_id($1)`

	var backtest model.BacktestDetails
	err := r.db.GetContext(ctx, &backtest, query, backtestID)
	if err != nil {
		r.logger.Error("Failed to get backtest details", zap.Error(err), zap.Int("id", backtestID))
		return nil, err
	}

	return &backtest, nil
}

// CountBacktests counts the total number of backtests for a user with filtering
func (r *BacktestRepository) CountBacktests(
	ctx context.Context,
	userID int,
	searchTerm string,
	status string,
) (int, error) {
	query := `SELECT count_backtests($1, $2, $3)`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID, searchTerm, status)
	if err != nil {
		r.logger.Error("Failed to count user backtests",
			zap.Error(err),
			zap.Int("userID", userID),
			zap.String("searchTerm", searchTerm),
			zap.String("status", status))
		return 0, err
	}

	return count, nil
}

// GetBacktestsByUser retrieves backtests for a user using get_backtests function with sorting and pagination
func (r *BacktestRepository) GetBacktestsByUser(
	ctx context.Context,
	userID int,
	searchTerm string,
	status string,
	sortBy string,
	sortDirection string,
	limit int,
	offset int,
) ([]model.BacktestSummary, error) {
	query := `SELECT * FROM get_backtests($1, $2, $3, $4, $5, $6, $7)`

	var backtests []model.BacktestSummary
	err := r.db.SelectContext(ctx, &backtests, query,
		userID, searchTerm, status, sortBy, sortDirection, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get backtest summary",
			zap.Error(err),
			zap.Int("userID", userID),
			zap.String("sortBy", sortBy),
			zap.String("sortDirection", sortDirection))
		return nil, err
	}

	return backtests, nil
}

// UpdateBacktestRunStatus updates the status of a backtest run using update_backtest_run_status function
func (r *BacktestRepository) UpdateBacktestRunStatus(
	ctx context.Context,
	runID int,
	status string,
) (bool, error) {
	query := `SELECT update_backtest_run_status($1, $2)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, runID, status)
	if err != nil {
		r.logger.Error("Failed to update backtest run status",
			zap.Error(err),
			zap.Int("runID", runID),
			zap.String("status", status))
		return false, err
	}

	return success, nil
}

// SaveBacktestResults saves results for a backtest run using save_backtest_result function
func (r *BacktestRepository) SaveBacktestResults(
	ctx context.Context,
	runID int,
	results *model.BacktestResults,
) (int, error) {
	query := `SELECT save_backtest_result($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	var resultID int
	err := r.db.GetContext(
		ctx,
		&resultID,
		query,
		runID,
		results.TotalTrades,
		results.WinningTrades,
		results.LosingTrades,
		results.ProfitFactor,
		results.SharpeRatio,
		results.MaxDrawdown,
		results.FinalCapital,
		results.TotalReturn,
		results.AnnualizedReturn,
		results.ResultsJSON,
	)

	if err != nil {
		r.logger.Error("Failed to save backtest results", zap.Error(err), zap.Int("runID", runID))
		return 0, err
	}

	return resultID, nil
}

// CountBacktestTrades counts the number of trades for a backtest run
func (r *BacktestRepository) CountBacktestTrades(
	ctx context.Context,
	runID int,
) (int, error) {
	query := `SELECT count_backtest_trades($1)`

	var count int
	err := r.db.GetContext(ctx, &count, query, runID)
	if err != nil {
		r.logger.Error("Failed to count backtest trades", zap.Error(err), zap.Int("runID", runID))
		return 0, err
	}

	return count, nil
}

// AddBacktestTrade adds a trade to a backtest run using add_backtest_trade function
func (r *BacktestRepository) AddBacktestTrade(
	ctx context.Context,
	trade *model.BacktestTrade,
) (int, error) {
	query := `SELECT add_backtest_trade($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	var tradeID int
	err := r.db.GetContext(
		ctx,
		&tradeID,
		query,
		trade.BacktestRunID,
		trade.SymbolID,
		trade.EntryTime,
		trade.ExitTime,
		trade.PositionType,
		trade.EntryPrice,
		trade.ExitPrice,
		trade.Quantity,
		trade.ProfitLoss,
		trade.ProfitLossPercent,
		trade.ExitReason,
	)

	if err != nil {
		r.logger.Error("Failed to add backtest trade", zap.Error(err))
		return 0, err
	}

	return tradeID, nil
}

// GetBacktestTrades retrieves trades for a backtest run using get_backtest_trades function with sorting and pagination
func (r *BacktestRepository) GetBacktestTrades(
	ctx context.Context,
	runID int,
	sortBy string,
	sortDirection string,
	limit int,
	offset int,
) ([]model.BacktestTrade, error) {
	query := `SELECT * FROM get_backtest_trades($1, $2, $3, $4, $5)`

	var trades []model.BacktestTrade
	err := r.db.SelectContext(ctx, &trades, query, runID, sortBy, sortDirection, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get backtest trades",
			zap.Error(err),
			zap.Int("runID", runID),
			zap.String("sortBy", sortBy),
			zap.String("sortDirection", sortDirection))
		return nil, err
	}

	return trades, nil
}

// DeleteBacktest deletes a backtest using delete_backtest function
func (r *BacktestRepository) DeleteBacktest(
	ctx context.Context,
	userID int,
	backtestID int,
) (bool, error) {
	query := `SELECT delete_backtest($1, $2)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, userID, backtestID)
	if err != nil {
		r.logger.Error("Failed to delete backtest",
			zap.Error(err),
			zap.Int("userID", userID),
			zap.Int("backtestID", backtestID))
		return false, err
	}

	if !success {
		return false, fmt.Errorf("backtest not found or not owned by user")
	}

	return true, nil
}

// GetBacktestUserID gets the user ID associated with a backtest
func (r *BacktestRepository) GetBacktestUserID(
	ctx context.Context,
	backtestID int,
) (int, error) {
	query := "SELECT user_id FROM backtests WHERE id = $1"

	var userID int
	err := r.db.GetContext(ctx, &userID, query, backtestID)
	if err != nil {
		r.logger.Error("Failed to get backtest user ID",
			zap.Error(err),
			zap.Int("backtestID", backtestID))
		return 0, err
	}
	return userID, nil
}

// CountBacktestRuns counts the number of runs for a backtest
func (r *BacktestRepository) CountBacktestRuns(
	ctx context.Context,
	backtestID int,
) (int, error) {
	query := `SELECT count_backtest_runs($1)`

	var count int
	err := r.db.GetContext(ctx, &count, query, backtestID)
	if err != nil {
		r.logger.Error("Failed to count backtest runs", zap.Error(err), zap.Int("backtestID", backtestID))
		return 0, err
	}

	return count, nil
}

// GetBacktestRuns retrieves all runs for a backtest with sorting and pagination
func (r *BacktestRepository) GetBacktestRuns(
	ctx context.Context,
	backtestID int,
	sortBy string,
	sortDirection string,
	limit int,
	offset int,
) ([]struct {
	ID          int        `db:"id"`
	BacktestID  int        `db:"backtest_id"`
	SymbolID    int        `db:"symbol_id"`
	Symbol      string     `db:"symbol"`
	Status      string     `db:"status"`
	CreatedAt   time.Time  `db:"created_at"`
	CompletedAt *time.Time `db:"completed_at"`
}, error) {
	query := `SELECT * FROM get_backtest_runs($1, $2, $3, $4, $5)`

	var runs []struct {
		ID          int        `db:"id"`
		BacktestID  int        `db:"backtest_id"`
		SymbolID    int        `db:"symbol_id"`
		Symbol      string     `db:"symbol"`
		Status      string     `db:"status"`
		CreatedAt   time.Time  `db:"created_at"`
		CompletedAt *time.Time `db:"completed_at"`
	}

	err := r.db.SelectContext(ctx, &runs, query, backtestID, sortBy, sortDirection, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get backtest runs",
			zap.Error(err),
			zap.Int("backtestID", backtestID),
			zap.String("sortBy", sortBy),
			zap.String("sortDirection", sortDirection))
		return nil, err
	}

	return runs, nil
}

// GetBacktestRunIDBySymbol finds the run ID for a specific backtest and symbol
func (r *BacktestRepository) GetBacktestRunIDBySymbol(
	ctx context.Context,
	backtestID int,
	symbolID int,
) (int, error) {
	query := `
		SELECT id FROM backtest_runs
		WHERE backtest_id = $1 AND symbol_id = $2
	`

	var runID int
	err := r.db.GetContext(ctx, &runID, query, backtestID, symbolID)
	if err != nil {
		r.logger.Error("Failed to find backtest run ID",
			zap.Error(err),
			zap.Int("backtestID", backtestID),
			zap.Int("symbolID", symbolID))
		return 0, err
	}

	return runID, nil
}

// GetBacktestSymbolIDs gets all symbol IDs for a backtest
func (r *BacktestRepository) GetBacktestSymbolIDs(
	ctx context.Context,
	backtestID int,
) ([]int, error) {
	query := `SELECT symbol_id FROM backtest_runs WHERE backtest_id = $1`

	var symbolIDs []int
	err := r.db.SelectContext(ctx, &symbolIDs, query, backtestID)
	if err != nil {
		r.logger.Error("Failed to get backtest symbol IDs",
			zap.Error(err),
			zap.Int("backtestID", backtestID))
		return nil, err
	}
	return symbolIDs, err
}

// GetQueuedBacktests retrieves backtests in queued status
func (r *BacktestRepository) GetQueuedBacktests(
	ctx context.Context,
	limit int,
) ([]model.BacktestSummary, error) {
	query := `
		SELECT 
			b.id AS backtest_id,
			b.name,
			b.strategy_id,
			b.created_at AS date,
			b.status,
			NULL AS symbol_results,
			0 AS completed_runs,
			COUNT(br.id) AS total_runs
		FROM 
			backtests b
		LEFT JOIN
			backtest_runs br ON b.id = br.backtest_id
		WHERE 
			b.status = 'pending'
		GROUP BY
			b.id, b.name, b.strategy_id, b.created_at, b.status
		ORDER BY 
			b.created_at ASC
		LIMIT $1
	`

	var backtests []model.BacktestSummary
	err := r.db.SelectContext(ctx, &backtests, query, limit)
	if err != nil {
		r.logger.Error("Failed to get queued backtests", zap.Error(err))
		return nil, err
	}

	return backtests, nil
}

// UpdateBacktestRunsStatusBulk updates all runs for a backtest to the given status
func (r *BacktestRepository) UpdateBacktestRunsStatusBulk(
	ctx context.Context,
	backtestID int,
	status string,
) error {
	query := `
		UPDATE backtest_runs
		SET status = $2, completed_at = NOW()
		WHERE backtest_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, backtestID, status)
	if err != nil {
		r.logger.Error("Failed to update backtest runs status",
			zap.Error(err),
			zap.Int("backtestID", backtestID),
			zap.String("status", status))
	}
	return err
}

// UpdateBacktestStatus updates a backtest status
func (r *BacktestRepository) UpdateBacktestStatus(
	ctx context.Context,
	backtestID int,
	status string,
	errorMessage string,
) error {
	query := `
		UPDATE backtests
		SET status = $2, error_message = $3, completed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, backtestID, status, errorMessage)
	if err != nil {
		r.logger.Error("Failed to update backtest status",
			zap.Error(err),
			zap.Int("backtestID", backtestID),
			zap.String("status", status))
	}
	return err
}

// GetBacktestDetails gets detailed information about a backtest
func (r *BacktestRepository) GetBacktestDetails(
	ctx context.Context,
	backtestID int,
) (*struct {
	StrategyID      int
	StrategyVersion int
	UserID          int
	Timeframe       string
	StartDate       time.Time
	EndDate         time.Time
	InitialCapital  float64
}, error) {
	query := `
		SELECT strategy_id, strategy_version, user_id, timeframe, 
               start_date, end_date, initial_capital 
        FROM backtests WHERE id = $1
	`

	var dbDetails struct {
		StrategyID      int       `db:"strategy_id"`
		StrategyVersion int       `db:"strategy_version"`
		UserID          int       `db:"user_id"`
		Timeframe       string    `db:"timeframe"`
		StartDate       time.Time `db:"start_date"`
		EndDate         time.Time `db:"end_date"`
		InitialCapital  float64   `db:"initial_capital"`
	}

	err := r.db.GetContext(ctx, &dbDetails, query, backtestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get backtest details",
			zap.Error(err),
			zap.Int("backtestID", backtestID))
		return nil, err
	}

	// Create a clean struct without DB tags for the return value
	result := struct {
		StrategyID      int
		StrategyVersion int
		UserID          int
		Timeframe       string
		StartDate       time.Time
		EndDate         time.Time
		InitialCapital  float64
	}{
		StrategyID:      dbDetails.StrategyID,
		StrategyVersion: dbDetails.StrategyVersion,
		UserID:          dbDetails.UserID,
		Timeframe:       dbDetails.Timeframe,
		StartDate:       dbDetails.StartDate,
		EndDate:         dbDetails.EndDate,
		InitialCapital:  dbDetails.InitialCapital,
	}

	return &result, nil
}
