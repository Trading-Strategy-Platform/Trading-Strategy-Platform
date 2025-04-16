package service

import (
	"context"
	"encoding/json"

	"services/historical-data-service/internal/model"

	"go.uber.org/zap"
)

// GetIndicators retrieves all available technical indicators for strategies
func (s *BacktestService) GetIndicators(ctx context.Context) ([]model.Indicator, error) {
	// Get indicators from the backtest client
	indicators, err := s.backtestClient.GetAvailableIndicators(ctx)
	if err != nil {
		s.logger.Error("Failed to retrieve indicators from backtesting service", zap.Error(err))
		return nil, err
	}

	return indicators, nil
}

// ValidateStrategy validates a strategy structure
func (s *BacktestService) ValidateStrategy(ctx context.Context, strategyRaw interface{}) (bool, string, error) {
	// Convert the strategy to JSON
	strategyJSON, err := json.Marshal(strategyRaw)
	if err != nil {
		s.logger.Error("Failed to marshal strategy", zap.Error(err))
		return false, "Failed to format strategy data", err
	}

	// Create RawMessage from JSON
	strategyStructure := json.RawMessage(strategyJSON)

	// Validate using the backtest client
	return s.backtestClient.ValidateStrategy(ctx, strategyStructure)
}
