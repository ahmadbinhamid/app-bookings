// Package services is deliberately plural, and its business-logic type is
// named MyService rather than Service — appointments/internal/modules/services
// hit the exact same problem (the domain entity is itself called "Service")
// and solved it the same way; mirrored here rather than inventing a
// different name, per "mirror the sibling apps' conventions exactly."
package services

import "time"

// Service is a bookable offering (design doc SERVICE entity) — e.g.
// "Haircut", 30 minutes, £25, with a 10-minute buffer.
type Service struct {
	ID              string  `json:"id"`
	LocationID      string  `json:"location_id"`
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	DurationMinutes int     `json:"duration_minutes"`
	// BufferMinutes blocks the employee only, never the client — see the
	// app-bookings design doc's Buffer semantics section. Confirmed
	// per-service for v1 (not per-employee).
	BufferMinutes int       `json:"buffer_minutes"`
	Price         float64   `json:"price"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Input is the create/update request body. LocationID is deliberately NOT
// part of it — it comes from the URL (:locationId, already ownership-
// checked), not a client-supplied field that could target another
// location's table.
type Input struct {
	Name            string  `json:"name" binding:"required"`
	Description     *string `json:"description"`
	DurationMinutes int     `json:"duration_minutes" binding:"required,min=1"`
	BufferMinutes   int     `json:"buffer_minutes" binding:"min=0"`
	Price           float64 `json:"price" binding:"min=0"`
	Active          *bool   `json:"active"`
}
