package timeoff

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("time off not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(employeeID string, in Input) (TimeOff, error) {
	t := TimeOff{
		ID:            uuid.NewString(),
		EmployeeID:    employeeID,
		StartDatetime: in.StartDatetime,
		EndDatetime:   in.EndDatetime,
		Reason:        in.Reason,
		CreatedAt:     time.Now().UTC(),
	}
	_, err := r.db.Exec(`
		INSERT INTO employee_time_off (id, employee_id, start_datetime, end_datetime, reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, t.ID, t.EmployeeID, t.StartDatetime, t.EndDatetime, t.Reason, t.CreatedAt)
	if err != nil {
		return TimeOff{}, err
	}
	return t, nil
}

func (r *Repository) ListByEmployee(employeeID string) ([]TimeOff, error) {
	rows, err := r.db.Query(`
		SELECT id, employee_id, start_datetime, end_datetime, reason, created_at
		FROM employee_time_off WHERE employee_id = ? ORDER BY start_datetime
	`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TimeOff, 0)
	for rows.Next() {
		var t TimeOff
		if err := rows.Scan(&t.ID, &t.EmployeeID, &t.StartDatetime, &t.EndDatetime, &t.Reason, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) Get(id string) (TimeOff, error) {
	var t TimeOff
	err := r.db.QueryRow(`
		SELECT id, employee_id, start_datetime, end_datetime, reason, created_at
		FROM employee_time_off WHERE id = ?
	`, id).Scan(&t.ID, &t.EmployeeID, &t.StartDatetime, &t.EndDatetime, &t.Reason, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return TimeOff{}, ErrNotFound
	}
	if err != nil {
		return TimeOff{}, err
	}
	return t, nil
}

func (r *Repository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM employee_time_off WHERE id = ?`, id)
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
