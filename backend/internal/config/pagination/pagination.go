// Package pagination holds the domain-agnostic value types for offset-based
// pagination, shared by every list endpoint — ported verbatim (module path
// aside) from appointments/internal/config/pagination.
package pagination

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// Params is a validated, clamped page request.
type Params struct {
	Page    int
	PerPage int
}

// Limit and Offset translate the page request into SQL terms.
func (p Params) Limit() int  { return p.PerPage }
func (p Params) Offset() int { return (p.Page - 1) * p.PerPage }

// Meta is the pagination block returned alongside a page of data.
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Page wraps a slice of results with its pagination metadata.
type Page[T any] struct {
	Data []T  `json:"data"`
	Meta Meta `json:"meta"`
}

// NewPage builds a response page, computing total_pages and ensuring Data
// serializes as [] rather than null when empty.
func NewPage[T any](data []T, total int, p Params) Page[T] {
	totalPages := 0
	if p.PerPage > 0 {
		totalPages = (total + p.PerPage - 1) / p.PerPage
	}
	if data == nil {
		data = []T{}
	}
	return Page[T]{
		Data: data,
		Meta: Meta{
			Page:       p.Page,
			Limit:      p.PerPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}
