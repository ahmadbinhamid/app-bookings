package installation

import "errors"

// Service holds the marketplace install/uninstall business rules and
// delegates persistence to the repository.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Install provisions the tenant's installation on first install, or
// re-activates an existing one — idempotent, since FlowPOS may call it more
// than once for the same tenant (e.g. reinstalling after an uninstall).
func (s *Service) Install(tenantID uint64, apiKey string) (Installation, error) {
	_, err := s.repo.GetByTenantID(tenantID)
	switch {
	case errors.Is(err, ErrNotFound):
		return s.repo.Create(tenantID, apiKey)
	case err != nil:
		return Installation{}, err
	}
	if err := s.repo.SetInstalled(tenantID, true, apiKey); err != nil {
		return Installation{}, err
	}
	return s.repo.GetByTenantID(tenantID)
}

// Uninstall soft-disables the tenant's installation without deleting any
// data — Install re-activates it. A no-op if the tenant was never installed.
func (s *Service) Uninstall(tenantID uint64) error {
	err := s.repo.SetInstalled(tenantID, false, "")
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// GetByTenant returns the tenant's installation. ok is false when the tenant
// has never installed the app (shouldn't normally happen once the dashboard
// only embeds this app post-install, but kept defensive).
func (s *Service) GetByTenant(tenantID uint64) (Installation, bool, error) {
	in, err := s.repo.GetByTenantID(tenantID)
	if errors.Is(err, ErrNotFound) {
		return Installation{}, false, nil
	}
	if err != nil {
		return Installation{}, false, err
	}
	return in, true, nil
}
