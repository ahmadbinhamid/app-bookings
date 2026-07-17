// Package timeoff manages EMPLOYEE_TIME_OFF (design doc): dated absences,
// distinct from schedules' recurring weekly pattern.
package timeoff

import "time"

type TimeOff struct {
	ID            string    `json:"id"`
	EmployeeID    string    `json:"employee_id"`
	StartDatetime time.Time `json:"start_datetime"`
	EndDatetime   time.Time `json:"end_datetime"`
	Reason        *string   `json:"reason"`
	CreatedAt     time.Time `json:"created_at"`
}

type Input struct {
	StartDatetime time.Time `json:"start_datetime" binding:"required"`
	EndDatetime   time.Time `json:"end_datetime" binding:"required"`
	Reason        *string   `json:"reason"`
}
