package booking

import (
	"database/sql"
	"testing"
	"time"

	loc "app-booking/internal/modules/location"
	"app-booking/internal/modules/services"
)

// Shared fixture builders for this package's white-box tests
// (service_test.go, concurrency_test.go, availability_test.go).

func newBookingService(conn *sql.DB) *Service {
	repo := NewRepository(conn)
	svcSvc := services.NewService(services.NewRepository(conn))
	locSvc := loc.NewService(loc.NewRepository(conn))
	return NewService(repo, svcSvc, locSvc)
}

func seedLocationTZ(t *testing.T, conn *sql.DB, tz string, confirmed bool) string {
	t.Helper()
	id := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO locations (id, tenant_id, name, timezone, timezone_confirmed, flowpos_location_id, created_at, updated_at)
		VALUES (?, 1, 'Test Location', ?, ?, ?, NOW(), NOW())
	`, id, tz, confirmed, newUUID()); err != nil {
		t.Fatalf("seed location: %v", err)
	}
	return id
}

func seedEmployeeAt(t *testing.T, conn *sql.DB, locationID, name string) string {
	t.Helper()
	id := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO employees (id, location_id, flowpos_employee_id, name, active, synced_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, TRUE, NOW(), NOW(), NOW())
	`, id, locationID, newUUID(), name); err != nil {
		t.Fatalf("seed employee: %v", err)
	}
	return id
}

func seedServiceAt(t *testing.T, conn *sql.DB, locationID, name string, duration, buffer int, price float64) string {
	t.Helper()
	id := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO services (id, location_id, name, duration_minutes, buffer_minutes, price, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, NOW(), NOW())
	`, id, locationID, name, duration, buffer, price); err != nil {
		t.Fatalf("seed service: %v", err)
	}
	return id
}

func seedAssignment(t *testing.T, conn *sql.DB, employeeID, serviceID string) {
	t.Helper()
	if _, err := conn.Exec(`
		INSERT INTO employee_services (id, employee_id, service_id, created_at) VALUES (?, ?, ?, NOW())
	`, newUUID(), employeeID, serviceID); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
}

func seedSchedule(t *testing.T, conn *sql.DB, employeeID string, dayOfWeek int, start, end string) {
	t.Helper()
	if _, err := conn.Exec(`
		INSERT INTO employee_schedules (id, employee_id, day_of_week, start_time, end_time, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
	`, newUUID(), employeeID, dayOfWeek, start, end); err != nil {
		t.Fatalf("seed schedule: %v", err)
	}
}

// seedRawSegment inserts a booking_segment directly (bypassing Confirm) so
// tests can set up a pre-existing conflict/completed-segment scenario
// without going through the service layer under test.
func seedRawSegment(t *testing.T, conn *sql.DB, bookingID, employeeID, serviceID, status string, start, end, blockedUntil time.Time, price float64) string {
	t.Helper()
	id := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO booking_segments (id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, NOW(), NOW())
	`, id, bookingID, employeeID, serviceID, start, end, blockedUntil, status, price, price); err != nil {
		t.Fatalf("seed raw segment: %v", err)
	}
	return id
}

func seedRawBooking(t *testing.T, conn *sql.DB, locationID, status string) string {
	t.Helper()
	id := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO bookings (id, location_id, customer_name, status, total_price, created_at, updated_at)
		VALUES (?, ?, 'Jane Doe', ?, 0, NOW(), NOW())
	`, id, locationID, status); err != nil {
		t.Fatalf("seed raw booking: %v", err)
	}
	return id
}

// nextWeekday returns the next date STRICTLY AFTER `from` (i.e. starting
// tomorrow, never today) that falls on `weekday`, as a plain UTC-midnight
// calendar date. Deliberately never returns today: Propose() itself calls
// time.Now() to clamp a past earliestStart, so a test picking "today" could
// flake depending on what time of day it happens to run.
func nextWeekday(from time.Time, weekday time.Weekday) time.Time {
	d := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	for d.Weekday() != weekday {
		d = d.AddDate(0, 0, 1)
	}
	return d
}
