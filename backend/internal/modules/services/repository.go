package services

import (
	"database/sql"
	"errors"
	"time"

	"app-booking/internal/config/pagination"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("service not found")

// ErrHasBookings blocks deleting a service that's actually been booked —
// see Delete's doc comment.
var ErrHasBookings = errors.New("this service is currently booked and can't be deleted")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(locationID string, p pagination.Params, search string) ([]Service, int, error) {
	where := " WHERE location_id = ?"
	args := []any{locationID}

	if search != "" {
		where += " AND name LIKE ?"
		args = append(args, "%"+search+"%")
	}

	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM services`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(`
		SELECT id, location_id, name, description, duration_minutes, buffer_minutes, price, active, created_at, updated_at
		FROM services`+where+`
		ORDER BY name ASC
		LIMIT ? OFFSET ?
	`, append(args, p.Limit(), p.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]Service, 0)
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.ID, &s.LocationID, &s.Name, &s.Description, &s.DurationMinutes, &s.BufferMinutes, &s.Price, &s.Active, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *Repository) Get(id string) (Service, error) {
	var s Service
	err := r.db.QueryRow(`
		SELECT id, location_id, name, description, duration_minutes, buffer_minutes, price, active, created_at, updated_at
		FROM services
		WHERE id = ?
	`, id).Scan(&s.ID, &s.LocationID, &s.Name, &s.Description, &s.DurationMinutes, &s.BufferMinutes, &s.Price, &s.Active, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Service{}, ErrNotFound
	}
	if err != nil {
		return Service{}, err
	}
	return s, nil
}

func (r *Repository) Create(locationID string, in Input) (Service, error) {
	now := time.Now().UTC()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	s := Service{
		ID:              uuid.NewString(),
		LocationID:      locationID,
		Name:            in.Name,
		Description:     in.Description,
		DurationMinutes: in.DurationMinutes,
		BufferMinutes:   in.BufferMinutes,
		Price:           in.Price,
		Active:          active,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_, err := r.db.Exec(`
		INSERT INTO services (id, location_id, name, description, duration_minutes, buffer_minutes, price, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.LocationID, s.Name, s.Description, s.DurationMinutes, s.BufferMinutes, s.Price, s.Active, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return Service{}, err
	}
	return s, nil
}

func (r *Repository) Update(id string, in Input) (Service, error) {
	now := time.Now().UTC()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	res, err := r.db.Exec(`
		UPDATE services
		SET name = ?, description = ?, duration_minutes = ?, buffer_minutes = ?, price = ?, active = ?, updated_at = ?
		WHERE id = ?
	`, in.Name, in.Description, in.DurationMinutes, in.BufferMinutes, in.Price, active, now, id)
	if err != nil {
		return Service{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return Service{}, err
	}
	if affected == 0 {
		return Service{}, ErrNotFound
	}
	return r.Get(id)
}

// isForeignKeyViolation reports whether err is MySQL error 1451 ("cannot
// delete or update a parent row: a foreign key constraint fails") — the
// signal that some other table still references the row being deleted.
func isForeignKeyViolation(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1451
}

// Delete removes a service, first clearing its employee_services rows (the
// "who can perform this" assignments — plain metadata, safe to drop). If a
// booking was ever actually made against this service, booking_segments'
// composite FK into employee_services blocks that cleanup, and this returns
// ErrHasBookings instead of deleting anything, rather than losing booking
// history or leaving a dangling reference.
func (r *Repository) Delete(id string) error {
	if _, err := r.db.Exec(`DELETE FROM employee_services WHERE service_id = ?`, id); err != nil {
		if isForeignKeyViolation(err) {
			return ErrHasBookings
		}
		return err
	}

	res, err := r.db.Exec(`DELETE FROM services WHERE id = ?`, id)
	if err != nil {
		if isForeignKeyViolation(err) {
			return ErrHasBookings
		}
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
