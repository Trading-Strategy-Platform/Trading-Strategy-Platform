package service

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

// UpdateProfilePhoto updates a user's profile photo URL
func (s *UserService) UpdateProfilePhoto(ctx context.Context, userID int, photoURL string) error {
	// Update user profile photo URL
	success, err := s.userRepo.UpdateUser(
		ctx,
		userID,
		nil, // username (unchanged)
		nil, // email (unchanged)
		&photoURL,
		nil, // theme (unchanged)
		nil, // default_timeframe (unchanged)
		nil, // chart_preferences (unchanged)
		nil, // notification_settings (unchanged)
	)

	if err != nil {
		s.logger.Error("Failed to update profile photo", zap.Error(err))
		return err
	}

	if !success {
		return errors.New("failed to update profile photo")
	}

	return nil
}
