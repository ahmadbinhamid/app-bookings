package sync

import (
	"context"
	"log"
	"time"

	"app-booking/internal/modules/installation"
)

// Scheduler re-syncs every installed tenant on a fixed interval.
//
// There is no existing cron/queue/scheduler convention anywhere in the
// sibling apps to mirror — checked appointments, quotes, ai-builder, and
// form_builder's backends, none of them have one. This is a plain
// time.Ticker goroutine started once from cmd/server/main.go: the simplest
// option, and it doesn't add a new dependency (a queue library) that
// nothing else in this codebase uses.
//
// Known limitation worth flagging: if this backend is ever run as more than
// one replica, each replica runs its own ticker, so a tenant would get
// synced once per replica per interval — harmless (sync is idempotent) but
// wasteful. If that becomes a real deployment shape, the fix is to move
// triggering onto external infra (e.g. a k8s CronJob or system cron hitting
// POST /api/v1/sync/trigger once) rather than an in-process ticker — no
// code changes needed here to switch, since the same Service.SyncTenant is
// what both paths call.
type Scheduler struct {
	installations *installation.Repository
	sync          *Service
	interval      time.Duration
}

func NewScheduler(installations *installation.Repository, sync *Service, interval time.Duration) *Scheduler {
	return &Scheduler{installations: installations, sync: sync, interval: interval}
}

// Start runs SyncAll once immediately, then every interval, until ctx is
// cancelled. Intended to be launched with `go scheduler.Start(ctx)` from
// main.go.
func (s *Scheduler) Start(ctx context.Context) {
	s.SyncAll(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SyncAll(ctx)
		}
	}
}

// SyncAll syncs every installed tenant, logging (not aborting on) a
// per-tenant failure so one broken installation's api_key can't block sync
// for everyone else.
func (s *Scheduler) SyncAll(ctx context.Context) {
	tenantIDs, err := s.installations.ListInstalledTenantIDs()
	if err != nil {
		log.Printf("sync scheduler: could not list installed tenants: %v", err)
		return
	}

	for _, tenantID := range tenantIDs {
		summary, err := s.sync.SyncTenant(ctx, tenantID)
		if err != nil {
			log.Printf("sync scheduler: tenant %d failed: %v", tenantID, err)
			continue
		}
		log.Printf("sync scheduler: tenant %d: mode=%s locations=%d employees=%d deactivated=%d",
			tenantID, summary.LocationMode, summary.LocationsSynced, summary.EmployeesSynced, summary.EmployeesDeactivated)
	}
}
