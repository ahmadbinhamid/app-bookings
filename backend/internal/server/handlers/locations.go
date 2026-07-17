package handlers

import (
	"net/http"

	"app-booking/internal/modules/location"

	"github.com/gin-gonic/gin"
)

// LocationHandler: List is read-only (locations are synced from FlowPOS —
// internal/modules/sync); SetTimezone is the one field an admin manages by
// hand (design resolution: "Location timezone is a manually-managed
// field").
type LocationHandler struct {
	locations *location.Service
}

func NewLocationHandler(locations *location.Service) *LocationHandler {
	return &LocationHandler{locations: locations}
}

func (h *LocationHandler) List(c *gin.Context) {
	locations, err := h.locations.ListByTenant(tenantID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"locations": locations})
}

// SetTimezone handles PATCH /locations/:locationId/timezone. Tenant
// ownership of :locationId is already verified by RequireLocationOwnership.
func (h *LocationHandler) SetTimezone(c *gin.Context) {
	var in struct {
		Timezone string `json:"timezone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	loc, err := h.locations.SetTimezone(locationFrom(c).ID, in.Timezone)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"location": loc})
}
