package sync_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"app-booking/internal/config"
	"app-booking/internal/db"
	"app-booking/internal/flowpos"
	"app-booking/internal/modules/employee"
	"app-booking/internal/modules/installation"
	"app-booking/internal/modules/location"
	"app-booking/internal/modules/sync"

	_ "github.com/go-sql-driver/mysql"
)

// These integration tests run the real sync.Service against a real MySQL
// database (see internal/db/schema_test.go for why: same skip-if-no-DB
// pattern) and a fake FlowPOS server (httptest), per the Phase 2 test list:
// sync twice doesn't duplicate; a new employee appears; a removed employee
// is deactivated, not deleted, and existing booking_segments referencing
// them are untouched; a malformed employee row is skipped and logged, not
// crashed on; the single-location fallback activates only when FlowPOS
// returns 404 for /locations, never on a legitimately empty list.

func connectOrSkip(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set — skipping sync integration tests")
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

// seedInstallation creates an installation row for a fresh tenant id and
// returns that tenant id — each test gets its own tenant so they don't
// interfere with each other's location/employee rows.
func seedInstallation(t *testing.T, instRepo *installation.Repository) uint64 {
	t.Helper()
	tenantID := uint64(time.Now().UnixNano())
	if _, err := installation.NewService(instRepo).Install(tenantID, "test-api-key"); err != nil {
		t.Fatalf("seed installation: %v", err)
	}
	return tenantID
}

// envelope wraps a payload the way FlowPOS's real API does, per
// internal/flowpos/client.go's doc comment.
func envelope(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": payload, "status": true})
}

// --- Fake FlowPOS server: two fixed locations; a single tenant-wide
// employee list, since FlowPOS has no employee-to-location scoping
// (confirmed against flowpos-backend's EmployeeController) — every location
// gets the same list synced under its own location_id. ---

func fakeFlowposServer(t *testing.T, employees []map[string]any) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/locations", func(w http.ResponseWriter, r *http.Request) {
		envelope(w, map[string]any{
			"locations": []map[string]any{
				{"id": 1, "name": "Main Store", "timezone": "Europe/London"},
				{"id": 2, "name": "Second Store", "timezone": "Europe/Paris"},
			},
		})
	})
	mux.HandleFunc("/employees", func(w http.ResponseWriter, r *http.Request) {
		envelope(w, map[string]any{
			"employees": map[string]any{
				"data":         employees,
				"current_page": 1,
				"last_page":    1,
				"total":        len(employees),
			},
		})
	})
	return httptest.NewServer(mux)
}

func TestSyncTenant_TwoLocations_TenantEmployeesAppliedToEachLocation(t *testing.T) {
	conn := connectOrSkip(t)
	instRepo := installation.NewRepository(conn)
	locRepo := location.NewRepository(conn)
	empRepo := employee.NewRepository(conn)

	server := fakeFlowposServer(t, []map[string]any{
		{"id": 101, "name": "Alice", "email": "alice@example.com"},
		{"id": 102, "name": "Bob"},
	})
	defer server.Close()

	svc := sync.NewService(locRepo, empRepo, instRepo, flowpos.NewClient(server.URL))
	tenantID := seedInstallation(t, instRepo)

	summary, err := svc.SyncTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("SyncTenant: %v", err)
	}
	if summary.LocationMode != "flowpos" {
		t.Fatalf("expected location_mode=flowpos, got %q", summary.LocationMode)
	}
	if summary.LocationsSynced != 2 {
		t.Fatalf("expected 2 locations synced, got %d", summary.LocationsSynced)
	}
	// FlowPOS has no location scoping for employees, so both employees are
	// upserted at both locations: 2 employees x 2 locations = 4.
	if summary.EmployeesSynced != 4 {
		t.Fatalf("expected 4 employees synced (2 employees x 2 locations), got %d", summary.EmployeesSynced)
	}
	if len(summary.LocationErrors) != 0 {
		t.Fatalf("expected no location errors, got %v", summary.LocationErrors)
	}

	locs, err := locRepo.ListByTenant(tenantID)
	if err != nil {
		t.Fatalf("ListByTenant: %v", err)
	}
	if len(locs) != 2 {
		t.Fatalf("expected 2 location rows in DB, got %d", len(locs))
	}
	for _, loc := range locs {
		emps, err := empRepo.ListByLocation(loc.ID)
		if err != nil {
			t.Fatalf("ListByLocation: %v", err)
		}
		if len(emps) != 2 {
			t.Fatalf("location %s: expected both tenant employees present, got %d", loc.FlowposLocationID, len(emps))
		}
	}
}

func TestSyncTenant_RunTwice_DoesNotDuplicate(t *testing.T) {
	conn := connectOrSkip(t)
	instRepo := installation.NewRepository(conn)
	locRepo := location.NewRepository(conn)
	empRepo := employee.NewRepository(conn)

	server := fakeFlowposServer(t, []map[string]any{
		{"id": 101, "name": "Alice"}, {"id": 102, "name": "Bob"}, {"id": 201, "name": "Carol"},
	})
	defer server.Close()

	svc := sync.NewService(locRepo, empRepo, instRepo, flowpos.NewClient(server.URL))
	tenantID := seedInstallation(t, instRepo)

	if _, err := svc.SyncTenant(context.Background(), tenantID); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if _, err := svc.SyncTenant(context.Background(), tenantID); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	locs, err := locRepo.ListByTenant(tenantID)
	if err != nil {
		t.Fatalf("ListByTenant: %v", err)
	}
	if len(locs) != 2 {
		t.Fatalf("expected exactly 2 location rows after two syncs, got %d (duplicates created)", len(locs))
	}

	for _, loc := range locs {
		emps, err := empRepo.ListByLocation(loc.ID)
		if err != nil {
			t.Fatalf("ListByLocation: %v", err)
		}
		if len(emps) != 3 {
			t.Fatalf("location %s: expected 3 employee rows after two syncs, got %d (duplicates created)", loc.FlowposLocationID, len(emps))
		}
	}
}

func TestSyncTenant_NewEmployeeAppears(t *testing.T) {
	conn := connectOrSkip(t)
	instRepo := installation.NewRepository(conn)
	locRepo := location.NewRepository(conn)
	empRepo := employee.NewRepository(conn)
	tenantID := seedInstallation(t, instRepo)

	employees := []map[string]any{{"id": 101, "name": "Alice"}}
	server := fakeFlowposServer(t, employees)
	svc := sync.NewService(locRepo, empRepo, instRepo, flowpos.NewClient(server.URL))
	if _, err := svc.SyncTenant(context.Background(), tenantID); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	server.Close()

	// FlowPOS now also has a second employee, tenant-wide.
	employees = append(employees, map[string]any{"id": 103, "name": "New Hire"})
	server2 := fakeFlowposServer(t, employees)
	defer server2.Close()
	svc2 := sync.NewService(locRepo, empRepo, instRepo, flowpos.NewClient(server2.URL))

	summary, err := svc2.SyncTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	// EmployeesSynced counts every employee upserted this run across both
	// locations (Alice again plus the new hire, x2 locations), not just
	// newly-created rows.
	if summary.EmployeesSynced != 4 {
		t.Fatalf("expected 4 employees synced on the second run (2 employees x 2 locations), got %d", summary.EmployeesSynced)
	}

	loc, err := locRepo.GetByTenantAndFlowposID(tenantID, "1")
	if err != nil {
		t.Fatalf("GetByTenantAndFlowposID: %v", err)
	}
	newHire, err := empRepo.GetByLocationAndFlowposID(loc.ID, "103")
	if err != nil {
		t.Fatalf("expected the new employee to exist after re-sync: %v", err)
	}
	if !newHire.Active {
		t.Fatal("expected the new employee to be active")
	}
}

func TestSyncTenant_RemovedEmployee_DeactivatedNotDeleted_BookingSegmentUntouched(t *testing.T) {
	conn := connectOrSkip(t)
	instRepo := installation.NewRepository(conn)
	locRepo := location.NewRepository(conn)
	empRepo := employee.NewRepository(conn)
	tenantID := seedInstallation(t, instRepo)

	employees := []map[string]any{{"id": 101, "name": "Alice"}}
	server := fakeFlowposServer(t, employees)
	svc := sync.NewService(locRepo, empRepo, instRepo, flowpos.NewClient(server.URL))
	if _, err := svc.SyncTenant(context.Background(), tenantID); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	server.Close()

	loc, err := locRepo.GetByTenantAndFlowposID(tenantID, "1")
	if err != nil {
		t.Fatalf("GetByTenantAndFlowposID: %v", err)
	}
	alice, err := empRepo.GetByLocationAndFlowposID(loc.ID, "101")
	if err != nil {
		t.Fatalf("GetByLocationAndFlowposID: %v", err)
	}

	// Give Alice a service and a booking_segment referencing her, using raw
	// SQL exactly like Phase 1's tests do — this module has no service/
	// booking repositories yet (Phase 3/5), so this is fixture setup, not
	// application code under test.
	svcID, bookingID := seedServiceAndBooking(t, conn, loc.ID, alice.ID)

	// FlowPOS no longer returns Alice at all (tenant-wide, so she disappears
	// from every location at once).
	employees = nil
	server2 := fakeFlowposServer(t, employees)
	defer server2.Close()
	svc2 := sync.NewService(locRepo, empRepo, instRepo, flowpos.NewClient(server2.URL))

	summary, err := svc2.SyncTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	// Alice is deactivated at both locations (1 per location x 2 locations).
	if summary.EmployeesDeactivated != 2 {
		t.Fatalf("expected 2 employees deactivated (1 per location), got %d", summary.EmployeesDeactivated)
	}

	aliceAfter, err := empRepo.GetByLocationAndFlowposID(loc.ID, "101")
	if err != nil {
		t.Fatalf("expected Alice's row to still exist (never deleted): %v", err)
	}
	if aliceAfter.Active {
		t.Fatal("expected Alice to be inactive after being removed from FlowPOS")
	}
	if aliceAfter.ID != alice.ID {
		t.Fatal("expected the same row (same id) — deactivation must not delete-and-recreate")
	}

	// The booking_segment referencing her must be completely untouched.
	var segEmployeeID, segStatus string
	err = conn.QueryRow(`SELECT employee_id, status FROM booking_segments WHERE booking_id = ?`, bookingID).
		Scan(&segEmployeeID, &segStatus)
	if err != nil {
		t.Fatalf("expected the booking_segment row to still exist: %v", err)
	}
	if segEmployeeID != alice.ID || segStatus != "booked" {
		t.Fatalf("expected booking_segment untouched (employee_id=%s status=booked), got employee_id=%s status=%s", alice.ID, segEmployeeID, segStatus)
	}
	_ = svcID
}

func TestSyncTenant_MalformedEmployeeRecord_SkippedNotCrashed(t *testing.T) {
	conn := connectOrSkip(t)
	instRepo := installation.NewRepository(conn)
	locRepo := location.NewRepository(conn)
	empRepo := employee.NewRepository(conn)
	tenantID := seedInstallation(t, instRepo)

	server := fakeFlowposServer(t, []map[string]any{
		{"id": 101, "name": "Alice"},
		{"id": "", "name": "Missing Id"}, // malformed: no id
		{"id": 104, "name": ""},          // malformed: no name
	})
	defer server.Close()

	svc := sync.NewService(locRepo, empRepo, instRepo, flowpos.NewClient(server.URL))
	summary, err := svc.SyncTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("expected malformed rows to be skipped, not to fail the sync: %v", err)
	}
	// 1 valid employee per location x 2 locations (malformed rows skipped at each).
	if summary.EmployeesSynced != 2 {
		t.Fatalf("expected exactly 2 valid employees synced (malformed rows skipped), got %d", summary.EmployeesSynced)
	}
}

func TestSyncTenant_LocationsEndpointMissing_FallsBackToSingleLocation(t *testing.T) {
	conn := connectOrSkip(t)
	instRepo := installation.NewRepository(conn)
	locRepo := location.NewRepository(conn)
	empRepo := employee.NewRepository(conn)
	tenantID := seedInstallation(t, instRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("/locations", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r) // simulates a FlowPOS install with no locations endpoint at all
	})
	mux.HandleFunc("/employees", func(w http.ResponseWriter, r *http.Request) {
		envelope(w, map[string]any{
			"employees": map[string]any{
				"data":         []map[string]any{{"id": 301, "name": "Only Employee"}},
				"current_page": 1,
				"last_page":    1,
				"total":        1,
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := sync.NewService(locRepo, empRepo, instRepo, flowpos.NewClient(server.URL))
	summary, err := svc.SyncTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("SyncTenant: %v", err)
	}
	if summary.LocationMode != "fallback_single_location" {
		t.Fatalf("expected fallback_single_location mode, got %q", summary.LocationMode)
	}
	if summary.LocationsSynced != 1 {
		t.Fatalf("expected exactly 1 fallback location, got %d", summary.LocationsSynced)
	}
	if summary.EmployeesSynced != 1 {
		t.Fatalf("expected 1 employee synced in fallback mode, got %d", summary.EmployeesSynced)
	}

	loc, err := locRepo.GetByTenantAndFlowposID(tenantID, location.DefaultFlowposLocationID)
	if err != nil {
		t.Fatalf("expected the default fallback location to exist: %v", err)
	}
	if loc.Timezone != location.DefaultTimezone {
		t.Fatalf("expected fallback location timezone %q, got %q", location.DefaultTimezone, loc.Timezone)
	}
}

func TestSyncTenant_LocationsEndpointReturnsEmptyArray_DoesNotTriggerFallback(t *testing.T) {
	conn := connectOrSkip(t)
	instRepo := installation.NewRepository(conn)
	locRepo := location.NewRepository(conn)
	empRepo := employee.NewRepository(conn)
	tenantID := seedInstallation(t, instRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("/locations", func(w http.ResponseWriter, r *http.Request) {
		envelope(w, map[string]any{"locations": []map[string]any{}}) // legitimately zero locations
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := sync.NewService(locRepo, empRepo, instRepo, flowpos.NewClient(server.URL))
	summary, err := svc.SyncTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("SyncTenant: %v", err)
	}
	if summary.LocationMode != "flowpos" {
		t.Fatalf("a legitimately empty locations array must NOT trigger single-location fallback, got mode %q", summary.LocationMode)
	}
	if summary.LocationsSynced != 0 {
		t.Fatalf("expected 0 locations synced, got %d", summary.LocationsSynced)
	}
}

// seedServiceAndBooking is raw-SQL fixture setup (Phase 3/5 repositories
// don't exist yet) — creates a service, qualifies employeeID for it, and a
// booking_segment referencing employeeID, returning (serviceID, bookingID).
func seedServiceAndBooking(t *testing.T, conn *sql.DB, locationID, employeeID string) (string, string) {
	t.Helper()
	svcID := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO services (id, location_id, name, duration_minutes, buffer_minutes, price, active, created_at, updated_at)
		VALUES (?, ?, 'Haircut', 30, 10, 25.00, TRUE, NOW(), NOW())
	`, svcID, locationID); err != nil {
		t.Fatalf("seed service: %v", err)
	}
	if _, err := conn.Exec(`
		INSERT INTO employee_services (id, employee_id, service_id, created_at) VALUES (?, ?, ?, NOW())
	`, newUUID(), employeeID, svcID); err != nil {
		t.Fatalf("seed employee_service: %v", err)
	}
	bookingID := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO bookings (id, location_id, customer_name, status, total_price, created_at, updated_at)
		VALUES (?, ?, 'Jane Doe', 'confirmed', 25.00, NOW(), NOW())
	`, bookingID, locationID); err != nil {
		t.Fatalf("seed booking: %v", err)
	}
	start := time.Now().Add(time.Hour).Truncate(time.Second)
	end := start.Add(30 * time.Minute)
	blockedUntil := end.Add(10 * time.Minute)
	if _, err := conn.Exec(`
		INSERT INTO booking_segments
		  (id, booking_id, employee_id, service_id, sequence_order, start_time, end_time, blocked_until, status, original_price, price, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?, ?, 'booked', 25.00, 25.00, NOW(), NOW())
	`, newUUID(), bookingID, employeeID, svcID, start, end, blockedUntil); err != nil {
		t.Fatalf("seed booking_segment: %v", err)
	}
	return svcID, bookingID
}
