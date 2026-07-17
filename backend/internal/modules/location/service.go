package location

import (
	"errors"
	"time"
)

// ErrInvalidTimezone is returned when SetTimezone is given a string that
// isn't a real IANA zone name (validated against the tz database via
// time.LoadLocation — see cmd/server/main.go's blank time/tzdata import for
// why that works reliably in the minimal Alpine runtime image).
var ErrInvalidTimezone = errors.New("timezone must be a valid IANA zone name (e.g. Europe/London)")

// Service is deliberately thin for reads — it only exposes reads plus
// SetTimezone. The FlowPOS sync logic that actually populates this table
// lives in internal/modules/sync, per the layering rule that a "module"
// isn't automatically where cross-cutting orchestration goes: sync depends
// on both this repository and the employee one, so it owns that
// orchestration itself rather than either module reaching into the other.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByTenant(tenantID uint64) ([]Location, error) {
	return s.repo.ListByTenant(tenantID)
}

func (s *Service) GetByID(id string) (Location, error) {
	return s.repo.GetByID(id)
}

// SetTimezone is the only path to timezone_confirmed=true (design
// resolution: "timezone is a manually-managed field"). Validates against
// the real tz database rather than just checking the string looks
// plausible, so a typo like "Europe/Lundon" is rejected here instead of
// silently breaking the Phase 4 solver's schedule-to-instant conversion.
func (s *Service) SetTimezone(id, timezone string) (Location, error) {
	if _, err := time.LoadLocation(timezone); err != nil {
		return Location{}, ErrInvalidTimezone
	}
	return s.repo.SetTimezone(id, timezone)
}
