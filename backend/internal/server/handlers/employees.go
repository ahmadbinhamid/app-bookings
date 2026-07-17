package handlers

import (
	"net/http"

	"app-booking/internal/modules/employee"

	"github.com/gin-gonic/gin"
)

// EmployeeHandler is read-only: employees are synced from FlowPOS
// (internal/modules/sync), never created by hand.
type EmployeeHandler struct {
	employees *employee.Service
}

func NewEmployeeHandler(employees *employee.Service) *EmployeeHandler {
	return &EmployeeHandler{employees: employees}
}

// ListByLocation handles GET /locations/:locationId/employees. Tenant
// ownership of :locationId is already verified by RequireLocationOwnership
// (mounted on this route's group in server.go) — see ownership.go.
func (h *EmployeeHandler) ListByLocation(c *gin.Context) {
	employees, err := h.employees.ListByLocation(locationFrom(c).ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"employees": employees})
}
