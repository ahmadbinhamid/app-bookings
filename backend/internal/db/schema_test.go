package db_test

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

// These are integration tests against a real MySQL instance — every rule in
// Phase 1's schema gets tried on purpose, once from a passing angle and once
// from a failing angle, per the implementation plan's Phase 1 test
// instructions ("try to break it on purpose and check the database says
// no"). They read DB_HOST/DB_PORT/etc from the environment (same env vars
// backend/internal/config uses) and skip entirely if no test DB is reachable,
// so `go test ./...` stays green in environments with no DB configured.
func connectOrSkip(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set — skipping schema integration tests (see README for how to run a test DB)")
	}
	conn, err := db.Connect(config.Load())
	if err != nil {
		t.Fatalf("could not connect to test database: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// seedLocation, seedEmployee, seedService, seedEmployeeService are minimal
// fixtures — Phase 1 has no repository layer yet (that's Phase 3+), so
// these are raw inserts, not calls into application code.

func seedLocation(t *testing.T, conn *sql.DB) string {
	t.Helper()
	id := newUUID()
	_, err := conn.Exec(`
		INSERT INTO locations (id, tenant_id, name, timezone, flowpos_location_id, created_at, updated_at)
		VALUES (?, 1, 'Test Location', 'Europe/London', ?, NOW(), NOW())
	`, id, newUUID())
	if err != nil {
		t.Fatalf("seed location: %v", err)
	}
	return id
}

func seedEmployee(t *testing.T, conn *sql.DB, locationID string) string {
	t.Helper()
	id := newUUID()
	_, err := conn.Exec(`
		INSERT INTO employees (id, location_id, flowpos_employee_id, name, active, synced_at, created_at, updated_at)
		VALUES (?, ?, ?, 'Test Employee', TRUE, NOW(), NOW(), NOW())
	`, id, locationID, newUUID())
	if err != nil {
		t.Fatalf("seed employee: %v", err)
	}
	return id
}

func seedService(t *testing.T, conn *sql.DB, locationID string) string {
	t.Helper()
	id := newUUID()
	_, err := conn.Exec(`
		INSERT INTO services (id, location_id, name, duration_minutes, buffer_minutes, price, active, created_at, updated_at)
		VALUES (?, ?, 'Haircut', 30, 10, 25.00, TRUE, NOW(), NOW())
	`, id, locationID)
	if err != nil {
		t.Fatalf("seed service: %v", err)
	}
	return id
}

func seedEmployeeService(t *testing.T, conn *sql.DB, employeeID, serviceID string) {
	t.Helper()
	_, err := conn.Exec(`
		INSERT INTO employee_services (id, employee_id, service_id, created_at)
		VALUES (?, ?, ?, NOW())
	`, newUUID(), employeeID, serviceID)
	if err != nil {
		t.Fatalf("seed employee_service: %v", err)
	}
}

func seedBooking(t *testing.T, conn *sql.DB, locationID string) string {
	t.Helper()
	id := newUUID()
	_, err := conn.Exec(`
		INSERT INTO bookings (id, location_id, customer_name, status, total_price, created_at, updated_at)
		VALUES (?, ?, 'Jane Doe', 'pending', 25.00, NOW(), NOW())
	`, id, locationID)
	if err != nil {
		t.Fatalf("seed booking: %v", err)
	}
	return id
}

// --- EMPLOYEE: composite unique (location_id, flowpos_employee_id) ---

func TestEmployee_DuplicateFlowposIDAtSameLocation_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	flowposID := newUUID()

	_, err := conn.Exec(`
		INSERT INTO employees (id, location_id, flowpos_employee_id, name, active, synced_at, created_at, updated_at)
		VALUES (?, ?, ?, 'Employee A', TRUE, NOW(), NOW(), NOW())
	`, newUUID(), loc, flowposID)
	if err != nil {
		t.Fatalf("first insert should succeed: %v", err)
	}

	_, err = conn.Exec(`
		INSERT INTO employees (id, location_id, flowpos_employee_id, name, active, synced_at, created_at, updated_at)
		VALUES (?, ?, ?, 'Employee A Duplicate', TRUE, NOW(), NOW(), NOW())
	`, newUUID(), loc, flowposID)
	if err == nil {
		t.Fatal("expected duplicate (location_id, flowpos_employee_id) to be rejected, got no error")
	}
}

func TestEmployee_SameFlowposIDAtDifferentLocations_Allowed(t *testing.T) {
	conn := connectOrSkip(t)
	locA := seedLocation(t, conn)
	locB := seedLocation(t, conn)
	flowposID := newUUID()

	_, err := conn.Exec(`
		INSERT INTO employees (id, location_id, flowpos_employee_id, name, active, synced_at, created_at, updated_at)
		VALUES (?, ?, ?, 'Employee A', TRUE, NOW(), NOW(), NOW())
	`, newUUID(), locA, flowposID)
	if err != nil {
		t.Fatalf("insert at location A should succeed: %v", err)
	}

	_, err = conn.Exec(`
		INSERT INTO employees (id, location_id, flowpos_employee_id, name, active, synced_at, created_at, updated_at)
		VALUES (?, ?, ?, 'Employee A at another store', TRUE, NOW(), NOW(), NOW())
	`, newUUID(), locB, flowposID)
	if err != nil {
		t.Fatalf("same flowpos_employee_id at a different location should be allowed: %v", err)
	}
}

// --- EMPLOYEE_SERVICE: unique (employee_id, service_id) ---

func TestEmployeeService_Duplicate_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	emp := seedEmployee(t, conn, loc)
	svc := seedService(t, conn, loc)

	seedEmployeeService(t, conn, emp, svc)

	_, err := conn.Exec(`
		INSERT INTO employee_services (id, employee_id, service_id, created_at)
		VALUES (?, ?, ?, NOW())
	`, newUUID(), emp, svc)
	if err == nil {
		t.Fatal("expected duplicate (employee_id, service_id) to be rejected, got no error")
	}
}

// --- EMPLOYEE_SCHEDULE: no unique on (employee_id, day_of_week) — split shifts ---

func TestEmployeeSchedule_MultipleRowsSameDay_Allowed(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	emp := seedEmployee(t, conn, loc)

	_, err := conn.Exec(`
		INSERT INTO employee_schedules (id, employee_id, day_of_week, start_time, end_time, created_at, updated_at)
		VALUES (?, ?, 1, '09:00:00', '13:00:00', NOW(), NOW())
	`, newUUID(), emp)
	if err != nil {
		t.Fatalf("first split-shift row should succeed: %v", err)
	}

	_, err = conn.Exec(`
		INSERT INTO employee_schedules (id, employee_id, day_of_week, start_time, end_time, created_at, updated_at)
		VALUES (?, ?, 1, '16:00:00', '20:00:00', NOW(), NOW())
	`, newUUID(), emp)
	if err != nil {
		t.Fatalf("a second, non-overlapping row for the same employee/day (split shift) should be allowed: %v", err)
	}
}

func TestEmployeeSchedule_StartAfterEnd_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	emp := seedEmployee(t, conn, loc)

	_, err := conn.Exec(`
		INSERT INTO employee_schedules (id, employee_id, day_of_week, start_time, end_time, created_at, updated_at)
		VALUES (?, ?, 1, '18:00:00', '09:00:00', NOW(), NOW())
	`, newUUID(), emp)
	if err == nil {
		t.Fatal("expected start_time >= end_time to be rejected by chk_schedule_time_range, got no error")
	}
}

func TestEmployeeSchedule_InvalidDayOfWeek_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	emp := seedEmployee(t, conn, loc)

	_, err := conn.Exec(`
		INSERT INTO employee_schedules (id, employee_id, day_of_week, start_time, end_time, created_at, updated_at)
		VALUES (?, ?, 7, '09:00:00', '17:00:00', NOW(), NOW())
	`, newUUID(), emp)
	if err == nil {
		t.Fatal("expected day_of_week=7 to be rejected by chk_schedule_day_of_week, got no error")
	}
}

// --- EMPLOYEE_TIME_OFF: start < end ---

func TestEmployeeTimeOff_StartAfterEnd_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	emp := seedEmployee(t, conn, loc)

	future := time.Now().Add(48 * time.Hour)
	past := time.Now().Add(24 * time.Hour)

	_, err := conn.Exec(`
		INSERT INTO employee_time_off (id, employee_id, start_datetime, end_datetime, created_at)
		VALUES (?, ?, ?, ?, NOW())
	`, newUUID(), emp, future, past)
	if err == nil {
		t.Fatal("expected start_datetime >= end_datetime to be rejected by chk_timeoff_range, got no error")
	}
}

// --- BOOKING_SEGMENT: composite FK into employee_services ---

func TestBookingSegment_EmployeeNotQualifiedForService_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	emp := seedEmployee(t, conn, loc)
	svc := seedService(t, conn, loc)
	// Deliberately NOT seeding employee_services — emp is not qualified for svc.
	booking := seedBooking(t, conn, loc)

	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	end := start.Add(30 * time.Minute)
	blockedUntil := end.Add(10 * time.Minute)

	_, err := conn.Exec(`
		INSERT INTO booking_segments
		  (id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at)
		VALUES
		  (?, ?, ?, ?, 0, ?, ?, ?, 'booked', 25.00, 25.00, NOW(), NOW())
	`, newUUID(), booking, emp, svc, start, end, blockedUntil)
	if err == nil {
		t.Fatal("expected an unqualified employee/service pair to be rejected by fk_segment_employee_service, got no error")
	}
}

func TestBookingSegment_QualifiedEmployee_Allowed(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	emp := seedEmployee(t, conn, loc)
	svc := seedService(t, conn, loc)
	seedEmployeeService(t, conn, emp, svc)
	booking := seedBooking(t, conn, loc)

	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	end := start.Add(30 * time.Minute)
	blockedUntil := end.Add(10 * time.Minute)

	_, err := conn.Exec(`
		INSERT INTO booking_segments
		  (id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at)
		VALUES
		  (?, ?, ?, ?, 0, ?, ?, ?, 'booked', 25.00, 25.00, NOW(), NOW())
	`, newUUID(), booking, emp, svc, start, end, blockedUntil)
	if err != nil {
		t.Fatalf("a qualified employee/service pair should be accepted: %v", err)
	}
}

// --- BOOKING_SEGMENT: end_time > start_time, blocked_until >= end_time ---

func TestBookingSegment_EndBeforeStart_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	emp := seedEmployee(t, conn, loc)
	svc := seedService(t, conn, loc)
	seedEmployeeService(t, conn, emp, svc)
	booking := seedBooking(t, conn, loc)

	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	end := start.Add(-30 * time.Minute) // before start
	blockedUntil := start.Add(10 * time.Minute)

	_, err := conn.Exec(`
		INSERT INTO booking_segments
		  (id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at)
		VALUES
		  (?, ?, ?, ?, 0, ?, ?, ?, 'booked', 25.00, 25.00, NOW(), NOW())
	`, newUUID(), booking, emp, svc, start, end, blockedUntil)
	if err == nil {
		t.Fatal("expected end_time <= start_time to be rejected by chk_segment_end_after_start, got no error")
	}
}

func TestBookingSegment_BlockedUntilBeforeEnd_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	emp := seedEmployee(t, conn, loc)
	svc := seedService(t, conn, loc)
	seedEmployeeService(t, conn, emp, svc)
	booking := seedBooking(t, conn, loc)

	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	end := start.Add(30 * time.Minute)
	blockedUntil := end.Add(-5 * time.Minute) // before end — buffer can't be negative

	_, err := conn.Exec(`
		INSERT INTO booking_segments
		  (id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at)
		VALUES
		  (?, ?, ?, ?, 0, ?, ?, ?, 'booked', 25.00, 25.00, NOW(), NOW())
	`, newUUID(), booking, emp, svc, start, end, blockedUntil)
	if err == nil {
		t.Fatal("expected blocked_until < end_time to be rejected by chk_segment_blocked_after_end, got no error")
	}
}

// --- BOOKING.status / BOOKING_SEGMENT.status: closed enum sets ---

func TestBookingStatus_UnknownValue_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)

	_, err := conn.Exec(`
		INSERT INTO bookings (id, location_id, customer_name, status, total_price, created_at, updated_at)
		VALUES (?, ?, 'Jane Doe', 'not_a_real_status', 0, NOW(), NOW())
	`, newUUID(), loc)
	if err == nil {
		t.Fatal("expected an unknown booking status value to be rejected, got no error")
	}
}

func TestBookingSegmentStatus_UnknownValue_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	emp := seedEmployee(t, conn, loc)
	svc := seedService(t, conn, loc)
	seedEmployeeService(t, conn, emp, svc)
	booking := seedBooking(t, conn, loc)

	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	end := start.Add(30 * time.Minute)

	_, err := conn.Exec(`
		INSERT INTO booking_segments
		  (id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at)
		VALUES
		  (?, ?, ?, ?, 0, ?, ?, ?, 'not_a_real_status', 25.00, 25.00, NOW(), NOW())
	`, newUUID(), booking, emp, svc, start, end, end)
	if err == nil {
		t.Fatal("expected an unknown booking_segment status value to be rejected, got no error")
	}
}

// --- Referential integrity: dangling FK references are rejected ---

func TestEmployee_UnknownLocation_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	_, err := conn.Exec(`
		INSERT INTO employees (id, location_id, flowpos_employee_id, name, active, synced_at, created_at, updated_at)
		VALUES (?, ?, ?, 'Ghost', TRUE, NOW(), NOW(), NOW())
	`, newUUID(), newUUID(), newUUID())
	if err == nil {
		t.Fatal("expected a dangling location_id to be rejected by fk_employees_location, got no error")
	}
}

func TestBookingSegment_UnknownBooking_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedLocation(t, conn)
	emp := seedEmployee(t, conn, loc)
	svc := seedService(t, conn, loc)
	seedEmployeeService(t, conn, emp, svc)

	start := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	end := start.Add(30 * time.Minute)

	_, err := conn.Exec(`
		INSERT INTO booking_segments
		  (id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at)
		VALUES
		  (?, ?, ?, ?, 0, ?, ?, ?, 'booked', 25.00, 25.00, NOW(), NOW())
	`, newUUID(), newUUID(), emp, svc, start, end, end)
	if err == nil {
		t.Fatal("expected a dangling booking_id to be rejected by fk_segment_booking, got no error")
	}
}

// --- LOCATION: composite unique (tenant_id, flowpos_location_id) ---

func TestLocation_DuplicateFlowposIDForSameTenant_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	flowposLocID := newUUID()

	_, err := conn.Exec(`
		INSERT INTO locations (id, tenant_id, name, timezone, flowpos_location_id, created_at, updated_at)
		VALUES (?, 42, 'Store A', 'Europe/London', ?, NOW(), NOW())
	`, newUUID(), flowposLocID)
	if err != nil {
		t.Fatalf("first insert should succeed: %v", err)
	}

	_, err = conn.Exec(`
		INSERT INTO locations (id, tenant_id, name, timezone, flowpos_location_id, created_at, updated_at)
		VALUES (?, 42, 'Store A Duplicate', 'Europe/London', ?, NOW(), NOW())
	`, newUUID(), flowposLocID)
	if err == nil {
		t.Fatal("expected duplicate (tenant_id, flowpos_location_id) to be rejected, got no error")
	}
}

// newUUID is a tiny local UUIDv4 generator so this test file has no
// dependency on a uuid package — Phase 1 has no application code that
// generates ids yet (that lands with the repository layer in Phase 3+).
func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
