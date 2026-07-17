package timeoff

import "errors"

var ErrInvalidTimeRange = errors.New("start_datetime must be before end_datetime")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByEmployee(employeeID string) ([]TimeOff, error) {
	return s.repo.ListByEmployee(employeeID)
}

func (s *Service) Create(employeeID string, in Input) (TimeOff, error) {
	if !in.StartDatetime.Before(in.EndDatetime) {
		return TimeOff{}, ErrInvalidTimeRange
	}
	return s.repo.Create(employeeID, in)
}

func (s *Service) Get(id string) (TimeOff, error) {
	return s.repo.Get(id)
}

func (s *Service) Delete(id string) error {
	return s.repo.Delete(id)
}
