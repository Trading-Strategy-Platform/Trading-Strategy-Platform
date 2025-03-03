package service

import (
	"context"
	"errors"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/repository"

	"go.uber.org/zap"
)

// SymbolService handles symbol operations
type SymbolService struct {
	symbolRepo *repository.SymbolRepository
	logger     *zap.Logger
}

// NewSymbolService creates a new symbol service
func NewSymbolService(symbolRepo *repository.SymbolRepository, logger *zap.Logger) *SymbolService {
	return &SymbolService{
		symbolRepo: symbolRepo,
		logger:     logger,
	}
}

// GetAllSymbols retrieves all available symbols
func (s *SymbolService) GetAllSymbols(ctx context.Context) ([]model.Symbol, error) {
	return s.symbolRepo.GetAllSymbols(ctx)
}

// GetSymbolByID retrieves a symbol by ID
func (s *SymbolService) GetSymbolByID(ctx context.Context, id int) (*model.Symbol, error) {
	symbol, err := s.symbolRepo.GetSymbolByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if symbol == nil {
		return nil, errors.New("symbol not found")
	}

	return symbol, nil
}

// CreateSymbol creates a new symbol
func (s *SymbolService) CreateSymbol(ctx context.Context, symbol *model.Symbol) (int, error) {
	// Validate symbol
	if symbol.Symbol == "" || symbol.Name == "" || symbol.Exchange == "" {
		return 0, errors.New("symbol, name, and exchange are required")
	}

	// Create symbol
	return s.symbolRepo.CreateSymbol(ctx, symbol)
}

// TimeframeService handles timeframe operations
type TimeframeService struct {
	timeframeRepo *repository.TimeframeRepository
	logger        *zap.Logger
}

// NewTimeframeService creates a new timeframe service
func NewTimeframeService(timeframeRepo *repository.TimeframeRepository, logger *zap.Logger) *TimeframeService {
	return &TimeframeService{
		timeframeRepo: timeframeRepo,
		logger:        logger,
	}
}

// GetAllTimeframes retrieves all available timeframes
func (s *TimeframeService) GetAllTimeframes(ctx context.Context) ([]model.Timeframe, error) {
	return s.timeframeRepo.GetAllTimeframes(ctx)
}

// GetTimeframeByID retrieves a timeframe by ID
func (s *TimeframeService) GetTimeframeByID(ctx context.Context, id int) (*model.Timeframe, error) {
	timeframe, err := s.timeframeRepo.GetTimeframeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if timeframe == nil {
		return nil, errors.New("timeframe not found")
	}

	return timeframe, nil
}

// CreateTimeframe creates a new timeframe
func (s *TimeframeService) CreateTimeframe(ctx context.Context, timeframe *model.Timeframe) (int, error) {
	// Validate timeframe
	if timeframe.Name == "" || timeframe.DisplayName == "" || timeframe.Minutes <= 0 {
		return 0, errors.New("name, display name, and valid minutes are required")
	}

	// Create timeframe
	return s.timeframeRepo.CreateTimeframe(ctx, timeframe)
}
