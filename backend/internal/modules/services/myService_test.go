package services_test

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"app-booking/internal/config"
	"app-booking/internal/config/pagination"
	"app-booking/internal/db"
	"app-booking/internal/modules/services"

	_ "github.com/go-sql-driver/mysql"
)

func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func connectOrSkip(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set — skipping services integration tests")
	}
	conn, err := db.Connect(config.Load())
	if err != nil {
		t.Fatalf("could not connect to test database: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func seedTestLocation(t *testing.T, conn *sql.DB) string {
	t.Helper()
	id := newUUID()
	_, err := conn.Exec(`
		INSERT INTO locations (id, tenant_id, name, timezone, timezone_confirmed, flowpos_location_id, created_at, updated_at)
		VALUES (?, 1, 'Test Location', 'UTC', FALSE, ?, NOW(), NOW())
	`, id, newUUID())
	if err != nil {
		t.Fatalf("seed location: %v", err)
	}
	return id
}

func TestMyService_CreateGetUpdateDelete(t *testing.T) {
	conn := connectOrSkip(t)
	loc := seedTestLocation(t, conn)
	svc := services.NewService(services.NewRepository(conn))

	desc := "A classic cut"
	created, err := svc.Create(loc, services.Input{
		Name: "Haircut", Description: &desc, DurationMinutes: 30, BufferMinutes: 10, Price: 25.00,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.LocationID != loc {
		t.Fatalf("expected location_id %s, got %s", loc, created.LocationID)
	}
	if !created.Active {
		t.Fatal("expected a new service to default to active")
	}

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Haircut" || got.DurationMinutes != 30 || got.BufferMinutes != 10 {
		t.Fatalf("Get returned unexpected data: %+v", got)
	}

	updated, err := svc.Update(created.ID, services.Input{Name: "Deluxe Haircut", DurationMinutes: 45, BufferMinutes: 15, Price: 35.00})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Deluxe Haircut" || updated.DurationMinutes != 45 {
		t.Fatalf("Update did not apply: %+v", updated)
	}

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(created.ID); err != services.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMyService_ListScopedToLocationAndSearch(t *testing.T) {
	conn := connectOrSkip(t)
	locA := seedTestLocation(t, conn)
	locB := seedTestLocation(t, conn)
	svc := services.NewService(services.NewRepository(conn))

	mustCreate := func(loc, name string) {
		if _, err := svc.Create(loc, services.Input{Name: name, DurationMinutes: 30, Price: 10}); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}
	mustCreate(locA, "Haircut")
	mustCreate(locA, "Beard Trim")
	mustCreate(locB, "Manicure")

	items, total, err := svc.List(locA, pagination.Params{Page: 1, PerPage: 20}, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("expected 2 services at location A (location B's must not leak in), got total=%d len=%d", total, len(items))
	}

	items, total, err = svc.List(locA, pagination.Params{Page: 1, PerPage: 20}, "Beard")
	if err != nil {
		t.Fatalf("List with search: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Name != "Beard Trim" {
		t.Fatalf("expected search to find exactly Beard Trim, got %+v", items)
	}
}
