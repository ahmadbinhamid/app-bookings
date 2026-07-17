package handlers

import (
	"errors"
	"net/http"

	"app-booking/internal/modules/assignments"
	"app-booking/internal/modules/employee"

	"github.com/gin-gonic/gin"
)

// AssignmentHandler is mounted under /locations/:locationId/services/:serviceId
// (after RequireLocationOwnership + RequireServiceInLocation). It also
// checks the employee_id it's given (body for Assign, :employeeId param for
// Unassign) belongs to the same location — RequireEmployeeInLocation isn't
// used here since Assign's employee_id is a body field, not a route param.
type AssignmentHandler struct {
	assignments *assignments.Service
	employees   *employee.Service
}

func NewAssignmentHandler(assignments *assignments.Service, employees *employee.Service) *AssignmentHandler {
	return &AssignmentHandler{assignments: assignments, employees: employees}
}

func (h *AssignmentHandler) ListForService(c *gin.Context) {
	ids, err := h.assignments.ListEmployeeIDsForService(serviceFrom(c).ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]employee.Employee, 0, len(ids))
	for _, id := range ids {
		if e, err := h.employees.GetByID(id); err == nil {
			out = append(out, e)
		}
	}
	c.JSON(http.StatusOK, gin.H{"employees": out})
}

func (h *AssignmentHandler) Assign(c *gin.Context) {
	var in struct {
		EmployeeID string `json:"employee_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}

	emp, err := h.employees.GetByID(in.EmployeeID)
	if errors.Is(err, employee.ErrNotFound) || (err == nil && emp.LocationID != locationFrom(c).ID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "employee not found at this location"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	a, err := h.assignments.Assign(emp.ID, serviceFrom(c).ID)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *AssignmentHandler) Unassign(c *gin.Context) {
	emp, err := h.employees.GetByID(c.Param("employeeId"))
	if errors.Is(err, employee.ErrNotFound) || (err == nil && emp.LocationID != locationFrom(c).ID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "employee not found at this location"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.assignments.Unassign(emp.ID, serviceFrom(c).ID); err != nil {
		respondErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
