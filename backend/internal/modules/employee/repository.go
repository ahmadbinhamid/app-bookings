package employee

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("employee not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Upsert idempotently creates or updates an employee keyed by
// (tenant_id, flowpos_employee_id) — safe to call every sync run. A newly
// created row starts with location_id = NULL (unassigned); an existing row's
// location_id is deliberately left untouched by ON DUPLICATE KEY UPDATE, so
// re-syncing never clobbers an admin's manual location assignment. Every
// upsert also reactivates the row (active=TRUE) and stamps synced_at, since
// being upserted means FlowPOS returned this employee in the current sync.
func (r *Repository) Upsert(tenantID uint64, flowposEmployeeID, name, email, phone string) (Employee, error) {
	now := time.Now().UTC()
	id := uuid.NewString()

	_, err := r.db.Exec(`
		INSERT INTO employees (id, tenant_id, location_id, flowpos_employee_id, name, email, phone, active, synced_at, created_at, updated_at)
		VALUES (?, ?, NULL, ?, ?, ?, ?, TRUE, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			email = VALUES(email),
			phone = VALUES(phone),
			active = TRUE,
			synced_at = VALUES(synced_at),
			updated_at = VALUES(updated_at)
	`, id, tenantID, flowposEmployeeID, name, nullable(email), nullable(phone), now, now, now)
	if err != nil {
		return Employee{}, err
	}
	return r.GetByTenantAndFlowposID(tenantID, flowposEmployeeID)
}

// DeactivateMissing sets active=FALSE for every currently-active employee of
// tenantID whose flowpos_employee_id was NOT in the current sync's result set
// (seenFlowposIDs) — this is what makes staff removed from FlowPOS disappear
// from scheduling without ever deleting their row. Returns the number of rows
// deactivated. An empty seenFlowposIDs deactivates every active employee for
// the tenant (a sync that legitimately found zero employees).
func (r *Repository) DeactivateMissing(tenantID uint64, seenFlowposIDs []string) (int64, error) {
	now := time.Now().UTC()

	if len(seenFlowposIDs) == 0 {
		res, err := r.db.Exec(`
			UPDATE employees SET active = FALSE, updated_at = ?
			WHERE tenant_id = ? AND active = TRUE
		`, now, tenantID)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(seenFlowposIDs)), ",")
	args := make([]any, 0, len(seenFlowposIDs)+2)
	args = append(args, now, tenantID)
	for _, id := range seenFlowposIDs {
		args = append(args, id)
	}

	res, err := r.db.Exec(`
		UPDATE employees SET active = FALSE, updated_at = ?
		WHERE tenant_id = ? AND active = TRUE AND flowpos_employee_id NOT IN (`+placeholders+`)
	`, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

const selectColumns = `id, tenant_id, location_id, flowpos_employee_id, name, email, phone, active, synced_at, created_at, updated_at`

func scanEmployee(row interface{ Scan(...any) error }) (Employee, error) {
	var e Employee
	var locationID, email, phone sql.NullString
	err := row.Scan(&e.ID, &e.TenantID, &locationID, &e.FlowposEmployeeID, &e.Name, &email, &phone, &e.Active, &e.SyncedAt, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Employee{}, ErrNotFound
	}
	if err != nil {
		return Employee{}, err
	}
	if locationID.Valid {
		e.LocationID = &locationID.String
	}
	e.Email = email.String
	e.Phone = phone.String
	return e, nil
}

func (r *Repository) GetByTenantAndFlowposID(tenantID uint64, flowposEmployeeID string) (Employee, error) {
	row := r.db.QueryRow(`SELECT `+selectColumns+` FROM employees WHERE tenant_id = ? AND flowpos_employee_id = ?`, tenantID, flowposEmployeeID)
	return scanEmployee(row)
}

func (r *Repository) GetByID(id string) (Employee, error) {
	row := r.db.QueryRow(`SELECT `+selectColumns+` FROM employees WHERE id = ?`, id)
	return scanEmployee(row)
}

func (r *Repository) ListByLocation(locationID string) ([]Employee, error) {
	rows, err := r.db.Query(`SELECT `+selectColumns+` FROM employees WHERE location_id = ? ORDER BY name`, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEmployees(rows)
}

// ListByTenant returns every employee synced for tenantID, assigned or not —
// the tenant-wide roster view (see internal/server/handlers/employees.go
// ListAll), as opposed to ListByLocation's "who works here" or
// ListUnassigned's "who's available to assign."
func (r *Repository) ListByTenant(tenantID uint64) ([]Employee, error) {
	rows, err := r.db.Query(`SELECT `+selectColumns+` FROM employees WHERE tenant_id = ? ORDER BY name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEmployees(rows)
}

// ListUnassigned returns every active employee of tenantID not yet assigned
// to any location — the pool an admin picks from when assigning employees to
// a location (see internal/server/handlers/employees.go ListUnassigned).
func (r *Repository) ListUnassigned(tenantID uint64) ([]Employee, error) {
	rows, err := r.db.Query(`
		SELECT `+selectColumns+` FROM employees
		WHERE tenant_id = ? AND location_id IS NULL AND active = TRUE
		ORDER BY name
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEmployees(rows)
}

// AssignLocation sets (or changes) which single location an employee belongs
// to. Business-rule enforcement (blocking a move while future bookings
// exist) lives in Service.AssignLocation, not here — this is a plain write.
func (r *Repository) AssignLocation(employeeID, locationID string) error {
	_, err := r.db.Exec(`UPDATE employees SET location_id = ?, updated_at = ? WHERE id = ?`, locationID, time.Now().UTC(), employeeID)
	return err
}

func scanEmployees(rows *sql.Rows) ([]Employee, error) {
	out := make([]Employee, 0)
	for rows.Next() {
		e, err := scanEmployee(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullable(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
