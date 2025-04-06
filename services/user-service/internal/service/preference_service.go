package service

import (
	"context"
	"encoding/json"
	"errors"

	"services/user-service/internal/model"
	"services/user-service/internal/repository"

	"go.uber.org/zap"
)

// PreferenceService handles user preference operations
type PreferenceService struct {
	preferenceRepo *repository.PreferenceRepository
	userRepo       *repository.UserRepository
	logger         *zap.Logger
}

// NewPreferenceService creates a new preference service
func NewPreferenceService(
	preferenceRepo *repository.PreferenceRepository,
	userRepo *repository.UserRepository,
	logger *zap.Logger,
) *PreferenceService {
	return &PreferenceService{
		preferenceRepo: preferenceRepo,
		userRepo:       userRepo,
		logger:         logger,
	}
}

// GetPreferences gets a user's preferences
func (s *PreferenceService) GetPreferences(ctx context.Context, userID int) (*model.UserPreferences, error) {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("user not found or inactive")
	}

	return s.preferenceRepo.GetPreferences(ctx, userID)
}

// UpdatePreferences updates a user's preferences
func (s *PreferenceService) UpdatePreferences(ctx context.Context, userID int, prefs *model.PreferencesUpdate) error {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, userID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("user not found or inactive")
	}

	// Validate JSON objects if provided
	if prefs.ChartPreferences != nil {
		var chartPrefs model.ChartPreference
		if err := json.Unmarshal(prefs.ChartPreferences, &chartPrefs); err != nil {
			return errors.New("invalid chart preferences format")
		}
	}

	if prefs.NotificationSettings != nil {
		var notifSettings model.NotificationSetting
		if err := json.Unmarshal(prefs.NotificationSettings, &notifSettings); err != nil {
			return errors.New("invalid notification settings format")
		}
	}

	// Update preferences
	success, err := s.preferenceRepo.UpdatePreferences(
		ctx,
		userID,
		prefs.Theme,
		prefs.DefaultTimeframe,
		prefs.ChartPreferences,
		prefs.NotificationSettings,
	)

	if err != nil {
		return err
	}

	if !success {
		return errors.New("failed to update preferences")
	}

	return nil
}

// CheckPreferencesExist checks if a user's preferences exist
func (s *PreferenceService) CheckPreferencesExist(ctx context.Context, userID int) (bool, error) {
	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, errors.New("user not found")
	}

	return s.preferenceRepo.CheckPreferencesExist(ctx, userID)
}

// DeletePreferences deletes a user's preferences
func (s *PreferenceService) DeletePreferences(ctx context.Context, userID int) error {
	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	success, err := s.preferenceRepo.DeletePreferences(ctx, userID)
	if err != nil {
		return err
	}

	if !success {
		return errors.New("preferences not found or already deleted")
	}

	return nil
}

// ResetPreferences resets a user's preferences to default values
func (s *PreferenceService) ResetPreferences(ctx context.Context, userID int) error {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, userID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("user not found or inactive")
	}

	// Delete existing preferences
	_, err = s.preferenceRepo.DeletePreferences(ctx, userID)
	if err != nil {
		return err
	}

	// Create new preferences with default values
	defaultTheme := "light"
	defaultTimeframe := "1h"
	defaultChartPrefs := json.RawMessage(`{}`)
	defaultNotifSettings := json.RawMessage(`{}`)

	_, err = s.preferenceRepo.UpdatePreferences(
		ctx,
		userID,
		&defaultTheme,
		&defaultTimeframe,
		defaultChartPrefs,
		defaultNotifSettings,
	)

	if err != nil {
		return err
	}

	return nil
}

// checkUserActive checks if a user exists and is active
func (s *PreferenceService) checkUserActive(ctx context.Context, userID int) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil || !user.IsActive {
		return false, nil
	}
	return true, nil
}
