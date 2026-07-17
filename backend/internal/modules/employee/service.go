package employee

// Service is thin in Phase 2 — reads only. Sync-driven writes go through
// internal/modules/sync.Service, which owns both this repository and
// location's. Phase 3 will add employee CRUD here (create/deactivate by
// hand, assign services) alongside these reads.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByLocation(locationID string) ([]Employee, error) {
	return s.repo.ListByLocation(locationID)
}

func (s *Service) GetByID(id string) (Employee, error) {
	return s.repo.GetByID(id)
}
