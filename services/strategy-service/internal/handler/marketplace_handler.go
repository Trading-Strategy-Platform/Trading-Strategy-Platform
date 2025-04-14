package handler

import (
	"net/http"
	"strconv"
	"strings"

	"services/strategy-service/internal/model"
	"services/strategy-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

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

// ListListings handles listing marketplace listings
// GET /api/v1/marketplace
func (h *MarketplaceHandler) ListListings(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Parse search term
	searchTerm := c.Query("search")

	// Parse price filters
	var minPrice *float64
	if minPriceStr := c.Query("min_price"); minPriceStr != "" {
		minPriceVal, err := strconv.ParseFloat(minPriceStr, 64)
		if err == nil {
			minPrice = &minPriceVal
		}
	}

	var maxPrice *float64
	if maxPriceStr := c.Query("max_price"); maxPriceStr != "" {
		maxPriceVal, err := strconv.ParseFloat(maxPriceStr, 64)
		if err == nil {
			maxPrice = &maxPriceVal
		}
	}

	// Parse is_free filter
	var isFree *bool
	if isFreeStr := c.Query("is_free"); isFreeStr != "" {
		isFreeBool := isFreeStr == "true"
		isFree = &isFreeBool
	}

	// Parse tag filters
	var tags []int
	if tagsStr := c.Query("tags"); tagsStr != "" {
		for _, tagStr := range strings.Split(tagsStr, ",") {
			if tagID, err := strconv.Atoi(tagStr); err == nil {
				tags = append(tags, tagID)
			}
		}
	}

	// Parse min_rating filter
	var minRating *float64
	if minRatingStr := c.Query("min_rating"); minRatingStr != "" {
		minRatingVal, err := strconv.ParseFloat(minRatingStr, 64)
		if err == nil {
			minRating = &minRatingVal
		}
	}

	// Parse sort_by parameter
	sortBy := c.DefaultQuery("sort_by", "popularity")
	validSortOptions := map[string]bool{
		"popularity": true,
		"rating":     true,
		"price_asc":  true,
		"price_desc": true,
		"newest":     true,
	}

	if !validSortOptions[sortBy] {
		sortBy = "popularity"
	}

	listings, total, err := h.marketplaceService.GetAllListings(
		c.Request.Context(),
		searchTerm,
		minPrice,
		maxPrice,
		isFree,
		tags,
		minRating,
		sortBy,
		page,
		limit,
	)

	if err != nil {
		h.logger.Error("Failed to get marketplace listings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch listings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"listings": listings,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (total + limit - 1) / limit,
		},
	})
}

// CreateListing handles creating a new marketplace listing
// POST /api/v1/marketplace
func (h *MarketplaceHandler) CreateListing(c *gin.Context) {
	var request model.MarketplaceCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.Price < 0 {
		request.Price = 0
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

// DeleteListing handles deleting a marketplace listing
// DELETE /api/v1/marketplace/{id}
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
// POST /api/v1/marketplace/{id}/purchase
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

// CancelSubscription handles canceling a marketplace subscription
// PUT /api/v1/marketplace/purchases/{id}/cancel
func (h *MarketplaceHandler) CancelSubscription(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid purchase ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err = h.marketplaceService.CancelSubscription(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to cancel subscription", zap.Error(err), zap.Int("purchase_id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetReviews handles retrieving reviews for a marketplace listing
// GET /api/v1/marketplace/{id}/reviews
func (h *MarketplaceHandler) GetReviews(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid listing ID"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

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
			"pages": (total + limit - 1) / limit,
		},
	})
}

// CreateReview handles creating a review for a purchased strategy
// POST /api/v1/marketplace/{id}/reviews
func (h *MarketplaceHandler) CreateReview(c *gin.Context) {
	idStr := c.Param("id")
	marketplaceID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid listing ID"})
		return
	}

	var request struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	reviewCreate := &model.ReviewCreate{
		MarketplaceID: marketplaceID,
		Rating:        request.Rating,
		Comment:       request.Comment,
	}

	review, err := h.marketplaceService.CreateReview(c.Request.Context(), reviewCreate, userID.(int))
	if err != nil {
		h.logger.Error("Failed to create review", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, review)
}

// UpdateReview handles updating a review
// PUT /api/v1/reviews/{id}
func (h *MarketplaceHandler) UpdateReview(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid review ID"})
		return
	}

	var request struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err = h.marketplaceService.UpdateReview(
		c.Request.Context(),
		id,
		userID.(int),
		request.Rating,
		request.Comment,
	)

	if err != nil {
		h.logger.Error("Failed to update review", zap.Error(err), zap.Int("review_id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteReview handles deleting a review
// DELETE /api/v1/reviews/{id}
func (h *MarketplaceHandler) DeleteReview(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid review ID"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err = h.marketplaceService.DeleteReview(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to delete review", zap.Error(err), zap.Int("review_id", id))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
