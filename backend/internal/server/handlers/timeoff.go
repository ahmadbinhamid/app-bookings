package handlers

import (
	"net/http"

	"app-booking/internal/modules/timeoff"

	"github.com/gin-gonic/gin"
)

// TimeOffHandler is mounted under /locations/:locationId/employees/:employeeId
// (after RequireLocationOwnership + RequireEmployeeInLocation).
type TimeOffHandler struct {
	svc *timeoff.Service
}

func NewTimeOffHandler(svc *timeoff.Service) *TimeOffHandler {
	return &TimeOffHandler{svc: svc}
}

func (h *TimeOffHandler) List(c *gin.Context) {
	list, err := h.svc.ListByEmployee(employeeFrom(c).ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"time_off": list})
}

func (h *TimeOffHandler) Create(c *gin.Context) {
	var in timeoff.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	t, err := h.svc.Create(employeeFrom(c).ID, in)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, t)
}

// Delete verifies the time-off id actually belongs to :employeeId first —
// same reasoning as ScheduleHandler.Delete.
func (h *TimeOffHandler) Delete(c *gin.Context) {
	id := c.Param("timeOffId")
	t, err := h.svc.Get(id)
	if err != nil {
		respondErr(c, err)
		return
	}
	if t.EmployeeID != employeeFrom(c).ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "time off not found"})
		return
	}
	if err := h.svc.Delete(id); err != nil {
		respondErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
