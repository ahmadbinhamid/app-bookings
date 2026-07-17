package schedules

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("schedule not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(employeeID string, in Input) (Schedule, error) {
	now := time.Now().UTC()
	s := Schedule{
		ID:         uuid.NewString(),
		EmployeeID: employeeID,
		DayOfWeek:  in.DayOfWeek,
		StartTime:  in.StartTime,
		EndTime:    in.EndTime,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err := r.db.Exec(`
		INSERT INTO employee_schedules (id, employee_id, day_of_week, start_time, end_time, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.EmployeeID, s.DayOfWeek, s.StartTime, s.EndTime, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return Schedule{}, err
	}
	return s, nil
}

func (r *Repository) ListByEmployee(employeeID string) ([]Schedule, error) {
	return r.query(`SELECT id, employee_id, day_of_week, start_time, end_time, created_at, updated_at
		FROM employee_schedules WHERE employee_id = ? ORDER BY day_of_week, start_time`, employeeID)
}

// ListByEmployeeAndDay backs the overlap check in service.go — deliberately
// a plain SELECT the Go layer compares in memory (see Service.Create),
// not a DB constraint, per the design doc's explicit decision that split
// shifts must remain possible.
func (r *Repository) ListByEmployeeAndDay(employeeID string, dayOfWeek int) ([]Schedule, error) {
	return r.query(`SELECT id, employee_id, day_of_week, start_time, end_time, created_at, updated_at
		FROM employee_schedules WHERE employee_id = ? AND day_of_week = ?`, employeeID, dayOfWeek)
}

func (r *Repository) query(query string, args ...any) ([]Schedule, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Schedule, 0)
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(&s.ID, &s.EmployeeID, &s.DayOfWeek, &s.StartTime, &s.EndTime, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM employee_schedules WHERE id = ?`, id)
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

func (r *Repository) Get(id string) (Schedule, error) {
	var s Schedule
	err := r.db.QueryRow(`
		SELECT id, employee_id, day_of_week, start_time, end_time, created_at, updated_at
		FROM employee_schedules WHERE id = ?
	`, id).Scan(&s.ID, &s.EmployeeID, &s.DayOfWeek, &s.StartTime, &s.EndTime, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Schedule{}, ErrNotFound
	}
	if err != nil {
		return Schedule{}, err
	}
	return s, nil
}
