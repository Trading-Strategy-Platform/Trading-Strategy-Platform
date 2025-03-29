package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// NotificationRepository handles database operations for notifications
type NotificationRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *sqlx.DB, logger *zap.Logger) *NotificationRepository {
	return &NotificationRepository{
		db:     db,
		logger: logger,
	}
}

// GetActiveNotifications retrieves active notifications for a user
func (r *NotificationRepository) GetActiveNotifications(ctx context.Context, userID int) (interface{}, error) {
	query := `SELECT * FROM get_active_notifications($1)`

	var notifications []map[string]interface{}
	err := r.db.SelectContext(ctx, &notifications, query, userID)
	if err != nil {
		r.logger.Error("Failed to get active notifications", zap.Error(err))
		return nil, err
	}

	return notifications, nil
}

// GetUnreadCount retrieves the count of unread notifications for a user
func (r *NotificationRepository) GetUnreadCount(ctx context.Context, userID int) (int, error) {
	query := `SELECT get_unread_notification_count($1)`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		r.logger.Error("Failed to get unread notification count", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// GetAllNotifications retrieves all notifications for a user with pagination
func (r *NotificationRepository) GetAllNotifications(ctx context.Context, userID, limit, offset int) (interface{}, error) {
	query := `SELECT * FROM get_all_notifications($1, $2, $3)`

	var notifications []map[string]interface{}
	err := r.db.SelectContext(ctx, &notifications, query, userID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get all notifications", zap.Error(err))
		return nil, err
	}

	return notifications, nil
}

// MarkNotificationAsRead marks a notification as read
func (r *NotificationRepository) MarkNotificationAsRead(ctx context.Context, notificationID int) (bool, error) {
	query := `SELECT mark_notification_as_read($1)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, notificationID)
	if err != nil {
		r.logger.Error("Failed to mark notification as read", zap.Error(err))
		return false, err
	}

	return success, nil
}

// MarkAllNotificationsAsRead marks all notifications for a user as read
func (r *NotificationRepository) MarkAllNotificationsAsRead(ctx context.Context, userID int) (int, error) {
	query := `SELECT mark_all_notifications_as_read($1)`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		r.logger.Error("Failed to mark all notifications as read", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// AddNotification adds a new notification for a user
func (r *NotificationRepository) AddNotification(ctx context.Context, userID int, notificationType, title, message, link string) (int, error) {
	query := `SELECT add_notification($1, $2, $3, $4, $5)`

	var id int
	err := r.db.GetContext(ctx, &id, query, userID, notificationType, title, message, link)
	if err != nil {
		r.logger.Error("Failed to add notification", zap.Error(err))
		return 0, err
	}

	return id, nil
}
