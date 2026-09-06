package utils

import (
	"github.com/gin-gonic/gin"
)

// Pagination contains normalized page, limit, and offset values.
type Pagination struct {
	Page   int
	Limit  int
	Offset int
}

// NewPagination normalizes page and limit values for list queries.
func NewPagination(page, limit, defaultLimit, maxLimit int) Pagination {
	if defaultLimit < 1 {
		defaultLimit = 20
	}
	if maxLimit < 1 {
		maxLimit = defaultLimit
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxLimit {
		limit = defaultLimit
	}

	return Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

// PaginationFromQuery reads page and limit query parameters from a Gin request.
func PaginationFromQuery(c *gin.Context, defaultLimit, maxLimit int) Pagination {
	page := intFromBindField(c, "page", 1)
	limit := intFromBindField(c, "limit", defaultLimit)
	return NewPagination(page, limit, defaultLimit, maxLimit)
}
