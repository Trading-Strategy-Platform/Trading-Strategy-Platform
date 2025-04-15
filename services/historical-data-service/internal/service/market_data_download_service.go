package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"services/historical-data-service/internal/client"
	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/repository"

	"go.uber.org/zap"
)

// MarketDataDownloadService handles market data download operations
type MarketDataDownloadService struct {
	downloadRepo   *repository.DownloadJobRepository
	symbolRepo     *repository.SymbolRepository
	marketDataRepo *repository.MarketDataRepository
	logger         *zap.Logger
}

// NewMarketDataDownloadService creates a new market data download service
func NewMarketDataDownloadService(
	downloadRepo *repository.DownloadJobRepository,
	symbolRepo *repository.SymbolRepository,
	marketDataRepo *repository.MarketDataRepository,
	logger *zap.Logger,
) *MarketDataDownloadService {
	return &MarketDataDownloadService{
		downloadRepo:   downloadRepo,
		symbolRepo:     symbolRepo,
		marketDataRepo: marketDataRepo,
		logger:         logger,
	}
}

// GetAvailableSymbols retrieves all available symbols from a specific source
func (s *MarketDataDownloadService) GetAvailableSymbols(ctx context.Context, source string) (interface{}, error) {
	switch source {
	case string(model.SourceBinance):
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

	// Add cases for other sources as needed
	default:
		return nil, fmt.Errorf("unsupported data source: %s", source)
	}
}

// CheckSymbolStatus checks if a symbol exists in the database and what date ranges are available
func (s *MarketDataDownloadService) CheckSymbolStatus(ctx context.Context, symbol, timeframe string) (*model.SymbolDataStatus, error) {
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

// InitiateDataDownload starts a download job for historical data
func (s *MarketDataDownloadService) InitiateDataDownload(ctx context.Context, request *model.MarketDataDownloadRequest) (int, error) {
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
		if request.Source == string(model.SourceBinance) {
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
		} else {
			// For other sources, create a generic symbol
			name := request.Symbol
			exchange := request.Source
			assetType := "unknown"

			// Try to determine asset type from the symbol name
			if len(name) > 3 && name[len(name)-3:] == "USD" {
				assetType = "forex"
			} else if len(name) > 4 && name[len(name)-4:] == "USDT" {
				assetType = "crypto"
			}

			newSymbol := &model.Symbol{
				Symbol:    request.Symbol,
				Name:      name,
				AssetType: assetType,
				Exchange:  exchange,
				IsActive:  true,
			}

			symbolID, err = s.symbolRepo.CreateSymbol(ctx, newSymbol)
			if err != nil {
				return 0, err
			}
		}
	}

	// Create a download job
	jobID, err := s.downloadRepo.CreateDownloadJob(
		ctx,
		symbolID,
		request.Symbol,
		request.Source,
		request.Timeframe,
		request.StartDate,
		request.EndDate,
	)

	if err != nil {
		return 0, err
	}

	// Start the download process in a background goroutine
	go s.processDownload(jobID, request.Symbol, symbolID, request.Source, request.Timeframe, request.StartDate, request.EndDate)

	return jobID, nil
}

// GetDownloadStatus gets the status of a download job
func (s *MarketDataDownloadService) GetDownloadStatus(ctx context.Context, jobID int) (*model.MarketDataDownloadStatus, error) {
	job, err := s.downloadRepo.GetDownloadJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if job == nil {
		return nil, nil
	}

	return &model.MarketDataDownloadStatus{
		JobID:             job.ID,
		Symbol:            job.Symbol,
		Source:            job.Source,
		Status:            job.Status,
		Progress:          job.Progress,
		ProcessedCandles:  job.ProcessedCandles,
		TotalCandles:      job.TotalCandles,
		Retries:           job.Retries,
		Error:             job.Error,
		StartedAt:         job.CreatedAt,
		Timeframe:         job.Timeframe,
		StartDate:         job.StartDate,
		EndDate:           job.EndDate,
		LastProcessedTime: job.LastProcessedTime,
	}, nil
}

// GetActiveDownloads gets all active download jobs
func (s *MarketDataDownloadService) GetActiveDownloads(ctx context.Context, source string) ([]model.MarketDataDownloadJob, error) {
	return s.downloadRepo.GetActiveDownloadJobs(ctx, source)
}

// CancelDownload cancels a download job
func (s *MarketDataDownloadService) CancelDownload(ctx context.Context, jobID int) (bool, error) {
	job, err := s.downloadRepo.GetDownloadJob(ctx, jobID)
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

	success, err := s.downloadRepo.UpdateDownloadJobStatus(
		ctx,
		jobID,
		"cancelled",
		job.Progress,
		job.ProcessedCandles,
		job.TotalCandles,
		job.Retries,
		"Cancelled by user",
	)
	if err != nil {
		return false, err
	}

	return success, nil
}

// GetJobsSummary gets a summary of all download jobs
func (s *MarketDataDownloadService) GetJobsSummary(ctx context.Context) (map[string]interface{}, error) {
	return s.downloadRepo.GetJobsSummary(ctx)
}

// GetDataInventory gets a summary of all available market data
func (s *MarketDataDownloadService) GetDataInventory(ctx context.Context, assetType, exchange string) ([]map[string]interface{}, error) {
	// Get all symbols of the specified type
	symbols, err := s.symbolRepo.GetSymbolsByFilter(ctx, "", assetType, exchange)
	if err != nil {
		return nil, err
	}

	var inventory []map[string]interface{}

	for _, symbol := range symbols {
		if !symbol.DataAvailable {
			continue
		}

		// Get available timeframes
		timeframes, err := s.getAvailableTimeframes(ctx, symbol.ID)
		if err != nil {
			s.logger.Error("Failed to get timeframes for symbol",
				zap.Error(err),
				zap.Int("symbolID", symbol.ID))
			continue
		}

		timeframeData := make(map[string]interface{})

		for _, tf := range timeframes {
			// Get data ranges for this timeframe
			ranges, err := s.marketDataRepo.GetDataRanges(ctx, symbol.ID, tf)
			if err != nil {
				s.logger.Error("Failed to get data ranges",
					zap.Error(err),
					zap.Int("symbolID", symbol.ID),
					zap.String("timeframe", tf))
				continue
			}

			if len(ranges) == 0 {
				continue
			}

			// Identify gaps in data if any
			var gaps []model.DateRange
			for i := 0; i < len(ranges)-1; i++ {
				if ranges[i].End.Add(time.Minute).Before(ranges[i+1].Start) {
					gaps = append(gaps, model.DateRange{
						Start: ranges[i].End.Add(time.Minute),
						End:   ranges[i+1].Start.Add(-time.Minute),
					})
				}
			}

			// Get candle count
			count, err := s.downloadRepo.GetCandleCount(ctx, symbol.ID)
			if err != nil {
				s.logger.Error("Failed to get candle count",
					zap.Error(err),
					zap.Int("symbolID", symbol.ID))
				count = 0
			}

			timeframeData[tf] = map[string]interface{}{
				"available_ranges": ranges,
				"gaps":             gaps,
				"earliest_date":    ranges[0].Start,
				"latest_date":      ranges[len(ranges)-1].End,
				"candle_count":     count,
			}
		}

		if len(timeframeData) > 0 {
			inventory = append(inventory, map[string]interface{}{
				"symbol_id":  symbol.ID,
				"symbol":     symbol.Symbol,
				"name":       symbol.Name,
				"asset_type": symbol.AssetType,
				"exchange":   symbol.Exchange,
				"timeframes": timeframeData,
			})
		}
	}

	return inventory, nil
}

// getAvailableTimeframes gets all timeframes that have data for a symbol
func (s *MarketDataDownloadService) getAvailableTimeframes(ctx context.Context, symbolID int) ([]string, error) {
	return s.downloadRepo.GetAvailableTimeframes(ctx, symbolID)
}

// processDownload processes a download job for market data
func (s *MarketDataDownloadService) processDownload(
	jobID int,
	symbol string,
	symbolID int,
	source string,
	timeframe string,
	startDate time.Time,
	endDate time.Time,
) {
	// Create a new context for this background process
	ctx := context.Background()

	// Update job status to in_progress
	s.downloadRepo.UpdateDownloadJobStatus(
		ctx,
		jobID,
		"in_progress",
		0,
		0,
		0,
		0,
		"",
	)

	// Process the download based on the source
	switch source {
	case string(model.SourceBinance):
		s.processBinanceDownload(jobID, symbol, symbolID, timeframe, startDate, endDate)
	// Add other sources as needed
	default:
		s.downloadRepo.UpdateDownloadJobStatus(
			ctx,
			jobID,
			"failed",
			0,
			0,
			0,
			0,
			fmt.Sprintf("Unsupported data source: %s", source),
		)
	}
}

// processBinanceDownload handles downloading data from Binance
func (s *MarketDataDownloadService) processBinanceDownload(
	jobID int,
	symbol string,
	symbolID int,
	timeframe string,
	startDate time.Time,
	endDate time.Time,
) {
	ctx := context.Background()

	// Create a Binance client
	binanceClient := client.NewBinanceClient(s.logger)

	// Map our timeframe to Binance interval
	interval := client.MapTimeframeToBinanceInterval(timeframe)
	if interval == "" {
		s.downloadRepo.UpdateDownloadJobStatus(
			ctx,
			jobID,
			"failed",
			0,
			0,
			0,
			0,
			fmt.Sprintf("Invalid timeframe: %s", timeframe),
		)
		return
	}

	// Calculate the total time range to download
	totalDuration := endDate.Sub(startDate)
	processedDuration := time.Duration(0)

	// Estimate total number of candles
	var totalCandlesEstimate int
	switch timeframe {
	case "1m":
		totalCandlesEstimate = int(totalDuration.Minutes())
	case "5m":
		totalCandlesEstimate = int(totalDuration.Minutes() / 5)
	case "15m":
		totalCandlesEstimate = int(totalDuration.Minutes() / 15)
	case "30m":
		totalCandlesEstimate = int(totalDuration.Minutes() / 30)
	case "1h":
		totalCandlesEstimate = int(totalDuration.Hours())
	case "4h":
		totalCandlesEstimate = int(totalDuration.Hours() / 4)
	case "1d":
		totalCandlesEstimate = int(totalDuration.Hours() / 24)
	case "1w":
		totalCandlesEstimate = int(totalDuration.Hours() / 168)
	default:
		totalCandlesEstimate = 1000 // Default estimate
	}

	// Update job with total candles estimate
	s.downloadRepo.UpdateDownloadJobStatus(
		ctx,
		jobID,
		"in_progress",
		0,
		0,
		totalCandlesEstimate,
		0,
		"",
	)

	// Process in chunks
	currentStart := startDate

	// Track progress
	var progress float64
	var processedCandles int
	var retryCount int

	for currentStart.Before(endDate) {
		// Check if job was cancelled
		job, err := s.downloadRepo.GetDownloadJob(ctx, jobID)
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
			// Implement exponential backoff for retries
			if retryCount < 5 {
				retryCount++
				backoffTime := time.Duration(math.Pow(2, float64(retryCount))) * time.Second
				s.logger.Warn("Failed to fetch klines, retrying after backoff",
					zap.Error(err),
					zap.String("symbol", symbol),
					zap.String("interval", interval),
					zap.Duration("backoff", backoffTime),
					zap.Int("retry", retryCount))

				s.downloadRepo.UpdateDownloadJobStatus(
					ctx,
					jobID,
					"in_progress",
					progress,
					processedCandles,
					totalCandlesEstimate,
					retryCount,
					fmt.Sprintf("Retry %d/5: %v", retryCount, err),
				)

				time.Sleep(backoffTime)
				continue
			}

			// Max retries reached, fail this chunk but continue with next
			s.logger.Error("Max retries reached for chunk, skipping to next chunk",
				zap.Error(err),
				zap.String("symbol", symbol),
				zap.String("interval", interval),
				zap.Time("chunkStart", currentStart),
				zap.Time("chunkEnd", chunkEnd))

			currentStart = chunkEnd
			retryCount = 0
			continue
		}

		// If no klines were returned but we haven't reached the end date, there might be a gap
		if len(klines) == 0 && currentStart.Before(endDate) {
			// Skip ahead by the chunk size
			currentStart = chunkEnd
			continue
		}

		// Verify kline sequence for gaps
		if len(klines) > 1 {
			for i := 1; i < len(klines); i++ {
				var expectedTime time.Time
				switch timeframe {
				case "1m":
					expectedTime = klines[i-1].OpenTime.Add(1 * time.Minute)
				case "5m":
					expectedTime = klines[i-1].OpenTime.Add(5 * time.Minute)
				case "15m":
					expectedTime = klines[i-1].OpenTime.Add(15 * time.Minute)
				case "30m":
					expectedTime = klines[i-1].OpenTime.Add(30 * time.Minute)
				case "1h":
					expectedTime = klines[i-1].OpenTime.Add(1 * time.Hour)
				case "4h":
					expectedTime = klines[i-1].OpenTime.Add(4 * time.Hour)
				case "1d":
					expectedTime = klines[i-1].OpenTime.Add(24 * time.Hour)
				case "1w":
					expectedTime = klines[i-1].OpenTime.Add(7 * 24 * time.Hour)
				}

				if !expectedTime.Equal(klines[i].OpenTime) {
					s.logger.Warn("Gap detected in kline data",
						zap.String("symbol", symbol),
						zap.Time("expected", expectedTime),
						zap.Time("actual", klines[i].OpenTime))
				}
			}
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
			// Update job status with error but continue
			s.logger.Error("Failed to import candles",
				zap.Error(err),
				zap.String("symbol", symbol),
				zap.Time("chunkStart", currentStart),
				zap.Time("chunkEnd", chunkEnd))

			// Try next chunk
			currentStart = chunkEnd
			continue
		}

		// Reset retry counter on success
		retryCount = 0

		// Update progress
		processedCandles += len(candles)
		processedDuration += chunkEnd.Sub(currentStart)
		progress = float64(processedDuration) / float64(totalDuration) * 100

		// Update progress in database
		s.downloadRepo.UpdateDownloadJobStatus(
			ctx,
			jobID,
			"in_progress",
			progress,
			processedCandles,
			totalCandlesEstimate,
			retryCount,
			"",
		)

		// Move to the next chunk
		currentStart = chunkEnd

		// Sleep to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}

	// Update job status to completed or partial if there were errors
	if progress >= 99.0 {
		s.downloadRepo.UpdateDownloadJobStatus(
			ctx,
			jobID,
			"completed",
			100,
			processedCandles,
			totalCandlesEstimate,
			retryCount,
			"",
		)
	} else {
		s.downloadRepo.UpdateDownloadJobStatus(
			ctx,
			jobID,
			"partial",
			progress,
			processedCandles,
			totalCandlesEstimate,
			retryCount,
			"Download completed with some gaps",
		)
	}

	// Update symbol data availability flag
	s.symbolRepo.UpdateDataAvailability(ctx, symbolID, true)
}
