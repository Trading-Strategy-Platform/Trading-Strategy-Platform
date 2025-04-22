package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
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

// CreateBacktest creates a new backtest and queues it for processing
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

		// Convert timestamps to date-only comparison by truncating time parts
		requestStartDay := time.Date(request.StartDate.Year(), request.StartDate.Month(), request.StartDate.Day(), 0, 0, 0, 0, time.UTC)
		requestEndDay := time.Date(request.EndDate.Year(), request.EndDate.Month(), request.EndDate.Day(), 23, 59, 59, 999999999, time.UTC)
		availableStartDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
		availableEndDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, time.UTC)

		// Add a small buffer (1 day) to account for potential timezone differences
		if requestStartDay.AddDate(0, 0, -1).After(availableStartDay) || requestEndDay.AddDate(0, 0, 1).Before(availableEndDay) {
			return 0, fmt.Errorf("requested date range (%s to %s) is outside available data range for symbol ID %d (%s to %s)",
				requestStartDay.Format("2006-01-02"),
				requestEndDay.Format("2006-01-02"),
				symbolID,
				availableStartDay.Format("2006-01-02"),
				availableEndDay.Format("2006-01-02"))
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

// GetBacktest retrieves a backtest by ID with access control
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

// ListBacktests lists backtests for a user with filtering, sorting, and pagination
func (s *BacktestService) ListBacktests(
	ctx context.Context,
	userID int,
	searchTerm string,
	status string,
	sortBy string,
	sortDirection string,
	page int,
	limit int,
) ([]model.BacktestSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Validate sort field
	validSortFields := map[string]bool{
		"name":        true,
		"created_at":  true,
		"status":      true,
		"strategy_id": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at" // Default sort by creation date
	}

	// Validate sort direction
	sortDirection = noramlizeSortDirection(sortDirection)

	// Calculate offset
	offset := (page - 1) * limit

	// Get total count
	total, err := s.backtestRepo.CountBacktests(ctx, userID, searchTerm, status)
	if err != nil {
		return nil, 0, err
	}

	// Get backtests for the user
	backtests, err := s.backtestRepo.GetBacktestsByUser(
		ctx,
		userID,
		searchTerm,
		status,
		sortBy,
		sortDirection,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, err
	}

	return backtests, total, nil
}

// DeleteBacktest deletes a backtest with owner verification
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

// GetBacktestTrades retrieves trades for a backtest run with sorting and pagination
func (s *BacktestService) GetBacktestTrades(
	ctx context.Context,
	runID int,
	sortBy string,
	sortDirection string,
	limit int,
	offset int,
) ([]model.BacktestTrade, int, error) {
	// Validate sort field
	validSortFields := map[string]bool{
		"entry_time":          true,
		"exit_time":           true,
		"position_type":       true,
		"profit_loss":         true,
		"profit_loss_percent": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "entry_time" // Default sort by entry time
	}

	// Validate sort direction
	sortDirection = noramlizeSortDirection(sortDirection)

	// Get total count
	total, err := s.backtestRepo.CountBacktestTrades(ctx, runID)
	if err != nil {
		return nil, 0, err
	}

	// Get trades
	trades, err := s.backtestRepo.GetBacktestTrades(ctx, runID, sortBy, sortDirection, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return trades, total, nil
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

// GetBacktestRuns retrieves all runs for a backtest with sorting and pagination
func (s *BacktestService) GetBacktestRuns(
	ctx context.Context,
	backtestID int,
	sortBy string,
	sortDirection string,
	page int,
	limit int,
) ([]struct {
	ID          int        `json:"id"`
	BacktestID  int        `json:"backtest_id"`
	SymbolID    int        `json:"symbol_id"`
	Symbol      string     `json:"symbol"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}, int, error) {
	// Validate sort field
	validSortFields := map[string]bool{
		"id":           true,
		"status":       true,
		"created_at":   true,
		"completed_at": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at" // Default sort by creation date
	}

	// Validate sort direction
	sortDirection = noramlizeSortDirection(sortDirection)

	// Calculate offset
	offset := (page - 1) * limit

	// Get total count
	total, err := s.backtestRepo.CountBacktestRuns(ctx, backtestID)
	if err != nil {
		return nil, 0, err
	}

	// Get runs
	runs, err := s.backtestRepo.GetBacktestRuns(ctx, backtestID, sortBy, sortDirection, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Convert DB result to API result
	result := make([]struct {
		ID          int        `json:"id"`
		BacktestID  int        `json:"backtest_id"`
		SymbolID    int        `json:"symbol_id"`
		Symbol      string     `json:"symbol"`
		Status      string     `json:"status"`
		CreatedAt   time.Time  `json:"created_at"`
		CompletedAt *time.Time `json:"completed_at,omitempty"`
	}, len(runs))

	for i, run := range runs {
		result[i] = struct {
			ID          int        `json:"id"`
			BacktestID  int        `json:"backtest_id"`
			SymbolID    int        `json:"symbol_id"`
			Symbol      string     `json:"symbol"`
			Status      string     `json:"status"`
			CreatedAt   time.Time  `json:"created_at"`
			CompletedAt *time.Time `json:"completed_at,omitempty"`
		}{
			ID:          run.ID,
			BacktestID:  run.BacktestID,
			SymbolID:    run.SymbolID,
			Symbol:      run.Symbol,
			Status:      run.Status,
			CreatedAt:   run.CreatedAt,
			CompletedAt: run.CompletedAt,
		}
	}

	return result, total, nil
}

// CheckBacktestServiceHealth checks if the backtesting service is healthy
func (s *BacktestService) CheckBacktestServiceHealth(ctx context.Context) (bool, error) {
	return s.backtestClient.CheckHealth(ctx)
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

	// Added safety check for nil services
	if s.strategyClient == nil {
		s.logger.Error("Strategy client is nil",
			zap.Int("backtestID", backtestID),
			zap.Int("strategyID", request.StrategyID))
		s.failBacktest(ctx, backtestID, "Internal service error: strategy client unavailable")
		return
	}

	if s.backtestClient == nil {
		s.logger.Error("Backtest client is nil",
			zap.Int("backtestID", backtestID),
			zap.Int("strategyID", request.StrategyID))
		s.failBacktest(ctx, backtestID, "Internal service error: backtest client unavailable")
		return
	}

	// Get strategy structure (either latest or specific version)
	var strategyStructure json.RawMessage
	var strategyVersion int
	var err error // Declare err variable upfront to avoid redeclaration issues

	if request.StrategyVersion > 0 {
		// Get specific version
		var version *struct {
			ID        int             `json:"id"`
			Version   int             `json:"version"`
			Structure json.RawMessage `json:"structure"`
		}
		version, err = s.strategyClient.GetStrategyVersion(
			ctx,
			request.StrategyID,
			request.StrategyVersion,
			token,
		)
		if err != nil {
			s.failBacktest(ctx, backtestID, fmt.Sprintf("Failed to get strategy version: %v", err))
			return
		}
		if version == nil {
			s.failBacktest(ctx, backtestID, "Strategy version not found")
			return
		}
		strategyStructure = version.Structure
		strategyVersion = version.Version
	} else {
		// Get latest version
		var strategy *struct {
			ID        int             `json:"id"`
			Name      string          `json:"name"`
			Version   int             `json:"version"`
			Structure json.RawMessage `json:"structure"`
		}
		strategy, err = s.strategyClient.GetStrategy(ctx, request.StrategyID, token)
		if err != nil {
			s.failBacktest(ctx, backtestID, fmt.Sprintf("Failed to get strategy: %v", err))
			return
		}
		if strategy == nil {
			s.failBacktest(ctx, backtestID, "Strategy not found")
			return
		}
		strategyStructure = strategy.Structure
		strategyVersion = strategy.Version
	}

	// Make sure the strategy structure is not nil
	if strategyStructure == nil {
		s.failBacktest(ctx, backtestID, "Strategy structure is empty")
		return
	}

	// Validate strategy structure
	var valid bool
	var message string
	valid, message, err = s.backtestClient.ValidateStrategy(ctx, strategyStructure)
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
		var runID int
		runID, err = s.backtestRepo.GetBacktestRunIDBySymbol(ctx, backtestID, symbolID)
		if err != nil {
			s.logger.Error("Failed to find backtest run ID",
				zap.Error(err),
				zap.Int("backtestID", backtestID),
				zap.Int("symbolID", symbolID))
			continue
		}

		// Update run status to 'running'
		var success bool
		success, err = s.backtestRepo.UpdateBacktestRunStatus(ctx, runID, "running")
		if err != nil || !success {
			s.logger.Error("Failed to update backtest run status",
				zap.Error(err),
				zap.Int("runID", runID))
			continue
		}

		// Use the /backtest/db endpoint which will fetch data directly from the database
		backtestRequest := map[string]interface{}{
			"symbol_id":       symbolID,
			"timeframe":       request.Timeframe,
			"start_date":      request.StartDate.Format(time.RFC3339),
			"end_date":        request.EndDate.Format(time.RFC3339),
			"strategy":        strategyStructure,
			"backtest_run_id": runID,
			"params": map[string]interface{}{
				"symbol_id":       symbolID,
				"initial_capital": request.InitialCapital,
				"market_type":     "spot",  // Default to spot trading
				"leverage":        1.0,     // Default leverage (1.0 means no leverage)
				"commission_rate": 0.1,     // Default commission rate (0.1%)
				"slippage_rate":   0.05,    // Default slippage rate (0.05%)
				"position_sizing": "fixed", // Default position sizing strategy
				"allow_short":     false,   // Default to long-only for spot trading
			},
		}

		// Create the request body
		var jsonData []byte
		jsonData, err = json.Marshal(backtestRequest)
		if err != nil {
			s.failBacktest(ctx, backtestID, fmt.Sprintf("Failed to marshal backtest request: %v", err))
			continue
		}

		// Create the request
		url := fmt.Sprintf("%s/backtest/db", s.backtestClient.BaseURL())
		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
		if err != nil {
			s.failBacktest(ctx, backtestID, fmt.Sprintf("Failed to create request: %v", err))
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		// Execute the request
		s.logger.Info("Sending backtest request with direct DB access",
			zap.String("url", url),
			zap.Int("symbolID", symbolID),
			zap.Int("runID", runID))

		client := &http.Client{
			Timeout: 5 * time.Minute, // Extended timeout for backtesting
		}
		var resp *http.Response
		resp, err = client.Do(req)
		if err != nil {
			s.logger.Error("Failed to send request to backtesting service",
				zap.Error(err),
				zap.Int("symbolID", symbolID),
				zap.Int("runID", runID))

			// Mark this run as failed
			s.backtestRepo.UpdateBacktestRunStatus(ctx, runID, "failed")
			continue
		}
		defer resp.Body.Close()

		// Check for error status
		if resp.StatusCode != http.StatusOK {
			var errorResp struct {
				Error string `json:"error"`
			}
			decodeErr := json.NewDecoder(resp.Body).Decode(&errorResp)
			if decodeErr != nil {
				s.logger.Error("Failed to decode error response",
					zap.Error(decodeErr),
					zap.Int("statusCode", resp.StatusCode))
			}

			// Mark this run as failed
			s.backtestRepo.UpdateBacktestRunStatus(ctx, runID, "failed")
			s.logger.Error("Backtest service error",
				zap.String("error", errorResp.Error),
				zap.Int("symbolID", symbolID),
				zap.Int("runID", runID))
			continue
		}

		// Parse response
		var result model.BacktestResult
		decodeErr := json.NewDecoder(resp.Body).Decode(&result)
		if decodeErr != nil {
			s.logger.Error("Failed to decode backtest response",
				zap.Error(decodeErr),
				zap.Int("symbolID", symbolID),
				zap.Int("runID", runID))

			// Mark this run as failed
			s.backtestRepo.UpdateBacktestRunStatus(ctx, runID, "failed")
			continue
		}

		s.logger.Info("Backtest completed successfully",
			zap.Int("runID", runID),
			zap.Int("symbolID", symbolID),
			zap.Int("totalTrades", result.Metrics.TotalTrades),
			zap.Float64("totalReturn", result.Metrics.TotalReturn))

		// No need to save trades or update status, as the backtest service has
		// already saved everything directly to the database
	}

	// Notify the Strategy Service that the backtest is complete
	// Using = instead of := since err was already declared
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
	if s.backtestRepo == nil {
		if s.logger != nil {
			s.logger.Error("Cannot fail backtest: backtest repository is nil",
				zap.Int("backtestID", backtestID),
				zap.String("errorMessage", errorMessage))
		}
		return
	}

	// Update all runs for this backtest to 'failed'
	err := s.backtestRepo.UpdateBacktestRunsStatusBulk(ctx, backtestID, "failed")
	if err != nil && s.logger != nil {
		s.logger.Error("Failed to mark backtest runs as failed",
			zap.Error(err),
			zap.Int("backtestID", backtestID))
	}

	// Update backtest status to 'failed'
	err = s.backtestRepo.UpdateBacktestStatus(ctx, backtestID, "failed", errorMessage)
	if err != nil && s.logger != nil {
		s.logger.Error("Failed to mark backtest as failed",
			zap.Error(err),
			zap.Int("backtestID", backtestID))
	}
}

// Helper function to normalize sort direction
func noramlizeSortDirection(direction string) string {
	direction = strings.ToUpper(direction)
	if direction != "ASC" && direction != "DESC" {
		return "DESC" // Default to descending
	}
	return direction
}
