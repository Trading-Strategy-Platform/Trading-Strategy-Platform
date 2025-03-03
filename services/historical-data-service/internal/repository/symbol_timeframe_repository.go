package repository

import (
	"context"
	"database/sql"
	"time"

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

// GetAllSymbols retrieves all available symbols
func (r *SymbolRepository) GetAllSymbols(ctx context.Context) ([]model.Symbol, error) {
	query := `
		SELECT 
			id, symbol, name, exchange, asset_type, 
			is_active, data_available, created_at, updated_at
		FROM symbols
		ORDER BY symbol
	`

	var symbols []model.Symbol
	err := r.db.SelectContext(ctx, &symbols, query)
	if err != nil {
		r.logger.Error("Failed to get all symbols", zap.Error(err))
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

// CreateSymbol creates a new symbol
func (r *SymbolRepository) CreateSymbol(ctx context.Context, symbol *model.Symbol) (int, error) {
	query := `
		INSERT INTO symbols (
			symbol, name, exchange, asset_type, 
			is_active, data_available, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		symbol.Symbol,
		symbol.Name,
		symbol.Exchange,
		symbol.AssetType,
		symbol.IsActive,
		symbol.DataAvailable,
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create symbol", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// TimeframeRepository handles database operations for timeframes
type TimeframeRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewTimeframeRepository creates a new timeframe repository
func NewTimeframeRepository(db *sqlx.DB, logger *zap.Logger) *TimeframeRepository {
	return &TimeframeRepository{
		db:     db,
		logger: logger,
	}
}

// GetAllTimeframes retrieves all available timeframes
func (r *TimeframeRepository) GetAllTimeframes(ctx context.Context) ([]model.Timeframe, error) {
	query := `
		SELECT id, name, minutes, display_name, created_at, updated_at
		FROM timeframes
		ORDER BY minutes
	`

	var timeframes []model.Timeframe
	err := r.db.SelectContext(ctx, &timeframes, query)
	if err != nil {
		r.logger.Error("Failed to get all timeframes", zap.Error(err))
		return nil, err
	}

	return timeframes, nil
}

// GetTimeframeByID retrieves a timeframe by ID
func (r *TimeframeRepository) GetTimeframeByID(ctx context.Context, id int) (*model.Timeframe, error) {
	query := `
		SELECT id, name, minutes, display_name, created_at, updated_at
		FROM timeframes
		WHERE id = $1
	`

	var timeframe model.Timeframe
	err := r.db.GetContext(ctx, &timeframe, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get timeframe by ID", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &timeframe, nil
}

// CreateTimeframe creates a new timeframe
func (r *TimeframeRepository) CreateTimeframe(ctx context.Context, timeframe *model.Timeframe) (int, error) {
	query := `
		INSERT INTO timeframes (name, minutes, display_name, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		timeframe.Name,
		timeframe.Minutes,
		timeframe.DisplayName,
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create timeframe", zap.Error(err))
		return 0, err
	}

	return id, nil
}
