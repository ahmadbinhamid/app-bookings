package employee

import (
	"errors"
	"time"
)

// ErrHasFutureBookings is AssignLocation's guard against silently orphaning
// an upcoming booking: an employee can't be moved off their current location
// while they still have booked (non-cancelled) future segments there. The
// admin must cancel/reschedule those first.
var ErrHasFutureBookings = errors.New("employee has upcoming bookings at their current location")

// Employee mirrors the design doc's EMPLOYEE entity. active=false means
// "deactivated in FlowPOS" — rows are never hard-deleted since
// booking_segments reference them (design doc, Decisions table).
//
// LocationID is nullable: FlowPOS has no employee-location relationship at
// all (just two flat, unrelated lists per tenant), so a newly-synced
// employee starts unassigned (nil) until an admin assigns them to exactly
// one location by hand — see Service.AssignLocation.
type Employee struct {
	ID                string    `json:"id"`
	TenantID          uint64    `json:"tenant_id"`
	LocationID        *string   `json:"location_id"`
	FlowposEmployeeID string    `json:"flowpos_employee_id"`
	Name              string    `json:"name"`
	Email             string    `json:"email,omitempty"`
	Phone             string    `json:"phone,omitempty"`
	Active            bool      `json:"active"`
	SyncedAt          time.Time `json:"synced_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
