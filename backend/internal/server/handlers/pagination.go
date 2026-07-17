package handlers

import (
	"strconv"

	"app-booking/internal/config/pagination"

	"github.com/gin-gonic/gin"
)

func parsePagination(c *gin.Context) pagination.Params {
	page := atoiDefault(c.Query("page"), pagination.DefaultPage)
	if page < 1 {
		page = pagination.DefaultPage
	}

	limit := atoiDefault(c.Query("limit"), pagination.DefaultPerPage)
	if limit < 1 {
		limit = pagination.DefaultPerPage
	}
	if limit > pagination.MaxPerPage {
		limit = pagination.MaxPerPage
	}

	return pagination.Params{Page: page, PerPage: limit}
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
