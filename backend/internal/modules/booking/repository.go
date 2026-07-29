package booking

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// QualifiedActiveEmployeeIDs returns every active employee assigned to
// serviceID (EMPLOYEE_SERVICE join, design doc) — the candidate pool
// Propose hands to the solver. An empty result is exactly what makes the
// solver return NO_EMPLOYEE_ASSIGNED.
func (r *Repository) QualifiedActiveEmployeeIDs(serviceID string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT es.employee_id
		FROM employee_services es
		JOIN employees e ON e.id = es.employee_id
		WHERE es.service_id = ? AND e.active = TRUE
	`, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// EmployeeEligibleForSegment is the confirm/reschedule-time re-check that a
// segment's employee could still have legitimately come out of Propose for
// THIS location: active, still assigned to the segment's service
// (EMPLOYEE_SERVICE), and belonging to locationID. Runs inside the caller's
// transaction, after LockEmployees, so it sees the employee row's current
// (locked, consistent) state rather than a possibly-stale snapshot from a
// separate connection.
func (r *Repository) EmployeeEligibleForSegment(tx *sql.Tx, employeeID, serviceID, locationID string) (bool, error) {
	var discard string
	err := tx.QueryRow(`
		SELECT es.id
		FROM employee_services es
		JOIN employees e ON e.id = es.employee_id
		WHERE es.employee_id = ? AND es.service_id = ? AND e.active = TRUE AND e.location_id = ?
	`, employeeID, serviceID, locationID).Scan(&discard)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// BeginTx starts a transaction — exported so service.go's confirm/cancel/
// reschedule flows (which need explicit BEGIN/COMMIT/ROLLBACK control, not
// a single query) can drive it directly.
func (r *Repository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

// LockEmployees takes a row lock on every distinct employee referenced by a
// proposal, in ascending id order, before any conflict check runs.
//
// This is NOT in the design doc's pseudocode, and is added deliberately:
// MySQL has no range-exclusion constraint (unlike the Postgres branch), so
// a bare "SELECT ... FOR UPDATE WHERE overlaps" only locks rows that
// already exist. Two transactions proposing brand-new, mutually
// overlapping times for the same employee — where neither row exists yet
// at check time — would BOTH see "no conflict" and both insert, a phantom-
// read double-booking that no amount of re-checking closes. Locking the
// employee row itself forces the second transaction to wait for the first
// to commit, so by the time it runs its own conflict check, the first
// transaction's segment is visible. Sorted order across every caller
// (Confirm and Reschedule both use this) prevents lock-order deadlocks
// between transactions touching multiple employees.
func (r *Repository) LockEmployees(tx *sql.Tx, employeeIDs []string) error {
	ids := uniqueSorted(employeeIDs)
	for _, id := range ids {
		var discard string
		if err := tx.QueryRow(`SELECT id FROM employees WHERE id = ? FOR UPDATE`, id).Scan(&discard); err != nil {
			return err
		}
	}
	return nil
}

func uniqueSorted(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// FindConflicts is the design doc's confirm/reschedule-time re-validation
// query — the actual double-booking guarantee (combined with LockEmployees
// above). Relies on the caller having already cancelled any of this
// booking's OWN old segments earlier in the same transaction (reschedule)
// so they're excluded by "status <> 'cancelled'" without needing a separate
// exclude-by-booking-id param.
func (r *Repository) FindConflicts(tx *sql.Tx, employeeID string, start, blockedUntil time.Time) ([]Segment, error) {
	rows, err := tx.Query(`
		SELECT id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at
		FROM booking_segments
		WHERE employee_id = ? AND status <> 'cancelled' AND start_time < ? AND blocked_until > ?
		FOR UPDATE
	`, employeeID, blockedUntil, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSegments(rows)
}

// HasFutureBookingsForEmployee reports whether employeeID has any booked
// (non-cancelled) segment starting after `after` — the guard behind
// employee.Service.AssignLocation's "block reassignment while future
// bookings exist" rule. Reuses idx_employee_overlap (employee_id, status,
// start_time, blocked_until); no new index needed.
func (r *Repository) HasFutureBookingsForEmployee(employeeID string, after time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM booking_segments
			WHERE employee_id = ? AND status = 'booked' AND start_time > ?
		)
	`, employeeID, after).Scan(&exists)
	return exists, err
}

func scanSegments(rows *sql.Rows) ([]Segment, error) {
	var out []Segment
	for rows.Next() {
		var s Segment
		if err := rows.Scan(&s.ID, &s.BookingID, &s.EmployeeID, &s.ServiceID, &s.SequenceOrder, &s.StartTime, &s.EndTime, &s.BlockedUntil, &s.Status, &s.OriginalPrice, &s.Price, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// InsertBooking creates the booking row as 'pending' (design doc pseudocode
// — flipped to 'confirmed' by UpdateBookingStatus once every segment is in)
// and returns its generated id.
func (r *Repository) InsertBooking(tx *sql.Tx, locationID string, adminID *uint64, customerName string, customerPhone, customerEmail *string, windowStart, windowEnd time.Time, totalPrice float64) (string, error) {
	id := uuid.NewString()
	now := time.Now().UTC()
	_, err := tx.Exec(`
		INSERT INTO bookings (id, location_id, created_by_admin_id, customer_name, customer_phone, customer_email, status, window_start, window_end, total_price, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?)
	`, id, locationID, adminID, customerName, customerPhone, customerEmail, windowStart, windowEnd, totalPrice, now, now)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *Repository) InsertSegment(tx *sql.Tx, bookingID, employeeID, serviceID string, sequenceOrder int, start, end, blockedUntil time.Time, originalPrice, price float64) error {
	now := time.Now().UTC()
	_, err := tx.Exec(`
		INSERT INTO booking_segments (id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'booked', ?, ?, ?, ?)
	`, uuid.NewString(), bookingID, employeeID, serviceID, sequenceOrder, start, end, blockedUntil, originalPrice, price, now, now)
	return err
}

func (r *Repository) UpdateBookingStatus(tx *sql.Tx, bookingID, status string) error {
	_, err := tx.Exec(`UPDATE bookings SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now().UTC(), bookingID)
	return err
}

// LockBookingForUpdate is the design doc's reschedule/cancel serialization
// point: "lock the parent booking row ... so concurrent reschedule/cancel
// on the same booking can't interleave." Every mutation that touches more
// than one segment (whole-booking cancel, reschedule) takes this lock
// first; per-segment cancel does too, for the same reason.
func (r *Repository) LockBookingForUpdate(tx *sql.Tx, bookingID string) (Booking, error) {
	var b Booking
	err := tx.QueryRow(`
		SELECT id, location_id, created_by_admin_id, customer_name, customer_phone, customer_email, status, window_start, window_end, total_price, created_at, updated_at
		FROM bookings WHERE id = ? FOR UPDATE
	`, bookingID).Scan(&b.ID, &b.LocationID, &b.CreatedByAdminID, &b.CustomerName, &b.CustomerPhone, &b.CustomerEmail, &b.Status, &b.WindowStart, &b.WindowEnd, &b.TotalPrice, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Booking{}, ErrNotFound
	}
	return b, err
}

// SegmentsForBooking reads a booking's segments INSIDE the current
// transaction — after LockBookingForUpdate, per the design doc's
// reschedule ordering ("read old segments INSIDE the transaction, after
// locking the parent booking row").
func (r *Repository) SegmentsForBooking(tx *sql.Tx, bookingID string, onlyNonCancelled bool) ([]Segment, error) {
	query := `SELECT id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at
		FROM booking_segments WHERE booking_id = ?`
	if onlyNonCancelled {
		query += ` AND status <> 'cancelled'`
	}
	query += ` ORDER BY sequence_order`

	rows, err := tx.Query(query, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSegments(rows)
}

func (r *Repository) GetSegment(tx *sql.Tx, segmentID string) (Segment, error) {
	var s Segment
	err := tx.QueryRow(`
		SELECT id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at
		FROM booking_segments WHERE id = ?
	`, segmentID).Scan(&s.ID, &s.BookingID, &s.EmployeeID, &s.ServiceID, &s.SequenceOrder, &s.StartTime, &s.EndTime, &s.BlockedUntil, &s.Status, &s.OriginalPrice, &s.Price, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Segment{}, ErrSegmentNotFound
	}
	return s, err
}

func (r *Repository) UpdateSegmentStatus(tx *sql.Tx, segmentID, status string) error {
	_, err := tx.Exec(`UPDATE booking_segments SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now().UTC(), segmentID)
	return err
}

// RecomputeBookingWindowAndTotal recomputes window_start/window_end/
// total_price from whatever non-cancelled segments remain — design doc:
// "recomputed from remaining non-cancelled segments any time segments
// change." If none remain (every segment just got cancelled), the
// existing values are left untouched — a deliberate choice, not an
// omission: they become the historical record of what the booking was
// before it was fully cancelled, which is more useful for display/audit
// than nulling them out with nothing to replace them.
func (r *Repository) RecomputeBookingWindowAndTotal(tx *sql.Tx, bookingID string) error {
	segments, err := r.SegmentsForBooking(tx, bookingID, true)
	if err != nil {
		return err
	}
	if len(segments) == 0 {
		return nil
	}

	windowStart := segments[0].StartTime
	windowEnd := segments[0].EndTime
	var total float64
	for _, s := range segments {
		if s.StartTime.Before(windowStart) {
			windowStart = s.StartTime
		}
		if s.EndTime.After(windowEnd) {
			windowEnd = s.EndTime
		}
		total += s.Price
	}

	_, err = tx.Exec(`
		UPDATE bookings SET window_start = ?, window_end = ?, total_price = ?, updated_at = ? WHERE id = ?
	`, windowStart, windowEnd, total, time.Now().UTC(), bookingID)
	return err
}

func (r *Repository) GetByID(bookingID string) (Booking, error) {
	var b Booking
	err := r.db.QueryRow(`
		SELECT id, location_id, created_by_admin_id, customer_name, customer_phone, customer_email, status, window_start, window_end, total_price, created_at, updated_at
		FROM bookings WHERE id = ?
	`, bookingID).Scan(&b.ID, &b.LocationID, &b.CreatedByAdminID, &b.CustomerName, &b.CustomerPhone, &b.CustomerEmail, &b.Status, &b.WindowStart, &b.WindowEnd, &b.TotalPrice, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Booking{}, ErrNotFound
	}
	if err != nil {
		return Booking{}, err
	}

	rows, err := r.db.Query(`
		SELECT id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at
		FROM booking_segments WHERE booking_id = ? ORDER BY sequence_order
	`, bookingID)
	if err != nil {
		return Booking{}, err
	}
	defer rows.Close()
	segments, err := scanSegments(rows)
	if err != nil {
		return Booking{}, err
	}
	b.Segments = segments
	return b, nil
}

// ListByLocation optionally filters by employeeID (via its segments) and a
// [from, to) window — backs the bookings list/calendar view.
func (r *Repository) ListByLocation(locationID string, employeeID *string, from, to *time.Time) ([]Booking, error) {
	query := `SELECT DISTINCT b.id, b.location_id, b.created_by_admin_id, b.customer_name, b.customer_phone, b.customer_email, b.status, b.window_start, b.window_end, b.total_price, b.created_at, b.updated_at
		FROM bookings b`
	args := []any{}
	where := []string{"b.location_id = ?"}
	args = append(args, locationID)

	if employeeID != nil {
		query += ` JOIN booking_segments bs ON bs.booking_id = b.id`
		where = append(where, "bs.employee_id = ?")
		args = append(args, *employeeID)
	}
	if from != nil {
		where = append(where, "b.window_end > ?")
		args = append(args, *from)
	}
	if to != nil {
		where = append(where, "b.window_start < ?")
		args = append(args, *to)
	}

	query += " WHERE " + strings.Join(where, " AND ") + " ORDER BY b.window_start"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Booking, 0)
	for rows.Next() {
		var b Booking
		if err := rows.Scan(&b.ID, &b.LocationID, &b.CreatedByAdminID, &b.CustomerName, &b.CustomerPhone, &b.CustomerEmail, &b.Status, &b.WindowStart, &b.WindowEnd, &b.TotalPrice, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		segs, err := r.segmentsForBookingNoTx(out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Segments = segs
	}
	return out, nil
}

func (r *Repository) segmentsForBookingNoTx(bookingID string) ([]Segment, error) {
	rows, err := r.db.Query(`
		SELECT id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at
		FROM booking_segments WHERE booking_id = ? ORDER BY sequence_order
	`, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSegments(rows)
}
