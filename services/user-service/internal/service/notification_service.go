package service

import (
	"context"

	"services/user-service/internal/repository"

	"go.uber.org/zap"
)

// NotificationService handles notification operations
type NotificationService struct {
	notificationRepo *repository.NotificationRepository
	logger           *zap.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(notificationRepo *repository.NotificationRepository, logger *zap.Logger) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		logger:           logger,
	}
}

// GetActiveNotifications retrieves active notifications for a user
func (s *NotificationService) GetActiveNotifications(ctx context.Context, userID int) (interface{}, error) {
	return s.notificationRepo.GetActiveNotifications(ctx, userID)
}

// GetUnreadCount retrieves the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID int) (int, error) {
	return s.notificationRepo.GetUnreadCount(ctx, userID)
}

// GetAllNotifications retrieves all notifications for a user with pagination
func (s *NotificationService) GetAllNotifications(ctx context.Context, userID, limit, offset int) (interface{}, error) {
	return s.notificationRepo.GetAllNotifications(ctx, userID, limit, offset)
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID int) (bool, error) {
	return s.notificationRepo.MarkNotificationAsRead(ctx, notificationID)
}

// MarkAllAsRead marks all notifications for a user as read
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID int) (int, error) {
	return s.notificationRepo.MarkAllNotificationsAsRead(ctx, userID)
}

// AddNotification adds a new notification for a user
func (s *NotificationService) AddNotification(ctx context.Context, userID int, notificationType string, title, message string, link string) (int, error) {
	return s.notificationRepo.AddNotification(ctx, userID, notificationType, title, message, link)
}
