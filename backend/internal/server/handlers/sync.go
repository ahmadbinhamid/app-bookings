package handlers

import (
	"net/http"

	"app-booking/internal/modules/sync"

	"github.com/gin-gonic/gin"
)

// SyncHandler is the controller layer only: parse the (tenant-scoped, no
// body) request, call the service, shape the response. All the actual sync
// logic lives in sync.Service — see its doc comment.
type SyncHandler struct {
	sync *sync.Service
}

func NewSyncHandler(syncSvc *sync.Service) *SyncHandler {
	return &SyncHandler{sync: syncSvc}
}

// Trigger is the manual on-demand sync endpoint (design doc Phase 2: "hourly
// job + manual trigger"). Synchronous by design — there's no job queue in
// this codebase to hand it off to (see sync.Scheduler's doc comment), and a
// location+employee sync is fast enough to run inline within the request's
// timeout.
func (h *SyncHandler) Trigger(c *gin.Context) {
	summary, err := h.sync.SyncTenant(c.Request.Context(), tenantID(c))
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": summary})
}
