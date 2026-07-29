package services

import (
	"database/sql"
	"errors"
	"time"

	"app-booking/internal/config/pagination"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("service not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List optionally excludes deleted (deactivated) services via activeOnly —
// the Services management page passes true so a deleted service actually
// disappears from it; the booking flows (new-booking service picker,
// historical booking display) pass false since a past booking still needs
// to resolve a deleted service's name, and a service picker for NEW bookings
// filters active itself where that matters — see Delete's doc comment for
// why deletion never removes the row at all.
func (r *Repository) List(locationID string, p pagination.Params, search string, activeOnly bool) ([]Service, int, error) {
	where := " WHERE location_id = ?"
	args := []any{locationID}

	if activeOnly {
		where += " AND active = TRUE"
	}
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

// Delete soft-deletes a service: active flips to FALSE, nothing is ever
// removed. A hard delete was tried first (clearing employee_services, then
// booking_segments, then the row itself) but that meant any service with
// completed or cancelled booking history got its segments wiped out too —
// booking_segments' composite FK into employee_services has no CASCADE, so
// there was no way to hard-delete a used service without also destroying the
// booking history that referenced it, which then showed up as blank
// service/employee/time fields on old bookings. Deactivating avoids that
// entirely: List(activeOnly=true) hides it from the Services page as
// requested, while every other list call (booking history display, etc.)
// still resolves it correctly since the row is untouched.
func (r *Repository) Delete(id string) error {
	res, err := r.db.Exec(`UPDATE services SET active = FALSE, updated_at = ? WHERE id = ?`, time.Now().UTC(), id)
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
