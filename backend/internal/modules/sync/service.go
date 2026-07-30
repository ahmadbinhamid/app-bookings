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
}

// SyncTenant fetches the tenant's location list from FlowPOS and upserts each
// into `locations`, then separately syncs the tenant's employees exactly
// once. Locations and employees are two independent, unrelated FlowPOS lists
// — FlowPOS has no employee-location relationship at all (confirmed against
// flowpos-backend's EmployeeController, see
// internal/flowpos/employees_unconfirmed.go's header) — so unlike locations,
// employees are NOT synced once per location; doing so would upsert the same
// tenant-wide list under every location and make "one employee, one
// location" impossible to express. Which single location an employee
// belongs to is an app-bookings-only concept an admin sets by hand — see
// internal/modules/employee.Service.AssignLocation.
func (s *Service) SyncTenant(ctx context.Context, tenantID uint64) (Summary, error) {
	inst, err := s.installations.GetByTenantID(tenantID)
	if err != nil {
		return Summary{}, fmt.Errorf("sync tenant %d: %w", tenantID, err)
	}

	locationsSynced, mode, err := s.syncLocations(ctx, tenantID, inst.APIKey)
	if err != nil {
		return Summary{}, fmt.Errorf("sync tenant %d: sync locations: %w", tenantID, err)
	}

	employeesSynced, deactivated, err := s.syncEmployees(ctx, tenantID, inst.APIKey)
	if err != nil {
		return Summary{}, fmt.Errorf("sync tenant %d: sync employees: %w", tenantID, err)
	}

	return Summary{
		TenantID:             tenantID,
		LocationMode:         mode,
		LocationsSynced:      locationsSynced,
		EmployeesSynced:      employeesSynced,
		EmployeesDeactivated: deactivated,
	}, nil
}

// syncLocations upserts every location FlowPOS returns for the tenant,
// falling back to exactly one location per tenant only if FlowPOS's
// /locations endpoint doesn't exist at all (ErrEndpointNotFound) — any other
// error is a real failure and is returned as-is, never silently treated as
// "must be single-location mode."
func (s *Service) syncLocations(ctx context.Context, tenantID uint64, apiKey string) (int, string, error) {
	flowposLocations, err := s.flowposClient.ListLocations(ctx, apiKey)
	switch {
	case errors.Is(err, flowpos.ErrEndpointNotFound):
		// This FlowPOS install exposes no locations list at all — fall back
		// to exactly one location per tenant (design resolution). Upsert is
		// idempotent, so re-running this on every sync is safe and won't
		// create duplicates.
		if _, err := s.locations.Upsert(tenantID, location.DefaultFlowposLocationID, "Default Location", location.DefaultTimezone); err != nil {
			return 0, "", fmt.Errorf("ensure default location: %w", err)
		}
		return 1, "fallback_single_location", nil
	case err != nil:
		return 0, "", err
	}

	for _, fl := range flowposLocations {
		// fl.Timezone may be empty if FlowPOS's payload doesn't supply one —
		// Upsert already falls back to DefaultTimezone ("UTC") in that case,
		// and never overwrites a timezone an admin has since confirmed.
		if _, err := s.locations.Upsert(tenantID, fl.FlowposID, fl.Name, fl.Timezone); err != nil {
			return 0, "", fmt.Errorf("upsert location %q: %w", fl.FlowposID, err)
		}
	}
	return len(flowposLocations), "flowpos", nil
}

// syncEmployees fetches the tenant's complete employee list from FlowPOS
// once — there is no location to scope by — and upserts it, idempotent by
// (tenant_id, flowpos_employee_id), deactivating (never deleting) any
// employee no longer returned. A malformed row (no id or name) is skipped
// and logged rather than guessed at or allowed to fail the whole sync.
func (s *Service) syncEmployees(ctx context.Context, tenantID uint64, apiKey string) (synced int, deactivated int64, err error) {
	flowposEmployees, err := s.flowposClient.ListEmployees(ctx, apiKey)
	if err != nil {
		return 0, 0, err
	}

	seenIDs := make([]string, 0, len(flowposEmployees))
	for _, fe := range flowposEmployees {
		if fe.FlowposID == "" || fe.Name == "" {
			log.Printf("sync: skipping malformed employee record for tenant %d: %+v", tenantID, fe)
			continue
		}
		if _, err := s.employees.Upsert(tenantID, fe.FlowposID, fe.Name, fe.Email, fe.Phone); err != nil {
			return synced, 0, fmt.Errorf("upsert employee %q: %w", fe.FlowposID, err)
		}
		seenIDs = append(seenIDs, fe.FlowposID)
		synced++
	}

	deactivated, err = s.employees.DeactivateMissing(tenantID, seenIDs)
	if err != nil {
		return synced, 0, fmt.Errorf("deactivate missing employees: %w", err)
	}
	return synced, deactivated, nil
}
