// Package booking is the service layer for the app-bookings design doc's
// Process/Reschedule/Algorithm sections: it fetches the data
// internal/solver needs (employee schedules, time off, existing segments),
// calls the pure solver to produce a proposal, and — separately — persists
// a confirmed proposal, cancels, and reschedules bookings, all under the
// transaction/locking rules the design doc specifies.
package booking

import (
	"errors"
	"time"
)

var (
	ErrNotFound        = errors.New("booking not found")
	ErrSegmentNotFound = errors.New("booking segment not found")
	// ErrInvalidProposal is the pairwise self-consistency failure from the
	// design doc: a proposal that books the same employee at overlapping
	// times against itself — distinct from ErrSlotNoLongerAvailable, which
	// means someone ELSE took the slot.
	ErrInvalidProposal = errors.New("proposal is internally inconsistent: an employee is double-booked against themselves")
	// ErrSlotNoLongerAvailable is the confirm/reschedule-time re-validation
	// failure — the actual double-booking guarantee (design doc,
	// Concurrency section).
	ErrSlotNoLongerAvailable = errors.New("one or more segments in this proposal are no longer available")
	// ErrEmployeeNoLongerAvailable is a DIFFERENT confirm/reschedule-time
	// re-validation failure than ErrSlotNoLongerAvailable: no other booking
	// conflicts with this segment, but the employee's EMPLOYEE_SCHEDULE or
	// EMPLOYEE_TIME_OFF changed between Propose and Confirm/Reschedule, so
	// the segment no longer fits their working hours. Kept distinct so the
	// UI can say "this employee's schedule changed" instead of "someone
	// else booked this" — either way the client's fix is the same, re-solve.
	ErrEmployeeNoLongerAvailable = errors.New("one or more segments' employees are no longer available at this time (schedule or time off changed)")
	// ErrInvalidSegment is a THIRD distinct confirm/reschedule-time
	// re-validation failure, for a ProposedSegment that could never have
	// legitimately come from this location's own Propose response: its
	// service belongs to a different location, or its employee isn't
	// active, isn't assigned to that service (EMPLOYEE_SERVICE), or
	// belongs to a different location. The design doc's "fully optimistic"
	// model means Confirm/Reschedule receive a client-controlled JSON body
	// with no server-side session tying it back to the original Propose
	// call — this is the check that a submitted segment couldn't have been
	// tampered with (or simply gone stale) into referencing the wrong
	// employee/service/location.
	ErrInvalidSegment        = errors.New("one or more segments reference an employee or service that isn't valid for this booking's location")
	ErrAlreadyCancelled      = errors.New("this is already cancelled")
	ErrAlreadyCompleted      = errors.New("this is already completed and can't be cancelled")
	ErrLocationTimezoneUnset = errors.New("this location's timezone must be confirmed before booking can be solved")
)

type Booking struct {
	ID               string     `json:"id"`
	LocationID       string     `json:"location_id"`
	CreatedByAdminID *uint64    `json:"created_by_admin_id,omitempty"`
	CustomerName     string     `json:"customer_name"`
	CustomerPhone    *string    `json:"customer_phone,omitempty"`
	CustomerEmail    *string    `json:"customer_email,omitempty"`
	Status           string     `json:"status"`
	WindowStart      *time.Time `json:"window_start,omitempty"`
	WindowEnd        *time.Time `json:"window_end,omitempty"`
	TotalPrice       float64    `json:"total_price"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	Segments []Segment `json:"segments,omitempty"`
}

type Segment struct {
	ID            string    `json:"id"`
	BookingID     string    `json:"booking_id"`
	EmployeeID    string    `json:"employee_id"`
	ServiceID     string    `json:"service_id"`
	SequenceOrder int       `json:"sequence_order"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	BlockedUntil  time.Time `json:"blocked_until"`
	Status        string    `json:"status"`
	OriginalPrice float64   `json:"original_price"`
	Price         float64   `json:"price"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ProposedSegment is what /bookings/propose returns and /bookings/confirm +
// /bookings/{id}/reschedule accept back — the design's fully-optimistic,
// stateless round trip (design doc, "Proposal holds" decision: no server-
// side proposal cache, the client hands back exactly what it was shown).
type ProposedSegment struct {
	ServiceID    string    `json:"service_id"`
	EmployeeID   string    `json:"employee_id"`
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	BlockedUntil time.Time `json:"blocked_until"`
	Price        float64   `json:"price"`
}

type Proposal struct {
	Segments    []ProposedSegment `json:"segments"`
	WindowStart time.Time         `json:"window_start"`
	WindowEnd   time.Time         `json:"window_end"`
	TotalPrice  float64           `json:"total_price"`
}

// ServiceRequest is one line of a propose request — just a service id and
// how many minutes it needs; qualification/scheduling data is resolved
// server-side (never trust the client for that).
type ServiceRequest struct {
	ServiceID string `json:"service_id" binding:"required"`
}

type ProposeInput struct {
	Services      []ServiceRequest `json:"services" binding:"required,min=1,dive"`
	EarliestStart time.Time        `json:"earliest_start" binding:"required"`
}

type ConfirmInput struct {
	CustomerName  string            `json:"customer_name" binding:"required"`
	CustomerPhone *string           `json:"customer_phone"`
	CustomerEmail *string           `json:"customer_email"`
	Segments      []ProposedSegment `json:"segments" binding:"required,min=1"`
}

type RescheduleInput struct {
	Segments []ProposedSegment `json:"segments" binding:"required,min=1"`
}
