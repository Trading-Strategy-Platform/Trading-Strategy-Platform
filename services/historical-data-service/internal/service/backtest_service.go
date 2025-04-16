package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"services/historical-data-service/internal/client"
	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/repository"

	"go.uber.org/zap"
)

// BacktestService handles backtest operations
type BacktestService struct {
	backtestRepo   *repository.BacktestRepository
	marketDataRepo *repository.MarketDataRepository
	strategyClient *client.StrategyClient
	backtestClient *client.BacktestClient
	logger         *zap.Logger
}

// NewBacktestService creates a new backtest service
func NewBacktestService(
	backtestRepo *repository.BacktestRepository,
	marketDataRepo *repository.MarketDataRepository,
	strategyClient *client.StrategyClient,
	logger *zap.Logger,
) *BacktestService {
	// Get backtest service URL from environment or use default
	backtestServiceURL := os.Getenv("BACKTEST_SERVICE_URL")
	if backtestServiceURL == "" {
		backtestServiceURL = "http://backtest-service:5000"
	}

	// Create backtest client
	backtestClient := client.NewBacktestClient(backtestServiceURL, logger)

	return &BacktestService{
		backtestRepo:   backtestRepo,
		marketDataRepo: marketDataRepo,
		strategyClient: strategyClient,
		backtestClient: backtestClient,
		logger:         logger,
	}
}

// CreateBacktest creates a new backtest for a strategy
func (s *BacktestService) CreateBacktest(
	ctx context.Context,
	request *model.BacktestRequest,
	userID int,
	token string,
) (int, error) {
	// Validate date range
	if request.EndDate.Before(request.StartDate) {
		return 0, errors.New("end date must be after start date")
	}

	// Get strategy details
	strategy, err := s.strategyClient.GetStrategy(ctx, request.StrategyID, token)
	if err != nil {
		return 0, fmt.Errorf("failed to get strategy details: %w", err)
	}

	if strategy == nil {
		return 0, errors.New("strategy not found")
	}

	// Verify data availability for all symbols
	for _, symbolID := range request.SymbolIDs {
		// Check if there's data available for the requested symbol and timeframe
		hasData, err := s.marketDataRepo.HasData(ctx, symbolID, request.Timeframe)
		if err != nil {
			return 0, err
		}

		if !hasData {
			return 0, fmt.Errorf("no market data available for symbol ID %d with timeframe %s",
				symbolID, request.Timeframe)
		}

		// Check data range
		startDate, endDate, err := s.marketDataRepo.GetDataRange(ctx, symbolID, request.Timeframe)
		if err != nil {
			return 0, err
		}

		if request.StartDate.Before(startDate) || request.EndDate.After(endDate) {
			return 0, fmt.Errorf("requested date range (%s to %s) is outside available data range for symbol ID %d (%s to %s)",
				request.StartDate.Format("2006-01-02"),
				request.EndDate.Format("2006-01-02"),
				symbolID,
				startDate.Format("2006-01-02"),
				endDate.Format("2006-01-02"))
		}
	}

	// Use strategy version from request or default to latest version
	strategyVersion := request.StrategyVersion
	if strategyVersion == 0 {
		strategyVersion = strategy.Version
	}

	// Set default name if not provided
	name := request.Name
	if name == "" {
		name = strategy.Name + " Backtest"
	}

	// Create backtest using repository function
	backtestID, err := s.backtestRepo.CreateBacktest(
		ctx,
		userID,
		request.StrategyID,
		strategyVersion,
		name,
		request.Description,
		request.Timeframe,
		request.StartDate,
		request.EndDate,
		request.InitialCapital,
		request.SymbolIDs,
	)
	if err != nil {
		return 0, err
	}

	// Start backtest in the background
	go s.runBacktest(backtestID, request, userID, token)

	return backtestID, nil
}

// runBacktest executes a backtest in the background
func (s *BacktestService) runBacktest(
	backtestID int,
	request *model.BacktestRequest,
	userID int,
	token string,
) {
	// Create a new context for background processing
	ctx := context.Background()

	// Get strategy structure (either latest or specific version)
	var strategyStructure json.RawMessage
	var strategyVersion int

	if request.StrategyVersion > 0 {
		// Get specific version
		version, err := s.strategyClient.GetStrategyVersion(
			ctx,
			request.StrategyID,
			request.StrategyVersion,
			token,
		)
		if err != nil {
			s.failBacktest(ctx, backtestID, fmt.Sprintf("Failed to get strategy version: %v", err))
			return
		}
		strategyStructure = version.Structure
		strategyVersion = version.Version
	} else {
		// Get latest version
		strategy, err := s.strategyClient.GetStrategy(ctx, request.StrategyID, token)
		if err != nil {
			s.failBacktest(ctx, backtestID, fmt.Sprintf("Failed to get strategy: %v", err))
			return
		}
		strategyStructure = strategy.Structure
		strategyVersion = strategy.Version
	}

	// Validate strategy structure
	valid, message, err := s.backtestClient.ValidateStrategy(ctx, strategyStructure)
	if err != nil {
		s.failBacktest(ctx, backtestID, fmt.Sprintf("Failed to validate strategy: %v", err))
		return
	}

	if !valid {
		s.failBacktest(ctx, backtestID, fmt.Sprintf("Strategy validation failed: %s", message))
		return
	}

	// Log strategy information
	s.logger.Debug("Running backtest with validated strategy",
		zap.Int("backtestID", backtestID),
		zap.Int("strategyID", request.StrategyID),
		zap.Int("strategyVersion", strategyVersion))

	// Process each symbol in the backtest
	for _, symbolID := range request.SymbolIDs {
		// Find the run ID for this symbol
		runID, err := s.backtestRepo.GetBacktestRunIDBySymbol(ctx, backtestID, symbolID)
		if err != nil {
			s.logger.Error("Failed to find backtest run ID",
				zap.Error(err),
				zap.Int("backtestID", backtestID),
				zap.Int("symbolID", symbolID))
			continue
		}

		// Update run status to 'running'
		success, err := s.backtestRepo.UpdateBacktestRunStatus(ctx, runID, "running")
		if err != nil || !success {
			s.logger.Error("Failed to update backtest run status",
				zap.Error(err),
				zap.Int("runID", runID))
			continue
		}

		// Get market data for backtesting
		candles, err := s.marketDataRepo.GetCandles(
			ctx,
			symbolID,
			request.Timeframe,
			&request.StartDate,
			&request.EndDate,
			nil,
		)
		if err != nil {
			// Mark this run as failed
			s.backtestRepo.UpdateBacktestRunStatus(ctx, runID, "failed")
			s.logger.Error("Failed to get market data",
				zap.Error(err),
				zap.Int("symbolID", symbolID))
			continue
		}

		if len(candles) == 0 {
			// Mark this run as failed
			s.backtestRepo.UpdateBacktestRunStatus(ctx, runID, "failed")
			s.logger.Error("No market data available",
				zap.Int("symbolID", symbolID))
			continue
		}

		// Prepare parameters for the backtest
		backtestParams := map[string]interface{}{
			"symbol_id":       symbolID,
			"initial_capital": request.InitialCapital,
			"market_type":     "spot",  // Default to spot trading
			"leverage":        1.0,     // Default leverage (1.0 means no leverage)
			"commission_rate": 0.1,     // Default commission rate (0.1%)
			"slippage_rate":   0.05,    // Default slippage rate (0.05%)
			"position_sizing": "fixed", // Default position sizing strategy
			"allow_short":     false,   // Default to long-only for spot trading
			// Additional risk management parameters can be added here
		}

		// Run the backtest using the Python service
		result, err := s.backtestClient.RunBacktest(ctx, candles, strategyStructure, backtestParams)
		if err != nil {
			s.logger.Error("Failed to execute backtest",
				zap.Error(err),
				zap.Int("symbolID", symbolID),
				zap.Int("runID", runID))

			// Mark this run as failed
			s.backtestRepo.UpdateBacktestRunStatus(ctx, runID, "failed")
			continue
		}

		// Convert backtesting results to our model
		metrics := &model.BacktestResults{
			TotalTrades:      result.Metrics.TotalTrades,
			WinningTrades:    result.Metrics.WinningTrades,
			LosingTrades:     result.Metrics.LosingTrades,
			ProfitFactor:     result.Metrics.ProfitFactor,
			SharpeRatio:      result.Metrics.SharpeRatio,
			MaxDrawdown:      result.Metrics.MaxDrawdown,
			FinalCapital:     result.Metrics.FinalCapital,
			TotalReturn:      result.Metrics.TotalReturn,
			AnnualizedReturn: result.Metrics.AnnualizedReturn,
		}

		// Convert results JSON
		resultsJSON, err := json.Marshal(map[string]interface{}{
			"equity_curve":  result.EquityCurve,
			"equity_times":  result.EquityTimes,
			"trades_count":  len(result.Trades),
			"average_trade": result.Metrics.AverageTrade,
			"average_win":   result.Metrics.AverageWin,
			"average_loss":  result.Metrics.AverageLoss,
			"largest_win":   result.Metrics.LargestWin,
			"largest_loss":  result.Metrics.LargestLoss,
			"win_rate":      result.Metrics.WinRate,
		})
		if err != nil {
			s.logger.Error("Failed to marshal results JSON", zap.Error(err))
		} else {
			raw := json.RawMessage(resultsJSON)
			metrics.ResultsJSON = raw
		}

		// Save backtest results
		resultID, err := s.backtestRepo.SaveBacktestResults(ctx, runID, metrics)
		if err != nil {
			s.logger.Error("Failed to save backtest results",
				zap.Error(err),
				zap.Int("runID", runID))
			continue
		}

		s.logger.Info("Backtest results saved successfully",
			zap.Int("runID", runID),
			zap.Int("resultID", resultID),
			zap.Int("totalTrades", result.Metrics.TotalTrades),
			zap.Float64("totalReturn", result.Metrics.TotalReturn))

		// Save all trades from the backtest
		s.saveTrades(ctx, runID, symbolID, result.Trades)
	}

	// Notify the Strategy Service that the backtest is complete
	err = s.strategyClient.NotifyBacktestComplete(
		ctx,
		backtestID,
		request.StrategyID,
		userID,
		"completed",
	)
	if err != nil {
		s.logger.Warn("Failed to notify strategy service of backtest completion",
			zap.Error(err),
			zap.Int("backtestID", backtestID))
	}
}

// failBacktest marks a backtest as failed with an error message
func (s *BacktestService) failBacktest(ctx context.Context, backtestID int, errorMessage string) {
	// Update all runs for this backtest to 'failed'
	err := s.backtestRepo.UpdateBacktestRunsStatusBulk(ctx, backtestID, "failed")
	if err != nil {
		s.logger.Error("Failed to mark backtest runs as failed",
			zap.Error(err),
			zap.Int("backtestID", backtestID))
	}

	// Update backtest status to 'failed'
	err = s.backtestRepo.UpdateBacktestStatus(ctx, backtestID, "failed", errorMessage)
	if err != nil {
		s.logger.Error("Failed to mark backtest as failed",
			zap.Error(err),
			zap.Int("backtestID", backtestID))
	}
}

// saveTrades saves all trades from a backtest result
func (s *BacktestService) saveTrades(
	ctx context.Context,
	runID int,
	symbolID int,
	trades []model.BacktestTrade,
) {
	s.logger.Info("Saving trades from backtest result",
		zap.Int("runID", runID),
		zap.Int("symbolID", symbolID),
		zap.Int("tradeCount", len(trades)))

	// Process each trade
	for _, trade := range trades {
		// Parse entry time
		var entryTime time.Time
		var err error

		if t, err := time.Parse(time.RFC3339, trade.EntryTime.Format(time.RFC3339)); err == nil {
			entryTime = t
		} else if t, err := time.Parse("2006-01-02T15:04:05", trade.EntryTime.Format(time.RFC3339)); err == nil {
			entryTime = t
		} else {
			s.logger.Error("Failed to parse entry time, using current time",
				zap.Error(err),
				zap.String("entryTime", trade.EntryTime.Format(time.RFC3339)))
			entryTime = time.Now()
		}

		// Parse exit time if present
		var exitTime *time.Time
		if trade.ExitTime != nil {
			t, err := time.Parse(time.RFC3339, trade.ExitTime.Format(time.RFC3339))
			if err != nil {
				s.logger.Error("Failed to parse exit time",
					zap.Error(err),
					zap.String("exitTime", trade.ExitTime.Format(time.RFC3339)))
			} else {
				exitTime = &t
			}
		}

		// Create trade record
		dbTrade := &model.BacktestTrade{
			BacktestRunID:     runID,
			SymbolID:          symbolID,
			EntryTime:         entryTime,
			ExitTime:          exitTime,
			PositionType:      trade.PositionType,
			EntryPrice:        trade.EntryPrice,
			ExitPrice:         trade.ExitPrice,
			Quantity:          trade.Quantity,
			ProfitLoss:        trade.ProfitLoss,
			ProfitLossPercent: trade.ProfitLossPercent,
			ExitReason:        trade.ExitReason,
		}

		// Save to database
		_, err = s.backtestRepo.AddBacktestTrade(ctx, dbTrade)
		if err != nil {
			s.logger.Error("Failed to save trade",
				zap.Error(err),
				zap.Int("runID", runID),
				zap.String("positionType", trade.PositionType))
		}
	}
}

// GetBacktest retrieves a backtest by ID
func (s *BacktestService) GetBacktest(
	ctx context.Context,
	backtestID int,
	userID int,
) (*model.BacktestDetails, error) {
	// Get backtest details using SQL function
	backtest, err := s.backtestRepo.GetBacktest(ctx, backtestID)
	if err != nil {
		return nil, err
	}

	// Check user access (we must do this in code since the SQL function doesn't have this check)
	backtestUserID, err := s.backtestRepo.GetBacktestUserID(ctx, backtestID)
	if err != nil {
		return nil, err
	}

	if backtestUserID != userID {
		return nil, errors.New("access denied")
	}

	return backtest, nil
}

// ListBacktests lists backtests for a user
func (s *BacktestService) ListBacktests(
	ctx context.Context,
	userID int,
	page int,
	limit int,
) ([]model.BacktestSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Get backtests for the user
	backtests, err := s.backtestRepo.GetBacktestsByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	total, err := s.backtestRepo.CountUserBacktests(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return backtests, total, nil
}

// DeleteBacktest deletes a backtest
func (s *BacktestService) DeleteBacktest(
	ctx context.Context,
	backtestID int,
	userID int,
) error {
	// Use SQL function to delete the backtest
	success, err := s.backtestRepo.DeleteBacktest(ctx, userID, backtestID)
	if err != nil {
		return err
	}

	if !success {
		return errors.New("backtest not found or not owned by user")
	}

	return nil
}

// UpdateBacktestRunStatus updates the status of a backtest run
func (s *BacktestService) UpdateBacktestRunStatus(
	ctx context.Context,
	runID int,
	status string,
) (bool, error) {
	return s.backtestRepo.UpdateBacktestRunStatus(ctx, runID, status)
}

// SaveBacktestResults saves results for a backtest run
func (s *BacktestService) SaveBacktestResults(
	ctx context.Context,
	runID int,
	results *model.BacktestResults,
) (int, error) {
	return s.backtestRepo.SaveBacktestResults(ctx, runID, results)
}

// AddBacktestTrade adds a trade to a backtest run
func (s *BacktestService) AddBacktestTrade(
	ctx context.Context,
	trade *model.BacktestTrade,
) (int, error) {
	return s.backtestRepo.AddBacktestTrade(ctx, trade)
}

// GetBacktestTrades retrieves trades for a backtest run
func (s *BacktestService) GetBacktestTrades(
	ctx context.Context,
	runID int,
	limit int,
	offset int,
) ([]model.BacktestTrade, error) {
	return s.backtestRepo.GetBacktestTrades(ctx, runID, limit, offset)
}

// ProcessQueuedBacktests processes queued backtests
func (s *BacktestService) ProcessQueuedBacktests(
	ctx context.Context,
	limit int,
) (int, error) {
	// Get queued backtests
	backtests, err := s.backtestRepo.GetQueuedBacktests(ctx, limit)
	if err != nil {
		return 0, err
	}

	processedCount := 0

	// Process each backtest
	for _, backtest := range backtests {
		// Extract the necessary information to create a backtest request
		// We need to query for additional details since the summary doesn't have everything
		details, err := s.backtestRepo.GetBacktestDetails(ctx, backtest.BacktestID)
		if err != nil {
			s.logger.Error("Failed to get backtest details",
				zap.Error(err),
				zap.Int("backtestID", backtest.BacktestID))
			continue
		}

		if details == nil {
			s.logger.Error("Backtest not found",
				zap.Int("backtestID", backtest.BacktestID))
			continue
		}

		// Get the symbol IDs for this backtest
		symbolIDs, err := s.backtestRepo.GetBacktestSymbolIDs(ctx, backtest.BacktestID)
		if err != nil {
			s.logger.Error("Failed to get symbol IDs",
				zap.Error(err),
				zap.Int("backtestID", backtest.BacktestID))
			continue
		}

		// Create a backtest request
		request := &model.BacktestRequest{
			StrategyID:      details.StrategyID,
			StrategyVersion: details.StrategyVersion,
			Name:            backtest.Name,
			Timeframe:       details.Timeframe,
			SymbolIDs:       symbolIDs,
			StartDate:       details.StartDate,
			EndDate:         details.EndDate,
			InitialCapital:  details.InitialCapital,
		}

		// Run backtest in background
		go s.runBacktest(backtest.BacktestID, request, details.UserID, "")

		processedCount++
	}

	return processedCount, nil
}

// CheckBacktestServiceHealth checks if the backtesting service is healthy
func (s *BacktestService) CheckBacktestServiceHealth(ctx context.Context) (bool, error) {
	return s.backtestClient.CheckHealth(ctx)
}
