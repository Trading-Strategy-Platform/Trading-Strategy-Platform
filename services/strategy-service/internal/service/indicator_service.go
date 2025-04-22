package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

// GetAllIndicators retrieves all technical indicators with their parameters and enum values
// Now includes isAdmin parameter to control visibility of parameters
func (s *IndicatorService) GetAllIndicators(
	ctx context.Context,
	searchTerm string,
	categories []string,
	active *bool,
	sortBy string,
	sortDirection string,
	page,
	limit int,
	isAdmin bool,
) ([]model.TechnicalIndicator, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Validate sort field
	validSortFields := map[string]bool{
		"name":       true,
		"category":   true,
		"created_at": true,
		"updated_at": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "name" // Default sort by name
	}

	// Validate sort direction
	sortDirection = strings.ToUpper(sortDirection)
	if sortDirection != "ASC" && sortDirection != "DESC" {
		sortDirection = "ASC" // Default ascending for indicators
	}

	// Forward the parameters to the repository layer
	return s.indicatorRepo.GetAllIndicators(ctx, searchTerm, categories, active, sortBy, sortDirection, page, limit, isAdmin)
}

// GetIndicator retrieves a specific indicator by ID with parameters and enum values
func (s *IndicatorService) GetIndicator(ctx context.Context, id int, isAdmin bool) (*model.TechnicalIndicator, error) {
	// Forward the isAdmin flag to the repository layer
	indicator, err := s.indicatorRepo.GetIndicatorByID(ctx, id, isAdmin)
	if err != nil {
		return nil, err
	}

	if indicator == nil {
		return nil, errors.New("indicator not found")
	}

	return indicator, nil
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
	indicatorID, err := s.indicatorRepo.CreateIndicator(ctx, indicator)
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
		paramID, err := s.indicatorRepo.CreateIndicatorParameter(ctx, &paramCreate)
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
			IsPublic:      paramCreate.IsPublic,
			EnumValues:    make([]model.ParameterEnumValue, 0),
		}

		// Add enum values if provided
		for _, enumValueCreate := range paramCreate.EnumValues {
			// Set the parameter ID for this enum value
			enumValueCreate.ParameterID = paramID

			// Create the enum value
			enumID, err := s.indicatorRepo.CreateIndicatorParameterEnumValue(ctx, &enumValueCreate)
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

// UpdateIndicator updates an indicator
func (s *IndicatorService) UpdateIndicator(ctx context.Context, id int, update *model.TechnicalIndicator) (*model.TechnicalIndicator, error) {
	// Check if indicator exists
	indicator, err := s.indicatorRepo.GetIndicatorByID(ctx, id, true) // Admin view for checking existence
	if err != nil {
		return nil, err
	}

	if indicator == nil {
		return nil, errors.New("indicator not found")
	}

	// Update indicator
	err = s.indicatorRepo.UpdateIndicator(ctx, id, update)
	if err != nil {
		return nil, err
	}

	// Get the updated indicator (admin view for complete data)
	updatedIndicator, err := s.indicatorRepo.GetIndicatorByID(ctx, id, true)
	if err != nil {
		return nil, err
	}

	return updatedIndicator, nil
}

// DeleteIndicator deletes an indicator by ID
func (s *IndicatorService) DeleteIndicator(ctx context.Context, id int) error {
	// Check if indicator exists
	indicator, err := s.indicatorRepo.GetIndicatorByID(ctx, id, true) // Admin view for checking existence
	if err != nil {
		return err
	}

	if indicator == nil {
		return errors.New("indicator not found")
	}

	return s.indicatorRepo.DeleteIndicator(ctx, id)
}

// GetIndicatorCategories retrieves indicator categories
func (s *IndicatorService) GetIndicatorCategories(ctx context.Context) ([]CategoryInfo, error) {
	repoCategories, err := s.indicatorRepo.GetIndicatorCategories(ctx)
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

// CategoryInfo represents category data returned by the service
type CategoryInfo struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// AddIndicatorParameter adds a parameter to an indicator
func (s *IndicatorService) AddIndicatorParameter(
	ctx context.Context,
	indicatorID int,
	paramName string,
	paramType string,
	isRequired bool,
	minValue *float64,
	maxValue *float64,
	defaultValue string,
	description string,
	isPublic bool,
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
	indicator, err := s.indicatorRepo.GetIndicatorByID(ctx, indicatorID, true) // Admin view for checking existence
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
		IsPublic:      isPublic,
	}

	// Add parameter to database
	paramID, err := s.indicatorRepo.CreateIndicatorParameter(ctx, paramCreate)
	if err != nil {
		s.logger.Error("Failed to create parameter",
			zap.Error(err),
			zap.Int("indicator_id", indicatorID),
			zap.String("parameter_name", paramName))
		return nil, fmt.Errorf("failed to create parameter: %w", err)
	}

	// Create the parameter response object
	newParam := &model.IndicatorParameter{
		ID:            paramID,
		IndicatorID:   indicatorID,
		ParameterName: paramName,
		ParameterType: paramType,
		IsRequired:    isRequired,
		MinValue:      minValue,
		MaxValue:      maxValue,
		DefaultValue:  defaultValue,
		Description:   description,
		IsPublic:      isPublic,
		EnumValues:    []model.ParameterEnumValue{},
	}

	s.logger.Info("Successfully created parameter",
		zap.Int("id", paramID),
		zap.String("name", paramName),
		zap.Int("indicator_id", indicatorID),
		zap.Bool("is_public", isPublic))

	return newParam, nil
}

// UpdateIndicatorParameter updates a parameter
func (s *IndicatorService) UpdateIndicatorParameter(ctx context.Context, id int, param *model.IndicatorParameter) (*model.IndicatorParameter, error) {
	if param == nil {
		return nil, errors.New("parameter data cannot be nil")
	}

	// Verify parameter exists before updating
	existingParam, err := s.getIndicatorParameterByID(ctx, id)
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
	err = s.indicatorRepo.UpdateIndicatorParameter(ctx, id, param)
	if err != nil {
		s.logger.Error("Failed to update parameter in repository",
			zap.Error(err),
			zap.Int("id", id),
			zap.String("parameter_name", param.ParameterName))
		return nil, fmt.Errorf("failed to update parameter: %w", err)
	}

	// Get the updated parameter to return (including enum values)
	updatedParam, err := s.getIndicatorParameterByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to retrieve updated parameter", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("parameter was updated but could not retrieve updated data: %w", err)
	}

	if updatedParam == nil {
		s.logger.Error("Updated parameter not found after update", zap.Int("id", id))
		return nil, errors.New("parameter was updated but could not be found")
	}

	s.logger.Info("Successfully updated parameter",
		zap.Int("id", id),
		zap.String("name", updatedParam.ParameterName),
		zap.Int("indicator_id", updatedParam.IndicatorID),
		zap.Bool("is_public", updatedParam.IsPublic))

	return updatedParam, nil
}

// DeleteIndicatorParameter deletes a parameter
func (s *IndicatorService) DeleteIndicatorParameter(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("invalid parameter ID")
	}

	// First check if parameter exists
	existingParam, err := s.getIndicatorParameterByID(ctx, id)
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
	err = s.indicatorRepo.DeleteIndicatorParameter(ctx, id)
	if err != nil {
		s.logger.Error("Failed to delete parameter", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("failed to delete parameter: %w", err)
	}

	return nil
}

// AddIndicatorParameterEnumValue adds an enum value to a parameter
func (s *IndicatorService) AddIndicatorParameterEnumValue(
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
	enumID, err := s.indicatorRepo.CreateIndicatorParameterEnumValue(ctx, enumCreate)
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

// UpdateIndicatorParameterEnumValue updates an enum value
func (s *IndicatorService) UpdateIndicatorParameterEnumValue(ctx context.Context, id int, enumVal *model.ParameterEnumValue) (*model.ParameterEnumValue, error) {
	// Update enum value
	err := s.indicatorRepo.UpdateIndicatorParameterEnumValue(ctx, id, enumVal)
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

// DeleteIndicatorParameterEnumValue deletes an enum value
func (s *IndicatorService) DeleteIndicatorParameterEnumValue(ctx context.Context, id int) error {
	return s.indicatorRepo.DeleteIndicatorParameterEnumValue(ctx, id)
}

// getIndicatorParameterByID retrieves a parameter by ID
func (s *IndicatorService) getIndicatorParameterByID(ctx context.Context, id int) (*model.IndicatorParameter, error) {
	if id <= 0 {
		return nil, errors.New("invalid parameter ID")
	}

	return s.indicatorRepo.GetIndicatorParameterByID(ctx, id)
}

// SyncIndicatorsFromBacktestingService syncs indicators from the backtesting service
func (s *IndicatorService) SyncIndicatorsFromBacktestingService(ctx context.Context) (int, error) {
	// Get backtesting service URL from environment or use default
	backtestingServiceURL := os.Getenv("BACKTEST_SERVICE_URL")
	if backtestingServiceURL == "" {
		// Default base URL if environment variable not set
		backtestingServiceURL = "http://backtesting-service:5000"
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
