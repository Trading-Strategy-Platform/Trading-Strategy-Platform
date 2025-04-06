package handler

import (
	"net/http"

	"services/user-service/internal/model"
	"services/user-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PreferenceHandler handles user preference requests
type PreferenceHandler struct {
	preferenceService *service.PreferenceService
	logger            *zap.Logger
}

// NewPreferenceHandler creates a new preference handler
func NewPreferenceHandler(preferenceService *service.PreferenceService, logger *zap.Logger) *PreferenceHandler {
	return &PreferenceHandler{
		preferenceService: preferenceService,
		logger:            logger,
	}
}

// GetUserPreferences handles fetching user preferences
// GET /api/v1/users/me/preferences
func (h *PreferenceHandler) GetUserPreferences(c *gin.Context) {
	userID, _ := c.Get("userID")

	preferences, err := h.preferenceService.GetPreferences(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("failed to get user preferences", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get preferences"})
		return
	}

	c.JSON(http.StatusOK, preferences)
}

// UpdateUserPreferences handles updating user preferences
// PUT /api/v1/users/me/preferences
func (h *PreferenceHandler) UpdateUserPreferences(c *gin.Context) {
	var request model.PreferencesUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")
	err := h.preferenceService.UpdatePreferences(c.Request.Context(), userID.(int), &request)
	if err != nil {
		h.logger.Error("failed to update preferences", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
		return
	}

	// Return updated preferences
	preferences, err := h.preferenceService.GetPreferences(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("failed to get updated preferences", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"message": "Preferences updated successfully"})
		return
	}

	c.JSON(http.StatusOK, preferences)
}

// ResetUserPreferences handles resetting user preferences to defaults
// POST /api/v1/users/me/preferences/reset
func (h *PreferenceHandler) ResetUserPreferences(c *gin.Context) {
	userID, _ := c.Get("userID")

	err := h.preferenceService.ResetPreferences(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("failed to reset preferences", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset preferences"})
		return
	}

	// Return the new default preferences
	preferences, err := h.preferenceService.GetPreferences(c.Request.Context(), userID.(int))
	if err != nil {
		h.logger.Error("failed to get reset preferences", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"message": "Preferences reset successfully"})
		return
	}

	c.JSON(http.StatusOK, preferences)
}
