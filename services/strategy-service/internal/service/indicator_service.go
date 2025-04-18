package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"services/strategy-service/internal/model"
	"services/strategy-service/internal/repository"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// IndicatorService handles technical indicator operations
type IndicatorService struct {
	db            *sqlx.DB
	indicatorRepo *repository.IndicatorRepository
	logger        *zap.Logger
}

// NewIndicatorService creates a new indicator service
func NewIndicatorService(db *sqlx.DB, indicatorRepo *repository.IndicatorRepository, logger *zap.Logger) *IndicatorService {
	return &IndicatorService{
		db:            db,
		indicatorRepo: indicatorRepo,
		logger:        logger,
	}
}

// GetDB provides direct access to the database for debugging purposes
func (s *IndicatorService) GetDB() *sqlx.DB {
	return s.db
}

// GetAllIndicators retrieves all technical indicators with their parameters and enum values
func (s *IndicatorService) GetAllIndicators(ctx context.Context, searchTerm string, categories []string, active *bool, page, limit int) ([]model.TechnicalIndicator, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	return s.indicatorRepo.GetAll(ctx, searchTerm, categories, active, page, limit)
}

// CreateIndicator creates a new technical indicator with parameters and enum values
func (s *IndicatorService) CreateIndicator(
	ctx context.Context,
	name, description, category, formula string,
	minValue, maxValue *float64,
	isActive *bool,
	parameters []model.IndicatorParameterCreate,
) (*model.TechnicalIndicator, error) {
	// Validate input
	if name == "" {
		return nil, errors.New("indicator name is required")
	}
	if category == "" {
		return nil, errors.New("indicator category is required")
	}

	// Create a transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, err
	}
	defer tx.Rollback() // Rollback if we don't commit

	// Set default value for isActive if it's nil
	active := true
	if isActive != nil {
		active = *isActive
	}

	// Create the indicator
	indicator := &model.TechnicalIndicator{
		Name:        name,
		Description: description,
		Category:    category,
		Formula:     formula,
		MinValue:    minValue,
		MaxValue:    maxValue,
		IsActive:    active,
		CreatedAt:   time.Now(),
	}

	// Insert the indicator
	indicatorID, err := s.indicatorRepo.Create(ctx, indicator)
	if err != nil {
		s.logger.Error("Failed to create indicator", zap.Error(err))
		return nil, err
	}

	// Set the ID in the returned model
	indicator.ID = indicatorID
	indicator.Parameters = make([]model.IndicatorParameter, 0, len(parameters))

	// Add parameters if provided
	for _, paramCreate := range parameters {
		// Set the indicator ID for this parameter
		paramCreate.IndicatorID = indicatorID

		// Create the parameter
		paramID, err := s.indicatorRepo.CreateParameter(ctx, &paramCreate)
		if err != nil {
			s.logger.Error("Failed to add parameter", zap.Error(err))
			return nil, err
		}

		// Create a parameter object for the response
		param := model.IndicatorParameter{
			ID:            paramID,
			IndicatorID:   indicatorID,
			ParameterName: paramCreate.ParameterName,
			ParameterType: paramCreate.ParameterType,
			IsRequired:    paramCreate.IsRequired,
			MinValue:      paramCreate.MinValue,
			MaxValue:      paramCreate.MaxValue,
			DefaultValue:  paramCreate.DefaultValue,
			Description:   paramCreate.Description,
			EnumValues:    make([]model.ParameterEnumValue, 0),
		}

		// Add enum values if provided
		for _, enumValueCreate := range paramCreate.EnumValues {
			// Set the parameter ID for this enum value
			enumValueCreate.ParameterID = paramID

			// Create the enum value
			enumID, err := s.indicatorRepo.CreateEnumValue(ctx, &enumValueCreate)
			if err != nil {
				s.logger.Error("Failed to add enum value", zap.Error(err))
				return nil, err
			}

			// Add the enum value to the parameter
			param.EnumValues = append(param.EnumValues, model.ParameterEnumValue{
				ID:          enumID,
				ParameterID: paramID,
				EnumValue:   enumValueCreate.EnumValue,
				DisplayName: enumValueCreate.DisplayName,
			})
		}

		// Add the parameter to the indicator
		indicator.Parameters = append(indicator.Parameters, param)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Successfully created indicator",
		zap.Int("id", indicatorID),
		zap.String("name", name),
		zap.Int("parameters_count", len(parameters)),
		zap.Bool("is_active", active))

	return indicator, nil
}

// AddEnumValue adds an enum value to a parameter
func (s *IndicatorService) AddEnumValue(
	ctx context.Context,
	parameterID int,
	enumValue string,
	displayName string,
) (*model.ParameterEnumValue, error) {
	// Create enum value object
	enumCreate := &model.ParameterEnumValueCreate{
		ParameterID: parameterID,
		EnumValue:   enumValue,
		DisplayName: displayName,
	}

	// Add enum value to database
	enumID, err := s.indicatorRepo.CreateEnumValue(ctx, enumCreate)
	if err != nil {
		return nil, err
	}

	// Return the created enum value
	return &model.ParameterEnumValue{
		ID:          enumID,
		ParameterID: parameterID,
		EnumValue:   enumValue,
		DisplayName: displayName,
	}, nil
}

// GetIndicator retrieves a specific indicator by ID with parameters and enum values
func (s *IndicatorService) GetIndicator(ctx context.Context, id int) (*model.TechnicalIndicator, error) {
	indicator, err := s.indicatorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if indicator == nil {
		return nil, errors.New("indicator not found")
	}

	return indicator, nil
}

// GetCategories retrieves indicator categories
type CategoryInfo struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// GetCategories retrieves indicator categories
func (s *IndicatorService) GetCategories(ctx context.Context) ([]CategoryInfo, error) {
	repoCategories, err := s.indicatorRepo.GetCategories(ctx)
	if err != nil {
		s.logger.Error("Failed to get indicator categories", zap.Error(err))
		return nil, err
	}

	// Convert repository type to service type
	serviceCategories := make([]CategoryInfo, len(repoCategories))
	for i, cat := range repoCategories {
		serviceCategories[i] = CategoryInfo{
			Category: cat.Category,
			Count:    cat.Count,
		}
	}

	// If no categories found, return empty array
	if len(serviceCategories) == 0 {
		return []CategoryInfo{}, nil
	}

	return serviceCategories, nil
}

// DeleteIndicator deletes an indicator by ID
func (s *IndicatorService) DeleteIndicator(ctx context.Context, id int) error {
	// Check if indicator exists
	indicator, err := s.indicatorRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if indicator == nil {
		return errors.New("indicator not found")
	}

	return s.indicatorRepo.Delete(ctx, id)
}

// UpdateIndicator updates an indicator
func (s *IndicatorService) UpdateIndicator(ctx context.Context, id int, update *model.TechnicalIndicator) (*model.TechnicalIndicator, error) {
	// Check if indicator exists
	indicator, err := s.indicatorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if indicator == nil {
		return nil, errors.New("indicator not found")
	}

	// Update indicator
	err = s.indicatorRepo.Update(ctx, id, update)
	if err != nil {
		return nil, err
	}

	// Get the updated indicator
	updatedIndicator, err := s.indicatorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return updatedIndicator, nil
}

// UpdateParameter updates a parameter with proper error handling
func (s *IndicatorService) UpdateParameter(ctx context.Context, id int, param *model.IndicatorParameter) (*model.IndicatorParameter, error) {
	if param == nil {
		return nil, errors.New("parameter data cannot be nil")
	}

	// First check if parameter exists
	existingParam, err := s.getParameterByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to check if parameter exists", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("error checking parameter: %w", err)
	}

	if existingParam == nil {
		s.logger.Warn("Parameter not found during update", zap.Int("id", id))
		return nil, fmt.Errorf("parameter with ID %d not found", id)
	}

	// Set the indicator ID from the existing parameter to maintain relationship
	param.IndicatorID = existingParam.IndicatorID

	// Now update the parameter in the repository
	err = s.indicatorRepo.UpdateParameter(ctx, id, param)
	if err != nil {
		s.logger.Error("Failed to update parameter in repository",
			zap.Error(err),
			zap.Int("id", id),
			zap.String("parameter_name", param.ParameterName))
		return nil, fmt.Errorf("failed to update parameter: %w", err)
	}

	// Get the updated parameter to return
	updatedParam, err := s.getParameterByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to retrieve updated parameter", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("parameter was updated but could not retrieve updated data: %w", err)
	}

	if updatedParam == nil {
		s.logger.Error("Updated parameter not found after update", zap.Int("id", id))
		return nil, errors.New("parameter was updated but could not be found")
	}

	// Get enum values if any
	enumValues, err := s.getEnumValuesByParameterID(ctx, id)
	if err != nil {
		s.logger.Warn("Failed to retrieve enum values for parameter",
			zap.Error(err),
			zap.Int("parameter_id", id))
		// Don't fail the whole operation, just log the warning
	}

	updatedParam.EnumValues = enumValues

	s.logger.Info("Successfully updated parameter",
		zap.Int("id", id),
		zap.String("name", updatedParam.ParameterName),
		zap.Int("indicator_id", updatedParam.IndicatorID))

	return updatedParam, nil
}

// DeleteEnumValue deletes an enum value by ID
func (s *IndicatorService) DeleteEnumValue(ctx context.Context, id int) error {
	return s.indicatorRepo.DeleteEnumValue(ctx, id)
}

// UpdateEnumValue updates an enum value
func (s *IndicatorService) UpdateEnumValue(ctx context.Context, id int, enumVal *model.ParameterEnumValue) (*model.ParameterEnumValue, error) {
	// Update enum value
	err := s.indicatorRepo.UpdateEnumValue(ctx, id, enumVal)
	if err != nil {
		return nil, err
	}

	// Get the updated enum value
	// Since there's no direct method to get an enum value by ID, we return the enum value that was passed in
	// with the ID that was specified
	result := *enumVal
	result.ID = id

	return &result, nil
}

// getParameterByID retrieves a parameter by ID with proper error handling
func (s *IndicatorService) getParameterByID(ctx context.Context, id int) (*model.IndicatorParameter, error) {
	if id <= 0 {
		return nil, errors.New("invalid parameter ID")
	}

	// Query to get parameter by ID
	query := `
        SELECT id, indicator_id, parameter_name, parameter_type, is_required, 
               min_value, max_value, default_value, description
        FROM indicator_parameters
        WHERE id = $1
    `

	// Use a temporary struct with sql.NullString for nullable string fields
	type tempParameter struct {
		ID            int             `db:"id"`
		IndicatorID   int             `db:"indicator_id"`
		ParameterName string          `db:"parameter_name"`
		ParameterType string          `db:"parameter_type"`
		IsRequired    bool            `db:"is_required"`
		MinValue      sql.NullFloat64 `db:"min_value"`
		MaxValue      sql.NullFloat64 `db:"max_value"`
		DefaultValue  sql.NullString  `db:"default_value"`
		Description   sql.NullString  `db:"description"`
	}

	var tempParam tempParameter
	err := s.db.GetContext(ctx, &tempParam, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Parameter not found
		}
		s.logger.Error("Database error while getting parameter", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Convert the temporary struct to a model struct
	param := &model.IndicatorParameter{
		ID:            tempParam.ID,
		IndicatorID:   tempParam.IndicatorID,
		ParameterName: tempParam.ParameterName,
		ParameterType: tempParam.ParameterType,
		IsRequired:    tempParam.IsRequired,
		EnumValues:    []model.ParameterEnumValue{},
	}

	// Only set string values if they are valid (not NULL)
	if tempParam.DefaultValue.Valid {
		param.DefaultValue = tempParam.DefaultValue.String
	}
	if tempParam.Description.Valid {
		param.Description = tempParam.Description.String
	}

	// Set float values if they are valid
	if tempParam.MinValue.Valid {
		minValue := tempParam.MinValue.Float64
		param.MinValue = &minValue
	}
	if tempParam.MaxValue.Valid {
		maxValue := tempParam.MaxValue.Float64
		param.MaxValue = &maxValue
	}

	return param, nil
}

// getEnumValuesByParameterID gets enum values for a parameter
func (s *IndicatorService) getEnumValuesByParameterID(ctx context.Context, parameterID int) ([]model.ParameterEnumValue, error) {
	if parameterID <= 0 {
		return []model.ParameterEnumValue{}, nil
	}

	query := `
        SELECT id, parameter_id, enum_value, display_name
        FROM parameter_enum_values
        WHERE parameter_id = $1
        ORDER BY id
    `

	var enumValues []model.ParameterEnumValue
	err := s.db.SelectContext(ctx, &enumValues, query, parameterID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []model.ParameterEnumValue{}, nil
		}
		return nil, fmt.Errorf("failed to retrieve enum values: %w", err)
	}

	return enumValues, nil
}

// DeleteParameter deletes a parameter with proper error handling
func (s *IndicatorService) DeleteParameter(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("invalid parameter ID")
	}

	// First check if parameter exists
	existingParam, err := s.getParameterByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to check if parameter exists", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("error checking parameter: %w", err)
	}

	if existingParam == nil {
		s.logger.Warn("Parameter not found during delete", zap.Int("id", id))
		return fmt.Errorf("parameter with ID %d not found", id)
	}

	// Log the operation
	s.logger.Info("Deleting parameter",
		zap.Int("id", id),
		zap.String("name", existingParam.ParameterName),
		zap.Int("indicator_id", existingParam.IndicatorID))

	// Delete the parameter
	err = s.indicatorRepo.DeleteParameter(ctx, id)
	if err != nil {
		s.logger.Error("Failed to delete parameter", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("failed to delete parameter: %w", err)
	}

	return nil
}

// AddParameter adds a parameter to an indicator with proper error handling
func (s *IndicatorService) AddParameter(
	ctx context.Context,
	indicatorID int,
	paramName string,
	paramType string,
	isRequired bool,
	minValue *float64,
	maxValue *float64,
	defaultValue string,
	description string,
) (*model.IndicatorParameter, error) {
	// Validate indicator ID
	if indicatorID <= 0 {
		return nil, errors.New("invalid indicator ID")
	}

	// Validate parameter name
	if paramName == "" {
		return nil, errors.New("parameter name is required")
	}

	// Check if indicator exists
	indicator, err := s.GetIndicator(ctx, indicatorID)
	if err != nil {
		s.logger.Error("Failed to get indicator", zap.Error(err), zap.Int("indicator_id", indicatorID))
		return nil, fmt.Errorf("error checking indicator: %w", err)
	}

	if indicator == nil {
		return nil, fmt.Errorf("indicator with ID %d not found", indicatorID)
	}

	// Create parameter object
	paramCreate := &model.IndicatorParameterCreate{
		IndicatorID:   indicatorID,
		ParameterName: paramName,
		ParameterType: paramType,
		IsRequired:    isRequired,
		MinValue:      minValue,
		MaxValue:      maxValue,
		DefaultValue:  defaultValue,
		Description:   description,
	}

	// Add parameter to database
	paramID, err := s.indicatorRepo.CreateParameter(ctx, paramCreate)
	if err != nil {
		s.logger.Error("Failed to create parameter",
			zap.Error(err),
			zap.Int("indicator_id", indicatorID),
			zap.String("parameter_name", paramName))
		return nil, fmt.Errorf("failed to create parameter: %w", err)
	}

	// Get the created parameter
	newParam, err := s.getParameterByID(ctx, paramID)
	if err != nil {
		s.logger.Error("Failed to retrieve newly created parameter", zap.Error(err), zap.Int("id", paramID))
		return nil, fmt.Errorf("parameter was created but could not retrieve: %w", err)
	}

	if newParam == nil {
		s.logger.Error("Created parameter not found after creation", zap.Int("id", paramID))
		return nil, errors.New("parameter was created but could not be found")
	}

	s.logger.Info("Successfully created parameter",
		zap.Int("id", paramID),
		zap.String("name", paramName),
		zap.Int("indicator_id", indicatorID))

	return newParam, nil
}

// SyncIndicatorsFromBacktestingService syncs indicators from the backtesting service
func (s *IndicatorService) SyncIndicatorsFromBacktestingService(ctx context.Context) (int, error) {
	// Get backtesting service URL from environment or use default
	backtestingServiceURL := os.Getenv("BACKTEST_SERVICE_URL")
	if backtestingServiceURL == "" {
		// Default base URL if environment variable not set
		backtestingServiceURL = "http://backtesting-service:5000" // Corregido: name es "backtesting-service", no "backtest-service"
	}

	// Add the endpoint path
	indicatorsURL := backtestingServiceURL + "/indicators"

	s.logger.Info("Connecting to backtesting service", zap.String("url", indicatorsURL))

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make request to backtesting service
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indicatorsURL, nil)
	if err != nil {
		s.logger.Error("Failed to create request to backtesting service", zap.Error(err))
		return 0, err
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Failed to get indicators from backtesting service",
			zap.Error(err),
			zap.String("url", indicatorsURL))

		// Try alternative names with different formats
		alternativeURLs := []string{
			"http://backtest-service:5000/indicators",
			"http://backtesting_service:5000/indicators",
			"http://backtest_service:5000/indicators",
		}

		for _, altURL := range alternativeURLs {
			s.logger.Info("Trying alternative URL", zap.String("url", altURL))
			altReq, err := http.NewRequestWithContext(ctx, http.MethodGet, altURL, nil)
			if err != nil {
				continue
			}

			altResp, err := client.Do(altReq)
			if err == nil {
				s.logger.Info("Successfully connected using alternative URL", zap.String("url", altURL))
				resp = altResp
				break
			}
		}

		if resp == nil {
			return 0, fmt.Errorf("all connection attempts to backtesting service failed: %w", err)
		}
	}

	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		s.logger.Error("Backtesting service returned non-200 status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(bodyBytes)))
		return 0, fmt.Errorf("backtesting service returned status code %d", resp.StatusCode)
	}

	// Parse response
	var indicators []model.IndicatorFromBacktesting
	err = json.NewDecoder(resp.Body).Decode(&indicators)
	if err != nil {
		s.logger.Error("Failed to decode indicators response", zap.Error(err))
		return 0, err
	}

	// Sync indicators using repository
	syncedCount, err := s.indicatorRepo.SyncIndicators(ctx, indicators)
	if err != nil {
		return 0, fmt.Errorf("failed to sync indicators: %w", err)
	}

	return syncedCount, nil
}
