package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"services/historical-data-service/internal/client"
	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/repository"
	"services/historical-data-service/internal/utils"

	"go.uber.org/zap"
)

// MarketDataDownloadService handles market data download operations
type MarketDataDownloadService struct {
	downloadRepo   *repository.DownloadJobRepository
	inventoryRepo  *repository.InventoryRepository
	symbolRepo     *repository.SymbolRepository
	marketDataRepo *repository.MarketDataRepository
	logger         *zap.Logger
}

// NewMarketDataDownloadService creates a new market data download service
func NewMarketDataDownloadService(
	downloadRepo *repository.DownloadJobRepository,
	inventoryRepo *repository.InventoryRepository,
	symbolRepo *repository.SymbolRepository,
	marketDataRepo *repository.MarketDataRepository,
	logger *zap.Logger,
) *MarketDataDownloadService {
	return &MarketDataDownloadService{
		downloadRepo:   downloadRepo,
		inventoryRepo:  inventoryRepo,
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
	symbols, err := s.symbolRepo.GetAllSymbols(ctx, symbol, "", "", "", "", 0, 0)
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
	symbols, err := s.symbolRepo.GetAllSymbols(ctx, request.Symbol, "", "", "", "", 0, 0)
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

// GetActiveDownloads gets all active download jobs with pagination and sorting
func (s *MarketDataDownloadService) GetActiveDownloads(
	ctx context.Context,
	source string,
	sortBy string,
	sortDirection string,
	page int,
	limit int,
) ([]model.MarketDataDownloadJob, int, error) {
	// Normalize sort parameters
	if sortBy == "" {
		sortBy = "created_at"
	}

	sortDirection = utils.NormalizeSortDirection(sortDirection)
	offset := utils.CalculateOffset(page, limit)

	// Get total count for pagination
	totalCount, err := s.downloadRepo.CountActiveDownloadJobs(ctx, source)
	if err != nil {
		return nil, 0, err
	}

	// Get active jobs with sorting and pagination
	jobs, err := s.downloadRepo.GetActiveDownloadJobs(
		ctx,
		source,
		sortBy,
		sortDirection,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, err
	}

	return jobs, totalCount, nil
}

// CancelDownload cancels a download job
func (s *MarketDataDownloadService) CancelDownload(ctx context.Context, jobID int, force bool) (bool, error) {
	return s.downloadRepo.CancelDownload(ctx, jobID, force)
}

// GetJobsSummary gets a summary of all download jobs
func (s *MarketDataDownloadService) GetJobsSummary(ctx context.Context) (map[string]interface{}, error) {
	summary, err := s.downloadRepo.GetJobsSummary(ctx)
	if err != nil {
		return nil, err
	}

	// Format summary data for the API response
	result := make(map[string]interface{})
	statusCounts := make(map[string]map[string]int64)

	for _, s := range summary {
		statusCounts[s.Status] = map[string]int64{
			"total":    s.Count,
			"last_24h": s.Last24h,
		}
	}

	result["status_counts"] = statusCounts

	// Get recent jobs for the summary
	recentJobs, err := s.downloadRepo.GetDownloadJobsByStatus(
		ctx,
		"", // all statuses
		"", // all sources
		"", // all symbols
		"created_at",
		"DESC",
		10, // limit to recent 10
		0,  // no offset
	)

	if err == nil && len(recentJobs) > 0 {
		result["recent_jobs"] = recentJobs
	}

	// Get source counts
	sources := make(map[string]int)
	for _, job := range summary {
		if sources[job.Status] == 0 {
			sources[job.Status] = int(job.Count)
		}
	}
	result["sources"] = sources

	return result, nil
}

// GetDataInventory gets a summary of all available market data with pagination
func (s *MarketDataDownloadService) GetDataInventory(
	ctx context.Context,
	assetType string,
	exchange string,
	page int,
	limit int,
) ([]model.DataInventoryItem, int, error) {
	// Set defaults
	if limit <= 0 {
		limit = 100
	}
	if page <= 0 {
		page = 1
	}

	offset := utils.CalculateOffset(page, limit)

	// Get total count for pagination
	totalCount, err := s.inventoryRepo.CountDataInventory(ctx, assetType, exchange)
	if err != nil {
		return nil, 0, err
	}

	// Get inventory items with pagination
	inventory, err := s.inventoryRepo.GetDataInventory(
		ctx,
		assetType,
		exchange,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, err
	}

	return inventory, totalCount, nil
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

	// Calculate minutes per candle based on timeframe
	var minutesPerCandle int
	switch timeframe {
	case "1m":
		minutesPerCandle = 1
	case "5m":
		minutesPerCandle = 5
	case "15m":
		minutesPerCandle = 15
	case "30m":
		minutesPerCandle = 30
	case "1h":
		minutesPerCandle = 60
	case "4h":
		minutesPerCandle = 240
	case "1d":
		minutesPerCandle = 1440
	case "1w":
		minutesPerCandle = 10080
	default:
		minutesPerCandle = 1
	}

	// Calculate optimal chunk size to get close to 1000 candles per request
	// Maximum is 1000 candles per request, let's aim for 900 to be safe
	targetCandlesPerChunk := 900
	chunkMinutes := targetCandlesPerChunk * minutesPerCandle
	chunkDuration := time.Duration(chunkMinutes) * time.Minute

	s.logger.Info("Calculated optimal chunk size",
		zap.String("timeframe", timeframe),
		zap.Int("minutesPerCandle", minutesPerCandle),
		zap.Int("targetCandlesPerChunk", targetCandlesPerChunk),
		zap.Int("chunkMinutes", chunkMinutes),
		zap.Duration("chunkDuration", chunkDuration))

	// Calculate the total time range to download
	totalDuration := endDate.Sub(startDate)

	// Estimate total number of candles
	totalCandlesEstimate := int(totalDuration.Minutes()) / minutesPerCandle

	s.logger.Info("Starting download job",
		zap.Int("jobID", jobID),
		zap.String("symbol", symbol),
		zap.String("timeframe", timeframe),
		zap.Time("startDate", startDate),
		zap.Time("endDate", endDate),
		zap.Int("estimatedTotalCandles", totalCandlesEstimate))

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
	processedCandles := 0
	totalDownloaded := 0
	retryCount := 0

	// Keep track of consecutive empty chunks for early termination
	emptyChunksInARow := 0
	maxEmptyChunks := 5 // If we get 5 empty chunks in a row, we'll stop

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
		chunkEnd := currentStart.Add(chunkDuration)
		if chunkEnd.After(endDate) {
			chunkEnd = endDate
		}

		// Log the time range we're fetching
		s.logger.Debug("Fetching data chunk",
			zap.String("symbol", symbol),
			zap.String("interval", interval),
			zap.Time("startTime", currentStart),
			zap.Time("endTime", chunkEnd))

		// Calculate expected candles in this chunk
		expectedCandlesInChunk := int(chunkEnd.Sub(currentStart).Minutes()) / minutesPerCandle

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
					float64(processedCandles)/float64(totalCandlesEstimate)*100,
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

		// If no klines were returned but we haven't reached the end date
		if len(klines) == 0 {
			emptyChunksInARow++
			s.logger.Info("No data returned for time range",
				zap.String("symbol", symbol),
				zap.String("interval", interval),
				zap.Time("start", currentStart),
				zap.Time("end", chunkEnd),
				zap.Int("emptyChunksInARow", emptyChunksInARow))

			// Skip ahead by the chunk size
			currentStart = chunkEnd

			// Check if we've hit the limit for empty chunks
			if emptyChunksInARow >= maxEmptyChunks {
				s.logger.Warn("Too many consecutive empty chunks, ending download early",
					zap.Int("emptyChunksInARow", emptyChunksInARow),
					zap.Int("maxEmptyChunks", maxEmptyChunks))
				break
			}

			continue
		}

		// Reset empty chunks counter if we got data
		emptyChunksInARow = 0

		totalDownloaded += len(klines)
		s.logger.Debug("Received klines from Binance",
			zap.Int("count", len(klines)),
			zap.String("symbol", symbol),
			zap.String("interval", interval),
			zap.Time("firstCandleTime", klines[0].OpenTime),
			zap.Time("lastCandleTime", klines[len(klines)-1].OpenTime),
			zap.Int("expectedCandlesInChunk", expectedCandlesInChunk),
			zap.Int("actualCandlesInChunk", len(klines)))

		// Convert klines to candles
		candles := make([]model.CandleBatch, 0, len(klines))
		for i, k := range klines {
			// Make extra sure the time is not zero
			if k.OpenTime.IsZero() {
				s.logger.Warn("Skipping candle with zero time",
					zap.Int("index", i),
					zap.String("symbol", symbol))
				continue
			}

			candles = append(candles, model.CandleBatch{
				SymbolID: symbolID,
				Time:     k.OpenTime, // This will use the "candle_time" JSON tag
				Open:     k.Open,
				High:     k.High,
				Low:      k.Low,
				Close:    k.Close,
				Volume:   k.Volume,
			})
		}

		// Import candles
		importedCount, err := s.marketDataRepo.BatchImportCandles(ctx, candles)
		if err != nil {
			// Log in detail
			s.logger.Error("Failed to import candles",
				zap.Error(err),
				zap.String("symbol", symbol),
				zap.Time("chunkStart", currentStart),
				zap.Time("chunkEnd", chunkEnd),
				zap.Int("candlesInBatch", len(candles)))

			// Try next chunk
			currentStart = chunkEnd
			continue
		}

		s.logger.Info("Imported candles",
			zap.Int("importedCount", importedCount),
			zap.String("symbol", symbol),
			zap.String("interval", interval),
			zap.Time("start", currentStart),
			zap.Time("end", chunkEnd),
			zap.Int("totalDownloadedSoFar", totalDownloaded),
			zap.Int("totalImportedSoFar", processedCandles+importedCount))

		// Reset retry counter on success
		retryCount = 0

		// Update progress
		processedCandles += importedCount
		progress := float64(processedCandles) / float64(totalCandlesEstimate) * 100

		// Cap progress at 99% until completely done
		if progress > 99.0 && currentStart.Before(endDate) {
			progress = 99.0
		}

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
		time.Sleep(300 * time.Millisecond)
	}

	// Calculate final progress percentage
	finalProgress := float64(processedCandles) / float64(totalCandlesEstimate) * 100
	if finalProgress > 100.0 {
		finalProgress = 100.0
	}

	// Update job status to completed or partial if there were errors
	if finalProgress >= 99.0 {
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
		s.logger.Info("Download job completed successfully",
			zap.Int("jobID", jobID),
			zap.String("symbol", symbol),
			zap.Int("processedCandles", processedCandles),
			zap.Int("totalCandlesEstimate", totalCandlesEstimate))
	} else {
		s.downloadRepo.UpdateDownloadJobStatus(
			ctx,
			jobID,
			"partial",
			finalProgress,
			processedCandles,
			totalCandlesEstimate,
			retryCount,
			fmt.Sprintf("Download completed with some gaps. Imported %d of ~%d candles (%.1f%%)",
				processedCandles, totalCandlesEstimate, finalProgress),
		)
		s.logger.Info("Download job completed partially",
			zap.Int("jobID", jobID),
			zap.String("symbol", symbol),
			zap.Int("processedCandles", processedCandles),
			zap.Int("totalCandlesEstimate", totalCandlesEstimate),
			zap.Float64("completionPercentage", finalProgress))
	}

	// Update symbol data availability flag if we imported any data
	if processedCandles > 0 {
		s.symbolRepo.UpdateDataAvailability(ctx, symbolID, true)
	}
}
