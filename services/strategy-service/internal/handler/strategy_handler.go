// services/strategy-service/internal/handler/strategy_handler.go
package handler

import (
	"net/http"
	"strconv"

	"services/strategy-service/internal/model"
	"services/strategy-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// StrategyHandler handles strategy-related HTTP requests
type StrategyHandler struct {
	strategyService *service.StrategyService
	logger          *zap.Logger
}

// NewStrategyHandler creates a new strategy handler
func NewStrategyHandler(strategyService *service.StrategyService, logger *zap.Logger) *StrategyHandler {
	return &StrategyHandler{
		strategyService: strategyService,
		logger:          logger,
	}
}

// CreateStrategy handles creating a new strategy
func (h *StrategyHandler) CreateStrategy(c *gin.Context) {
	var request model.StrategyCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	strategy, err := h.strategyService.CreateStrategy(c.Request.Context(), &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to create strategy", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, strategy)
}

// GetStrategy handles retrieving a strategy by ID
func (h *StrategyHandler) GetStrategy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	strategy, err := h.strategyService.GetStrategy(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to get strategy", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, strategy)
}

// ListUserStrategies handles listing strategies for a user
func (h *StrategyHandler) ListUserStrategies(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse request parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Parse isPublic filter
	var isPublic *bool
	if isPublicStr := c.Query("is_public"); isPublicStr != "" {
		isPublicBool := isPublicStr == "true"
		isPublic = &isPublicBool
	}

	// Parse tag filter
	var tagID *int
	if tagIDStr := c.Query("tag_id"); tagIDStr != "" {
		tagIDInt, err := strconv.Atoi(tagIDStr)
		if err == nil {
			tagID = &tagIDInt
		}
	}

	strategies, total, err := h.strategyService.GetUserStrategies(
		c.Request.Context(),
		userID.(int),
		userID.(int),
		isPublic,
		tagID,
		page,
		limit,
	)

	if err != nil {
		h.logger.Error("Failed to get user strategies", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch strategies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"strategies": strategies,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// ListPublicStrategies handles listing public strategies
func (h *StrategyHandler) ListPublicStrategies(c *gin.Context) {
	// Parse request parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Parse tag filter
	var tagID *int
	if tagIDStr := c.Query("tag_id"); tagIDStr != "" {
		tagIDInt, err := strconv.Atoi(tagIDStr)
		if err == nil {
			tagID = &tagIDInt
		}
	}

	strategies, total, err := h.strategyService.GetPublicStrategies(
		c.Request.Context(),
		tagID,
		page,
		limit,
	)

	if err != nil {
		h.logger.Error("Failed to get public strategies", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch strategies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"strategies": strategies,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// UpdateStrategy handles updating a strategy
func (h *StrategyHandler) UpdateStrategy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var request model.StrategyUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	strategy, err := h.strategyService.UpdateStrategy(c.Request.Context(), id, &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to update strategy", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, strategy)
}

// DeleteStrategy handles deleting a strategy
func (h *StrategyHandler) DeleteStrategy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err = h.strategyService.DeleteStrategy(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to delete strategy", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateVersion handles creating a new version of a strategy
func (h *StrategyHandler) CreateVersion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var request model.VersionCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	version, err := h.strategyService.CreateVersion(c.Request.Context(), id, &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to create version", zap.Error(err), zap.Int("strategy_id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, version)
}

// GetVersions handles retrieving all versions of a strategy
func (h *StrategyHandler) GetVersions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	versions, err := h.strategyService.GetVersions(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to get versions", zap.Error(err), zap.Int("strategy_id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, versions)
}

// GetVersion handles retrieving a specific version of a strategy
func (h *StrategyHandler) GetVersion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version number"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	versionObj, err := h.strategyService.GetVersion(c.Request.Context(), id, version, userID.(int))
	if err != nil {
		h.logger.Error("Failed to get version", zap.Error(err), zap.Int("strategy_id", id), zap.Int("version", version))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, versionObj)
}

// RestoreVersion handles restoring a strategy to a previous version
func (h *StrategyHandler) RestoreVersion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version number"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	strategy, err := h.strategyService.RestoreVersion(c.Request.Context(), id, version, userID.(int))
	if err != nil {
		h.logger.Error("Failed to restore version", zap.Error(err), zap.Int("strategy_id", id), zap.Int("version", version))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, strategy)
}

// CloneStrategy handles cloning a strategy
func (h *StrategyHandler) CloneStrategy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var request struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	strategy, err := h.strategyService.CloneStrategy(c.Request.Context(), id, userID.(int), request.Name)
	if err != nil {
		h.logger.Error("Failed to clone strategy", zap.Error(err), zap.Int("source_id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, strategy)
}

// StartBacktest handles starting a backtest for a strategy
func (h *StrategyHandler) StartBacktest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var request model.BacktestRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set strategy ID from URL parameter
	request.StrategyID = id

	backtestID, err := h.strategyService.StartBacktest(c.Request.Context(), &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to start backtest", zap.Error(err), zap.Int("strategy_id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"backtest_id": backtestID,
		"message":     "Backtest started successfully",
	})
}

// TagHandler handles tag-related HTTP requests
type TagHandler struct {
	tagService *service.TagService
	logger     *zap.Logger
}

// NewTagHandler creates a new tag handler
func NewTagHandler(tagService *service.TagService, logger *zap.Logger) *TagHandler {
	return &TagHandler{
		tagService: tagService,
		logger:     logger,
	}
}

// GetAllTags handles retrieving all tags
func (h *TagHandler) GetAllTags(c *gin.Context) {
	tags, err := h.tagService.GetAllTags(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get tags", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags"})
		return
	}

	c.JSON(http.StatusOK, tags)
}

// CreateTag handles creating a new tag
func (h *TagHandler) CreateTag(c *gin.Context) {
	var request struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tag, err := h.tagService.CreateTag(c.Request.Context(), request.Name)
	if err != nil {
		h.logger.Error("Failed to create tag", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tag)
}

// IndicatorHandler handles indicator-related HTTP requests
type IndicatorHandler struct {
	indicatorService *service.IndicatorService
	logger           *zap.Logger
}

// NewIndicatorHandler creates a new indicator handler
func NewIndicatorHandler(indicatorService *service.IndicatorService, logger *zap.Logger) *IndicatorHandler {
	return &IndicatorHandler{
		indicatorService: indicatorService,
		logger:           logger,
	}
}

// GetAllIndicators handles retrieving all indicators
func (h *IndicatorHandler) GetAllIndicators(c *gin.Context) {
	// Parse category filter
	var category *string
	if categoryStr := c.Query("category"); categoryStr != "" {
		category = &categoryStr
	}

	indicators, err := h.indicatorService.GetAllIndicators(c.Request.Context(), category)
	if err != nil {
		h.logger.Error("Failed to get indicators", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch indicators"})
		return
	}

	c.JSON(http.StatusOK, indicators)
}

// GetIndicator handles retrieving a specific indicator
func (h *IndicatorHandler) GetIndicator(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid indicator ID"})
		return
	}

	indicator, err := h.indicatorService.GetIndicator(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get indicator", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, indicator)
}

// MarketplaceHandler handles marketplace-related HTTP requests
type MarketplaceHandler struct {
	marketplaceService *service.MarketplaceService
	logger             *zap.Logger
}

// NewMarketplaceHandler creates a new marketplace handler
func NewMarketplaceHandler(marketplaceService *service.MarketplaceService, logger *zap.Logger) *MarketplaceHandler {
	return &MarketplaceHandler{
		marketplaceService: marketplaceService,
		logger:             logger,
	}
}

// CreateListing handles creating a new marketplace listing
func (h *MarketplaceHandler) CreateListing(c *gin.Context) {
	var request model.MarketplaceCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	listing, err := h.marketplaceService.CreateListing(c.Request.Context(), &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to create listing", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, listing)
}

// GetListing handles retrieving a marketplace listing
func (h *MarketplaceHandler) GetListing(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid listing ID"})
		return
	}

	listing, err := h.marketplaceService.GetListing(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get listing", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, listing)
}

// ListListings handles listing marketplace listings
func (h *MarketplaceHandler) ListListings(c *gin.Context) {
	// Parse request parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Parse isActive filter
	var isActive *bool
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActiveBool := isActiveStr == "true"
		isActive = &isActiveBool
	}

	// Parse userID filter
	var userID *int
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		userIDInt, err := strconv.Atoi(userIDStr)
		if err == nil {
			userID = &userIDInt
		}
	}

	listings, total, err := h.marketplaceService.GetAllListings(
		c.Request.Context(),
		isActive,
		userID,
		page,
		limit,
	)

	if err != nil {
		h.logger.Error("Failed to get listings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch listings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"listings": listings,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// UpdateListing handles updating a marketplace listing
func (h *MarketplaceHandler) UpdateListing(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid listing ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var request struct {
		Price       *float64 `json:"price"`
		IsActive    *bool    `json:"is_active"`
		Description *string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	listing, err := h.marketplaceService.UpdateListing(
		c.Request.Context(),
		id,
		request.Price,
		request.IsActive,
		request.Description,
		userID.(int),
	)

	if err != nil {
		h.logger.Error("Failed to update listing", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, listing)
}

// DeleteListing handles deleting a marketplace listing
func (h *MarketplaceHandler) DeleteListing(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid listing ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err = h.marketplaceService.DeleteListing(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to delete listing", zap.Error(err), zap.Int("id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// PurchaseStrategy handles purchasing a strategy from the marketplace
func (h *MarketplaceHandler) PurchaseStrategy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid listing ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	purchase, err := h.marketplaceService.PurchaseStrategy(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to purchase strategy", zap.Error(err), zap.Int("listing_id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, purchase)
}

// GetPurchases handles retrieving a user's purchases
func (h *MarketplaceHandler) GetPurchases(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse request parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	purchases, total, err := h.marketplaceService.GetPurchases(
		c.Request.Context(),
		userID.(int),
		page,
		limit,
	)

	if err != nil {
		h.logger.Error("Failed to get purchases", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch purchases"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"purchases": purchases,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// CreateReview handles creating a review for a purchased strategy
func (h *MarketplaceHandler) CreateReview(c *gin.Context) {
	var request model.ReviewCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	review, err := h.marketplaceService.CreateReview(c.Request.Context(), &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to create review", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, review)
}

// GetReviews handles retrieving reviews for a marketplace listing
func (h *MarketplaceHandler) GetReviews(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid listing ID"})
		return
	}

	// Parse request parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	reviews, total, err := h.marketplaceService.GetReviews(
		c.Request.Context(),
		id,
		page,
		limit,
	)

	if err != nil {
		h.logger.Error("Failed to get reviews", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews": reviews,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}
