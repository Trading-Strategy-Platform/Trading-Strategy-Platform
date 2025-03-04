package model

// Pagination represents pagination parameters
type Pagination struct {
	Page    int `json:"page" form:"page"`
	PerPage int `json:"per_page" form:"per_page"`
}

// PaginationDefaults holds default pagination values
var PaginationDefaults = struct {
	Page       int
	PerPage    int
	MaxPerPage int
}{
	Page:       1,
	PerPage:    20,
	MaxPerPage: 100,
}

// GetPage returns the page number with defaults
func (p *Pagination) GetPage() int {
	if p.Page < 1 {
		return PaginationDefaults.Page
	}
	return p.Page
}

// GetPerPage returns the per page value with defaults
func (p *Pagination) GetPerPage() int {
	if p.PerPage < 1 {
		return PaginationDefaults.PerPage
	}
	if p.PerPage > PaginationDefaults.MaxPerPage {
		return PaginationDefaults.MaxPerPage
	}
	return p.PerPage
}

// GetOffset returns the offset for database queries
func (p *Pagination) GetOffset() int {
	return (p.GetPage() - 1) * p.GetPerPage()
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// NewPaginationMeta creates pagination metadata
func NewPaginationMeta(pagination *Pagination, totalItems int) PaginationMeta {
	page := pagination.GetPage()
	perPage := pagination.GetPerPage()

	totalPages := totalItems / perPage
	if totalItems%perPage > 0 {
		totalPages++
	}

	return PaginationMeta{
		Page:       page,
		PerPage:    perPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
