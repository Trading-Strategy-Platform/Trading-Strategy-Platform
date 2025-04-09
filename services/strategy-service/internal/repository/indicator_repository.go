package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"services/strategy-service/internal/model"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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

// GetAll retrieves all indicators with filtering options
func (r *IndicatorRepository) GetAll(ctx context.Context, searchTerm string, categories []string, page, limit int) ([]model.TechnicalIndicator, int, error) {
	// Use the new get_indicators function with enhanced parameters
	query := `
        SELECT id, name, description, category, formula, created_at, updated_at 
        FROM get_indicators($1, $2)
    `

	var args []interface{}
	args = append(args, searchTerm) // Can be empty string

	// Convert categories slice to PostgreSQL array
	if len(categories) > 0 {
		args = append(args, pq.Array(categories))
	} else {
		args = append(args, nil) // NULL for empty categories
	}

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to execute get indicators query", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	// Scan rows into indicator objects
	var allIndicators []model.TechnicalIndicator
	for rows.Next() {
		var indicator model.TechnicalIndicator
		var updatedAt sql.NullTime

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

// GetByID retrieves an indicator by ID using the get_indicator_by_id function
func (r *IndicatorRepository) GetByID(ctx context.Context, id int) (*model.TechnicalIndicator, error) {
	// Use the get_indicator_by_id function that includes parameters
	query := `
        SELECT id, name, description, category, formula, created_at, updated_at, parameters
        FROM get_indicator_by_id($1)
    `

	// Execute query
	var indicator model.TechnicalIndicator
	var updatedAt sql.NullTime
	var parametersJSON []byte // To store the JSON array of parameters

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&indicator.ID,
		&indicator.Name,
		&indicator.Description,
		&indicator.Category,
		&indicator.Formula,
		&indicator.CreatedAt,
		&updatedAt,
		&parametersJSON,
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

	// Parse parameters JSON
	if parametersJSON != nil {
		var params []model.IndicatorParameter

		if err := json.Unmarshal(parametersJSON, &params); err != nil {
			r.logger.Error("Failed to unmarshal parameters JSON", zap.Error(err))
		} else {
			indicator.Parameters = params
		}
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
