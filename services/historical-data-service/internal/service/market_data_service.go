package service

import (
	"context"
	"encoding/json"
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

// GetAssetTypes retrieves all available asset types
func (s *MarketDataService) GetAssetTypes(ctx context.Context) (interface{}, error) {
	query := `SELECT * FROM get_asset_types()`

	var assetTypes []map[string]interface{}
	err := s.db.SelectContext(ctx, &assetTypes, query)
	if err != nil {
		s.logger.Error("Failed to get asset types", zap.Error(err))
		return nil, err
	}

	return assetTypes, nil
}

// GetExchanges retrieves all available exchanges
func (s *MarketDataService) GetExchanges(ctx context.Context) (interface{}, error) {
	query := `SELECT * FROM get_exchanges()`

	var exchanges []map[string]interface{}
	err := s.db.SelectContext(ctx, &exchanges, query)
	if err != nil {
		s.logger.Error("Failed to get exchanges", zap.Error(err))
		return nil, err
	}

	return exchanges, nil
}

// GetCandles retrieves candle data with dynamic timeframe
func (s *MarketDataService) GetCandles(
	ctx context.Context,
	symbolID int,
	timeframe string,
	startDate *time.Time,
	endDate *time.Time,
	limit *int,
) (interface{}, error) {
	query := `SELECT * FROM get_candles($1, $2, $3, $4, $5)`

	var candles []map[string]interface{}
	err := s.db.SelectContext(
		ctx,
		&candles,
		query,
		symbolID,
		timeframe,
		startDate,
		endDate,
		limit,
	)

	if err != nil {
		s.logger.Error("Failed to get candles", zap.Error(err))
		return nil, err
	}

	return candles, nil
}

// BatchImportCandles handles batch importing of candle data
func (s *MarketDataService) BatchImportCandles(ctx context.Context, candles []model.CandleBatch) (int, error) {
	// Convert to JSONB
	candlesJSON, err := json.Marshal(candles)
	if err != nil {
		s.logger.Error("Failed to marshal candles", zap.Error(err))
		return 0, err
	}

	query := `SELECT insert_candles($1)`

	var count int
	err = s.db.GetContext(ctx, &count, query, candlesJSON)
	if err != nil {
		s.logger.Error("Failed to insert candles", zap.Error(err))
		return 0, err
	}

	return count, nil
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
