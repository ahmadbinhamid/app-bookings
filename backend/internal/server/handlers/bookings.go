package handlers

import (
	"net/http"
	"time"

	"app-booking/internal/modules/booking"

	"github.com/gin-gonic/gin"
)

// BookingHandler is mounted under /locations/:locationId (after
// RequireLocationOwnership); :bookingId routes additionally go through
// RequireBookingInLocation.
type BookingHandler struct {
	bookings *booking.Service
}

func NewBookingHandler(bookings *booking.Service) *BookingHandler {
	return &BookingHandler{bookings: bookings}
}

// Propose handles POST /locations/:locationId/bookings/propose — the
// read-only "work out a slot" step. Nothing is persisted.
func (h *BookingHandler) Propose(c *gin.Context) {
	var in booking.ProposeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	proposal, err := h.bookings.Propose(locationFrom(c).ID, in)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, proposal)
}

// Confirm handles POST /locations/:locationId/bookings/confirm — the
// client hands back exactly the proposal it was shown (design doc: fully
// optimistic, no server-side proposal cache).
func (h *BookingHandler) Confirm(c *gin.Context) {
	var in booking.ConfirmInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	b, err := h.bookings.Confirm(locationFrom(c).ID, adminIDFrom(c), in)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, b)
}

// List handles GET /locations/:locationId/bookings?employee_id=&from=&to=
func (h *BookingHandler) List(c *gin.Context) {
	var employeeID *string
	if v := c.Query("employee_id"); v != "" {
		employeeID = &v
	}
	from, err := parseOptionalTime(c.Query("from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' — expected RFC3339"})
		return
	}
	to, err := parseOptionalTime(c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' — expected RFC3339"})
		return
	}

	list, err := h.bookings.ListByLocation(locationFrom(c).ID, employeeID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookings": list})
}

func parseOptionalTime(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Get handles GET /locations/:locationId/bookings/:bookingId — the
// booking is already resolved by RequireBookingInLocation.
func (h *BookingHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, bookingFrom(c))
}

// Cancel handles POST /locations/:locationId/bookings/:bookingId/cancel.
func (h *BookingHandler) Cancel(c *gin.Context) {
	if err := h.bookings.CancelBooking(bookingFrom(c).ID); err != nil {
		respondErr(c, err)
		return
	}
	b, err := h.bookings.GetByID(bookingFrom(c).ID)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, b)
}

// CancelSegment handles POST /locations/:locationId/bookings/:bookingId/segments/:segmentId/cancel.
func (h *BookingHandler) CancelSegment(c *gin.Context) {
	if err := h.bookings.CancelSegment(bookingFrom(c).ID, c.Param("segmentId")); err != nil {
		respondErr(c, err)
		return
	}
	b, err := h.bookings.GetByID(bookingFrom(c).ID)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, b)
}

// Reschedule handles POST /locations/:locationId/bookings/:bookingId/reschedule
// — body is a fresh proposal (from Propose) for the new preferences.
func (h *BookingHandler) Reschedule(c *gin.Context) {
	var in booking.RescheduleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	b, err := h.bookings.Reschedule(bookingFrom(c).ID, in)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, b)
}
