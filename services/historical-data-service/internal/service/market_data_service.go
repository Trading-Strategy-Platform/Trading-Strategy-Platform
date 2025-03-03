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
	logger         *zap.Logger
}

// NewMarketDataService creates a new market data service
func NewMarketDataService(marketDataRepo *repository.MarketDataRepository, logger *zap.Logger) *MarketDataService {
	return &MarketDataService{
		marketDataRepo: marketDataRepo,
		logger:         logger,
	}
}

// GetMarketData retrieves market data based on query parameters
func (s *MarketDataService) GetMarketData(
	ctx context.Context,
	query *model.MarketDataQuery,
) ([]model.MarketData, error) {
	// Validate query
	if query.SymbolID <= 0 || query.TimeframeID <= 0 {
		return nil, errors.New("invalid symbol ID or timeframe ID")
	}

	// Get market data
	data, err := s.marketDataRepo.GetMarketData(
		ctx,
		query.SymbolID,
		query.TimeframeID,
		query.StartDate,
		query.EndDate,
		query.Limit,
	)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// ImportMarketData imports market data for a symbol and timeframe
func (s *MarketDataService) ImportMarketData(
	ctx context.Context,
	request *model.MarketDataImport,
) error {
	// Validate request
	if request.SymbolID <= 0 || request.TimeframeID <= 0 {
		return errors.New("invalid symbol ID or timeframe ID")
	}

	if len(request.Data) == 0 {
		return errors.New("no data provided")
	}

	// Insert market data
	err := s.marketDataRepo.InsertMarketData(
		ctx,
		request.SymbolID,
		request.TimeframeID,
		request.Data,
	)
	if err != nil {
		return err
	}

	// Update symbol data availability flag
	err = s.marketDataRepo.UpdateSymbolDataAvailability(ctx, request.SymbolID, true)
	if err != nil {
		s.logger.Warn("Failed to update symbol data availability",
			zap.Error(err),
			zap.Int("symbolID", request.SymbolID))
	}

	return nil
}

// BatchImportMarketData handles batch importing of market data
func (s *MarketDataService) BatchImportMarketData(
	ctx context.Context,
	requests []model.MarketDataImport,
) error {
	for _, request := range requests {
		err := s.ImportMarketData(ctx, &request)
		if err != nil {
			s.logger.Error("Failed to import batch data",
				zap.Error(err),
				zap.Int("symbolID", request.SymbolID),
				zap.Int("timeframeID", request.TimeframeID))
			return err
		}
	}
	return nil
}

// GetDataAvailabilityRange gets the date range for which data is available
func (s *MarketDataService) GetDataAvailabilityRange(
	ctx context.Context,
	symbolID int,
	timeframeID int,
) (*time.Time, *time.Time, error) {
	// Check if data exists
	hasData, err := s.marketDataRepo.HasData(ctx, symbolID, timeframeID)
	if err != nil {
		return nil, nil, err
	}

	if !hasData {
		return nil, nil, nil
	}

	// Get data range
	startDate, endDate, err := s.marketDataRepo.GetDataRange(ctx, symbolID, timeframeID)
	if err != nil {
		return nil, nil, err
	}

	return &startDate, &endDate, nil
}
