package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"services/historical-data-service/internal/model"
	"services/historical-data-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DataDownloadHandler handles market data download HTTP requests
type DataDownloadHandler struct {
	downloadService *service.MarketDataDownloadService
	logger          *zap.Logger
}

// NewDataDownloadHandler creates a new data download handler
func NewDataDownloadHandler(downloadService *service.MarketDataDownloadService, logger *zap.Logger) *DataDownloadHandler {
	return &DataDownloadHandler{
		downloadService: downloadService,
		logger:          logger,
	}
}

// GetAvailableSymbols handles retrieving all available symbols from a specific source
// GET /api/v1/market-data/downloads/sources/:source/symbols
func (h *DataDownloadHandler) GetAvailableSymbols(c *gin.Context) {
	source := c.Param("source")
	if source == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source is required"})
		return
	}

	symbols, err := h.downloadService.GetAvailableSymbols(c.Request.Context(), source)
	if err != nil {
		h.logger.Error("Failed to get available symbols",
			zap.Error(err),
			zap.String("source", source))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get symbols from source"})
		return
	}

	c.JSON(http.StatusOK, symbols)
}

// CheckSymbolStatus handles checking if a symbol exists in the database and what date ranges are available
// GET /api/v1/market-data/downloads/symbols/:symbol/status
func (h *DataDownloadHandler) CheckSymbolStatus(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.DefaultQuery("timeframe", "1h")

	status, err := h.downloadService.CheckSymbolStatus(c.Request.Context(), symbol, timeframe)
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

// InitiateDataDownload handles initiating a data download
// POST /api/v1/market-data/downloads
func (h *DataDownloadHandler) InitiateDataDownload(c *gin.Context) {
	var request model.MarketDataDownloadRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the request
	if request.EndDate.Before(request.StartDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date must be after start date"})
		return
	}

	jobID, err := h.downloadService.InitiateDataDownload(c.Request.Context(), &request)
	if err != nil {
		h.logger.Error("Failed to start data download",
			zap.Error(err),
			zap.String("symbol", request.Symbol),
			zap.String("source", request.Source),
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
// GET /api/v1/market-data/downloads/:id/status
func (h *DataDownloadHandler) GetDownloadStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	status, err := h.downloadService.GetDownloadStatus(c.Request.Context(), id)
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
// GET /api/v1/market-data/downloads/active
func (h *DataDownloadHandler) GetActiveDownloads(c *gin.Context) {
	source := c.Query("source")
	jobs, err := h.downloadService.GetActiveDownloads(c.Request.Context(), source)
	if err != nil {
		h.logger.Error("Failed to get active downloads", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active downloads"})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

// CancelDownload handles cancelling a download job
// DELETE /api/v1/market-data/downloads/:id
func (h *DataDownloadHandler) CancelDownload(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	// Get the force flag from the query
	force := c.DefaultQuery("force", "false") == "true"

	// First get the job to check its status
	job, err := h.downloadService.GetDownloadStatus(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get download job", zap.Error(err), zap.Int("jobID", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve download job"})
		return
	}

	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Download job not found"})
		return
	}

	// Check if job is in a state that can be cancelled
	if !force && job.Status != "pending" && job.Status != "in_progress" {
		// Return success with clear information that nothing was changed
		c.JSON(http.StatusOK, gin.H{
			"message":      fmt.Sprintf("Job is already in '%s' state. Use ?force=true to cancel anyway.", job.Status),
			"status":       job.Status,
			"job_id":       job.JobID,
			"cancelled":    false,
			"already_done": true,
		})
		return
	}

	// We're actually going to change the status
	success, err := h.downloadService.CancelDownload(c.Request.Context(), id, force)
	if err != nil {
		h.logger.Error("Failed to cancel download", zap.Error(err), zap.Int("jobID", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel download"})
		return
	}

	if !success {
		c.JSON(http.StatusNotFound, gin.H{"error": "Download job not found or cannot be cancelled"})
		return
	}

	// Refresh the job status after cancellation
	updatedJob, _ := h.downloadService.GetDownloadStatus(c.Request.Context(), id)

	// Prepare current status value
	var currentStatus string
	if updatedJob != nil {
		currentStatus = updatedJob.Status
	} else {
		currentStatus = "cancelled"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Download cancelled successfully",
		"job_id":          id,
		"cancelled":       true,
		"previous_status": job.Status,
		"current_status":  currentStatus,
	})
}

// GetJobsSummary handles retrieving a summary of download jobs
// GET /api/v1/market-data/downloads/summary
func (h *DataDownloadHandler) GetJobsSummary(c *gin.Context) {
	summary, err := h.downloadService.GetJobsSummary(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get jobs summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get jobs summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetDataInventory handles retrieving data inventory information
// GET /api/v1/market-data/inventory
func (h *DataDownloadHandler) GetDataInventory(c *gin.Context) {
	assetType := c.DefaultQuery("asset_type", "")
	exchange := c.DefaultQuery("exchange", "")

	// Add more detailed logging
	h.logger.Info("GetDataInventory request received",
		zap.String("assetType", assetType),
		zap.String("exchange", exchange))

	inventory, err := h.downloadService.GetDataInventory(c.Request.Context(), assetType, exchange)
	if err != nil {
		h.logger.Error("Failed to get data inventory", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get data inventory"})
		return
	}

	// Add debug info about result
	h.logger.Info("Inventory result",
		zap.Any("inventoryLength", len(inventory)),
		zap.Any("inventoryIsNil", inventory == nil))

	c.JSON(http.StatusOK, inventory)
}
