// Package assignments manages the EMPLOYEE_SERVICE join (design doc): which
// employees can perform which services. Kept as its own module rather than
// folded into employee or services, since it depends on both and neither
// should reach into the other's tables directly.
package assignments

import "time"

type Assignment struct {
	ID         string    `json:"id"`
	EmployeeID string    `json:"employee_id"`
	ServiceID  string    `json:"service_id"`
	CreatedAt  time.Time `json:"created_at"`
}
