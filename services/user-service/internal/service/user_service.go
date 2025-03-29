// internal/service/user_service.go
package service

import (
	"context"
	"errors"

	"services/user-service/internal/model"
	"services/user-service/internal/repository"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles user operations
type UserService struct {
	userRepo         *repository.UserRepository
	notificationRepo *repository.NotificationRepository
	preferencesRepo  *repository.PreferencesRepository
	logger           *zap.Logger
}

// NewUserService creates a new user service
func NewUserService(
	userRepo *repository.UserRepository,
	notificationRepo *repository.NotificationRepository,
	preferencesRepo *repository.PreferencesRepository,
	logger *zap.Logger,
) *UserService {
	return &UserService{
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		preferencesRepo:  preferencesRepo,
		logger:           logger,
	}
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(ctx context.Context, id int) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// GetCurrentUser gets the current user by ID from context
func (s *UserService) GetCurrentUser(ctx context.Context, userID int) (*model.UserDetails, error) {
	return s.userRepo.GetUserDetails(ctx, userID)
}

// Update updates a user's details
func (s *UserService) Update(ctx context.Context, id int, update *model.UserUpdate) error {
	// Use the update_user function from the database
	success, err := s.userRepo.UpdateUser(
		ctx,
		id,
		update.Username,
		update.Email,
		update.ProfilePhotoURL,
		nil, // theme
		nil, // default_timeframe
		nil, // chart_preferences
		nil, // notification_settings
	)

	if err != nil {
		return err
	}

	if !success {
		return errors.New("failed to update user")
	}

	return nil
}

// UpdatePreferences updates a user's preferences
func (s *UserService) UpdatePreferences(ctx context.Context, id int, prefs *model.PreferencesUpdate) error {
	// Use the update_user function from the database to update preferences
	success, err := s.userRepo.UpdateUser(
		ctx,
		id,
		nil, // username
		nil, // email
		nil, // profile_photo_url
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

// GetPreferences gets a user's preferences
func (s *UserService) GetPreferences(ctx context.Context, id int) (*model.UserPreferences, error) {
	// Get user details which include preferences
	details, err := s.userRepo.GetUserDetails(ctx, id)
	if err != nil {
		return nil, err
	}

	if details == nil {
		return nil, errors.New("user not found")
	}

	// Extract and return only the preferences part
	prefs := &model.UserPreferences{
		Theme:                details.Theme,
		DefaultTimeframe:     details.DefaultTimeframe,
		ChartPreferences:     details.ChartPreferences,
		NotificationSettings: details.NotificationSettings,
	}

	return prefs, nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(ctx context.Context, id int, change *model.UserChangePassword) error {
	// Get user to verify current password
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(change.CurrentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(change.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return errors.New("failed to process new password")
	}

	// Update password in database using update_user_password function
	success, err := s.userRepo.UpdateUserPassword(ctx, id, string(newHash))
	if err != nil {
		return err
	}

	if !success {
		return errors.New("failed to update password")
	}

	return nil
}

// DeleteUser marks a user as inactive
func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	// Use the delete_user function from the database
	success, err := s.userRepo.DeleteUser(ctx, id)
	if err != nil {
		return err
	}

	if !success {
		return errors.New("failed to delete user")
	}

	return nil
}

// ListUsers gets a paginated list of users
func (s *UserService) ListUsers(ctx context.Context, page, limit int) ([]model.User, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit

	users, err := s.userRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}

// GetNotifications gets a user's notifications
func (s *UserService) GetNotifications(ctx context.Context, userID int, limit, offset int) (interface{}, error) {
	return s.notificationRepo.GetAllNotifications(ctx, userID, limit, offset)
}

// GetUnreadNotificationCount gets a user's unread notification count
func (s *UserService) GetUnreadNotificationCount(ctx context.Context, userID int) (int, error) {
	return s.notificationRepo.GetUnreadCount(ctx, userID)
}

// MarkNotificationAsRead marks a notification as read
func (s *UserService) MarkNotificationAsRead(ctx context.Context, notificationID int) (bool, error) {
	return s.notificationRepo.MarkNotificationAsRead(ctx, notificationID)
}

// MarkAllNotificationsAsRead marks all notifications for a user as read
func (s *UserService) MarkAllNotificationsAsRead(ctx context.Context, userID int) (int, error) {
	return s.notificationRepo.MarkAllNotificationsAsRead(ctx, userID)
}

// GetActiveNotifications gets all unread notifications for a user
func (s *UserService) GetActiveNotifications(ctx context.Context, userID int) (interface{}, error) {
	return s.notificationRepo.GetActiveNotifications(ctx, userID)
}
