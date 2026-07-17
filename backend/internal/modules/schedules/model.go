// Package schedules manages EMPLOYEE_SCHEDULE (design doc): an employee's
// recurring weekly working hours. Multiple rows per employee per day are
// valid on purpose (split shifts) — see the design doc's Constraints table:
// no DB uniqueness on (employee_id, day_of_week), only a service-layer
// overlap check (this package's Service.Create).
package schedules

import "time"

// StartTime/EndTime are "HH:MM:SS" wall-clock strings, not time.Time — a
// schedule entry has no date, and MySQL's TIME column round-trips cleanly
// as a plain string through database/sql without needing a custom scanner.
type Schedule struct {
	ID         string    `json:"id"`
	EmployeeID string    `json:"employee_id"`
	DayOfWeek  int       `json:"day_of_week"`
	StartTime  string    `json:"start_time"`
	EndTime    string    `json:"end_time"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Input struct {
	DayOfWeek int    `json:"day_of_week" binding:"min=0,max=6"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
}
