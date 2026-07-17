package services

import "app-booking/internal/config/pagination"

// MyService is named MyService, not Service, to avoid colliding with the
// domain model type above (also called Service) — see this package's doc
// comment in model.go.
type MyService struct {
	repo *Repository
}

func NewService(repo *Repository) *MyService {
	return &MyService{repo: repo}
}

func (s *MyService) List(locationID string, p pagination.Params, search string) ([]Service, int, error) {
	return s.repo.List(locationID, p, search)
}

func (s *MyService) Get(id string) (Service, error) {
	return s.repo.Get(id)
}

func (s *MyService) Create(locationID string, in Input) (Service, error) {
	return s.repo.Create(locationID, in)
}

func (s *MyService) Update(id string, in Input) (Service, error) {
	return s.repo.Update(id, in)
}

func (s *MyService) Delete(id string) error {
	return s.repo.Delete(id)
}
