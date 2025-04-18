package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
func (r *IndicatorRepository) GetAll(ctx context.Context, searchTerm string, categories []string, active *bool, page, limit int) ([]model.TechnicalIndicator, int, error) {
	// Use the get_indicators function directly
	query := `SELECT * FROM get_indicators($1, $2, $3)`

	var args []interface{}
	args = append(args, searchTerm)

	// Handle categories
	if len(categories) > 0 {
		args = append(args, pq.Array(categories))
	} else {
		args = append(args, nil)
	}

	// Handle active filter
	args = append(args, active)

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
		// Use sql.NullString for fields that can be NULL
		var formulaNull sql.NullString
		var minValueNull sql.NullFloat64
		var maxValueNull sql.NullFloat64

		err := rows.Scan(
			&indicator.ID,
			&indicator.Name,
			&indicator.Description,
			&indicator.Category,
			&formulaNull,
			&minValueNull,
			&maxValueNull,
			&indicator.IsActive,
			&indicator.CreatedAt,
			&updatedAt,
			&parametersJSON,
		)
		if err != nil {
			r.logger.Error("Failed to scan indicator row", zap.Error(err))
			return nil, 0, err
		}

		// Assign values of NULL fields
		if formulaNull.Valid {
			indicator.Formula = formulaNull.String
		} else {
			indicator.Formula = ""
		}

		// Assign values for min_value and max_value only if valid
		if minValueNull.Valid {
			indicator.MinValue = &minValueNull.Float64
		}

		if maxValueNull.Valid {
			indicator.MaxValue = &maxValueNull.Float64
		}

		// Convert nullable time
		if updatedAt.Valid {
			indicator.UpdatedAt = &updatedAt.Time
		}

		// Parse parameters from JSON - now we expect a direct JSON array instead of an array type
		indicator.Parameters = []model.IndicatorParameter{} // Initialize with empty array

		if len(parametersJSON) > 0 && string(parametersJSON) != "[]" && string(parametersJSON) != "null" {
			// Parse the JSON array of parameters
			var paramsArray []json.RawMessage
			err = json.Unmarshal(parametersJSON, &paramsArray)
			if err != nil {
				r.logger.Error("Failed to unmarshal parameters array",
					zap.Error(err),
					zap.String("parametersJSON", string(parametersJSON)))
				continue // Skip but don't fail completely
			}

			for _, paramJSON := range paramsArray {
				var param struct {
					ID           int             `json:"id"`
					Name         string          `json:"name"`
					Type         string          `json:"type"`
					IsRequired   bool            `json:"is_required"`
					MinValue     *float64        `json:"min_value,omitempty"`
					MaxValue     *float64        `json:"max_value,omitempty"`
					DefaultValue string          `json:"default_value,omitempty"`
					Description  string          `json:"description,omitempty"`
					EnumValues   json.RawMessage `json:"enum_values,omitempty"`
				}

				err = json.Unmarshal(paramJSON, &param)
				if err != nil {
					r.logger.Error("Failed to unmarshal parameter",
						zap.Error(err),
						zap.String("paramJSON", string(paramJSON)))
					continue
				}

				// Create parameter with basic properties
				parameterObj := model.IndicatorParameter{
					ID:            param.ID,
					IndicatorID:   indicator.ID,
					ParameterName: param.Name,
					ParameterType: param.Type,
					IsRequired:    param.IsRequired,
					MinValue:      param.MinValue,
					MaxValue:      param.MaxValue,
					DefaultValue:  param.DefaultValue,
					Description:   param.Description,
					EnumValues:    []model.ParameterEnumValue{},
				}

				// Parse enum values separately if they exist
				if len(param.EnumValues) > 0 && string(param.EnumValues) != "null" && string(param.EnumValues) != "[]" {
					var enumValues []struct {
						ID          int    `json:"id"`
						EnumValue   string `json:"enum_value"`
						DisplayName string `json:"display_name"`
					}

					if err := json.Unmarshal(param.EnumValues, &enumValues); err != nil {
						r.logger.Warn("Failed to unmarshal enum values",
							zap.Error(err),
							zap.String("enum_values_json", string(param.EnumValues)))
					} else {
						for _, ev := range enumValues {
							parameterObj.EnumValues = append(parameterObj.EnumValues, model.ParameterEnumValue{
								ID:          ev.ID,
								ParameterID: param.ID,
								EnumValue:   ev.EnumValue,
								DisplayName: ev.DisplayName,
							})
						}
					}
				}

				indicator.Parameters = append(indicator.Parameters, parameterObj)
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
	// Use sql.NullString and sql.NullFloat64 for fields that can be NULL
	var formulaNull sql.NullString
	var minValueNull sql.NullFloat64
	var maxValueNull sql.NullFloat64

	err = rows.Scan(
		&indicator.ID,
		&indicator.Name,
		&indicator.Description,
		&indicator.Category,
		&formulaNull,
		&minValueNull,
		&maxValueNull,
		&indicator.IsActive,
		&indicator.CreatedAt,
		&updatedAt,
		&parametersJSON,
	)

	if err != nil {
		r.logger.Error("Failed to scan indicator row", zap.Error(err))
		return nil, err
	}

	// Asignar valores de los campos NULL
	if formulaNull.Valid {
		indicator.Formula = formulaNull.String
	} else {
		indicator.Formula = "" // Asignar string vacío cuando es NULL
	}

	// Asignar valores para min_value y max_value solo si son válidos
	if minValueNull.Valid {
		indicator.MinValue = &minValueNull.Float64
	}

	if maxValueNull.Valid {
		indicator.MaxValue = &maxValueNull.Float64
	}

	// Convert nullable time
	if updatedAt.Valid {
		indicator.UpdatedAt = &updatedAt.Time
	}

	// Initialize parameters with empty array
	indicator.Parameters = []model.IndicatorParameter{}

	// Parse parameters from JSON - now we expect a direct JSON array
	if len(parametersJSON) > 0 && string(parametersJSON) != "[]" && string(parametersJSON) != "null" {
		// Parse the JSON array of parameters
		var paramsArray []json.RawMessage
		err = json.Unmarshal(parametersJSON, &paramsArray)
		if err != nil {
			r.logger.Error("Failed to unmarshal parameters array",
				zap.Error(err),
				zap.String("parametersJSON", string(parametersJSON)))
			return &indicator, nil // Return indicator without parameters rather than failing
		}

		for _, paramJSON := range paramsArray {
			var param struct {
				ID           int             `json:"id"`
				Name         string          `json:"name"`
				Type         string          `json:"type"`
				IsRequired   bool            `json:"is_required"`
				MinValue     *float64        `json:"min_value,omitempty"`
				MaxValue     *float64        `json:"max_value,omitempty"`
				DefaultValue string          `json:"default_value,omitempty"`
				Description  string          `json:"description,omitempty"`
				EnumValues   json.RawMessage `json:"enum_values,omitempty"`
			}

			err = json.Unmarshal(paramJSON, &param)
			if err != nil {
				r.logger.Error("Failed to unmarshal parameter",
					zap.Error(err),
					zap.String("paramJSON", string(paramJSON)))
				continue
			}

			// Create parameter with basic properties
			parameterObj := model.IndicatorParameter{
				ID:            param.ID,
				IndicatorID:   indicator.ID,
				ParameterName: param.Name,
				ParameterType: param.Type,
				IsRequired:    param.IsRequired,
				MinValue:      param.MinValue,
				MaxValue:      param.MaxValue,
				DefaultValue:  param.DefaultValue,
				Description:   param.Description,
				EnumValues:    []model.ParameterEnumValue{},
			}

			// Parse enum values separately if they exist
			if len(param.EnumValues) > 0 && string(param.EnumValues) != "null" && string(param.EnumValues) != "[]" {
				var enumValues []struct {
					ID          int    `json:"id"`
					EnumValue   string `json:"enum_value"`
					DisplayName string `json:"display_name"`
				}

				if err := json.Unmarshal(param.EnumValues, &enumValues); err != nil {
					r.logger.Warn("Failed to unmarshal enum values",
						zap.Error(err),
						zap.String("enum_values_json", string(param.EnumValues)))
				} else {
					for _, ev := range enumValues {
						parameterObj.EnumValues = append(parameterObj.EnumValues, model.ParameterEnumValue{
							ID:          ev.ID,
							ParameterID: param.ID,
							EnumValue:   ev.EnumValue,
							DisplayName: ev.DisplayName,
						})
					}
				}
			}

			indicator.Parameters = append(indicator.Parameters, parameterObj)
		}
	}

	return &indicator, nil
}

// Create adds a new indicator to the database
func (r *IndicatorRepository) Create(ctx context.Context, indicator *model.TechnicalIndicator) (int, error) {
	query := `
		INSERT INTO indicators (name, description, category, formula, min_value, max_value, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
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
		indicator.MinValue,
		indicator.MaxValue,
		indicator.IsActive, // Added the is_active field
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
type CategoryData struct {
	Category string `db:"category" json:"category"`
	Count    int64  `db:"count" json:"count"`
}

// GetCategories retrieves indicator categories
func (r *IndicatorRepository) GetCategories(ctx context.Context) ([]CategoryData, error) {
	query := `SELECT * FROM get_indicator_categories()`

	var categories []CategoryData

	err := r.db.SelectContext(ctx, &categories, query)
	if err != nil {
		r.logger.Error("Failed to get indicator categories", zap.Error(err))
		return nil, err
	}

	// Return an empty array instead of nil if no categories found
	if categories == nil {
		categories = []CategoryData{}
	}

	return categories, nil
}

// Delete an indicator by ID
func (r *IndicatorRepository) Delete(ctx context.Context, id int) error {
	query := `SELECT delete_indicator($1)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to delete indicator", zap.Error(err), zap.Int("id", id))
		return err
	}

	if !success {
		return errors.New("indicator not found")
	}

	return nil
}

// Update an indicator
func (r *IndicatorRepository) Update(ctx context.Context, id int, indicator *model.TechnicalIndicator) error {
	query := `SELECT update_indicator($1, $2, $3, $4, $5, $6, $7, $8)`

	var success bool
	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
		indicator.Name,
		indicator.Description,
		indicator.Category,
		indicator.Formula,
		indicator.MinValue,
		indicator.MaxValue,
		indicator.IsActive,
	).Scan(&success)

	if err != nil {
		r.logger.Error("Failed to update indicator", zap.Error(err), zap.Int("id", id))
		return err
	}

	if !success {
		return errors.New("indicator not found")
	}

	return nil
}

// Delete a parameter by ID
func (r *IndicatorRepository) DeleteParameter(ctx context.Context, id int) error {
	query := `SELECT delete_parameter($1)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to delete parameter", zap.Error(err), zap.Int("id", id))
		return err
	}

	if !success {
		return errors.New("parameter not found")
	}

	return nil
}

// Update a parameter
func (r *IndicatorRepository) UpdateParameter(ctx context.Context, id int, param *model.IndicatorParameter) error {
	query := `SELECT update_parameter($1, $2, $3, $4, $5, $6, $7, $8)`

	var success bool
	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
		param.ParameterName,
		param.ParameterType,
		param.IsRequired,
		param.MinValue,
		param.MaxValue,
		param.DefaultValue,
		param.Description,
	).Scan(&success)

	if err != nil {
		r.logger.Error("Failed to update parameter", zap.Error(err), zap.Int("id", id))
		return err
	}

	if !success {
		return errors.New("parameter not found")
	}

	return nil
}

// Delete an enum value by ID
func (r *IndicatorRepository) DeleteEnumValue(ctx context.Context, id int) error {
	query := `SELECT delete_enum_value($1)`

	var success bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&success)
	if err != nil {
		r.logger.Error("Failed to delete enum value", zap.Error(err), zap.Int("id", id))
		return err
	}

	if !success {
		return errors.New("enum value not found")
	}

	return nil
}

// Update an enum value
func (r *IndicatorRepository) UpdateEnumValue(ctx context.Context, id int, enumVal *model.ParameterEnumValue) error {
	query := `SELECT update_enum_value($1, $2, $3)`

	var success bool
	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
		enumVal.EnumValue,
		enumVal.DisplayName,
	).Scan(&success)

	if err != nil {
		r.logger.Error("Failed to update enum value", zap.Error(err), zap.Int("id", id))
		return err
	}

	if !success {
		return errors.New("enum value not found")
	}

	return nil
}

// SyncIndicators syncs indicators from the provided list
func (r *IndicatorRepository) SyncIndicators(ctx context.Context, indicators []model.IndicatorFromBacktesting) (int, error) {
	// Start a transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction", zap.Error(err))
		return 0, err
	}
	defer tx.Rollback() // Rollback if not committed

	// Get existing indicators from database
	var existingIndicators []struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}
	err = tx.SelectContext(ctx, &existingIndicators, "SELECT id, name FROM indicators")
	if err != nil {
		r.logger.Error("Failed to get existing indicators", zap.Error(err))
		return 0, err
	}

	// Create map of existing indicators by name for easy lookup
	existingIndicatorMap := make(map[string]int)
	for _, indicator := range existingIndicators {
		existingIndicatorMap[indicator.Name] = indicator.ID
	}

	// Count of synced indicators
	syncedCount := 0

	// Process each indicator
	for _, indicator := range indicators {
		// Determine if this is an update or insert
		indicatorID, exists := existingIndicatorMap[indicator.Name]

		if exists {
			// Update existing indicator - now setting is_active=false
			_, err = tx.ExecContext(ctx,
				"UPDATE indicators SET description = $1, is_active = $2, updated_at = NOW() WHERE id = $3",
				indicator.Description, false, indicatorID)
			if err != nil {
				r.logger.Error("Failed to update indicator",
					zap.Error(err),
					zap.String("name", indicator.Name))
				return 0, err
			}
		} else {
			// Insert new indicator - explicitly set is_active=false
			// Categorize indicator based on name
			category := categorizeIndicator(indicator.Name)

			err = tx.QueryRowContext(ctx,
				`INSERT INTO indicators 
                (name, description, category, is_active, created_at, updated_at) 
                VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id`,
				indicator.Name, indicator.Description, category, false).Scan(&indicatorID)
			if err != nil {
				r.logger.Error("Failed to insert indicator",
					zap.Error(err),
					zap.String("name", indicator.Name))
				return 0, err
			}
		}

		// Get existing parameters for this indicator
		var existingParams []struct {
			ID            int    `db:"id"`
			ParameterName string `db:"parameter_name"`
		}
		err = tx.SelectContext(ctx, &existingParams,
			`SELECT id, parameter_name FROM indicator_parameters WHERE indicator_id = $1`,
			indicatorID)
		if err != nil {
			r.logger.Error("Failed to get existing parameters",
				zap.Error(err),
				zap.Int("indicator_id", indicatorID))
			return 0, err
		}

		// Create map of existing parameters by name for easy lookup
		existingParamMap := make(map[string]int)
		for _, param := range existingParams {
			existingParamMap[param.ParameterName] = param.ID
		}

		// Process parameters
		for _, param := range indicator.Parameters {
			paramType := param.Type

			// IMPORTANTE: Truncar el valor por defecto si es demasiado largo
			defaultValue := param.Default
			if len(defaultValue) > 50 {
				r.logger.Warn("Truncating parameter default value that exceeds 50 characters",
					zap.String("param_name", param.Name),
					zap.String("indicator_name", indicator.Name),
					zap.Int("original_length", len(defaultValue)))
				defaultValue = defaultValue[:50]
			}

			// Determine if this is an enum parameter
			hasOptions := len(param.Options) > 0
			if hasOptions {
				paramType = "enum"
			}

			paramID, paramExists := existingParamMap[param.Name]

			if paramExists {
				// Update existing parameter
				_, err = tx.ExecContext(ctx,
					`UPDATE indicator_parameters 
                     SET parameter_type = $1, default_value = $2 
                     WHERE id = $3`,
					paramType, defaultValue, paramID)
				if err != nil {
					r.logger.Error("Failed to update parameter",
						zap.Error(err),
						zap.String("name", param.Name),
						zap.String("default_value", defaultValue),
						zap.Int("default_value_length", len(defaultValue)))
					return 0, err
				}
			} else {
				// Insert new parameter
				err = tx.QueryRowContext(ctx,
					`INSERT INTO indicator_parameters 
                     (indicator_id, parameter_name, parameter_type, default_value, is_required) 
                     VALUES ($1, $2, $3, $4, $5) RETURNING id`,
					indicatorID, param.Name, paramType, defaultValue, true).Scan(&paramID)
				if err != nil {
					r.logger.Error("Failed to insert parameter",
						zap.Error(err),
						zap.String("name", param.Name),
						zap.String("default_value", defaultValue),
						zap.Int("default_value_length", len(defaultValue)))
					return 0, err
				}
			}

			// Handle enum values if present
			if hasOptions {
				// Get existing enum values
				var existingEnums []struct {
					ID        int    `db:"id"`
					EnumValue string `db:"enum_value"`
				}
				err = tx.SelectContext(ctx, &existingEnums,
					`SELECT id, enum_value FROM parameter_enum_values WHERE parameter_id = $1`,
					paramID)
				if err != nil {
					r.logger.Error("Failed to get existing enum values",
						zap.Error(err),
						zap.Int("parameter_id", paramID))
					return 0, err
				}

				// Create map of existing enum values by value for easy lookup
				existingEnumMap := make(map[string]int)
				for _, enum := range existingEnums {
					existingEnumMap[enum.EnumValue] = enum.ID
				}

				// Process each option
				for _, option := range param.Options {
					optionStr := fmt.Sprintf("%v", option) // Convert to string

					// Truncar el valor de enumeración si es demasiado largo (también varchar(50))
					if len(optionStr) > 50 {
						r.logger.Warn("Truncating enum value that exceeds 50 characters",
							zap.String("original_value", optionStr),
							zap.Int("original_length", len(optionStr)))
						optionStr = optionStr[:50]
					}

					if _, enumExists := existingEnumMap[optionStr]; !enumExists {
						// Insert new enum value
						_, err = tx.ExecContext(ctx,
							`INSERT INTO parameter_enum_values 
                             (parameter_id, enum_value, display_name) 
                             VALUES ($1, $2, $3)`,
							paramID, optionStr, optionStr)
						if err != nil {
							r.logger.Error("Failed to insert enum value",
								zap.Error(err),
								zap.String("value", optionStr),
								zap.Int("value_length", len(optionStr)))
							return 0, err
						}
					}
				}
			}
		}

		syncedCount++
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		r.logger.Error("Failed to commit transaction", zap.Error(err))
		return 0, err
	}

	return syncedCount, nil
}

// Helper function to categorize indicators based on their name
func categorizeIndicator(name string) string {
	// Default category
	category := "Other"

	// Check for trend indicators
	trendIndicators := []string{"MA", "EMA", "SMA", "MACD", "ADX", "Ichimoku"}
	for _, indicator := range trendIndicators {
		if strings.Contains(strings.ToUpper(name), strings.ToUpper(indicator)) {
			return "Trend"
		}
	}

	// Check for momentum indicators
	momentumIndicators := []string{"RSI", "CCI", "Stochastic", "TRIX", "ROC", "Momentum"}
	for _, indicator := range momentumIndicators {
		if strings.Contains(strings.ToUpper(name), strings.ToUpper(indicator)) {
			return "Momentum"
		}
	}

	// Check for volatility indicators
	volatilityIndicators := []string{"Bollinger", "ATR", "Volatility", "Standard Deviation"}
	for _, indicator := range volatilityIndicators {
		if strings.Contains(strings.ToUpper(name), strings.ToUpper(indicator)) {
			return "Volatility"
		}
	}

	// Check for volume indicators
	volumeIndicators := []string{"Volume", "OBV", "Money Flow", "Accumulation"}
	for _, indicator := range volumeIndicators {
		if strings.Contains(strings.ToUpper(name), strings.ToUpper(indicator)) {
			return "Volume"
		}
	}

	return category
}
