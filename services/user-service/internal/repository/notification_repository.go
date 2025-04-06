// internal/repository/notification_repository.go
package repository

import (
	"context"
	"database/sql"
	"errors"

	"services/user-service/internal/model"

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

// GetActiveNotifications retrieves active notifications for a user using get_active_notifications function
func (r *NotificationRepository) GetActiveNotifications(ctx context.Context, userID int) ([]model.Notification, error) {
	query := `SELECT * FROM get_active_notifications($1)`

	var notifications []model.Notification
	err := r.db.SelectContext(ctx, &notifications, query, userID)
	if err != nil {
		r.logger.Error("Failed to get active notifications", zap.Error(err))
		return nil, err
	}

	return notifications, nil
}

// GetUnreadCount retrieves the count of unread notifications using get_unread_notification_count function
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

// GetAllNotifications retrieves all notifications with pagination using get_all_notifications function
func (r *NotificationRepository) GetAllNotifications(ctx context.Context, userID, limit, offset int) ([]model.Notification, error) {
	query := `SELECT * FROM get_all_notifications($1, $2, $3)`

	var notifications []model.Notification
	err := r.db.SelectContext(ctx, &notifications, query, userID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to get all notifications", zap.Error(err))
		return nil, err
	}

	return notifications, nil
}

// MarkNotificationAsRead marks a notification as read using mark_notification_as_read function
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

// MarkAllNotificationsAsRead marks all notifications as read using mark_all_notifications_as_read function
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

// AddNotification adds a new notification using add_notification function
func (r *NotificationRepository) AddNotification(
	ctx context.Context,
	userID int,
	notificationType,
	title,
	message,
	link string,
) (int, error) {
	query := `SELECT add_notification($1, $2::notification_type, $3, $4, $5)`

	var id int
	err := r.db.GetContext(ctx, &id, query, userID, notificationType, title, message, link)
	if err != nil {
		r.logger.Error("Failed to add notification", zap.Error(err))
		return 0, err
	}

	return id, nil
}

// DeleteUserNotifications deletes all notifications for a user using delete_user_notifications function
func (r *NotificationRepository) DeleteUserNotifications(ctx context.Context, userID int) (int, error) {
	query := `SELECT delete_user_notifications($1)`

	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		r.logger.Error("Failed to delete user notifications", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// GetNotificationByID gets a notification by ID using get_notification_by_id function
func (r *NotificationRepository) GetNotificationByID(ctx context.Context, notificationID int) (*model.Notification, error) {
	query := `SELECT * FROM get_notification_by_id($1)`

	var notification model.Notification
	err := r.db.GetContext(ctx, &notification, query, notificationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("Failed to get notification by ID", zap.Error(err))
		return nil, err
	}

	return &notification, nil
}
