package assignments_test

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"app-booking/internal/config"
	"app-booking/internal/db"
	"app-booking/internal/modules/assignments"

	_ "github.com/go-sql-driver/mysql"
)

func connectOrSkip(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set — skipping assignments integration tests")
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

func seedLocationEmployeeService(t *testing.T, conn *sql.DB) (employeeID, serviceID string) {
	t.Helper()
	loc := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO locations (id, tenant_id, name, timezone, timezone_confirmed, flowpos_location_id, created_at, updated_at)
		VALUES (?, 1, 'Test Location', 'UTC', FALSE, ?, NOW(), NOW())
	`, loc, newUUID()); err != nil {
		t.Fatalf("seed location: %v", err)
	}
	employeeID = newUUID()
	if _, err := conn.Exec(`
		INSERT INTO employees (id, location_id, flowpos_employee_id, name, active, synced_at, created_at, updated_at)
		VALUES (?, ?, ?, 'Alice', TRUE, NOW(), NOW(), NOW())
	`, employeeID, loc, newUUID()); err != nil {
		t.Fatalf("seed employee: %v", err)
	}
	serviceID = newUUID()
	if _, err := conn.Exec(`
		INSERT INTO services (id, location_id, name, duration_minutes, buffer_minutes, price, active, created_at, updated_at)
		VALUES (?, ?, 'Haircut', 30, 10, 25.00, TRUE, NOW(), NOW())
	`, serviceID, loc); err != nil {
		t.Fatalf("seed service: %v", err)
	}
	return employeeID, serviceID
}

func TestService_Assign_DuplicateRejected(t *testing.T) {
	conn := connectOrSkip(t)
	employeeID, serviceID := seedLocationEmployeeService(t, conn)
	svc := assignments.NewService(assignments.NewRepository(conn))

	if _, err := svc.Assign(employeeID, serviceID); err != nil {
		t.Fatalf("first assign: %v", err)
	}
	_, err := svc.Assign(employeeID, serviceID)
	if err != assignments.ErrDuplicate {
		t.Fatalf("expected ErrDuplicate assigning the same employee to the same service twice, got %v", err)
	}
}

func TestService_AssignThenUnassign(t *testing.T) {
	conn := connectOrSkip(t)
	employeeID, serviceID := seedLocationEmployeeService(t, conn)
	svc := assignments.NewService(assignments.NewRepository(conn))

	if _, err := svc.Assign(employeeID, serviceID); err != nil {
		t.Fatalf("assign: %v", err)
	}
	ids, err := svc.ListEmployeeIDsForService(serviceID)
	if err != nil {
		t.Fatalf("ListEmployeeIDsForService: %v", err)
	}
	if len(ids) != 1 || ids[0] != employeeID {
		t.Fatalf("expected [%s], got %v", employeeID, ids)
	}

	if err := svc.Unassign(employeeID, serviceID); err != nil {
		t.Fatalf("unassign: %v", err)
	}
	ids, err = svc.ListEmployeeIDsForService(serviceID)
	if err != nil {
		t.Fatalf("ListEmployeeIDsForService after unassign: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected no assignments after unassign, got %v", ids)
	}

	// Re-assigning after unassign must work — proves Delete actually
	// removed the row rather than leaving a "cancelled" ghost the unique
	// key would still collide against.
	if _, err := svc.Assign(employeeID, serviceID); err != nil {
		t.Fatalf("re-assign after unassign should succeed: %v", err)
	}
}
