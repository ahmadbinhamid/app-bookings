package schedules

import (
	"errors"
	"time"
)

var (
	ErrInvalidTimeRange = errors.New("start_time must be before end_time")
	// ErrOverlap is deliberately per-employee-per-day, not a DB constraint —
	// see this package's doc comment and the design doc's Constraints table:
	// split shifts (multiple non-overlapping rows on the same day) must
	// stay valid, so this can't be a simple uniqueness rule.
	ErrOverlap = errors.New("this time range overlaps an existing schedule entry for this employee on this day")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByEmployee(employeeID string) ([]Schedule, error) {
	return s.repo.ListByEmployee(employeeID)
}

// Create validates start < end, then checks the new range against every
// existing schedule row for this employee on the same day — two ranges
// overlap iff each starts before the other ends. A non-overlapping second
// row on the same day (a split shift) is accepted, not rejected.
func (s *Service) Create(employeeID string, in Input) (Schedule, error) {
	start, err := parseTimeOfDay(in.StartTime)
	if err != nil {
		return Schedule{}, ErrInvalidTimeRange
	}
	end, err := parseTimeOfDay(in.EndTime)
	if err != nil {
		return Schedule{}, ErrInvalidTimeRange
	}
	if !start.Before(end) {
		return Schedule{}, ErrInvalidTimeRange
	}

	existing, err := s.repo.ListByEmployeeAndDay(employeeID, in.DayOfWeek)
	if err != nil {
		return Schedule{}, err
	}
	for _, ex := range existing {
		exStart, err := parseTimeOfDay(ex.StartTime)
		if err != nil {
			continue
		}
		exEnd, err := parseTimeOfDay(ex.EndTime)
		if err != nil {
			continue
		}
		if start.Before(exEnd) && exStart.Before(end) {
			return Schedule{}, ErrOverlap
		}
	}

	return s.repo.Create(employeeID, in)
}

func (s *Service) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *Service) Get(id string) (Schedule, error) {
	return s.repo.Get(id)
}

// parseTimeOfDay accepts "HH:MM:SS" or "HH:MM" (admins typing a time in a
// form rarely include seconds) and returns a same-day time.Time purely so
// Before() can compare two wall-clock times — the date component is never
// used or persisted.
func parseTimeOfDay(s string) (time.Time, error) {
	if t, err := time.Parse("15:04:05", s); err == nil {
		return t, nil
	}
	return time.Parse("15:04", s)
}
