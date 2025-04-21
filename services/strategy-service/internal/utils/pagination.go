package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationParams holds pagination-related query parameters
type PaginationParams struct {
	Page  int
	Limit int
}

// ParsePaginationParams parses and validates pagination parameters from the request
// with support for default and maximum limits
func ParsePaginationParams(c *gin.Context, defaultLimit int, maxLimit int) PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))

	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = defaultLimit
	} else if limit > maxLimit {
		limit = maxLimit // Cap the maximum limit
	}

	return PaginationParams{
		Page:  page,
		Limit: limit,
	}
}

// CalculateOffset calculates the offset for pagination in SQL queries
func CalculateOffset(page, limit int) int {
	return (page - 1) * limit
}

// CalculateTotalPages calculates the total number of pages based on total items and limit
func CalculateTotalPages(totalItems, limit int) int {
	totalPages := (totalItems + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}
	return totalPages
}

// PaginationMetadata represents the standardized pagination metadata
type PaginationMetadata struct {
	TotalItems   int `json:"totalItems"`
	CurrentPage  int `json:"currentPage"`
	TotalPages   int `json:"totalPages"`
	ItemsPerPage int `json:"itemsPerPage"`
}

// NewPaginationMetadata creates a new pagination metadata object
func NewPaginationMetadata(totalItems, page, limit int) PaginationMetadata {
	return PaginationMetadata{
		TotalItems:   totalItems,
		CurrentPage:  page,
		TotalPages:   CalculateTotalPages(totalItems, limit),
		ItemsPerPage: limit,
	}
}

// SendPaginatedResponse sends a standardized paginated API response
func SendPaginatedResponse(c *gin.Context, statusCode int, data interface{}, totalItems, page, limit int) {
	c.JSON(statusCode, gin.H{
		"data":       data,
		"pagination": NewPaginationMetadata(totalItems, page, limit),
	})
}

// SendErrorResponse sends a standardized error response
func SendErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{"error": message})
}
