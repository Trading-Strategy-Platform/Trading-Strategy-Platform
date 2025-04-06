package handler

import (
	"net/http"

	"services/historical-data-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TimeframeHandler handles timeframe HTTP requests
type TimeframeHandler struct {
	timeframeService *service.TimeframeService
	logger           *zap.Logger
}

// NewTimeframeHandler creates a new timeframe handler
func NewTimeframeHandler(timeframeService *service.TimeframeService, logger *zap.Logger) *TimeframeHandler {
	return &TimeframeHandler{
		timeframeService: timeframeService,
		logger:           logger,
	}
}

// GetAllTimeframes handles retrieving all timeframes
// GET /api/v1/timeframes
func (h *TimeframeHandler) GetAllTimeframes(c *gin.Context) {
	timeframes, err := h.timeframeService.GetAllTimeframes(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get all timeframes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve timeframes"})
		return
	}

	c.JSON(http.StatusOK, timeframes)
}

// ValidateTimeframe handles validating a timeframe
// GET /api/v1/timeframes/validate/:timeframe
func (h *TimeframeHandler) ValidateTimeframe(c *gin.Context) {
	timeframe := c.Param("timeframe")

	valid, err := h.timeframeService.ValidateTimeframe(c.Request.Context(), timeframe)
	if err != nil {
		h.logger.Error("Failed to validate timeframe", zap.Error(err), zap.String("timeframe", timeframe))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate timeframe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"timeframe": timeframe,
		"valid":     valid,
	})
}
