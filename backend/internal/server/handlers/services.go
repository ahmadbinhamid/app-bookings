package handlers

import (
	"net/http"
	"strings"

	"app-booking/internal/config/pagination"
	"app-booking/internal/modules/services"

	"github.com/gin-gonic/gin"
)

// ServiceHandler is mounted under /locations/:locationId — every method
// here relies on RequireLocationOwnership having already run and stashed
// the location in context.
type ServiceHandler struct {
	svc *services.MyService
}

func NewServiceHandler(svc *services.MyService) *ServiceHandler {
	return &ServiceHandler{svc: svc}
}

// List defaults to every service (active or deleted/deactivated) — the
// Services management page passes ?active=true to hide deleted ones, while
// booking flows that need to resolve a possibly-deleted service (new-booking
// picker aside, which filters active itself client-side; historical booking
// display) rely on this default so old bookings still show the right name.
func (h *ServiceHandler) List(c *gin.Context) {
	p := parsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	activeOnly := c.Query("active") == "true"
	items, total, err := h.svc.List(locationFrom(c).ID, p, search, activeOnly)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, pagination.NewPage(items, total, p))
}

// Get returns the service already resolved by RequireServiceInLocation —
// no extra repository call needed.
func (h *ServiceHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, serviceFrom(c))
}

func (h *ServiceHandler) Create(c *gin.Context) {
	var in services.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	s, err := h.svc.Create(locationFrom(c).ID, in)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, s)
}

func (h *ServiceHandler) Update(c *gin.Context) {
	var in services.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	s, err := h.svc.Update(serviceFrom(c).ID, in)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, s)
}

// Delete soft-deletes the service (see services.Repository.Delete's doc
// comment for why) — always a plain 204, there's no failure mode left to
// report.
func (h *ServiceHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(serviceFrom(c).ID); err != nil {
		respondErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
