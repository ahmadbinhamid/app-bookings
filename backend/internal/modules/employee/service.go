package employee

import "time"

// BookingConflictChecker is the narrow slice of booking.Repository this
// package needs — accepted as an interface (rather than importing
// internal/modules/booking directly) so employee stays a leaf dependency,
// same shape sync.Service already uses for its own cross-module repo access.
type BookingConflictChecker interface {
	HasFutureBookingsForEmployee(employeeID string, after time.Time) (bool, error)
}

// Service is thin in Phase 2 — reads only, plus the one hand-managed write
// FlowPOS can't give us: location assignment (see model.go's doc comment on
// LocationID for why this can't come from sync).
type Service struct {
	repo     *Repository
	bookings BookingConflictChecker
}

func NewService(repo *Repository, bookings BookingConflictChecker) *Service {
	return &Service{repo: repo, bookings: bookings}
}

func (s *Service) ListByLocation(locationID string) ([]Employee, error) {
	return s.repo.ListByLocation(locationID)
}

func (s *Service) ListByTenant(tenantID uint64) ([]Employee, error) {
	return s.repo.ListByTenant(tenantID)
}

func (s *Service) ListUnassigned(tenantID uint64) ([]Employee, error) {
	return s.repo.ListUnassigned(tenantID)
}

func (s *Service) GetByID(id string) (Employee, error) {
	return s.repo.GetByID(id)
}

// AssignLocation assigns an employee to exactly one location. If they're
// already assigned somewhere else, the move is blocked while they still have
// upcoming (booked, non-cancelled) bookings at the current location — the
// admin must cancel/reschedule those first rather than this silently
// orphaning them. Assigning a not-yet-assigned employee (LocationID == nil)
// never needs this check, since there's nothing to orphan.
func (s *Service) AssignLocation(employeeID, targetLocationID string) (Employee, error) {
	emp, err := s.repo.GetByID(employeeID)
	if err != nil {
		return Employee{}, err
	}

	if emp.LocationID != nil && *emp.LocationID != targetLocationID {
		hasFuture, err := s.bookings.HasFutureBookingsForEmployee(employeeID, time.Now().UTC())
		if err != nil {
			return Employee{}, err
		}
		if hasFuture {
			return Employee{}, ErrHasFutureBookings
		}
	}

	if err := s.repo.AssignLocation(employeeID, targetLocationID); err != nil {
		return Employee{}, err
	}
	return s.repo.GetByID(employeeID)
}
