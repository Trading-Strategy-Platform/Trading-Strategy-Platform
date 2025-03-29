package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	logger         *zap.Logger
}

// NewBacktestService creates a new backtest service
func NewBacktestService(
	backtestRepo *repository.BacktestRepository,
	marketDataRepo *repository.MarketDataRepository,
	strategyClient *client.StrategyClient,
	logger *zap.Logger,
) *BacktestService {
	return &BacktestService{
		backtestRepo:   backtestRepo,
		marketDataRepo: marketDataRepo,
		strategyClient: strategyClient,
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
	} else {
		// Get latest version
		strategy, err := s.strategyClient.GetStrategy(ctx, request.StrategyID, token)
		if err != nil {
			s.failBacktest(ctx, backtestID, fmt.Sprintf("Failed to get strategy: %v", err))
			return
		}
		strategyStructure = strategy.Structure
	}

	// Log strategy structure to use the variable
	s.logger.Debug("Running backtest with strategy structure",
		zap.Int("backtestID", backtestID),
		zap.String("strategyType", string(strategyStructure)[:100]+"..."))

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

		// TODO: Implement the actual backtest execution logic
		// This would involve processing the strategy rules against the market data
		// For now, we'll simulate a backtest with dummy results

		// Simulate processing time
		time.Sleep(2 * time.Second)

		// Generate dummy results
		results := &model.BacktestResults{
			TotalTrades:      25,
			WinningTrades:    15,
			LosingTrades:     10,
			ProfitFactor:     1.5,
			SharpeRatio:      1.8,
			MaxDrawdown:      5.5,
			FinalCapital:     request.InitialCapital * 1.15,
			TotalReturn:      15.0,
			AnnualizedReturn: 10.0,
			ResultsJSON:      json.RawMessage(`{"equityCurve": [1000, 1025, 1050, 1075, 1100, 1125, 1150]}`),
		}

		// Save backtest results
		resultID, err := s.backtestRepo.SaveBacktestResults(ctx, runID, results)
		if err != nil {
			s.logger.Error("Failed to save backtest results",
				zap.Error(err),
				zap.Int("runID", runID))
			continue
		}

		s.logger.Info("Backtest run completed and results saved",
			zap.Int("runID", runID),
			zap.Int("resultID", resultID))

		// Add some dummy trades
		s.addDummyTrades(ctx, runID, symbolID, &request.StartDate)
	}

	// Once all runs are processed, the update_backtest_run_status function will
	// automatically update the parent backtest status if all runs are complete

	// Notify the Strategy Service that the backtest is complete
	err := s.strategyClient.NotifyBacktestComplete(
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

// addDummyTrades adds some dummy trades for testing
func (s *BacktestService) addDummyTrades(
	ctx context.Context,
	runID int,
	symbolID int,
	startDate *time.Time,
) {
	// Add a few dummy trades
	tradeDate := startDate.AddDate(0, 0, 5)

	longTrade := &model.BacktestTrade{
		BacktestRunID:     runID,
		SymbolID:          symbolID,
		EntryTime:         tradeDate,
		ExitTime:          &[]time.Time{tradeDate.AddDate(0, 0, 2)}[0],
		PositionType:      "long",
		EntryPrice:        100.0,
		ExitPrice:         &[]float64{105.0}[0],
		Quantity:          10.0,
		ProfitLoss:        &[]float64{50.0}[0],
		ProfitLossPercent: &[]float64{5.0}[0],
		ExitReason:        &[]string{"take_profit"}[0],
	}

	_, err := s.backtestRepo.AddBacktestTrade(ctx, longTrade)
	if err != nil {
		s.logger.Error("Failed to add dummy long trade", zap.Error(err))
	}

	shortTrade := &model.BacktestTrade{
		BacktestRunID:     runID,
		SymbolID:          symbolID,
		EntryTime:         tradeDate.AddDate(0, 0, 5),
		ExitTime:          &[]time.Time{tradeDate.AddDate(0, 0, 7)}[0],
		PositionType:      "short",
		EntryPrice:        110.0,
		ExitPrice:         &[]float64{105.0}[0],
		Quantity:          10.0,
		ProfitLoss:        &[]float64{50.0}[0],
		ProfitLossPercent: &[]float64{4.5}[0],
		ExitReason:        &[]string{"take_profit"}[0],
	}

	_, err = s.backtestRepo.AddBacktestTrade(ctx, shortTrade)
	if err != nil {
		s.logger.Error("Failed to add dummy short trade", zap.Error(err))
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
