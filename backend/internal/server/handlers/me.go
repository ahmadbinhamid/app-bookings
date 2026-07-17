package handlers

import (
	"net/http"

	"app-booking/internal/modules/installation"

	"github.com/gin-gonic/gin"
)

// MeHandler exposes the caller's installation state, used by the frontend to
// confirm the JWT round-trip works and the tenant is installed.
type MeHandler struct {
	installations *installation.Service
}

func NewMeHandler(installations *installation.Service) *MeHandler {
	return &MeHandler{installations: installations}
}

// Me returns the caller's installation (null if the tenant never installed
// the app — shouldn't normally happen since the dashboard only embeds this
// app post-install, but kept defensive).
func (h *MeHandler) Me(c *gin.Context) {
	inst, ok, err := h.installations.GetByTenant(tenantID(c))
	if err != nil {
		respondErr(c, err)
		return
	}
	if !ok {
		c.JSON(http.StatusOK, gin.H{"installation": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"installation": inst})
}
