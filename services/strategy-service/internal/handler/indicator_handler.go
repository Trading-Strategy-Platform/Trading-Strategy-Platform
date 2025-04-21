package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"services/strategy-service/internal/client"
	"services/strategy-service/internal/model"
	"services/strategy-service/internal/service"
	"services/strategy-service/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// IndicatorHandler handles indicator-related HTTP requests
type IndicatorHandler struct {
	indicatorService *service.IndicatorService
	userClient       *client.UserClient
	logger           *zap.Logger
}

// NewIndicatorHandler creates a new indicator handler
func NewIndicatorHandler(indicatorService *service.IndicatorService, userClient *client.UserClient, logger *zap.Logger) *IndicatorHandler {
	return &IndicatorHandler{
		indicatorService: indicatorService,
		userClient:       userClient,
		logger:           logger,
	}
}

// checkIsAdmin checks if the current user has admin role
func (h *IndicatorHandler) checkIsAdmin(c *gin.Context) bool {
	userID, exists := c.Get("userID")
	if !exists {
		return false
	}

	// Get the token from the Authorization header
	authHeader := c.GetHeader("Authorization")
	token := ""
	if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
		token = authHeader[7:]
	}

	// Check if user has admin role
	isAdmin, err := h.userClient.CheckUserRole(c.Request.Context(), userID.(int), "admin", token)
	if err != nil {
		h.logger.Warn("Failed to check user role", zap.Error(err), zap.Int("user_id", userID.(int)))
		return false
	}

	return isAdmin
}

// GetAllIndicators handles retrieving all indicators with filtering options
// GET /api/v1/indicators
func (h *IndicatorHandler) GetAllIndicators(c *gin.Context) {
	// Parse search parameter
	searchTerm := c.Query("search")

	// Parse categories filter (comma-separated)
	var categories []string
	if categoriesStr := c.Query("categories"); categoriesStr != "" {
		categories = strings.Split(categoriesStr, ",")
	}

	// Parse active filter
	var active *bool
	if activeStr := c.Query("active"); activeStr != "" {
		activeBool := activeStr == "true"
		active = &activeBool
	}

	// Parse pagination parameters using the utility function
	params := utils.ParsePaginationParams(c, 20, 100) // default limit: 20, max limit: 100

	// Check if user is admin
	isAdmin := h.checkIsAdmin(c)

	indicators, total, err := h.indicatorService.GetAllIndicators(
		c.Request.Context(),
		searchTerm,
		categories,
		active,
		params.Page,
		params.Limit,
		isAdmin,
	)

	if err != nil {
		h.logger.Error("Failed to get indicators", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to fetch indicators")
		return
	}

	// Use standardized pagination response
	utils.SendPaginatedResponse(c, http.StatusOK, indicators, total, params.Page, params.Limit)
}

// GetCategories handles retrieving indicator categories
// GET /api/v1/indicators/categories
func (h *IndicatorHandler) GetCategories(c *gin.Context) {
	// Log the request with a distinctive message
	h.logger.Info("CATEGORIES ENDPOINT EXPLICITLY CALLED",
		zap.String("path", c.Request.URL.Path),
		zap.String("query", c.Request.URL.RawQuery),
		zap.String("client_ip", c.ClientIP()))

	// Get the timestamp (will be ignored, just for cache busting)
	_ = c.Query("t")

	categories, err := h.indicatorService.GetCategories(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get indicator categories", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to fetch indicator categories")
		return
	}

	// Add cache control headers here too for good measure
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	// Log what we're returning
	h.logger.Info("Returning categories data", zap.Any("categories", categories))

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// GetIndicator handles retrieving a specific indicator
// GET /api/v1/indicators/{id}
func (h *IndicatorHandler) GetIndicator(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid indicator ID")
		return
	}

	// Check if user is admin
	isAdmin := h.checkIsAdmin(c)

	indicator, err := h.indicatorService.GetIndicator(c.Request.Context(), id, isAdmin)
	if err != nil {
		h.logger.Error("Failed to get indicator", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": indicator})
}

// CreateIndicator handles creating a new indicator
// POST /api/v1/indicators
func (h *IndicatorHandler) CreateIndicator(c *gin.Context) {
	// Check if user has admin role
	if !h.checkIsAdmin(c) {
		utils.SendErrorResponse(c, http.StatusForbidden, "Admin access required to create indicators")
		return
	}

	// Get user ID from context
	userID, _ := c.Get("userID")
	h.logger.Info("Creating indicator by user", zap.Int("userID", userID.(int)))

	// Define request struct to handle parameters
	var request struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description" binding:"required"`
		Category    string   `json:"category" binding:"required"`
		Formula     string   `json:"formula"`
		MinValue    *float64 `json:"min_value"`
		MaxValue    *float64 `json:"max_value"`
		IsActive    *bool    `json:"is_active"`
		Parameters  []struct {
			Name         string   `json:"name" binding:"required"`
			Type         string   `json:"type" binding:"required"`
			IsRequired   bool     `json:"is_required"`
			MinValue     *float64 `json:"min_value"`
			MaxValue     *float64 `json:"max_value"`
			DefaultValue string   `json:"default_value"`
			Description  string   `json:"description"`
			IsPublic     bool     `json:"is_public"`
			EnumValues   []struct {
				Value       string `json:"value" binding:"required"`
				DisplayName string `json:"display_name"`
			} `json:"enum_values"`
		} `json:"parameters"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Log what we're trying to insert
	h.logger.Info("Creating indicator",
		zap.String("name", request.Name),
		zap.String("category", request.Category),
		zap.Int("parameters_count", len(request.Parameters)),
		zap.Any("is_active", request.IsActive))

	// Convert request parameters to model parameters
	var parameters []model.IndicatorParameterCreate
	for _, param := range request.Parameters {
		// Convert enum values if present
		var enumValues []model.ParameterEnumValueCreate
		for _, enum := range param.EnumValues {
			enumValues = append(enumValues, model.ParameterEnumValueCreate{
				EnumValue:   enum.Value,
				DisplayName: enum.DisplayName,
			})
		}

		// Create parameter
		parameters = append(parameters, model.IndicatorParameterCreate{
			ParameterName: param.Name,
			ParameterType: param.Type,
			IsRequired:    param.IsRequired,
			MinValue:      param.MinValue,
			MaxValue:      param.MaxValue,
			DefaultValue:  param.DefaultValue,
			Description:   param.Description,
			IsPublic:      param.IsPublic,
			EnumValues:    enumValues,
		})
	}

	// Create the indicator using the service
	indicator, err := h.indicatorService.CreateIndicator(
		c.Request.Context(),
		request.Name,
		request.Description,
		request.Category,
		request.Formula,
		request.MinValue,
		request.MaxValue,
		request.IsActive,
		parameters,
	)

	if err != nil {
		h.logger.Error("Failed to create indicator", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to create indicator: "+err.Error())
		return
	}

	h.logger.Info("Successfully created indicator",
		zap.Int("id", indicator.ID),
		zap.Bool("is_active", indicator.IsActive))

	c.JSON(http.StatusCreated, gin.H{"data": indicator})
}

// DeleteIndicator handles deleting an indicator
// DELETE /api/v1/indicators/{id}
func (h *IndicatorHandler) DeleteIndicator(c *gin.Context) {
	// Check if user has admin role
	if !h.checkIsAdmin(c) {
		utils.SendErrorResponse(c, http.StatusForbidden, "Admin access required to delete indicators")
		return
	}

	// Parse indicator ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid indicator ID")
		return
	}

	// Delete the indicator
	err = h.indicatorService.DeleteIndicator(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete indicator", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateIndicator handles updating an indicator
// PUT /api/v1/indicators/{id}
func (h *IndicatorHandler) UpdateIndicator(c *gin.Context) {
	// Check if user has admin role
	if !h.checkIsAdmin(c) {
		utils.SendErrorResponse(c, http.StatusForbidden, "Admin access required to update indicators")
		return
	}

	// Parse indicator ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid indicator ID")
		return
	}

	// Bind request body
	var request struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Category    string   `json:"category" binding:"required"`
		Formula     string   `json:"formula"`
		MinValue    *float64 `json:"min_value"`
		MaxValue    *float64 `json:"max_value"`
		IsActive    *bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Create indicator object
	indicator := &model.TechnicalIndicator{
		Name:        request.Name,
		Description: request.Description,
		Category:    request.Category,
		Formula:     request.Formula,
		MinValue:    request.MinValue,
		MaxValue:    request.MaxValue,
	}

	// Set IsActive only if provided
	if request.IsActive != nil {
		indicator.IsActive = *request.IsActive
		h.logger.Info("Updating indicator active status",
			zap.Int("id", id),
			zap.Bool("is_active", indicator.IsActive))
	} else {
		// Fetch current indicator to get current active status
		currentIndicator, err := h.indicatorService.GetIndicator(c.Request.Context(), id, true)
		if err == nil && currentIndicator != nil {
			indicator.IsActive = currentIndicator.IsActive
		} else {
			indicator.IsActive = true // Default to true if we can't get current value
		}
	}

	// Update the indicator
	updatedIndicator, err := h.indicatorService.UpdateIndicator(c.Request.Context(), id, indicator)
	if err != nil {
		h.logger.Error("Failed to update indicator", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updatedIndicator})
}

// AddParameter handles adding a parameter to an indicator
// POST /api/v1/indicators/{id}/parameters
func (h *IndicatorHandler) AddParameter(c *gin.Context) {
	// Check if user has admin role
	if !h.checkIsAdmin(c) {
		utils.SendErrorResponse(c, http.StatusForbidden, "Admin access required to add parameters")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid indicator ID")
		return
	}

	var request ParameterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Just add "Number" to the list of valid types
	validTypes := map[string]bool{
		"number":  true,
		"boolean": true,
		"string":  true,
		"enum":    true,
	}

	if !validTypes[request.ParameterType] {
		utils.SendErrorResponse(c, http.StatusBadRequest,
			"Invalid parameter_type. Must be one of: integer, float, boolean, string, enum, price, timeframe")
		return
	}

	parameter, err := h.indicatorService.AddParameter(
		c.Request.Context(),
		id,
		request.ParameterName,
		request.ParameterType,
		request.IsRequired,
		request.MinValue,
		request.MaxValue,
		request.DefaultValue,
		request.Description,
		request.IsPublic,
	)

	if err != nil {
		h.logger.Error("Failed to add parameter", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": parameter})
}

type ParameterRequest struct {
	ParameterName string   `json:"parameter_name" binding:"required"`
	ParameterType string   `json:"parameter_type" binding:"required"`
	IsRequired    bool     `json:"is_required"`
	MinValue      *float64 `json:"min_value"`
	MaxValue      *float64 `json:"max_value"`
	DefaultValue  string   `json:"default_value"`
	Description   string   `json:"description"`
	IsPublic      bool     `json:"is_public"`
}

// AddEnumValue handles adding an enum value to a parameter
// POST /api/v1/parameters/{id}/enum-values
func (h *IndicatorHandler) AddEnumValue(c *gin.Context) {
	// Check if user has admin role
	if !h.checkIsAdmin(c) {
		utils.SendErrorResponse(c, http.StatusForbidden, "Admin access required to add enum values")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid parameter ID")
		return
	}

	var request struct {
		EnumValue   string `json:"enum_value" binding:"required"`
		DisplayName string `json:"display_name"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	enumValue, err := h.indicatorService.AddEnumValue(
		c.Request.Context(),
		id,
		request.EnumValue,
		request.DisplayName,
	)

	if err != nil {
		h.logger.Error("Failed to add enum value", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": enumValue})
}

// DeleteParameter handles deleting a parameter
// DELETE /api/v1/parameters/{id}
func (h *IndicatorHandler) DeleteParameter(c *gin.Context) {
	// Check if user has admin role
	if !h.checkIsAdmin(c) {
		utils.SendErrorResponse(c, http.StatusForbidden, "Admin access required to delete parameters")
		return
	}

	// Parse parameter ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid parameter ID")
		return
	}

	// Delete the parameter
	err = h.indicatorService.DeleteParameter(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete parameter", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateParameter handles updating a parameter
// PUT /api/v1/parameters/{id}
func (h *IndicatorHandler) UpdateParameter(c *gin.Context) {
	// Check if user has admin role
	if !h.checkIsAdmin(c) {
		utils.SendErrorResponse(c, http.StatusForbidden, "Admin access required to update parameters")
		return
	}

	// Parse parameter ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid parameter ID")
		return
	}

	// Bind request body
	var request struct {
		ParameterName string   `json:"parameter_name" binding:"required"`
		ParameterType string   `json:"parameter_type" binding:"required"`
		IsRequired    bool     `json:"is_required"`
		MinValue      *float64 `json:"min_value"`
		MaxValue      *float64 `json:"max_value"`
		DefaultValue  string   `json:"default_value"`
		Description   string   `json:"description"`
		IsPublic      bool     `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Create parameter object
	parameter := &model.IndicatorParameter{
		ID:            id,
		ParameterName: request.ParameterName,
		ParameterType: request.ParameterType,
		IsRequired:    request.IsRequired,
		MinValue:      request.MinValue,
		MaxValue:      request.MaxValue,
		DefaultValue:  request.DefaultValue,
		Description:   request.Description,
		IsPublic:      request.IsPublic,
	}

	// Update the parameter
	updatedParameter, err := h.indicatorService.UpdateParameter(c.Request.Context(), id, parameter)
	if err != nil {
		h.logger.Error("Failed to update parameter", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updatedParameter})
}

// DeleteEnumValue handles deleting an enum value
// DELETE /api/v1/enum-values/{id}
func (h *IndicatorHandler) DeleteEnumValue(c *gin.Context) {
	// Check if user has admin role
	if !h.checkIsAdmin(c) {
		utils.SendErrorResponse(c, http.StatusForbidden, "Admin access required to delete enum values")
		return
	}

	// Parse enum value ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid enum value ID")
		return
	}

	// Delete the enum value
	err = h.indicatorService.DeleteEnumValue(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete enum value", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateEnumValue handles updating an enum value
// PUT /api/v1/enum-values/{id}
func (h *IndicatorHandler) UpdateEnumValue(c *gin.Context) {
	// Check if user has admin role
	if !h.checkIsAdmin(c) {
		utils.SendErrorResponse(c, http.StatusForbidden, "Admin access required to update enum values")
		return
	}

	// Parse enum value ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid enum value ID")
		return
	}

	// Bind request body
	var request struct {
		EnumValue   string `json:"enum_value" binding:"required"`
		DisplayName string `json:"display_name"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Create enum value object
	enumValue := &model.ParameterEnumValue{
		EnumValue:   request.EnumValue,
		DisplayName: request.DisplayName,
	}

	// Update the enum value
	updatedEnumValue, err := h.indicatorService.UpdateEnumValue(c.Request.Context(), id, enumValue)
	if err != nil {
		h.logger.Error("Failed to update enum value", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updatedEnumValue})
}

// POST /api/v1/indicators/sync
func (h *IndicatorHandler) SyncIndicators(c *gin.Context) {
	// Check if user has admin role
	if !h.checkIsAdmin(c) {
		utils.SendErrorResponse(c, http.StatusForbidden, "Admin access required to sync indicators")
		return
	}

	// Get user ID from context
	userID, _ := c.Get("userID")
	h.logger.Info("Syncing indicators", zap.Int("userID", userID.(int)))

	// Sync indicators
	count, err := h.indicatorService.SyncIndicatorsFromBacktestingService(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to sync indicators", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to sync indicators: "+err.Error())
		return
	}

	h.logger.Info("Successfully synced indicators", zap.Int("count", count))
	c.JSON(http.StatusOK, gin.H{
		"status":            "success",
		"message":           fmt.Sprintf("Successfully synced %d indicators", count),
		"indicators_synced": count,
	})
}
