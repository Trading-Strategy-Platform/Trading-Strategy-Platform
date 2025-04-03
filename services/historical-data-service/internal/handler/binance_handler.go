package handler

import (
	"net/http"
	"strconv"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BinanceHandler handles Binance API related HTTP requests
type BinanceHandler struct {
	marketDataService *service.MarketDataService
	logger            *zap.Logger
}

// NewBinanceHandler creates a new Binance handler
func NewBinanceHandler(marketDataService *service.MarketDataService, logger *zap.Logger) *BinanceHandler {
	return &BinanceHandler{
		marketDataService: marketDataService,
		logger:            logger,
	}
}

// GetAvailableSymbols handles retrieving all available symbols from Binance
// GET /api/v1/binance/symbols
func (h *BinanceHandler) GetAvailableSymbols(c *gin.Context) {
	symbols, err := h.marketDataService.GetBinanceSymbols(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get available symbols from Binance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get symbols from Binance"})
		return
	}

	c.JSON(http.StatusOK, symbols)
}

// CheckSymbolStatus handles checking if a symbol exists in the database and what date ranges are available
// GET /api/v1/binance/symbols/:symbol/status
func (h *BinanceHandler) CheckSymbolStatus(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.DefaultQuery("timeframe", "1h")

	status, err := h.marketDataService.CheckSymbolDataStatus(c.Request.Context(), symbol, timeframe)
	if err != nil {
		h.logger.Error("Failed to check symbol status",
			zap.Error(err),
			zap.String("symbol", symbol),
			zap.String("timeframe", timeframe))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check symbol status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// InitiateDataDownload handles initiating a data download from Binance
// POST /api/v1/binance/download
func (h *BinanceHandler) InitiateDataDownload(c *gin.Context) {
	var request model.BinanceDownloadRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the request
	if request.EndDate.Before(request.StartDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date must be after start date"})
		return
	}

	jobID, err := h.marketDataService.StartBinanceDataDownload(c.Request.Context(), &request)
	if err != nil {
		h.logger.Error("Failed to start Binance data download",
			zap.Error(err),
			zap.String("symbol", request.Symbol),
			zap.String("timeframe", request.Timeframe))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start data download"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data download initiated",
		"job_id":  jobID,
	})
}

// GetDownloadStatus handles checking the status of a data download job
// GET /api/v1/binance/download/:id/status
func (h *BinanceHandler) GetDownloadStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	status, err := h.marketDataService.GetBinanceDownloadStatus(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get download status", zap.Error(err), zap.Int("jobID", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get download status"})
		return
	}

	if status == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Download job not found"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetActiveDownloads handles retrieving all active download jobs
// GET /api/v1/binance/downloads/active
func (h *BinanceHandler) GetActiveDownloads(c *gin.Context) {
	jobs, err := h.marketDataService.GetActiveBinanceDownloads(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get active downloads", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active downloads"})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

// CancelDownload handles cancelling a download job
// DELETE /api/v1/binance/download/:id
func (h *BinanceHandler) CancelDownload(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	success, err := h.marketDataService.CancelBinanceDownload(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to cancel download", zap.Error(err), zap.Int("jobID", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel download"})
		return
	}

	if !success {
		c.JSON(http.StatusNotFound, gin.H{"error": "Download job not found or cannot be cancelled"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Download cancelled successfully"})
}
