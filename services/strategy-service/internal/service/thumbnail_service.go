package service

import (
	"context"
	"errors"
	"fmt"

	"services/strategy-service/internal/model"

	"go.uber.org/zap"
)

// UpdateThumbnail updates a strategy's thumbnail URL
func (s *StrategyService) UpdateThumbnail(ctx context.Context, id int, userID int, thumbnailURL string) error {
	// Check if strategy exists and belongs to the user
	strategy, err := s.strategyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if strategy == nil {
		return errors.New("strategy not found")
	}

	if strategy.UserID != userID {
		return errors.New("access denied")
	}

	// Create an update object with just the thumbnail URL
	thumbnailUpdate := &model.StrategyUpdate{
		ThumbnailURL: &thumbnailURL,
	}

	// Update the strategy
	err = s.strategyRepo.Update(ctx, id, thumbnailUpdate, userID)
	if err != nil {
		s.logger.Error("Failed to update strategy thumbnail",
			zap.Error(err),
			zap.Int("id", id),
			zap.Int("userID", userID),
			zap.String("thumbnailURL", thumbnailURL))
		return fmt.Errorf("failed to update strategy thumbnail: %w", err)
	}

	s.logger.Info("Strategy thumbnail updated successfully",
		zap.Int("id", id),
		zap.Int("userID", userID),
		zap.String("thumbnailURL", thumbnailURL))

	return nil
}
