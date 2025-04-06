package service

import (
	"context"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/repository"

	"go.uber.org/zap"
)

// TimeframeService handles timeframe operations
type TimeframeService struct {
	timeframeRepo *repository.TimeframeRepository
	logger        *zap.Logger
}

// NewTimeframeService creates a new timeframe service
func NewTimeframeService(timeframeRepo *repository.TimeframeRepository, logger *zap.Logger) *TimeframeService {
	return &TimeframeService{
		timeframeRepo: timeframeRepo,
		logger:        logger,
	}
}

// GetAllTimeframes retrieves all available timeframes
func (s *TimeframeService) GetAllTimeframes(ctx context.Context) ([]model.Timeframe, error) {
	return s.timeframeRepo.GetAllTimeframes(ctx)
}

// ValidateTimeframe checks if a timeframe is valid
func (s *TimeframeService) ValidateTimeframe(ctx context.Context, timeframe string) (bool, error) {
	return s.timeframeRepo.ValidateTimeframe(ctx, timeframe)
}
