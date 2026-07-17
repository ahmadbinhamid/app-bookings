package installation

import (
	"database/sql"
	"errors"
	"time"
)

// ErrNotFound is returned when a tenant has no installation row.
var ErrNotFound = errors.New("installation not found")

// Repository is the data-access layer for installations.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByTenantID(tenantID uint64) (Installation, error) {
	var in Installation
	err := r.db.QueryRow(`
		SELECT id, tenant_id, api_key, installed, created_at, updated_at
		FROM installations
		WHERE tenant_id = ?
	`, tenantID).Scan(&in.ID, &in.TenantID, &in.APIKey, &in.Installed, &in.CreatedAt, &in.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Installation{}, ErrNotFound
	}
	if err != nil {
		return Installation{}, err
	}
	return in, nil
}

func (r *Repository) Create(tenantID uint64, apiKey string) (Installation, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`
		INSERT INTO installations (tenant_id, api_key, installed, created_at, updated_at)
		VALUES (?, ?, TRUE, ?, ?)
	`, tenantID, apiKey, now, now)
	if err != nil {
		return Installation{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Installation{}, err
	}
	return Installation{
		ID:        uint64(id),
		TenantID:  tenantID,
		APIKey:    apiKey,
		Installed: true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// SetInstalled flips the installed flag for the tenant, refreshing api_key
// too when a non-empty one is given. Returns ErrNotFound if the tenant has no
// installation row.
func (r *Repository) SetInstalled(tenantID uint64, installed bool, apiKey string) error {
	now := time.Now().UTC()
	var res sql.Result
	var err error
	if apiKey != "" {
		res, err = r.db.Exec(`
			UPDATE installations SET installed = ?, api_key = ?, updated_at = ?
			WHERE tenant_id = ?
		`, installed, apiKey, now, tenantID)
	} else {
		res, err = r.db.Exec(`
			UPDATE installations SET installed = ?, updated_at = ?
			WHERE tenant_id = ?
		`, installed, now, tenantID)
	}
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
