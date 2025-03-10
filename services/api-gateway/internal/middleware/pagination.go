package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	sharedModel "github.com/yourorg/trading-platform/shared/go/model"
)

// ExtractPagination parses pagination parameters and sets them in context
func ExtractPagination() gin.HandlerFunc {
	return func(c *gin.Context) {
		var pagination sharedModel.Pagination

		// Extract page parameter
		if pageStr := c.Query("page"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
				pagination.Page = page
			}
		}

		// Extract per_page parameter
		if perPageStr := c.Query("per_page"); perPageStr != "" {
			if perPage, err := strconv.Atoi(perPageStr); err == nil && perPage > 0 {
				pagination.PerPage = perPage
			}
		}

		// Set defaults if not provided
		if pagination.Page == 0 {
			pagination.Page = sharedModel.PaginationDefaults.Page
		}

		if pagination.PerPage == 0 {
			pagination.PerPage = sharedModel.PaginationDefaults.PerPage
		}

		// Store in context for handlers to use
		c.Set("pagination", pagination)

		// Continue processing
		c.Next()
	}
}
