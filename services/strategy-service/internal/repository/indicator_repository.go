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

// GetAll retrieves all indicators with parameters and enum values using get_indicators function
func (r *IndicatorRepository) GetAll(ctx context.Context, searchTerm string, categories []string, page, limit int) ([]model.TechnicalIndicator, int, error) {
	// Use the get_indicators function directly
	query := `SELECT * FROM get_indicators($1, $2)`

	var args []interface{}
	args = append(args, searchTerm)

	// Handle categories
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
		var parametersJSON []byte

		err := rows.Scan(
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
			r.logger.Error("Failed to scan indicator row", zap.Error(err))
			return nil, 0, err
		}

		// Convert nullable time
		if updatedAt.Valid {
			indicator.UpdatedAt = &updatedAt.Time
		}

		// Parse parameters from JSON
		if len(parametersJSON) > 0 {
			var paramsArray []json.RawMessage
			err = json.Unmarshal(parametersJSON, &paramsArray)
			if err != nil {
				r.logger.Error("Failed to unmarshal parameters array", zap.Error(err))
				continue
			}

			indicator.Parameters = make([]model.IndicatorParameter, 0, len(paramsArray))

			for _, paramJSON := range paramsArray {
				var param struct {
					ID           int                        `json:"id"`
					Name         string                     `json:"name"`
					Type         string                     `json:"type"`
					IsRequired   bool                       `json:"is_required"`
					MinValue     *float64                   `json:"min_value,omitempty"`
					MaxValue     *float64                   `json:"max_value,omitempty"`
					DefaultValue string                     `json:"default_value,omitempty"`
					Description  string                     `json:"description,omitempty"`
					EnumValues   []model.ParameterEnumValue `json:"enum_values,omitempty"`
				}

				err = json.Unmarshal(paramJSON, &param)
				if err != nil {
					r.logger.Error("Failed to unmarshal parameter", zap.Error(err))
					continue
				}

				indicator.Parameters = append(indicator.Parameters, model.IndicatorParameter{
					ID:            param.ID,
					IndicatorID:   indicator.ID,
					ParameterName: param.Name,
					ParameterType: param.Type,
					IsRequired:    param.IsRequired,
					MinValue:      param.MinValue,
					MaxValue:      param.MaxValue,
					DefaultValue:  param.DefaultValue,
					Description:   param.Description,
					EnumValues:    param.EnumValues,
				})
			}
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

// GetByID retrieves an indicator by ID with parameters and enum values
func (r *IndicatorRepository) GetByID(ctx context.Context, id int) (*model.TechnicalIndicator, error) {
	// Use the get_indicator_by_id function
	query := `SELECT * FROM get_indicator_by_id($1)`

	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to execute get indicator by ID query", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	// We should get only one row
	if !rows.Next() {
		return nil, nil // No indicator found
	}

	var indicator model.TechnicalIndicator
	var updatedAt sql.NullTime
	var parametersJSON []byte

	err = rows.Scan(
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
		r.logger.Error("Failed to scan indicator row", zap.Error(err))
		return nil, err
	}

	// Convert nullable time
	if updatedAt.Valid {
		indicator.UpdatedAt = &updatedAt.Time
	}

	// Parse parameters from JSON
	if len(parametersJSON) > 0 {
		var paramsArray []json.RawMessage
		err = json.Unmarshal(parametersJSON, &paramsArray)
		if err != nil {
			r.logger.Error("Failed to unmarshal parameters array", zap.Error(err))
		} else {
			indicator.Parameters = make([]model.IndicatorParameter, 0, len(paramsArray))

			for _, paramJSON := range paramsArray {
				var param struct {
					ID           int                        `json:"id"`
					Name         string                     `json:"name"`
					Type         string                     `json:"type"`
					IsRequired   bool                       `json:"is_required"`
					MinValue     *float64                   `json:"min_value,omitempty"`
					MaxValue     *float64                   `json:"max_value,omitempty"`
					DefaultValue string                     `json:"default_value,omitempty"`
					Description  string                     `json:"description,omitempty"`
					EnumValues   []model.ParameterEnumValue `json:"enum_values,omitempty"`
				}

				err = json.Unmarshal(paramJSON, &param)
				if err != nil {
					r.logger.Error("Failed to unmarshal parameter", zap.Error(err))
					continue
				}

				indicator.Parameters = append(indicator.Parameters, model.IndicatorParameter{
					ID:            param.ID,
					IndicatorID:   indicator.ID,
					ParameterName: param.Name,
					ParameterType: param.Type,
					IsRequired:    param.IsRequired,
					MinValue:      param.MinValue,
					MaxValue:      param.MaxValue,
					DefaultValue:  param.DefaultValue,
					Description:   param.Description,
					EnumValues:    param.EnumValues,
				})
			}
		}
	}

	return &indicator, nil
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

// CreateParameter adds a parameter to an indicator
func (r *IndicatorRepository) CreateParameter(ctx context.Context, parameter *model.IndicatorParameterCreate) (int, error) {
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
		parameter.IndicatorID,
		parameter.ParameterName,
		parameter.ParameterType,
		parameter.IsRequired,
		parameter.MinValue,
		parameter.MaxValue,
		parameter.DefaultValue,
		parameter.Description,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create parameter", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// CreateEnumValue adds an enum value to a parameter
func (r *IndicatorRepository) CreateEnumValue(ctx context.Context, enumValue *model.ParameterEnumValueCreate) (int, error) {
	query := `
		INSERT INTO parameter_enum_values (parameter_id, enum_value, display_name)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		enumValue.ParameterID,
		enumValue.EnumValue,
		enumValue.DisplayName,
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to create enum value", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// GetCategories retrieves indicator categories
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
