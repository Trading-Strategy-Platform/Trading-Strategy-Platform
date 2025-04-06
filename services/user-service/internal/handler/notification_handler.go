package handler

import (
	"net/http"
	"strconv"

	"services/user-service/internal/model"
	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NotificationHandler handles notification-related HTTP requests
type NotificationHandler struct {
	notificationService *service.NotificationService
	logger              *zap.Logger
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(notificationService *service.NotificationService, logger *zap.Logger) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		logger:              logger,
	}
}

// GetNotifications handles retrieving user notifications
// GET /api/v1/users/me/notifications
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, _ := c.Get("userID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	unreadOnly := c.Query("unread_only") == "true"

	var notifications []model.Notification
	var err error

	if unreadOnly {
		// If unread_only is true, get only active (unread) notifications
		notifications, err = h.notificationService.GetActiveNotifications(c.Request.Context(), userID.(int))
	} else {
		notifications, err = h.notificationService.GetAllNotifications(
			c.Request.Context(),
			userID.(int),
			limit,
			offset,
		)
	}

	if err != nil {
		h.logger.Error("Failed to get notifications", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notifications"})
		return
	}

	// Get unread count for metadata
	unreadCount, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Warn("Failed to get unread count", zap.Error(err))
		unreadCount = 0
	}

	response := model.NotificationListResponse{
		Notifications: notifications,
		Total:         len(notifications),
		Unread:        unreadCount,
	}

	c.JSON(http.StatusOK, response)
}

// GetUnreadCount handles retrieving unread notification count
// GET /api/v1/users/me/notifications/count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, _ := c.Get("userID")

	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("Failed to get unread notification count", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notification count"})
		return
	}

	response := model.NotificationCountResponse{
		Count: count,
	}

	c.JSON(http.StatusOK, response)
}

// MarkNotificationAsRead handles marking a notification as read
// PUT /api/v1/users/me/notifications/:id/read
func (h *NotificationHandler) MarkNotificationAsRead(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	success, err := h.notificationService.MarkNotificationAsRead(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to mark notification as read", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification"})
		return
	}

	if !success {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MarkAllAsRead handles marking all notifications as read
// PUT /api/v1/users/me/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, _ := c.Get("userID")

	count, err := h.notificationService.MarkAllNotificationsAsRead(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("Failed to mark all notifications as read", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notifications"})
		return
	}

	response := model.NotificationMarkResponse{
		Success:     true,
		MarkedCount: count,
	}

	c.JSON(http.StatusOK, response)
}

// CreateNotification handles creating a new notification (admin or service-to-service)
// POST /api/v1/admin/notifications
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	var request model.NotificationCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.notificationService.AddNotification(c.Request.Context(), &request)
	if err != nil {
		h.logger.Error("Failed to create notification", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "success": true})
}
