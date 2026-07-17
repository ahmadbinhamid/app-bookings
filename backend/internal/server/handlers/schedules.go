package handlers

import (
	"net/http"

	"app-booking/internal/modules/schedules"

	"github.com/gin-gonic/gin"
)

// ScheduleHandler is mounted under
// /locations/:locationId/employees/:employeeId (after RequireLocationOwnership
// + RequireEmployeeInLocation).
type ScheduleHandler struct {
	svc *schedules.Service
}

func NewScheduleHandler(svc *schedules.Service) *ScheduleHandler {
	return &ScheduleHandler{svc: svc}
}

func (h *ScheduleHandler) List(c *gin.Context) {
	list, err := h.svc.ListByEmployee(employeeFrom(c).ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"schedules": list})
}

func (h *ScheduleHandler) Create(c *gin.Context) {
	var in schedules.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	s, err := h.svc.Create(employeeFrom(c).ID, in)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, s)
}

// Delete verifies the schedule id actually belongs to :employeeId before
// deleting — otherwise an authenticated caller could delete a schedule row
// belonging to a different employee (possibly a different location/tenant
// entirely) just by knowing its id.
func (h *ScheduleHandler) Delete(c *gin.Context) {
	id := c.Param("scheduleId")
	s, err := h.svc.Get(id)
	if err != nil {
		respondErr(c, err)
		return
	}
	if s.EmployeeID != employeeFrom(c).ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}
	if err := h.svc.Delete(id); err != nil {
		respondErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
