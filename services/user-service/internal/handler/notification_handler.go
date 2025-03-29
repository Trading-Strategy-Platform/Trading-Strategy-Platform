package handler

import (
	"net/http"
	"strconv"

	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NotificationHandler handles notification-related HTTP requests
type NotificationHandler struct {
	userService *service.UserService
	logger      *zap.Logger
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(userService *service.UserService, logger *zap.Logger) *NotificationHandler {
	return &NotificationHandler{
		userService: userService,
		logger:      logger,
	}
}

// GetNotifications handles retrieving user notifications
// GET /api/v1/users/me/notifications
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, _ := c.Get("userID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	unreadOnly := c.Query("unread_only") == "true"

	var notifications interface{}
	var err error

	if unreadOnly {
		// If unread_only is true, get only active (unread) notifications
		notifications, err = h.userService.GetActiveNotifications(c.Request.Context(), userID.(int))
	} else {
		notifications, err = h.userService.GetNotifications(c.Request.Context(), userID.(int), limit, offset)
	}

	if err != nil {
		h.logger.Error("Failed to get notifications", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notifications"})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// GetUnreadCount handles retrieving unread notification count
// GET /api/v1/users/me/notifications/count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, _ := c.Get("userID")

	count, err := h.userService.GetUnreadNotificationCount(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("Failed to get unread notification count", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notification count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// MarkNotificationAsRead handles marking a notification as read
// PUT /api/v1/users/me/notifications/{id}/read
func (h *NotificationHandler) MarkNotificationAsRead(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	success, err := h.userService.MarkNotificationAsRead(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to mark notification as read", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification"})
		return
	}

	if !success {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// MarkAllAsRead handles marking all notifications as read
// PUT /api/v1/users/me/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, _ := c.Get("userID")

	count, err := h.userService.MarkAllNotificationsAsRead(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("Failed to mark all notifications as read", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"marked_count": count})
}
