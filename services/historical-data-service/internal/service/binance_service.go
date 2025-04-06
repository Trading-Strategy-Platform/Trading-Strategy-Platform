package service

import (
	"context"
	"fmt"
	"time"

	"services/historical-data-service/internal/client"
	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/repository"

	"go.uber.org/zap"
)

// BinanceService handles Binance data operations
type BinanceService struct {
	binanceRepo    *repository.BinanceRepository
	symbolRepo     *repository.SymbolRepository
	marketDataRepo *repository.MarketDataRepository
	logger         *zap.Logger
}

// NewBinanceService creates a new Binance service
func NewBinanceService(
	binanceRepo *repository.BinanceRepository,
	symbolRepo *repository.SymbolRepository,
	marketDataRepo *repository.MarketDataRepository,
	logger *zap.Logger,
) *BinanceService {
	return &BinanceService{
		binanceRepo:    binanceRepo,
		symbolRepo:     symbolRepo,
		marketDataRepo: marketDataRepo,
		logger:         logger,
	}
}

// GetAvailableSymbols retrieves all available symbols from Binance
func (s *BinanceService) GetAvailableSymbols(ctx context.Context) ([]model.BinanceSymbol, error) {
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

// CheckSymbolStatus checks if a symbol exists in the database and what date ranges are available
func (s *BinanceService) CheckSymbolStatus(ctx context.Context, symbol, timeframe string) (*model.SymbolDataStatus, error) {
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

// InitiateDataDownload starts a download job for historical data from Binance
func (s *BinanceService) InitiateDataDownload(ctx context.Context, request *model.BinanceDownloadRequest) (int, error) {
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
	jobID, err := s.binanceRepo.CreateDownloadJob(
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
	go s.processDownload(jobID, request.Symbol, symbolID, request.Timeframe, request.StartDate, request.EndDate)

	return jobID, nil
}

// GetDownloadStatus gets the status of a download job
func (s *BinanceService) GetDownloadStatus(ctx context.Context, jobID int) (*model.BinanceDownloadStatus, error) {
	job, err := s.binanceRepo.GetDownloadJob(ctx, jobID)
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

// GetActiveDownloads gets all active download jobs
func (s *BinanceService) GetActiveDownloads(ctx context.Context) ([]model.BinanceDownloadJob, error) {
	return s.binanceRepo.GetActiveDownloadJobs(ctx)
}

// CancelDownload cancels a download job
func (s *BinanceService) CancelDownload(ctx context.Context, jobID int) (bool, error) {
	job, err := s.binanceRepo.GetDownloadJob(ctx, jobID)
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

	success, err := s.binanceRepo.UpdateDownloadJobStatus(ctx, jobID, "cancelled", job.Progress, "Cancelled by user")
	if err != nil {
		return false, err
	}

	return success, nil
}

// processDownload processes a download job for Binance data
func (s *BinanceService) processDownload(
	jobID int,
	symbol string,
	symbolID int,
	timeframe string,
	startDate time.Time,
	endDate time.Time,
) {
	// Create a new context for this background process
	ctx := context.Background()

	// Update job status to in_progress
	_, err := s.binanceRepo.UpdateDownloadJobStatus(ctx, jobID, "in_progress", 0, "")
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
		s.binanceRepo.UpdateDownloadJobStatus(
			ctx,
			jobID,
			"failed",
			0,
			fmt.Sprintf("Invalid timeframe: %s", timeframe),
		)
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
		job, err := s.binanceRepo.GetDownloadJob(ctx, jobID)
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
				zap.String("interval", interval))

			// Wait before retrying to respect rate limits
			time.Sleep(5 * time.Second)
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
			s.binanceRepo.UpdateDownloadJobStatus(
				ctx,
				jobID,
				"failed",
				progress,
				fmt.Sprintf("Failed to import candles: %v", err),
			)
			return
		}

		// Update progress
		processedDuration += chunkEnd.Sub(currentStart)
		progress = float64(processedDuration) / float64(totalDuration) * 100

		// Update progress in database every 10 batches
		batchCount++
		if batchCount%10 == 0 {
			s.binanceRepo.UpdateDownloadJobStatus(ctx, jobID, "in_progress", progress, "")
		}

		// Move to the next chunk
		currentStart = chunkEnd

		// Sleep to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}

	// Update job status to completed
	s.binanceRepo.UpdateDownloadJobStatus(ctx, jobID, "completed", 100, "")

	// Update symbol data availability flag
	s.symbolRepo.UpdateDataAvailability(ctx, symbolID, true)
}
