package schedules_test

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"app-booking/internal/config"
	"app-booking/internal/db"
	"app-booking/internal/modules/schedules"

	_ "github.com/go-sql-driver/mysql"
)

func connectOrSkip(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set — skipping schedules integration tests")
	}
	conn, err := db.Connect(config.Load())
	if err != nil {
		t.Fatalf("could not connect to test database: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func seedEmployee(t *testing.T, conn *sql.DB) string {
	t.Helper()
	loc := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO locations (id, tenant_id, name, timezone, timezone_confirmed, flowpos_location_id, created_at, updated_at)
		VALUES (?, 1, 'Test Location', 'UTC', FALSE, ?, NOW(), NOW())
	`, loc, newUUID()); err != nil {
		t.Fatalf("seed location: %v", err)
	}
	id := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO employees (id, tenant_id, location_id, flowpos_employee_id, name, active, synced_at, created_at, updated_at)
		VALUES (?, 1, ?, ?, 'Alice', TRUE, NOW(), NOW(), NOW())
	`, id, loc, newUUID()); err != nil {
		t.Fatalf("seed employee: %v", err)
	}
	return id
}

func TestService_Create_InvalidTimeRange_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	emp := seedEmployee(t, conn)
	svc := schedules.NewService(schedules.NewRepository(conn))

	_, err := svc.Create(emp, schedules.Input{DayOfWeek: 1, StartTime: "18:00", EndTime: "09:00"})
	if err != schedules.ErrInvalidTimeRange {
		t.Fatalf("expected ErrInvalidTimeRange for start after end, got %v", err)
	}
}

func TestService_Create_SplitShift_Allowed(t *testing.T) {
	conn := connectOrSkip(t)
	emp := seedEmployee(t, conn)
	svc := schedules.NewService(schedules.NewRepository(conn))

	if _, err := svc.Create(emp, schedules.Input{DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00"}); err != nil {
		t.Fatalf("first shift: %v", err)
	}
	if _, err := svc.Create(emp, schedules.Input{DayOfWeek: 1, StartTime: "16:00", EndTime: "20:00"}); err != nil {
		t.Fatalf("second, non-overlapping shift on the same day should be allowed (split shift): %v", err)
	}

	list, err := svc.ListByEmployee(emp)
	if err != nil {
		t.Fatalf("ListByEmployee: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 schedule rows, got %d", len(list))
	}
}

func TestService_Create_OverlappingSameDay_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	emp := seedEmployee(t, conn)
	svc := schedules.NewService(schedules.NewRepository(conn))

	if _, err := svc.Create(emp, schedules.Input{DayOfWeek: 2, StartTime: "09:00", EndTime: "17:00"}); err != nil {
		t.Fatalf("first shift: %v", err)
	}

	// Overlaps the middle of the existing 09:00-17:00 shift.
	_, err := svc.Create(emp, schedules.Input{DayOfWeek: 2, StartTime: "12:00", EndTime: "14:00"})
	if err != schedules.ErrOverlap {
		t.Fatalf("expected ErrOverlap for a range inside an existing shift, got %v", err)
	}

	// Partial overlap at the boundary must also be rejected.
	_, err = svc.Create(emp, schedules.Input{DayOfWeek: 2, StartTime: "16:00", EndTime: "18:00"})
	if err != schedules.ErrOverlap {
		t.Fatalf("expected ErrOverlap for a range partially overlapping an existing shift, got %v", err)
	}
}

func TestService_Create_SameTimesDifferentDay_Allowed(t *testing.T) {
	conn := connectOrSkip(t)
	emp := seedEmployee(t, conn)
	svc := schedules.NewService(schedules.NewRepository(conn))

	if _, err := svc.Create(emp, schedules.Input{DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00"}); err != nil {
		t.Fatalf("monday shift: %v", err)
	}
	if _, err := svc.Create(emp, schedules.Input{DayOfWeek: 2, StartTime: "09:00", EndTime: "17:00"}); err != nil {
		t.Fatalf("identical hours on a different day should never be treated as an overlap: %v", err)
	}
}
