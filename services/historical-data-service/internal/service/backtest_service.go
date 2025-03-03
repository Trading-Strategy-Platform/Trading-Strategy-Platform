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

	// Check if there's data available for the requested symbol and timeframe
	hasData, err := s.marketDataRepo.HasData(ctx, request.SymbolID, request.TimeframeID)
	if err != nil {
		return 0, err
	}

	if !hasData {
		return 0, errors.New("no market data available for the requested symbol and timeframe")
	}

	// Check if data range is available
	startDate, endDate, err := s.marketDataRepo.GetDataRange(ctx, request.SymbolID, request.TimeframeID)
	if err != nil {
		return 0, err
	}

	if request.StartDate.Before(startDate) || request.EndDate.After(endDate) {
		return 0, fmt.Errorf("requested date range (%s to %s) is outside available data range (%s to %s)",
			request.StartDate.Format("2006-01-02"),
			request.EndDate.Format("2006-01-02"),
			startDate.Format("2006-01-02"),
			endDate.Format("2006-01-02"))
	}

	// Get strategy details
	strategy, err := s.strategyClient.GetStrategy(ctx, request.StrategyID, token)
	if err != nil {
		return 0, fmt.Errorf("failed to get strategy details: %w", err)
	}

	if strategy == nil {
		return 0, errors.New("strategy not found")
	}

	// Create backtest record
	backtest := &model.Backtest{
		UserID:          userID,
		StrategyID:      request.StrategyID,
		StrategyName:    strategy.Name,
		StrategyVersion: request.StrategyVersion,
		SymbolID:        request.SymbolID,
		TimeframeID:     request.TimeframeID,
		StartDate:       request.StartDate,
		EndDate:         request.EndDate,
		InitialCapital:  request.InitialCapital,
		Status:          model.BacktestStatusQueued,
	}

	// Insert backtest
	backtestID, err := s.backtestRepo.CreateBacktest(ctx, backtest)
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

	// Mark backtest as running
	err := s.backtestRepo.UpdateBacktestStatus(ctx, backtestID, model.BacktestStatusRunning)
	if err != nil {
		s.logger.Error("Failed to update backtest status",
			zap.Error(err),
			zap.Int("backtestID", backtestID))
		return
	}

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

	// Use strategyStructure - log information about it
	if len(strategyStructure) > 0 {
		s.logger.Info("Retrieved strategy structure for backtest",
			zap.Int("backtest_id", backtestID),
			zap.Int("bytes_length", len(strategyStructure)))
	} else {
		s.logger.Warn("Empty strategy structure for backtest",
			zap.Int("backtest_id", backtestID))
	}

	// Get market data for backtesting
	marketData, err := s.marketDataRepo.GetMarketData(
		ctx,
		request.SymbolID,
		request.TimeframeID,
		&request.StartDate,
		&request.EndDate,
		nil,
	)
	if err != nil {
		s.failBacktest(ctx, backtestID, fmt.Sprintf("Failed to get market data: %v", err))
		return
	}

	if len(marketData) == 0 {
		s.failBacktest(ctx, backtestID, "No market data available for the requested period")
		return
	}

	// TODO: Implement the actual backtest execution logic
	// This would involve processing the strategy rules against the market data
	// For now, we'll simulate a backtest with dummy results

	// Simulate processing time
	time.Sleep(2 * time.Second)

	// Generate dummy results
	results := model.BacktestResults{
		NetProfit:          1000.50,
		ProfitFactor:       1.5,
		TotalTrades:        25,
		WinningTrades:      15,
		LosingTrades:       10,
		WinRate:            60.0,
		MaxDrawdown:        250.30,
		MaxDrawdownPercent: 5.5,
		CAGR:               15.2,
		SharpeRatio:        1.8,
		SortinoRatio:       2.2,
		Trades: []model.Trade{
			{
				EntryTime:     time.Date(2023, 1, 5, 10, 0, 0, 0, time.UTC),
				ExitTime:      time.Date(2023, 1, 7, 14, 30, 0, 0, time.UTC),
				EntryPrice:    100.50,
				ExitPrice:     105.75,
				Direction:     "long",
				Quantity:      10,
				ProfitLoss:    52.50,
				ProfitLossPct: 5.22,
			},
			{
				EntryTime:     time.Date(2023, 1, 12, 11, 0, 0, 0, time.UTC),
				ExitTime:      time.Date(2023, 1, 14, 15, 45, 0, 0, time.UTC),
				EntryPrice:    107.25,
				ExitPrice:     104.50,
				Direction:     "long",
				Quantity:      10,
				ProfitLoss:    -27.50,
				ProfitLossPct: -2.56,
			},
		},
		EquityCurve: []float64{1000.0, 1025.5, 1052.5, 1040.0, 1055.0, 1025.0, 1015.0, 1055.0, 1075.0, 1100.0},
	}

	// Complete the backtest with results
	err = s.backtestRepo.CompleteBacktest(ctx, backtestID, results)
	if err != nil {
		s.logger.Error("Failed to complete backtest",
			zap.Error(err),
			zap.Int("backtestID", backtestID))
		return
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
	err := s.backtestRepo.FailBacktest(ctx, backtestID, errorMessage)
	if err != nil {
		s.logger.Error("Failed to mark backtest as failed",
			zap.Error(err),
			zap.Int("backtestID", backtestID))
	}
}

// GetBacktest retrieves a backtest by ID
func (s *BacktestService) GetBacktest(ctx context.Context, id int, userID int) (*model.Backtest, error) {
	backtest, err := s.backtestRepo.GetBacktest(ctx, id)
	if err != nil {
		return nil, err
	}

	if backtest == nil {
		return nil, errors.New("backtest not found")
	}

	// Check user access
	if backtest.UserID != userID {
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
) ([]model.Backtest, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	return s.backtestRepo.GetBacktestsByUser(ctx, userID, page, limit)
}

// DeleteBacktest deletes a backtest
func (s *BacktestService) DeleteBacktest(ctx context.Context, id int, userID int) error {
	return s.backtestRepo.DeleteBacktest(ctx, id, userID)
}

// ProcessQueuedBacktests processes queued backtests
func (s *BacktestService) ProcessQueuedBacktests(ctx context.Context, limit int) (int, error) {
	// Get queued backtests
	backtests, err := s.backtestRepo.GetQueuedBacktests(ctx, limit)
	if err != nil {
		return 0, err
	}

	count := 0

	// Process each backtest
	for _, backtest := range backtests {
		// Update status to running
		err := s.backtestRepo.UpdateBacktestStatus(ctx, backtest.ID, model.BacktestStatusRunning)
		if err != nil {
			s.logger.Error("Failed to update backtest status",
				zap.Error(err),
				zap.Int("backtestID", backtest.ID))
			continue
		}

		// Create backtest request
		request := &model.BacktestRequest{
			StrategyID:      backtest.StrategyID,
			StrategyVersion: backtest.StrategyVersion,
			SymbolID:        backtest.SymbolID,
			TimeframeID:     backtest.TimeframeID,
			StartDate:       backtest.StartDate,
			EndDate:         backtest.EndDate,
			InitialCapital:  backtest.InitialCapital,
		}

		// Run backtest in background
		go s.runBacktest(backtest.ID, request, backtest.UserID, "")

		count++
	}

	return count, nil
}

// RunBacktest is a placeholder for the RunBacktest method
func (s *BacktestService) RunBacktest(ctx context.Context, backtest *model.Backtest) error {
	// ... existing code ...

	// Get strategy structure (if needed)
	var strategyStructure json.RawMessage
	// Actually get and use strategy structure from strategy client
	if backtest.StrategyVersion > 0 {
		version, err := s.strategyClient.GetStrategyVersion(ctx, backtest.StrategyID, backtest.StrategyVersion, "")
		if err == nil && version != nil {
			strategyStructure = version.Structure
		}
	} else {
		strategy, err := s.strategyClient.GetStrategy(ctx, backtest.StrategyID, "")
		if err == nil && strategy != nil {
			strategyStructure = strategy.Structure
		}
	}

	// Log if we have strategy structure
	if len(strategyStructure) > 0 {
		s.logger.Info("Using strategy structure for backtest", zap.Int("backtest_id", backtest.ID))
	} else {
		s.logger.Warn("Missing strategy structure for backtest", zap.Int("backtest_id", backtest.ID))
	}

	// ... rest of the implementation
	return nil
}
