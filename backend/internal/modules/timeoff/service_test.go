package timeoff_test

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"app-booking/internal/config"
	"app-booking/internal/db"
	"app-booking/internal/modules/timeoff"

	_ "github.com/go-sql-driver/mysql"
)

func connectOrSkip(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set — skipping time-off integration tests")
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

func TestService_Create_InvalidRange_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	emp := seedEmployee(t, conn)
	svc := timeoff.NewService(timeoff.NewRepository(conn))

	start := time.Now().Add(48 * time.Hour)
	end := time.Now().Add(24 * time.Hour)
	_, err := svc.Create(emp, timeoff.Input{StartDatetime: start, EndDatetime: end})
	if err != timeoff.ErrInvalidTimeRange {
		t.Fatalf("expected ErrInvalidTimeRange, got %v", err)
	}
}

func TestService_CreateAndList(t *testing.T) {
	conn := connectOrSkip(t)
	emp := seedEmployee(t, conn)
	svc := timeoff.NewService(timeoff.NewRepository(conn))

	reason := "Annual leave"
	start := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	end := start.Add(48 * time.Hour)
	created, err := svc.Create(emp, timeoff.Input{StartDatetime: start, EndDatetime: end, Reason: &reason})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	list, err := svc.ListByEmployee(emp)
	if err != nil {
		t.Fatalf("ListByEmployee: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("expected the created row back, got %+v", list)
	}

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(created.ID); err != timeoff.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
