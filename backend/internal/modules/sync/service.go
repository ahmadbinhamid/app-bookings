// Package sync orchestrates pulling locations and employees from FlowPOS
// for a tenant and upserting them locally. It depends on both the location
// and employee repositories directly rather than going through their
// services, because this cross-cutting orchestration doesn't belong to
// either module individually — see location.Service's doc comment.
package sync

import (
	"context"
	"errors"
	"fmt"
	"log"

	"app-booking/internal/flowpos"
	"app-booking/internal/modules/employee"
	"app-booking/internal/modules/installation"
	"app-booking/internal/modules/location"
)

type Service struct {
	locations     *location.Repository
	employees     *employee.Repository
	installations *installation.Repository
	flowposClient *flowpos.Client
}

func NewService(locations *location.Repository, employees *employee.Repository, installations *installation.Repository, flowposClient *flowpos.Client) *Service {
	return &Service{
		locations:     locations,
		employees:     employees,
		installations: installations,
		flowposClient: flowposClient,
	}
}

// Summary reports what a single SyncTenant call actually did, returned to
// both the manual-trigger HTTP handler and the background scheduler's logs.
type Summary struct {
	TenantID             uint64 `json:"tenant_id"`
	LocationMode         string `json:"location_mode"` // "flowpos" or "fallback_single_location"
	LocationsSynced      int    `json:"locations_synced"`
	EmployeesSynced      int    `json:"employees_synced"`
	EmployeesDeactivated int64  `json:"employees_deactivated"`
	// LocationErrors holds one entry per location whose employee sync
	// failed — the whole tenant sync still completes for the other
	// locations rather than aborting outright (design doc's Phase 2 tests
	// call for "skip and log," not "crash").
	LocationErrors []string `json:"location_errors,omitempty"`
}

// syncedLocation pairs a local location row with the FlowPOS-side id to use
// when fetching its employees ("" in fallback mode, meaning "no location
// filter — fetch every employee").
type syncedLocation struct {
	local             location.Location
	flowposLocationID string
}

// SyncTenant fetches the tenant's location list from FlowPOS, upserts each
// into `locations`, then syncs employees per location — idempotent upsert
// by (location_id, flowpos_employee_id), and deactivates (never deletes)
// any employee no longer returned. Resolved per the design discussion:
// multiple locations are the norm; a single-store tenant is just the
// one-item case, no special-casing. Falls back to exactly one location per
// tenant only if FlowPOS's /locations endpoint doesn't exist at all
// (ErrEndpointNotFound) — any other error is a real failure and is returned
// as-is, never silently treated as "must be single-location mode."
func (s *Service) SyncTenant(ctx context.Context, tenantID uint64) (Summary, error) {
	inst, err := s.installations.GetByTenantID(tenantID)
	if err != nil {
		return Summary{}, fmt.Errorf("sync tenant %d: %w", tenantID, err)
	}

	synced, mode, err := s.resolveLocations(ctx, tenantID, inst.APIKey)
	if err != nil {
		return Summary{}, fmt.Errorf("sync tenant %d: resolve locations: %w", tenantID, err)
	}

	summary := Summary{TenantID: tenantID, LocationMode: mode, LocationsSynced: len(synced)}

	for _, sl := range synced {
		employeesSynced, deactivated, err := s.syncEmployeesForLocation(ctx, sl, inst.APIKey)
		if err != nil {
			msg := fmt.Sprintf("location %s (flowpos id %q): %v", sl.local.ID, sl.flowposLocationID, err)
			log.Printf("sync tenant %d: %s", tenantID, msg)
			summary.LocationErrors = append(summary.LocationErrors, msg)
			continue
		}
		summary.EmployeesSynced += employeesSynced
		summary.EmployeesDeactivated += deactivated
	}

	return summary, nil
}

func (s *Service) resolveLocations(ctx context.Context, tenantID uint64, apiKey string) ([]syncedLocation, string, error) {
	flowposLocations, err := s.flowposClient.ListLocations(ctx, apiKey)
	switch {
	case errors.Is(err, flowpos.ErrEndpointNotFound):
		// This FlowPOS install exposes no locations list at all — fall back
		// to exactly one location per tenant (design resolution). Upsert is
		// idempotent, so re-running this on every sync is safe and won't
		// create duplicates.
		def, err := s.locations.Upsert(tenantID, location.DefaultFlowposLocationID, "Default Location", location.DefaultTimezone)
		if err != nil {
			return nil, "", fmt.Errorf("ensure default location: %w", err)
		}
		return []syncedLocation{{local: def, flowposLocationID: ""}}, "fallback_single_location", nil
	case err != nil:
		return nil, "", err
	}

	out := make([]syncedLocation, 0, len(flowposLocations))
	for _, fl := range flowposLocations {
		local, err := s.locations.Upsert(tenantID, fl.FlowposID, fl.Name, fl.Timezone)
		if err != nil {
			return nil, "", fmt.Errorf("upsert location %q: %w", fl.FlowposID, err)
		}
		out = append(out, syncedLocation{local: local, flowposLocationID: fl.FlowposID})
	}
	return out, "flowpos", nil
}

func (s *Service) syncEmployeesForLocation(ctx context.Context, sl syncedLocation, apiKey string) (synced int, deactivated int64, err error) {
	// FlowPOS has no employee-to-location scoping (confirmed against
	// flowpos-backend's EmployeeController), so every location gets the
	// tenant's full employee list upserted under its own location_id.
	flowposEmployees, err := s.flowposClient.ListEmployees(ctx, apiKey)
	if err != nil {
		return 0, 0, err
	}

	seenIDs := make([]string, 0, len(flowposEmployees))
	for _, fe := range flowposEmployees {
		// A row with no id or name isn't a real employee we can safely
		// upsert (our composite unique key is (location_id,
		// flowpos_employee_id) — an empty id would collide across every
		// malformed row). Skip and log rather than guess or crash, per the
		// design doc's Phase 2 test requirement.
		if fe.FlowposID == "" || fe.Name == "" {
			log.Printf("sync: skipping malformed employee record at location %s: %+v", sl.local.ID, fe)
			continue
		}
		if _, err := s.employees.Upsert(sl.local.ID, fe.FlowposID, fe.Name, fe.Email, fe.Phone); err != nil {
			return synced, 0, fmt.Errorf("upsert employee %q: %w", fe.FlowposID, err)
		}
		seenIDs = append(seenIDs, fe.FlowposID)
		synced++
	}

	deactivated, err = s.employees.DeactivateMissing(sl.local.ID, seenIDs)
	if err != nil {
		return synced, 0, fmt.Errorf("deactivate missing employees: %w", err)
	}
	return synced, deactivated, nil
}
