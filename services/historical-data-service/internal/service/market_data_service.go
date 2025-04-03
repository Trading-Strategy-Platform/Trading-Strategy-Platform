package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"services/historical-data-service/internal/client"
	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/repository"

	"go.uber.org/zap"
)

// MarketDataService handles market data operations
type MarketDataService struct {
	marketDataRepo *repository.MarketDataRepository
	symbolRepo     *repository.SymbolRepository
	logger         *zap.Logger
}

// NewMarketDataService creates a new market data service
func NewMarketDataService(
	marketDataRepo *repository.MarketDataRepository,
	symbolRepo *repository.SymbolRepository,
	logger *zap.Logger,
) *MarketDataService {
	return &MarketDataService{
		marketDataRepo: marketDataRepo,
		symbolRepo:     symbolRepo,
		logger:         logger,
	}
}

// GetBinanceSymbols retrieves all available symbols from Binance
func (s *MarketDataService) GetBinanceSymbols(ctx context.Context) ([]model.BinanceSymbol, error) {
	// Create a Binance client
	binanceClient := client.NewBinanceClient(s.logger)

	// Get exchange info from Binance
	exchangeInfo, err := binanceClient.GetExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}

	// Filter for active spot trading symbols
	var activeSymbols []model.BinanceSymbol
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.Status == "TRADING" && symbol.IsSpotTradingAllowed {
			activeSymbols = append(activeSymbols, symbol)
		}
	}

	return activeSymbols, nil
}

// CheckSymbolDataStatus checks if a symbol exists in the database and what date ranges are available
func (s *MarketDataService) CheckSymbolDataStatus(ctx context.Context, symbol, timeframe string) (*model.SymbolDataStatus, error) {
	// Check if symbol exists in our database
	symbols, err := s.symbolRepo.GetSymbolsByFilter(ctx, symbol, "", "")
	if err != nil {
		return nil, err
	}

	var symbolID int
	var foundSymbol bool

	// Find the exact symbol
	for _, s := range symbols {
		if s.Symbol == symbol {
			symbolID = s.ID
			foundSymbol = true
			break
		}
	}

	// If symbol doesn't exist in our database yet, create a bare status response
	if !foundSymbol {
		// Note: We'll need to create this symbol before downloading data
		return &model.SymbolDataStatus{
			Symbol:        symbol,
			SymbolID:      0,
			HasData:       false,
			AvailableData: []model.DateRange{},
			MissingData:   []model.DateRange{},
		}, nil
	}

	// Check if we have any data for this symbol and timeframe
	hasData, err := s.marketDataRepo.HasData(ctx, symbolID, timeframe)
	if err != nil {
		return nil, err
	}

	var availableData []model.DateRange
	if hasData {
		// Get the ranges of data we already have
		availableData, err = s.marketDataRepo.GetDataRanges(ctx, symbolID, timeframe)
		if err != nil {
			return nil, err
		}
	}

	// Create a full time range from the first available day to today
	var fullRange model.DateRange

	if hasData && len(availableData) > 0 {
		// Use the earliest data point we have as start
		fullRange.Start = availableData[0].Start
	} else {
		// If no data, use a reasonable default start date (e.g., 5 years ago)
		fullRange.Start = time.Now().AddDate(-5, 0, 0)
	}

	// End date is today
	fullRange.End = time.Now()

	// Calculate missing data ranges
	missingData, err := s.marketDataRepo.CalculateMissingDataRanges(
		ctx,
		symbolID,
		timeframe,
		fullRange.Start,
		fullRange.End,
	)

	if err != nil {
		return nil, err
	}

	// Return the status
	return &model.SymbolDataStatus{
		Symbol:        symbol,
		SymbolID:      symbolID,
		HasData:       hasData,
		AvailableData: availableData,
		MissingData:   missingData,
	}, nil
}

// StartBinanceDataDownload initiates a data download from Binance
func (s *MarketDataService) StartBinanceDataDownload(ctx context.Context, request *model.BinanceDownloadRequest) (int, error) {
	// Check if the symbol already exists in our database
	symbols, err := s.symbolRepo.GetSymbolsByFilter(ctx, request.Symbol, "", "")
	if err != nil {
		return 0, err
	}

	var symbolID int
	var foundSymbol bool

	// Find the exact symbol
	for _, s := range symbols {
		if s.Symbol == request.Symbol {
			symbolID = s.ID
			foundSymbol = true
			break
		}
	}

	// If symbol doesn't exist in our database yet, we need to create it
	if !foundSymbol {
		// Create a new Binance client
		binanceClient := client.NewBinanceClient(s.logger)

		// Get exchange info to get more details about the symbol
		exchangeInfo, err := binanceClient.GetExchangeInfo(ctx)
		if err != nil {
			return 0, err
		}

		var symbolInfo *model.BinanceSymbol
		for _, s := range exchangeInfo.Symbols {
			if s.Symbol == request.Symbol {
				symbolInfo = &s
				break
			}
		}

		if symbolInfo == nil {
			return 0, fmt.Errorf("symbol '%s' not found on Binance", request.Symbol)
		}

		// Create the symbol in our database
		newSymbol := &model.Symbol{
			Symbol:    symbolInfo.Symbol,
			Name:      symbolInfo.BaseAsset + "/" + symbolInfo.QuoteAsset,
			AssetType: "crypto", // Assuming all Binance symbols are crypto
			Exchange:  "Binance",
			IsActive:  true,
		}

		symbolID, err = s.symbolRepo.CreateSymbol(ctx, newSymbol)
		if err != nil {
			return 0, err
		}
	}

	// Create a download job
	jobID, err := s.marketDataRepo.CreateBinanceDownloadJob(
		ctx,
		symbolID,
		request.Symbol,
		request.Timeframe,
		request.StartDate,
		request.EndDate,
	)

	if err != nil {
		return 0, err
	}

	// Start the download process in a background goroutine
	go s.processBinanceDownload(jobID, request.Symbol, symbolID, request.Timeframe, request.StartDate, request.EndDate)

	return jobID, nil
}

// processBinanceDownload processes a Binance data download job
func (s *MarketDataService) processBinanceDownload(
	jobID int,
	symbol string,
	symbolID int,
	timeframe string,
	startDate,
	endDate time.Time,
) {
	// Create a new context for this background process
	ctx := context.Background()

	// Update job status to in_progress
	err := s.marketDataRepo.UpdateBinanceDownloadJobStatus(ctx, jobID, "in_progress", 0, "")
	if err != nil {
		s.logger.Error("Failed to update job status",
			zap.Error(err),
			zap.Int("jobID", jobID))
		return
	}

	// Create a Binance client
	binanceClient := client.NewBinanceClient(s.logger)

	// Map our timeframe to Binance interval
	interval := client.MapTimeframeToBinanceInterval(timeframe)
	if interval == "" {
		err := s.marketDataRepo.UpdateBinanceDownloadJobStatus(
			ctx,
			jobID,
			"failed",
			0,
			fmt.Sprintf("Invalid timeframe: %s", timeframe),
		)
		if err != nil {
			s.logger.Error("Failed to update job status",
				zap.Error(err),
				zap.Int("jobID", jobID))
		}
		return
	}

	// Calculate the total time range to download
	totalDuration := endDate.Sub(startDate)
	processedDuration := time.Duration(0)

	// Process in chunks
	currentStart := startDate

	// Track progress
	var progress float64
	var batchCount int

	for currentStart.Before(endDate) {
		// Check if job was cancelled
		job, err := s.marketDataRepo.GetBinanceDownloadJob(ctx, jobID)
		if err != nil {
			s.logger.Error("Failed to check job status",
				zap.Error(err),
				zap.Int("jobID", jobID))
			break
		}

		if job.Status == "cancelled" {
			s.logger.Info("Job was cancelled",
				zap.Int("jobID", jobID),
				zap.String("symbol", symbol))
			break
		}

		// Calculate chunk end time
		chunkEnd := currentStart.Add(24 * time.Hour) // Download in daily chunks
		if chunkEnd.After(endDate) {
			chunkEnd = endDate
		}

		// Fetch klines for this chunk
		klines, err := binanceClient.GetKlines(ctx, symbol, interval, &currentStart, &chunkEnd, 1000)
		if err != nil {
			// Log error and retry after a delay
			s.logger.Error("Failed to fetch klines, retrying after delay",
				zap.Error(err),
				zap.String("symbol", symbol),
				zap.String("interval", interval),
				zap.Time("startTime", currentStart),
				zap.Time("endTime", chunkEnd))

			// Wait before retrying to respect rate limits
			time.Sleep(5 * time.Second)

			// Increment retry count and potentially fail if too many retries
			continue
		}

		// If no klines were returned but we haven't reached the end date, there might be a gap
		if len(klines) == 0 && currentStart.Before(endDate) {
			// Skip ahead by the chunk size
			currentStart = chunkEnd
			continue
		}

		// Convert klines to candles
		candles := make([]model.CandleBatch, len(klines))
		for i, k := range klines {
			candles[i] = model.CandleBatch{
				SymbolID: symbolID,
				Time:     k.OpenTime,
				Open:     k.Open,
				High:     k.High,
				Low:      k.Low,
				Close:    k.Close,
				Volume:   k.Volume,
			}
		}

		// Import candles
		_, err = s.marketDataRepo.BatchImportCandles(ctx, candles)
		if err != nil {
			// Update job status to failed
			updateErr := s.marketDataRepo.UpdateBinanceDownloadJobStatus(
				ctx,
				jobID,
				"failed",
				progress,
				fmt.Sprintf("Failed to import candles: %v", err),
			)
			if updateErr != nil {
				s.logger.Error("Failed to update job status",
					zap.Error(updateErr),
					zap.Int("jobID", jobID))
			}
			return
		}

		// Update progress
		processedDuration += chunkEnd.Sub(currentStart)
		progress = float64(processedDuration) / float64(totalDuration) * 100

		// Update progress in database every 10 batches
		batchCount++
		if batchCount%10 == 0 {
			err = s.marketDataRepo.UpdateBinanceDownloadJobStatus(ctx, jobID, "in_progress", progress, "")
			if err != nil {
				s.logger.Error("Failed to update job progress",
					zap.Error(err),
					zap.Int("jobID", jobID),
					zap.Float64("progress", progress))
			}
		}

		// Move to the next chunk
		currentStart = chunkEnd

		// Sleep to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}

	// After all chunks are processed, verify completeness
	verificationStatus := s.verifyDataCompleteness(ctx, symbolID, timeframe, startDate, endDate)
	if !verificationStatus.Complete {
		// There are still gaps, try to fill them
		for i := 0; i < 3; i++ { // Try up to 3 times
			if len(verificationStatus.Gaps) == 0 {
				break
			}

			s.logger.Info("Found gaps in downloaded data, attempting to fill",
				zap.Int("jobID", jobID),
				zap.String("symbol", symbol),
				zap.Int("gapCount", len(verificationStatus.Gaps)))

			// Update status to indicate filling gaps
			err = s.marketDataRepo.UpdateBinanceDownloadJobStatus(
				ctx,
				jobID,
				"in_progress",
				95, // Show high but not complete progress
				"Filling data gaps")

			// Try to fill each gap
			for _, gap := range verificationStatus.Gaps {
				s.fillDataGap(ctx, binanceClient, symbol, interval, symbolID, gap.Start, gap.End)
			}

			// Check again for completeness
			verificationStatus = s.verifyDataCompleteness(ctx, symbolID, timeframe, startDate, endDate)
		}
	}

	// Update job status to completed
	err = s.marketDataRepo.UpdateBinanceDownloadJobStatus(ctx, jobID, "completed", 100, "")
	if err != nil {
		s.logger.Error("Failed to update job status",
			zap.Error(err),
			zap.Int("jobID", jobID))
	}

	// Update symbol data availability flag
	err = s.marketDataRepo.UpdateSymbolDataAvailability(ctx, symbolID, true)
	if err != nil {
		s.logger.Error("Failed to update symbol data availability",
			zap.Error(err),
			zap.Int("symbolID", symbolID))
	}
}

// VerificationResult represents the result of a data completeness verification
type VerificationResult struct {
	Complete bool
	Gaps     []model.DateRange
}

// verifyDataCompleteness checks if there are any gaps in the downloaded data
func (s *MarketDataService) verifyDataCompleteness(
	ctx context.Context,
	symbolID int,
	timeframe string,
	startDate time.Time,
	endDate time.Time,
) VerificationResult {
	// Get missing data ranges
	gaps, err := s.marketDataRepo.CalculateMissingDataRanges(ctx, symbolID, timeframe, startDate, endDate)
	if err != nil {
		s.logger.Error("Failed to calculate missing data ranges",
			zap.Error(err),
			zap.Int("symbolID", symbolID),
			zap.String("timeframe", timeframe))

		// If we can't verify, assume there are gaps
		return VerificationResult{Complete: false, Gaps: []model.DateRange{}}
	}

	return VerificationResult{
		Complete: len(gaps) == 0,
		Gaps:     gaps,
	}
}

// fillDataGap attempts to fill a gap in the downloaded data
func (s *MarketDataService) fillDataGap(
	ctx context.Context,
	binanceClient *client.BinanceClient,
	symbol string,
	interval string,
	symbolID int,
	startDate time.Time,
	endDate time.Time,
) {
	s.logger.Info("Attempting to fill data gap",
		zap.String("symbol", symbol),
		zap.String("interval", interval),
		zap.Time("startDate", startDate),
		zap.Time("endDate", endDate))

	// Fetch klines for this gap
	klines, err := binanceClient.GetKlines(ctx, symbol, interval, &startDate, &endDate, 1000)
	if err != nil {
		s.logger.Error("Failed to fetch klines for gap",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.String("interval", interval))
		return
	}

	if len(klines) == 0 {
		s.logger.Info("No data available for gap",
			zap.String("symbol", symbol),
			zap.String("interval", interval),
			zap.Time("startDate", startDate),
			zap.Time("endDate", endDate))
		return
	}

	// Convert klines to candles
	candles := make([]model.CandleBatch, len(klines))
	for i, k := range klines {
		candles[i] = model.CandleBatch{
			SymbolID: symbolID,
			Time:     k.OpenTime,
			Open:     k.Open,
			High:     k.High,
			Low:      k.Low,
			Close:    k.Close,
			Volume:   k.Volume,
		}
	}

	// Import candles
	_, err = s.marketDataRepo.BatchImportCandles(ctx, candles)
	if err != nil {
		s.logger.Error("Failed to import candles for gap",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.String("interval", interval))
	}
}

// GetBinanceDownloadStatus gets the status of a Binance download job
func (s *MarketDataService) GetBinanceDownloadStatus(ctx context.Context, jobID int) (*model.BinanceDownloadStatus, error) {
	job, err := s.marketDataRepo.GetBinanceDownloadJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if job == nil {
		return nil, nil
	}

	return &model.BinanceDownloadStatus{
		JobID:     job.ID,
		Symbol:    job.Symbol,
		Status:    job.Status,
		Progress:  job.Progress,
		Error:     job.Error,
		StartedAt: job.CreatedAt,
		Timeframe: job.Timeframe,
		StartDate: job.StartDate,
		EndDate:   job.EndDate,
	}, nil
}

// GetActiveBinanceDownloads gets all active Binance download jobs
func (s *MarketDataService) GetActiveBinanceDownloads(ctx context.Context) ([]model.BinanceDownloadJob, error) {
	return s.marketDataRepo.GetActiveBinanceDownloadJobs(ctx)
}

// CancelBinanceDownload cancels a Binance download job
func (s *MarketDataService) CancelBinanceDownload(ctx context.Context, jobID int) (bool, error) {
	job, err := s.marketDataRepo.GetBinanceDownloadJob(ctx, jobID)
	if err != nil {
		return false, err
	}

	if job == nil {
		return false, nil
	}

	// Only pending or in-progress jobs can be cancelled
	if job.Status != "pending" && job.Status != "in_progress" {
		return false, fmt.Errorf("job is already %s", job.Status)
	}

	err = s.marketDataRepo.UpdateBinanceDownloadJobStatus(ctx, jobID, "cancelled", job.Progress, "Cancelled by user")
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetCandles retrieves candle data with dynamic timeframe
func (s *MarketDataService) GetCandles(
	ctx context.Context,
	query *model.MarketDataQuery,
) ([]model.Candle, error) {
	// Validate inputs
	if query.SymbolID <= 0 {
		return nil, errors.New("invalid symbol ID")
	}

	if query.Timeframe == "" {
		return nil, errors.New("timeframe is required")
	}

	// Call repository function
	candles, err := s.marketDataRepo.GetCandles(
		ctx,
		query.SymbolID,
		query.Timeframe,
		query.StartDate,
		query.EndDate,
		query.Limit,
	)

	if err != nil {
		return nil, err
	}

	return candles, nil
}

// BatchImportCandles handles batch importing of candle data
func (s *MarketDataService) BatchImportCandles(
	ctx context.Context,
	candles []model.CandleBatch,
) (int, error) {
	if len(candles) == 0 {
		return 0, errors.New("no candle data provided")
	}

	// Call repository function
	insertedCount, err := s.marketDataRepo.BatchImportCandles(ctx, candles)
	if err != nil {
		return 0, err
	}

	// Update symbol data availability for all unique symbols
	symbolsMap := make(map[int]bool)
	for _, candle := range candles {
		symbolsMap[candle.SymbolID] = true
	}

	// Update data availability flag for each symbol
	for symbolID := range symbolsMap {
		err := s.marketDataRepo.UpdateSymbolDataAvailability(ctx, symbolID, true)
		if err != nil {
			s.logger.Warn("Failed to update symbol data availability",
				zap.Error(err),
				zap.Int("symbolID", symbolID))
		}
	}

	return insertedCount, nil
}

// GetDataAvailabilityRange gets the date range for which data is available
func (s *MarketDataService) GetDataAvailabilityRange(
	ctx context.Context,
	symbolID int,
	timeframe string,
) (*time.Time, *time.Time, error) {
	// Validate inputs
	if symbolID <= 0 {
		return nil, nil, errors.New("invalid symbol ID")
	}

	if timeframe == "" {
		return nil, nil, errors.New("timeframe is required")
	}

	// Check if data exists
	hasData, err := s.marketDataRepo.HasData(ctx, symbolID, timeframe)
	if err != nil {
		return nil, nil, err
	}

	if !hasData {
		return nil, nil, nil
	}

	// Get data range
	startDate, endDate, err := s.marketDataRepo.GetDataRange(ctx, symbolID, timeframe)
	if err != nil {
		return nil, nil, err
	}

	return &startDate, &endDate, nil
}

// GetAssetTypes retrieves all available asset types
func (s *MarketDataService) GetAssetTypes(ctx context.Context) (interface{}, error) {
	return s.symbolRepo.GetAssetTypes(ctx)
}

// GetExchanges retrieves all available exchanges
func (s *MarketDataService) GetExchanges(ctx context.Context) (interface{}, error) {
	return s.symbolRepo.GetExchanges(ctx)
}
