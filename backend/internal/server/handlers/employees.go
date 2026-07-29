package handlers

import (
	"errors"
	"net/http"

	"app-booking/internal/modules/employee"
	"app-booking/internal/modules/location"

	"github.com/gin-gonic/gin"
)

// EmployeeHandler: ListByLocation/ListUnassigned are read-only — employees
// are synced from FlowPOS (internal/modules/sync), never created by hand.
// AssignLocation is the one field an admin manages by hand, since FlowPOS
// has no employee-location relationship at all to sync (design resolution:
// see internal/modules/employee.Service.AssignLocation's doc comment).
type EmployeeHandler struct {
	employees *employee.Service
	locations *location.Service
}

func NewEmployeeHandler(employees *employee.Service, locations *location.Service) *EmployeeHandler {
	return &EmployeeHandler{employees: employees, locations: locations}
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

// ListAll handles GET /api/v1/employees — every employee synced for the
// tenant, assigned or not. This is the Employees page's roster view; a
// per-location "who works here" view is still available at
// ListByLocation for the Locations/booking-flow use cases that need it.
func (h *EmployeeHandler) ListAll(c *gin.Context) {
	employees, err := h.employees.ListByTenant(tenantID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"employees": employees})
}

// ListUnassigned handles GET /api/v1/employees/unassigned — the tenant-wide
// pool of synced employees not yet assigned to any location. This is a
// tenant-level route (no :locationId in the path), so it scopes by the JWT's
// tenant_id directly rather than via RequireLocationOwnership.
func (h *EmployeeHandler) ListUnassigned(c *gin.Context) {
	employees, err := h.employees.ListUnassigned(tenantID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"employees": employees})
}

// AssignLocation handles PATCH /api/v1/employees/:employeeId/location — sets
// or changes the single location an employee belongs to. Not nested under
// /locations/:locationId (an employee being assigned for the first time, or
// moved from a different location, isn't yet "in" the target location), so
// ownership of both the employee and the target location is verified here
// directly against the JWT's tenant_id, mirroring the checks
// RequireLocationOwnership/RequireEmployeeInLocation do for nested routes.
func (h *EmployeeHandler) AssignLocation(c *gin.Context) {
	var in struct {
		LocationID string `json:"location_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}

	emp, err := h.employees.GetByID(c.Param("employeeId"))
	if errors.Is(err, employee.ErrNotFound) || (err == nil && emp.TenantID != tenantID(c)) {
		c.JSON(http.StatusNotFound, gin.H{"error": "employee not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	loc, err := h.locations.GetByID(in.LocationID)
	if errors.Is(err, location.ErrNotFound) || (err == nil && loc.TenantID != tenantID(c)) {
		c.JSON(http.StatusNotFound, gin.H{"error": "location not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.employees.AssignLocation(emp.ID, loc.ID)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"employee": updated})
}
