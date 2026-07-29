package handlers

import (
	"errors"
	"net/http"

	"app-booking/internal/modules/booking"
	"app-booking/internal/modules/employee"
	"app-booking/internal/modules/location"
	"app-booking/internal/modules/services"

	"github.com/gin-gonic/gin"
)

// This file holds every "does this path param actually belong to the
// calling tenant/location" check, extracted out of individual handlers
// (originally added ad hoc in the Phase 2 employees handler) so every
// location-scoped route from Phase 3 onward uses the same guard instead of
// each handler re-implementing its own — see server.go for how these are
// mounted as route-group middleware.

const (
	locationKey = "fp_location"
	employeeKey = "fp_employee"
	serviceKey  = "fp_service"
	bookingKey  = "fp_booking"
)

// RequireLocationOwnership checks that :locationId both exists and belongs
// to the calling tenant, and stashes the resolved location in context —
// mount on any route group whose path starts with /locations/:locationId.
// Without this, an authenticated caller from a different tenant could read
// or mutate another tenant's data just by guessing/incrementing an id.
func RequireLocationOwnership(locations *location.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		loc, err := locations.GetByID(c.Param("locationId"))
		if errors.Is(err, location.ErrNotFound) || (err == nil && loc.TenantID != tenantID(c)) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "location not found"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Set(locationKey, loc)
		c.Next()
	}
}

func locationFrom(c *gin.Context) location.Location {
	v, _ := c.Get(locationKey)
	loc, _ := v.(location.Location)
	return loc
}

// RequireEmployeeInLocation checks that :employeeId exists and belongs to
// the location already resolved by RequireLocationOwnership — mount after
// it on any route group nested under /locations/:locationId/employees/:employeeId.
// An employee not yet assigned to any location (LocationID == nil) never
// belongs here either.
func RequireEmployeeInLocation(employees *employee.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		emp, err := employees.GetByID(c.Param("employeeId"))
		if errors.Is(err, employee.ErrNotFound) || (err == nil && (emp.LocationID == nil || *emp.LocationID != locationFrom(c).ID)) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "employee not found"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Set(employeeKey, emp)
		c.Next()
	}
}

func employeeFrom(c *gin.Context) employee.Employee {
	v, _ := c.Get(employeeKey)
	emp, _ := v.(employee.Employee)
	return emp
}

// RequireServiceInLocation checks that :serviceId exists and belongs to the
// location already resolved by RequireLocationOwnership — mount after it on
// any route group nested under /locations/:locationId/services/:serviceId.
func RequireServiceInLocation(svc *services.MyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := svc.Get(c.Param("serviceId"))
		if errors.Is(err, services.ErrNotFound) || (err == nil && s.LocationID != locationFrom(c).ID) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "service not found"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Set(serviceKey, s)
		c.Next()
	}
}

func serviceFrom(c *gin.Context) services.Service {
	v, _ := c.Get(serviceKey)
	s, _ := v.(services.Service)
	return s
}

// RequireBookingInLocation checks that :bookingId exists and belongs to the
// location already resolved by RequireLocationOwnership — mount after it on
// any route group nested under /locations/:locationId/bookings/:bookingId.
func RequireBookingInLocation(bookings *booking.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		b, err := bookings.GetByID(c.Param("bookingId"))
		if errors.Is(err, booking.ErrNotFound) || (err == nil && b.LocationID != locationFrom(c).ID) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "booking not found"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Set(bookingKey, b)
		c.Next()
	}
}

func bookingFrom(c *gin.Context) booking.Booking {
	v, _ := c.Get(bookingKey)
	b, _ := v.(booking.Booking)
	return b
}
