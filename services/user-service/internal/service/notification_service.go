package service

import (
	"context"
	"errors"

	"services/user-service/internal/model"
	"services/user-service/internal/repository"

	"go.uber.org/zap"
)

// NotificationService handles notification operations
type NotificationService struct {
	notificationRepo *repository.NotificationRepository
	userRepo         *repository.UserRepository
	logger           *zap.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	notificationRepo *repository.NotificationRepository,
	userRepo *repository.UserRepository,
	logger *zap.Logger,
) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		logger:           logger,
	}
}

// GetActiveNotifications retrieves active notifications for a user
func (s *NotificationService) GetActiveNotifications(ctx context.Context, userID int) ([]model.Notification, error) {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("user not found or inactive")
	}

	return s.notificationRepo.GetActiveNotifications(ctx, userID)
}

// GetUnreadCount retrieves the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID int) (int, error) {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, userID)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, errors.New("user not found or inactive")
	}

	return s.notificationRepo.GetUnreadCount(ctx, userID)
}

// GetAllNotifications retrieves all notifications for a user with pagination
func (s *NotificationService) GetAllNotifications(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]model.Notification, error) {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("user not found or inactive")
	}

	// Set default values for limit and offset
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return s.notificationRepo.GetAllNotifications(ctx, userID, limit, offset)
}

// MarkNotificationAsRead marks a notification as read
func (s *NotificationService) MarkNotificationAsRead(ctx context.Context, notificationID int) (bool, error) {
	// Get notification details to check ownership
	notification, err := s.notificationRepo.GetNotificationByID(ctx, notificationID)
	if err != nil {
		return false, err
	}
	if notification == nil {
		return false, errors.New("notification not found")
	}

	// Make sure user exists and is active
	exists, err := s.checkUserActive(ctx, notification.UserID)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, errors.New("user not found or inactive")
	}

	return s.notificationRepo.MarkNotificationAsRead(ctx, notificationID)
}

// MarkAllNotificationsAsRead marks all notifications for a user as read
func (s *NotificationService) MarkAllNotificationsAsRead(ctx context.Context, userID int) (int, error) {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, userID)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, errors.New("user not found or inactive")
	}

	return s.notificationRepo.MarkAllNotificationsAsRead(ctx, userID)
}

// AddNotification adds a new notification for a user
func (s *NotificationService) AddNotification(ctx context.Context, notification *model.NotificationCreate) (int, error) {
	// Check if user exists and is active
	exists, err := s.checkUserActive(ctx, notification.UserID)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, errors.New("user not found or inactive")
	}

	return s.notificationRepo.AddNotification(
		ctx,
		notification.UserID,
		notification.Type,
		notification.Title,
		notification.Message,
		notification.Link,
	)
}

// DeleteUserNotifications deletes all notifications for a user
func (s *NotificationService) DeleteUserNotifications(ctx context.Context, userID int) (int, error) {
	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if user == nil {
		return 0, errors.New("user not found")
	}

	return s.notificationRepo.DeleteUserNotifications(ctx, userID)
}

// checkUserActive checks if a user exists and is active
func (s *NotificationService) checkUserActive(ctx context.Context, userID int) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil || !user.IsActive {
		return false, nil
	}
	return true, nil
}
