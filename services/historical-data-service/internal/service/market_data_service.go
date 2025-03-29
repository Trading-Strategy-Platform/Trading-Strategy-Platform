package service

import (
	"context"
	"errors"
	"time"

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
