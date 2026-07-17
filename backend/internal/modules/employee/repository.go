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
// (location_id, flowpos_employee_id) — safe to call every sync run. Every
// upsert also reactivates the row (active=TRUE) and stamps synced_at, since
// being upserted means FlowPOS returned this employee in the current sync.
func (r *Repository) Upsert(locationID, flowposEmployeeID, name, email, phone string) (Employee, error) {
	now := time.Now().UTC()
	id := uuid.NewString()

	_, err := r.db.Exec(`
		INSERT INTO employees (id, location_id, flowpos_employee_id, name, email, phone, active, synced_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			email = VALUES(email),
			phone = VALUES(phone),
			active = TRUE,
			synced_at = VALUES(synced_at),
			updated_at = VALUES(updated_at)
	`, id, locationID, flowposEmployeeID, name, nullable(email), nullable(phone), now, now, now)
	if err != nil {
		return Employee{}, err
	}
	return r.GetByLocationAndFlowposID(locationID, flowposEmployeeID)
}

// DeactivateMissing sets active=FALSE for every currently-active employee at
// locationID whose flowpos_employee_id was NOT in the current sync's result
// set (seenFlowposIDs) — this is what makes staff removed from FlowPOS
// disappear from scheduling without ever deleting their row (design doc,
// Decisions table: FlowPOS sync). Returns the number of rows deactivated.
// An empty seenFlowposIDs deactivates every active employee at the location
// (a sync that legitimately found zero employees).
func (r *Repository) DeactivateMissing(locationID string, seenFlowposIDs []string) (int64, error) {
	now := time.Now().UTC()

	if len(seenFlowposIDs) == 0 {
		res, err := r.db.Exec(`
			UPDATE employees SET active = FALSE, updated_at = ?
			WHERE location_id = ? AND active = TRUE
		`, now, locationID)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(seenFlowposIDs)), ",")
	args := make([]any, 0, len(seenFlowposIDs)+2)
	args = append(args, now, locationID)
	for _, id := range seenFlowposIDs {
		args = append(args, id)
	}

	res, err := r.db.Exec(`
		UPDATE employees SET active = FALSE, updated_at = ?
		WHERE location_id = ? AND active = TRUE AND flowpos_employee_id NOT IN (`+placeholders+`)
	`, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *Repository) GetByLocationAndFlowposID(locationID, flowposEmployeeID string) (Employee, error) {
	var e Employee
	var email, phone sql.NullString
	err := r.db.QueryRow(`
		SELECT id, location_id, flowpos_employee_id, name, email, phone, active, synced_at, created_at, updated_at
		FROM employees
		WHERE location_id = ? AND flowpos_employee_id = ?
	`, locationID, flowposEmployeeID).Scan(&e.ID, &e.LocationID, &e.FlowposEmployeeID, &e.Name, &email, &phone, &e.Active, &e.SyncedAt, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Employee{}, ErrNotFound
	}
	if err != nil {
		return Employee{}, err
	}
	e.Email = email.String
	e.Phone = phone.String
	return e, nil
}

func (r *Repository) GetByID(id string) (Employee, error) {
	var e Employee
	var email, phone sql.NullString
	err := r.db.QueryRow(`
		SELECT id, location_id, flowpos_employee_id, name, email, phone, active, synced_at, created_at, updated_at
		FROM employees
		WHERE id = ?
	`, id).Scan(&e.ID, &e.LocationID, &e.FlowposEmployeeID, &e.Name, &email, &phone, &e.Active, &e.SyncedAt, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Employee{}, ErrNotFound
	}
	if err != nil {
		return Employee{}, err
	}
	e.Email = email.String
	e.Phone = phone.String
	return e, nil
}

func (r *Repository) ListByLocation(locationID string) ([]Employee, error) {
	rows, err := r.db.Query(`
		SELECT id, location_id, flowpos_employee_id, name, email, phone, active, synced_at, created_at, updated_at
		FROM employees
		WHERE location_id = ?
		ORDER BY name
	`, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Employee, 0)
	for rows.Next() {
		var e Employee
		var email, phone sql.NullString
		if err := rows.Scan(&e.ID, &e.LocationID, &e.FlowposEmployeeID, &e.Name, &email, &phone, &e.Active, &e.SyncedAt, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.Email = email.String
		e.Phone = phone.String
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullable(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
