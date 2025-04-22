package service

import (
	"context"
	"errors"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/repository"
	"services/historical-data-service/internal/utils"

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

// GetAllSymbols retrieves all symbols without filtering (for backward compatibility)
func (s *SymbolService) GetAllSymbols(ctx context.Context) ([]model.Symbol, error) {
	symbols, err := s.symbolRepo.GetAllSymbols(
		ctx,
		"",       // no search term
		"",       // no asset type filter
		"",       // no exchange filter
		"symbol", // default sort by symbol
		"ASC",    // default sort ascending
		1000,     // default large limit
		0,        // no offset
	)
	return symbols, err
}

// GetSymbolsWithPagination retrieves all available symbols with pagination and sorting
func (s *SymbolService) GetSymbolsWithPagination(
	ctx context.Context,
	searchTerm string,
	assetType string,
	exchange string,
	sortBy string,
	sortDirection string,
	page int,
	limit int,
) ([]model.Symbol, int, error) {
	// Validate and set default sort parameters
	if sortBy == "" {
		sortBy = "symbol"
	}

	// Normalize sort direction
	sortDirection = utils.NormalizeSortDirection(sortDirection)

	// Calculate offset
	offset := utils.CalculateOffset(page, limit)

	// Get total count for pagination
	total, err := s.symbolRepo.CountSymbols(ctx, searchTerm, assetType, exchange)
	if err != nil {
		return nil, 0, err
	}

	// Get symbols with pagination and sorting
	symbols, err := s.symbolRepo.GetAllSymbols(
		ctx,
		searchTerm,
		assetType,
		exchange,
		sortBy,
		sortDirection,
		limit,
		offset,
	)
	if err != nil {
		return nil, 0, err
	}

	return symbols, total, nil
}

// GetSymbolsByFilter retrieves symbols with filter parameters (for backward compatibility)
func (s *SymbolService) GetSymbolsByFilter(ctx context.Context, filter *model.SymbolFilter) ([]model.Symbol, error) {
	return s.symbolRepo.GetAllSymbols(
		ctx,
		filter.SearchTerm,
		filter.AssetType,
		filter.Exchange,
		"symbol", // default sort by symbol
		"ASC",    // default sort ascending
		1000,     // default large limit
		0,        // no offset
	)
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
	if symbol.Symbol == "" || symbol.Name == "" || symbol.AssetType == "" {
		return 0, errors.New("symbol, name, and asset type are required")
	}

	// Set defaults if not provided
	symbol.IsActive = true
	symbol.DataAvailable = false

	// Create symbol using the database function via repository
	return s.symbolRepo.CreateSymbol(ctx, symbol)
}

// UpdateSymbol updates an existing symbol
func (s *SymbolService) UpdateSymbol(ctx context.Context, symbol *model.Symbol) (bool, error) {
	// Validate symbol
	if symbol.ID <= 0 {
		return false, errors.New("invalid symbol ID")
	}

	// Check if symbol exists
	existingSymbol, err := s.symbolRepo.GetSymbolByID(ctx, symbol.ID)
	if err != nil {
		return false, err
	}

	if existingSymbol == nil {
		return false, errors.New("symbol not found")
	}

	// Update symbol using the database function via repository
	return s.symbolRepo.UpdateSymbol(ctx, symbol)
}

// DeleteSymbol marks a symbol as inactive
func (s *SymbolService) DeleteSymbol(ctx context.Context, id int) (bool, error) {
	// Validate ID
	if id <= 0 {
		return false, errors.New("invalid symbol ID")
	}

	// Check if symbol exists
	existingSymbol, err := s.symbolRepo.GetSymbolByID(ctx, id)
	if err != nil {
		return false, err
	}

	if existingSymbol == nil {
		return false, errors.New("symbol not found")
	}

	// Delete symbol using the database function via repository
	return s.symbolRepo.DeleteSymbol(ctx, id)
}

// GetAssetTypes retrieves all available asset types
func (s *SymbolService) GetAssetTypes(ctx context.Context) (interface{}, error) {
	return s.symbolRepo.GetAssetTypes(ctx)
}

// GetExchanges retrieves all available exchanges
func (s *SymbolService) GetExchanges(ctx context.Context) (interface{}, error) {
	return s.symbolRepo.GetExchanges(ctx)
}
