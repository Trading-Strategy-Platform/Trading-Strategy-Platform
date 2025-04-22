package handler

import (
	"net/http"
	"strconv"
	"strings"

	"services/strategy-service/internal/model"
	"services/strategy-service/internal/service"
	"services/strategy-service/internal/utils"

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
	// Parse pagination parameters using the utility function
	params := utils.ParsePaginationParams(c, 20, 100) // default limit: 20, max limit: 100

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
		params.Page,
		params.Limit,
	)

	if err != nil {
		h.logger.Error("Failed to get marketplace listings", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to fetch listings")
		return
	}

	// Use standardized pagination response
	utils.SendPaginatedResponse(c, http.StatusOK, listings, total, params.Page, params.Limit)
}

// CreateListing handles creating a new marketplace listing
// POST /api/v1/marketplace
func (h *MarketplaceHandler) CreateListing(c *gin.Context) {
	var request model.MarketplaceCreate
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if request.Price < 0 {
		request.Price = 0
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	listing, err := h.marketplaceService.CreateListing(c.Request.Context(), &request, userID.(int))
	if err != nil {
		h.logger.Error("Failed to create listing", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": listing})
}

// DeleteListing handles deleting a marketplace listing
// DELETE /api/v1/marketplace/{id}
func (h *MarketplaceHandler) DeleteListing(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid listing ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err = h.marketplaceService.DeleteListing(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to delete listing", zap.Error(err), zap.Int("id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
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
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid listing ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	purchase, err := h.marketplaceService.PurchaseStrategy(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to purchase strategy", zap.Error(err), zap.Int("listing_id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": purchase})
}

// CancelSubscription handles canceling a marketplace subscription
// PUT /api/v1/marketplace/purchases/{id}/cancel
func (h *MarketplaceHandler) CancelSubscription(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid purchase ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err = h.marketplaceService.CancelSubscription(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to cancel subscription", zap.Error(err), zap.Int("purchase_id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
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
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid listing ID")
		return
	}

	// Parse pagination parameters using the utility function
	params := utils.ParsePaginationParams(c, 10, 50) // default limit: 10, max limit: 50

	// Parse min_rating filter
	var minRating *float64
	if minRatingStr := c.Query("min_rating"); minRatingStr != "" {
		minRatingVal, err := strconv.ParseFloat(minRatingStr, 64)
		if err == nil {
			minRating = &minRatingVal
		}
	}

	reviews, total, err := h.marketplaceService.GetReviews(
		c.Request.Context(),
		id,
		minRating,
		params.Page,
		params.Limit,
	)

	if err != nil {
		h.logger.Error("Failed to get reviews", zap.Error(err))
		utils.SendErrorResponse(c, http.StatusInternalServerError, "Failed to fetch reviews")
		return
	}

	// Use standardized pagination response
	utils.SendPaginatedResponse(c, http.StatusOK, reviews, total, params.Page, params.Limit)
}

// CreateReview handles creating a review for a purchased strategy
// POST /api/v1/marketplace/{id}/reviews
func (h *MarketplaceHandler) CreateReview(c *gin.Context) {
	idStr := c.Param("id")
	marketplaceID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid listing ID")
		return
	}

	var request struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
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
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": review})
}

// UpdateReview handles updating a review
// PUT /api/v1/reviews/{id}
func (h *MarketplaceHandler) UpdateReview(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid review ID")
		return
	}

	var request struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
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
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
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
		utils.SendErrorResponse(c, http.StatusBadRequest, "Invalid review ID")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		utils.SendErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err = h.marketplaceService.DeleteReview(c.Request.Context(), id, userID.(int))
	if err != nil {
		h.logger.Error("Failed to delete review", zap.Error(err), zap.Int("review_id", id))
		utils.SendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}
