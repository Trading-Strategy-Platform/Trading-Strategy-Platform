package repository

import (
	"context"
	"database/sql"

	"services/historical-data-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// SymbolRepository handles database operations for symbols
type SymbolRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewSymbolRepository creates a new symbol repository
func NewSymbolRepository(db *sqlx.DB, logger *zap.Logger) *SymbolRepository {
	return &SymbolRepository{
		db:     db,
		logger: logger,
	}
}

// GetAllSymbols retrieves all available symbols using get_symbols function
func (r *SymbolRepository) GetAllSymbols(ctx context.Context) ([]model.Symbol, error) {
	query := `SELECT * FROM get_symbols(NULL, NULL, NULL)`

	var symbols []model.Symbol
	err := r.db.SelectContext(ctx, &symbols, query)
	if err != nil {
		r.logger.Error("Failed to get all symbols", zap.Error(err))
		return nil, err
	}

	return symbols, nil
}

// GetSymbolsByFilter retrieves symbols with filtering using get_symbols function
func (r *SymbolRepository) GetSymbolsByFilter(ctx context.Context, searchTerm, assetType, exchange string) ([]model.Symbol, error) {
	query := `SELECT * FROM get_symbols($1, $2, $3)`

	var searchTermParam *string
	if searchTerm != "" {
		searchTermParam = &searchTerm
	}

	var assetTypeParam *string
	if assetType != "" {
		assetTypeParam = &assetType
	}

	var exchangeParam *string
	if exchange != "" {
		exchangeParam = &exchange
	}

	var symbols []model.Symbol
	err := r.db.SelectContext(ctx, &symbols, query, searchTermParam, assetTypeParam, exchangeParam)
	if err != nil {
		r.logger.Error("Failed to get symbols by filter",
			zap.Error(err),
			zap.String("searchTerm", searchTerm),
			zap.String("assetType", assetType),
			zap.String("exchange", exchange))
		return nil, err
	}

	return symbols, nil
}

// GetSymbolByID retrieves a symbol by ID
func (r *SymbolRepository) GetSymbolByID(ctx context.Context, id int) (*model.Symbol, error) {
	query := `
		SELECT 
			id, symbol, name, exchange, asset_type, 
			is_active, data_available, created_at, updated_at
		FROM symbols
		WHERE id = $1
	`

	var symbol model.Symbol
	err := r.db.GetContext(ctx, &symbol, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get symbol by ID", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &symbol, nil
}

// CreateSymbol creates a new symbol using add_symbol function
func (r *SymbolRepository) CreateSymbol(ctx context.Context, symbol *model.Symbol) (int, error) {
	query := `SELECT add_symbol($1, $2, $3, $4)`

	var id int
	err := r.db.GetContext(
		ctx,
		&id,
		query,
		symbol.Symbol,
		symbol.Name,
		symbol.AssetType,
		symbol.Exchange,
	)

	if err != nil {
		r.logger.Error("Failed to create symbol", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// UpdateSymbol updates a symbol using update_symbol function
func (r *SymbolRepository) UpdateSymbol(ctx context.Context, symbol *model.Symbol) (bool, error) {
	query := `SELECT update_symbol($1, $2, $3, $4, $5)`

	var success bool
	err := r.db.GetContext(
		ctx,
		&success,
		query,
		symbol.ID,
		symbol.Symbol,
		symbol.Name,
		symbol.AssetType,
		symbol.Exchange,
	)

	if err != nil {
		r.logger.Error("Failed to update symbol", zap.Error(err), zap.Int("id", symbol.ID))
		return false, err
	}

	return success, nil
}

// DeleteSymbol marks a symbol as inactive using delete_symbol function
func (r *SymbolRepository) DeleteSymbol(ctx context.Context, id int) (bool, error) {
	query := `SELECT delete_symbol($1)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, id)
	if err != nil {
		r.logger.Error("Failed to delete symbol", zap.Error(err), zap.Int("id", id))
		return false, err
	}

	return success, nil
}

// GetAssetTypes retrieves all available asset types using get_asset_types function
func (r *SymbolRepository) GetAssetTypes(ctx context.Context) ([]struct {
	AssetType string `db:"asset_type"`
	Count     int    `db:"count"`
}, error) {
	query := `SELECT * FROM get_asset_types()`

	var assetTypes []struct {
		AssetType string `db:"asset_type"`
		Count     int    `db:"count"`
	}
	err := r.db.SelectContext(ctx, &assetTypes, query)
	if err != nil {
		r.logger.Error("Failed to get asset types", zap.Error(err))
		return nil, err
	}

	return assetTypes, nil
}

// GetExchanges retrieves all available exchanges using get_exchanges function
func (r *SymbolRepository) GetExchanges(ctx context.Context) ([]struct {
	Exchange string `db:"exchange"`
	Count    int    `db:"count"`
}, error) {
	query := `SELECT * FROM get_exchanges()`

	var exchanges []struct {
		Exchange string `db:"exchange"`
		Count    int    `db:"count"`
	}
	err := r.db.SelectContext(ctx, &exchanges, query)
	if err != nil {
		r.logger.Error("Failed to get exchanges", zap.Error(err))
		return nil, err
	}

	return exchanges, nil
}

// UpdateDataAvailability updates the data_available flag for a symbol
func (r *SymbolRepository) UpdateDataAvailability(ctx context.Context, symbolID int, available bool) (bool, error) {
	query := `SELECT update_symbol_data_availability($1, $2)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, symbolID, available)
	if err != nil {
		r.logger.Error("Failed to update symbol data availability",
			zap.Error(err),
			zap.Int("symbolID", symbolID))
		return false, err
	}

	return success, nil
}
