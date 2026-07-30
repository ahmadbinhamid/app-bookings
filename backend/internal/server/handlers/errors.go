package handlers

import (
	"errors"
	"net/http"

	"app-booking/internal/flowpos"
	"app-booking/internal/modules/assignments"
	"app-booking/internal/modules/booking"
	"app-booking/internal/modules/employee"
	"app-booking/internal/modules/installation"
	"app-booking/internal/modules/location"
	"app-booking/internal/modules/schedules"
	"app-booking/internal/modules/services"
	"app-booking/internal/modules/timeoff"
	"app-booking/internal/solver"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// respondErr maps domain errors to HTTP status codes. Shared by every handler
// in this package — add new domains' sentinel errors here as features grow.
func respondErr(c *gin.Context, err error) {
	var noAvail *solver.NoAvailabilityError
	if errors.As(err, &noAvail) {
		// A structured body, not just a string — the frontend needs
		// service_id + reason to tell "no one offers this service" apart
		// from "no one's free today" (design doc, Solver edge cases).
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":      err.Error(),
			"code":       "NO_AVAILABILITY",
			"service_id": noAvail.ServiceID,
			"reason":     noAvail.Reason,
		})
		return
	}

	switch {
	case errors.Is(err, installation.ErrNotFound),
		errors.Is(err, location.ErrNotFound),
		errors.Is(err, employee.ErrNotFound),
		errors.Is(err, services.ErrNotFound),
		errors.Is(err, assignments.ErrNotFound),
		errors.Is(err, schedules.ErrNotFound),
		errors.Is(err, timeoff.ErrNotFound),
		errors.Is(err, booking.ErrNotFound),
		errors.Is(err, booking.ErrSegmentNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, assignments.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, schedules.ErrInvalidTimeRange),
		errors.Is(err, timeoff.ErrInvalidTimeRange),
		errors.Is(err, location.ErrInvalidTimezone),
		errors.Is(err, booking.ErrLocationTimezoneUnset),
		errors.Is(err, solver.ErrNoServicesRequested),
		errors.Is(err, booking.ErrInvalidProposal),
		errors.Is(err, booking.ErrInvalidSegment):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, schedules.ErrOverlap),
		errors.Is(err, booking.ErrSlotNoLongerAvailable),
		errors.Is(err, booking.ErrEmployeeNoLongerAvailable),
		errors.Is(err, booking.ErrAlreadyCancelled),
		errors.Is(err, booking.ErrAlreadyCompleted),
		errors.Is(err, employee.ErrHasFutureBookings):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": errCode(err)})
	case errors.Is(err, flowpos.ErrUpstreamUnavailable):
		// FlowPOS itself is down/erroring/unreachable — not this tenant's api_key
		// or anything about the request. 503 (not 502) since retrying later, not
		// changing anything about the request, is the only thing that helps.
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "FlowPOS is currently unavailable. Please try again in a few minutes.",
			"code":  "FLOWPOS_UNAVAILABLE",
		})
	case errors.Is(err, flowpos.ErrUpstreamRejected):
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	case errors.Is(err, flowpos.ErrInvalidInput), errors.Is(err, flowpos.ErrEndpointNotFound):
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// errCode gives the frontend a stable machine-readable string for the few
// errors where the design doc names one explicitly (SLOT_NO_LONGER_AVAILABLE
// etc) — everything else relies on the HTTP status + message alone.
func errCode(err error) string {
	switch {
	case errors.Is(err, booking.ErrSlotNoLongerAvailable):
		return "SLOT_NO_LONGER_AVAILABLE"
	case errors.Is(err, booking.ErrEmployeeNoLongerAvailable):
		return "EMPLOYEE_NO_LONGER_AVAILABLE"
	case errors.Is(err, booking.ErrAlreadyCancelled):
		return "ALREADY_CANCELLED"
	case errors.Is(err, booking.ErrAlreadyCompleted):
		return "ALREADY_COMPLETED"
	case errors.Is(err, schedules.ErrOverlap):
		return "SCHEDULE_OVERLAP"
	case errors.Is(err, employee.ErrHasFutureBookings):
		return "HAS_FUTURE_BOOKINGS"
	default:
		return ""
	}
}

// respondBindErr handles request-body binding failures. Field validation
// failures become a 422 with a {"errors": {field: message}} map; a malformed
// or unparseable body (which can't be attributed to a field) is a 400.
func respondBindErr(c *gin.Context, err error) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make(map[string]string, len(ve))
		for _, fe := range ve {
			fields[fe.Field()] = validationMessage(fe)
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": fields})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

// validationMessage turns a single field error into a human-readable message.
func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	default:
		return fe.Field() + " is invalid"
	}
}
