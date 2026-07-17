package employee

import "time"

// Employee mirrors the design doc's EMPLOYEE entity. active=false means
// "deactivated in FlowPOS" — rows are never hard-deleted since
// booking_segments reference them (design doc, Decisions table).
type Employee struct {
	ID                string    `json:"id"`
	LocationID        string    `json:"location_id"`
	FlowposEmployeeID string    `json:"flowpos_employee_id"`
	Name              string    `json:"name"`
	Email             string    `json:"email,omitempty"`
	Phone             string    `json:"phone,omitempty"`
	Active            bool      `json:"active"`
	SyncedAt          time.Time `json:"synced_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
