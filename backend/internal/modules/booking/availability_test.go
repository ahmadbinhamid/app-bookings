package booking

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"app-booking/internal/config"
	"app-booking/internal/db"

	_ "github.com/go-sql-driver/mysql"
)

// newUUID is shared by every *_test.go file in this package (white-box
// tests, package booking).
func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// White-box test file (package booking, not booking_test) since
// combineDateAndTimeOfDay/freeIntervalsFor are unexported — they're the
// DST-safety-critical internals the design doc calls out specifically.

func connectOrSkip(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set — skipping booking integration tests")
	}
	conn, err := db.Connect(config.Load())
	if err != nil {
		t.Fatalf("could not connect to test database: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// --- DST safety: combineDateAndTimeOfDay must resolve the correct UTC
// offset PER DATE, not a cached constant — proven by checking a schedule
// either side of a real US DST transition (2026-03-08, clocks spring
// forward at 2am). No database needed for this one.

func TestCombineDateAndTimeOfDay_DSTSafe_SpringForward(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}

	// Before the transition: New York is on EST (UTC-5).
	before, err := combineDateAndTimeOfDay(2026, time.March, 7, "09:00:00", loc)
	if err != nil {
		t.Fatalf("combineDateAndTimeOfDay: %v", err)
	}
	if h := before.UTC().Hour(); h != 14 {
		t.Fatalf("expected 09:00 EST (2026-03-07) to be 14:00 UTC, got %d:00", h)
	}

	// After the transition: New York is on EDT (UTC-4) — the SAME
	// wall-clock time now resolves to a DIFFERENT UTC instant. A fixed-
	// offset shortcut (e.g. always subtracting 5 hours) would get this
	// wrong; time.Date via loc gets it right because it looks up the
	// offset for this specific date.
	after, err := combineDateAndTimeOfDay(2026, time.March, 9, "09:00:00", loc)
	if err != nil {
		t.Fatalf("combineDateAndTimeOfDay: %v", err)
	}
	if h := after.UTC().Hour(); h != 13 {
		t.Fatalf("expected 09:00 EDT (2026-03-09) to be 13:00 UTC, got %d:00", h)
	}
}

func seedEmployeeWithLocationTZ(t *testing.T, conn *sql.DB, tz string) (employeeID string) {
	t.Helper()
	loc := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO locations (id, tenant_id, name, timezone, timezone_confirmed, flowpos_location_id, created_at, updated_at)
		VALUES (?, 1, 'Test Location', ?, TRUE, ?, NOW(), NOW())
	`, loc, tz, newUUID()); err != nil {
		t.Fatalf("seed location: %v", err)
	}
	employeeID = newUUID()
	if _, err := conn.Exec(`
		INSERT INTO employees (id, tenant_id, location_id, flowpos_employee_id, name, active, synced_at, created_at, updated_at)
		VALUES (?, 1, ?, ?, 'Alice', TRUE, NOW(), NOW(), NOW())
	`, employeeID, loc, newUUID()); err != nil {
		t.Fatalf("seed employee: %v", err)
	}
	return employeeID
}

func TestFreeIntervalsFor_DSTTransitionDay_UsesCorrectOffset(t *testing.T) {
	conn := connectOrSkip(t)
	loc, _ := time.LoadLocation("America/New_York")
	emp := seedEmployeeWithLocationTZ(t, conn, "America/New_York")

	// Monday schedule 09:00-17:00 (2026-03-09 is a Monday, day_of_week=1).
	if _, err := conn.Exec(`
		INSERT INTO employee_schedules (id, employee_id, day_of_week, start_time, end_time, created_at, updated_at)
		VALUES (?, ?, 1, '09:00:00', '17:00:00', NOW(), NOW())
	`, newUUID(), emp); err != nil {
		t.Fatalf("seed schedule: %v", err)
	}

	targetDate := time.Date(2026, time.March, 9, 0, 0, 0, 0, time.UTC)
	intervals, err := freeIntervalsFor(conn, emp, targetDate, loc)
	if err != nil {
		t.Fatalf("freeIntervalsFor: %v", err)
	}
	if len(intervals) != 1 {
		t.Fatalf("expected 1 free interval, got %d", len(intervals))
	}
	// Post-transition day → EDT (UTC-4): 09:00 local = 13:00 UTC.
	if h := intervals[0].Start.Hour(); h != 13 {
		t.Fatalf("expected free interval to start at 13:00 UTC (09:00 EDT), got %d:00", h)
	}
}

func TestFreeIntervalsFor_SubtractsTimeOffAndExistingSegments(t *testing.T) {
	conn := connectOrSkip(t)
	loc, _ := time.LoadLocation("UTC")
	emp := seedEmployeeWithLocationTZ(t, conn, "UTC")

	// 2026-07-20 is a Monday → day_of_week=1.
	if _, err := conn.Exec(`
		INSERT INTO employee_schedules (id, employee_id, day_of_week, start_time, end_time, created_at, updated_at)
		VALUES (?, ?, 1, '09:00:00', '17:00:00', NOW(), NOW())
	`, newUUID(), emp); err != nil {
		t.Fatalf("seed schedule: %v", err)
	}

	// Time off 12:00-13:00.
	if _, err := conn.Exec(`
		INSERT INTO employee_time_off (id, employee_id, start_datetime, end_datetime, created_at)
		VALUES (?, ?, '2026-07-20 12:00:00', '2026-07-20 13:00:00', NOW())
	`, newUUID(), emp); err != nil {
		t.Fatalf("seed time off: %v", err)
	}

	// An existing (non-cancelled) booking occupying 09:00-09:40 (incl buffer).
	svcID := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO services (id, location_id, name, duration_minutes, buffer_minutes, price, active, created_at, updated_at)
		SELECT ?, location_id, 'Haircut', 30, 10, 25, TRUE, NOW(), NOW() FROM employees WHERE id = ?
	`, svcID, emp); err != nil {
		t.Fatalf("seed service: %v", err)
	}
	if _, err := conn.Exec(`INSERT INTO employee_services (id, employee_id, service_id, created_at) VALUES (?, ?, ?, NOW())`, newUUID(), emp, svcID); err != nil {
		t.Fatalf("seed employee_service: %v", err)
	}
	bookingID := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO bookings (id, location_id, customer_name, status, total_price, created_at, updated_at)
		SELECT ?, location_id, 'Jane', 'confirmed', 25, NOW(), NOW() FROM employees WHERE id = ?
	`, bookingID, emp); err != nil {
		t.Fatalf("seed booking: %v", err)
	}
	if _, err := conn.Exec(`
		INSERT INTO booking_segments (id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, '2026-07-20 09:00:00', '2026-07-20 09:30:00', '2026-07-20 09:40:00', 'booked', 25, 25, NOW(), NOW())
	`, newUUID(), bookingID, emp, svcID); err != nil {
		t.Fatalf("seed booking_segment: %v", err)
	}

	// A CANCELLED segment at 14:00-14:40 must NOT block availability.
	cancelledBookingID := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO bookings (id, location_id, customer_name, status, total_price, created_at, updated_at)
		SELECT ?, location_id, 'Cancelled Customer', 'cancelled', 25, NOW(), NOW() FROM employees WHERE id = ?
	`, cancelledBookingID, emp); err != nil {
		t.Fatalf("seed cancelled booking: %v", err)
	}
	if _, err := conn.Exec(`
		INSERT INTO booking_segments (id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, '2026-07-20 14:00:00', '2026-07-20 14:30:00', '2026-07-20 14:40:00', 'cancelled', 25, 25, NOW(), NOW())
	`, newUUID(), cancelledBookingID, emp, svcID); err != nil {
		t.Fatalf("seed cancelled booking_segment: %v", err)
	}

	targetDate := time.Date(2026, time.July, 20, 0, 0, 0, 0, time.UTC)
	intervals, err := freeIntervalsFor(conn, emp, targetDate, loc)
	if err != nil {
		t.Fatalf("freeIntervalsFor: %v", err)
	}

	// Expect: 09:40-12:00, 13:00-17:00 (14:00-14:40 NOT removed — it's cancelled).
	if len(intervals) != 2 {
		t.Fatalf("expected 2 free intervals, got %d: %+v", len(intervals), intervals)
	}
	if intervals[0].Start.Hour() != 9 || intervals[0].Start.Minute() != 40 || intervals[0].End.Hour() != 12 {
		t.Fatalf("unexpected first interval: %+v", intervals[0])
	}
	if intervals[1].Start.Hour() != 13 || intervals[1].End.Hour() != 17 {
		t.Fatalf("unexpected second interval (cancelled segment must not have carved out 14:00-14:40): %+v", intervals[1])
	}
}

func TestFreeIntervalsFor_NoScheduleThatDay_ReturnsEmpty(t *testing.T) {
	conn := connectOrSkip(t)
	loc, _ := time.LoadLocation("UTC")
	emp := seedEmployeeWithLocationTZ(t, conn, "UTC")
	// No schedule rows seeded at all.

	targetDate := time.Date(2026, time.July, 20, 0, 0, 0, 0, time.UTC)
	intervals, err := freeIntervalsFor(conn, emp, targetDate, loc)
	if err != nil {
		t.Fatalf("freeIntervalsFor: %v", err)
	}
	if len(intervals) != 0 {
		t.Fatalf("expected no free intervals when there's no schedule that day, got %+v", intervals)
	}
}
