package service

import (
	"context"
	"errors"
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
func (s *IndicatorService) GetAllIndicators(ctx context.Context, searchTerm string, categories []string, page, limit int) ([]model.TechnicalIndicator, int, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	return s.indicatorRepo.GetAll(ctx, searchTerm, categories, page, limit)
}

// CreateIndicator creates a new technical indicator with parameters and enum values
func (s *IndicatorService) CreateIndicator(ctx context.Context, name, description, category, formula string, parameters []model.IndicatorParameterCreate) (*model.TechnicalIndicator, error) {
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

	// Create the indicator
	indicator := &model.TechnicalIndicator{
		Name:        name,
		Description: description,
		Category:    category,
		Formula:     formula,
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
		zap.Int("parameters_count", len(parameters)))

	return indicator, nil
}

// AddParameter adds a parameter to an indicator
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
	// Validate indicator exists
	indicator, err := s.indicatorRepo.GetByID(ctx, indicatorID)
	if err != nil {
		return nil, err
	}
	if indicator == nil {
		return nil, errors.New("indicator not found")
	}

	// Create parameter object
	parameterCreate := &model.IndicatorParameterCreate{
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
	paramID, err := s.indicatorRepo.CreateParameter(ctx, parameterCreate)
	if err != nil {
		return nil, err
	}

	// Return the created parameter
	return &model.IndicatorParameter{
		ID:            paramID,
		IndicatorID:   indicatorID,
		ParameterName: paramName,
		ParameterType: paramType,
		IsRequired:    isRequired,
		MinValue:      minValue,
		MaxValue:      maxValue,
		DefaultValue:  defaultValue,
		Description:   description,
		EnumValues:    []model.ParameterEnumValue{},
	}, nil
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

// DeleteParameter deletes a parameter by ID
func (s *IndicatorService) DeleteParameter(ctx context.Context, id int) error {
	return s.indicatorRepo.DeleteParameter(ctx, id)
}

// UpdateParameter updates a parameter
func (s *IndicatorService) UpdateParameter(ctx context.Context, id int, param *model.IndicatorParameter) (*model.IndicatorParameter, error) {
	// Update parameter
	err := s.indicatorRepo.UpdateParameter(ctx, id, param)
	if err != nil {
		return nil, err
	}

	// Get the indicator that this parameter belongs to
	indicator, err := s.indicatorRepo.GetByID(ctx, param.IndicatorID)
	if err != nil {
		return nil, err
	}

	// Find the updated parameter in the indicator's parameters
	for _, p := range indicator.Parameters {
		if p.ID == id {
			return &p, nil
		}
	}

	return nil, errors.New("parameter not found after update")
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
