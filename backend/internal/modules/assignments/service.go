package assignments

// Service is thin — deliberately no "does this employee/service pair belong
// to the same location" check here. Assignments are only ever created via
// routes nested under one /locations/:locationId (see server.go), where
// RequireEmployeeInLocation and RequireServiceInLocation have both already
// verified employeeID and serviceID belong to that same location before
// this is ever called. Re-checking here would just be the same query twice.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Assign(employeeID, serviceID string) (Assignment, error) {
	return s.repo.Create(employeeID, serviceID)
}

func (s *Service) Unassign(employeeID, serviceID string) error {
	return s.repo.Delete(employeeID, serviceID)
}

func (s *Service) ListEmployeeIDsForService(serviceID string) ([]string, error) {
	return s.repo.ListEmployeeIDsForService(serviceID)
}

func (s *Service) ListServiceIDsForEmployee(employeeID string) ([]string, error) {
	return s.repo.ListServiceIDsForEmployee(employeeID)
}
