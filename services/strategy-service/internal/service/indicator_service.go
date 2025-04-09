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
func NewIndicatorService(indicatorRepo *repository.IndicatorRepository, logger *zap.Logger) *IndicatorService {
	return &IndicatorService{
		indicatorRepo: indicatorRepo,
		logger:        logger,
	}
}

// GetDB provides direct access to the database for debugging purposes
func (s *IndicatorService) GetDB() *sqlx.DB {
	return s.db
}

// GetAllIndicators retrieves all technical indicators using get_indicators function
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

// CreateIndicator creates a new technical indicator
func (s *IndicatorService) CreateIndicator(ctx context.Context, name, description, category, formula string, parameters []model.IndicatorParameterCreate) (*model.TechnicalIndicator, error) {
	// Validate input
	if name == "" {
		return nil, errors.New("indicator name is required")
	}
	if category == "" {
		return nil, errors.New("indicator category is required")
	}

	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
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

	// Call the repository to insert the indicator
	indicatorID, err := s.indicatorRepo.Create(ctx, indicator)
	if err != nil {
		s.logger.Error("Failed to create indicator", zap.Error(err))
		return nil, err
	}

	// Set the ID in the returned model
	indicator.ID = indicatorID

	// Add parameters if provided
	if parameters != nil && len(parameters) > 0 {
		for i := range parameters {
			// Set the indicator ID for each parameter
			parameters[i].IndicatorID = indicatorID

			// Create the parameter
			paramID, err := s.indicatorRepo.AddParameter(ctx, &parameters[i])
			if err != nil {
				s.logger.Error("Failed to add parameter", zap.Error(err))
				return nil, err
			}

			// Add the parameter to the indicator model
			param := model.IndicatorParameter{
				ID:            paramID,
				IndicatorID:   indicatorID,
				ParameterName: parameters[i].ParameterName,
				ParameterType: parameters[i].ParameterType,
				IsRequired:    parameters[i].IsRequired,
				MinValue:      parameters[i].MinValue,
				MaxValue:      parameters[i].MaxValue,
				DefaultValue:  parameters[i].DefaultValue,
				Description:   parameters[i].Description,
			}

			indicator.Parameters = append(indicator.Parameters, param)

			// Add enum values if provided
			if parameters[i].EnumValues != nil && len(parameters[i].EnumValues) > 0 {
				for _, enumValue := range parameters[i].EnumValues {
					// Set the parameter ID for the enum value
					enumCreate := model.ParameterEnumValueCreate{
						ParameterID: paramID,
						EnumValue:   enumValue.EnumValue,
						DisplayName: enumValue.DisplayName,
					}

					// Create the enum value
					enumID, err := s.indicatorRepo.AddEnumValue(ctx, &enumCreate)
					if err != nil {
						s.logger.Error("Failed to add enum value", zap.Error(err))
						return nil, err
					}

					// Add the enum value to the parameter
					param.EnumValues = append(param.EnumValues, model.ParameterEnumValue{
						ID:          enumID,
						ParameterID: paramID,
						EnumValue:   enumValue.EnumValue,
						DisplayName: enumValue.DisplayName,
					})
				}
			}
		}
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
	paramID, err := s.indicatorRepo.AddParameter(ctx, parameterCreate)
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
	enumID, err := s.indicatorRepo.AddEnumValue(ctx, enumCreate)
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

// GetIndicator retrieves a specific indicator by ID using get_indicator_by_id function
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

// GetCategories retrieves indicator categories using get_indicator_categories function
func (s *IndicatorService) GetCategories(ctx context.Context) ([]struct {
	Category string `db:"category"`
	Count    int    `db:"count"`
}, error) {
	return s.indicatorRepo.GetCategories(ctx)
}
