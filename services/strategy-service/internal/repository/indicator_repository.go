package repository

import (
	"context"
	"database/sql"
	"time"

	"services/strategy-service/internal/model"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// IndicatorRepository handles database operations for technical indicators
type IndicatorRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewIndicatorRepository creates a new indicator repository
func NewIndicatorRepository(db *sqlx.DB, logger *zap.Logger) *IndicatorRepository {
	return &IndicatorRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll retrieves all indicators with optional category filter
func (r *IndicatorRepository) GetAll(ctx context.Context, category *string, page, limit int) ([]model.TechnicalIndicator, int, error) {
	var query string
	var args []interface{}

	// Build query with explicit column selection
	if category != nil {
		query = `
			SELECT id, name, description, category, formula, created_at, updated_at 
			FROM get_indicators($1::VARCHAR)
		`
		args = append(args, *category)
	} else {
		query = `
			SELECT id, name, description, category, formula, created_at, updated_at 
			FROM get_indicators(NULL::VARCHAR)
		`
	}

	// Execute query using QueryContext instead of SelectContext
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to execute get indicators query", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	// Manually scan rows into slices
	var allIndicators []model.TechnicalIndicator
	for rows.Next() {
		var indicator model.TechnicalIndicator
		var updatedAt sql.NullTime

		// Scan row data into variables
		err := rows.Scan(
			&indicator.ID,
			&indicator.Name,
			&indicator.Description,
			&indicator.Category,
			&indicator.Formula,
			&indicator.CreatedAt,
			&updatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan indicator row", zap.Error(err))
			return nil, 0, err
		}

		// Convert nullable time
		if updatedAt.Valid {
			indicator.UpdatedAt = &updatedAt.Time
		}

		allIndicators = append(allIndicators, indicator)
	}

	// Check for any errors encountered during iteration
	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating indicator rows", zap.Error(err))
		return nil, 0, err
	}

	// Get total count
	total := len(allIndicators)

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		return []model.TechnicalIndicator{}, total, nil
	}

	if end > total {
		end = total
	}

	return allIndicators[start:end], total, nil
}

// GetByID retrieves an indicator by ID
func (r *IndicatorRepository) GetByID(ctx context.Context, id int) (*model.TechnicalIndicator, error) {
	// Use explicit column selection to avoid the parameters column
	query := `
		SELECT id, name, description, category, formula, created_at, updated_at
		FROM get_indicator_by_id($1)
	`

	// Execute query
	var indicator model.TechnicalIndicator
	var updatedAt sql.NullTime

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&indicator.ID,
		&indicator.Name,
		&indicator.Description,
		&indicator.Category,
		&indicator.Formula,
		&indicator.CreatedAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get indicator by ID", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	// Convert nullable time
	if updatedAt.Valid {
		indicator.UpdatedAt = &updatedAt.Time
	}

	return &indicator, nil
}

// GetCategories retrieves indicator categories using get_indicator_categories function
func (r *IndicatorRepository) GetCategories(ctx context.Context) ([]struct {
	Category string `db:"category"`
	Count    int    `db:"count"`
}, error) {
	query := `SELECT * FROM get_indicator_categories()`

	var categories []struct {
		Category string `db:"category"`
		Count    int    `db:"count"`
	}
	err := r.db.SelectContext(ctx, &categories, query)
	if err != nil {
		r.logger.Error("Failed to get indicator categories", zap.Error(err))
		return nil, err
	}

	return categories, nil
}

// Create adds a new indicator to the database
func (r *IndicatorRepository) Create(ctx context.Context, indicator *model.TechnicalIndicator) (int, error) {
	query := `
		INSERT INTO indicators (name, description, category, formula, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		indicator.Name,
		indicator.Description,
		indicator.Category,
		indicator.Formula,
		time.Now(),
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create indicator", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// AddParameter adds a parameter to an indicator
func (r *IndicatorRepository) AddParameter(ctx context.Context, parameterCreate *model.IndicatorParameterCreate) (int, error) {
	query := `
		INSERT INTO indicator_parameters (
			indicator_id, parameter_name, parameter_type, is_required, 
			min_value, max_value, default_value, description
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		parameterCreate.IndicatorID,
		parameterCreate.ParameterName,
		parameterCreate.ParameterType,
		parameterCreate.IsRequired,
		parameterCreate.MinValue,
		parameterCreate.MaxValue,
		parameterCreate.DefaultValue,
		parameterCreate.Description,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to add parameter", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// AddEnumValue adds an enum value to a parameter
func (r *IndicatorRepository) AddEnumValue(ctx context.Context, enumCreate *model.ParameterEnumValueCreate) (int, error) {
	query := `
		INSERT INTO parameter_enum_values (parameter_id, enum_value, display_name)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		enumCreate.ParameterID,
		enumCreate.EnumValue,
		enumCreate.DisplayName,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to add enum value", zap.Error(err))
		return 0, err
	}

	return id, nil
}
