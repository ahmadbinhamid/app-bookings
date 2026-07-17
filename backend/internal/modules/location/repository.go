package location

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("location not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Upsert idempotently creates or updates a location keyed by
// (tenant_id, flowpos_location_id) — the unique index the migration added
// specifically so this can be a single statement, safe to call every sync
// run without creating duplicates.
//
// timezone here is always treated as sync-supplied, never confirmed: the
// SQL only writes it when timezone_confirmed is still FALSE (an admin
// hasn't set one yet) — once an admin calls SetTimezone, sync can never
// overwrite it again, per the design resolution ("a manual value always
// wins and is never overwritten by sync").
func (r *Repository) Upsert(tenantID uint64, flowposLocationID, name, timezone string) (Location, error) {
	if timezone == "" {
		timezone = DefaultTimezone
	}
	now := time.Now().UTC()
	id := uuid.NewString()

	_, err := r.db.Exec(`
		INSERT INTO locations (id, tenant_id, name, timezone, timezone_confirmed, flowpos_location_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, FALSE, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			timezone = IF(timezone_confirmed, timezone, VALUES(timezone)),
			updated_at = VALUES(updated_at)
	`, id, tenantID, name, timezone, flowposLocationID, now, now)
	if err != nil {
		return Location{}, err
	}
	return r.GetByTenantAndFlowposID(tenantID, flowposLocationID)
}

// SetTimezone is the ONLY way timezone_confirmed becomes TRUE — an explicit
// admin action (Phase 3 UI), validated by location.Service before this is
// called. Once set, Upsert above will never overwrite it again.
func (r *Repository) SetTimezone(id, timezone string) (Location, error) {
	res, err := r.db.Exec(`
		UPDATE locations SET timezone = ?, timezone_confirmed = TRUE, updated_at = ?
		WHERE id = ?
	`, timezone, time.Now().UTC(), id)
	if err != nil {
		return Location{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return Location{}, err
	}
	if affected == 0 {
		return Location{}, ErrNotFound
	}
	return r.GetByID(id)
}

const selectColumns = `id, tenant_id, name, timezone, timezone_confirmed, flowpos_location_id, created_at, updated_at`

func scanLocation(row interface {
	Scan(dest ...any) error
}) (Location, error) {
	var l Location
	err := row.Scan(&l.ID, &l.TenantID, &l.Name, &l.Timezone, &l.TimezoneConfirmed, &l.FlowposLocationID, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Location{}, ErrNotFound
	}
	if err != nil {
		return Location{}, err
	}
	return l, nil
}

func (r *Repository) GetByTenantAndFlowposID(tenantID uint64, flowposLocationID string) (Location, error) {
	row := r.db.QueryRow(`SELECT `+selectColumns+` FROM locations WHERE tenant_id = ? AND flowpos_location_id = ?`, tenantID, flowposLocationID)
	return scanLocation(row)
}

func (r *Repository) GetByID(id string) (Location, error) {
	row := r.db.QueryRow(`SELECT `+selectColumns+` FROM locations WHERE id = ?`, id)
	return scanLocation(row)
}

func (r *Repository) ListByTenant(tenantID uint64) ([]Location, error) {
	rows, err := r.db.Query(`SELECT `+selectColumns+` FROM locations WHERE tenant_id = ? ORDER BY created_at`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Location, 0)
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.TenantID, &l.Name, &l.Timezone, &l.TimezoneConfirmed, &l.FlowposLocationID, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
