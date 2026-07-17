package location_test

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"app-booking/internal/config"
	"app-booking/internal/db"
	"app-booking/internal/modules/location"

	_ "github.com/go-sql-driver/mysql"
)

func connectOrSkip(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set — skipping location integration tests")
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

// No database needed — SetTimezone validates the zone name before ever
// touching the repository, so a nil repo is safe here.
func TestService_SetTimezone_RejectsInvalidZone(t *testing.T) {
	loc := location.NewService(nil)
	_, err := loc.SetTimezone("irrelevant-id", "Europe/Lundon") // typo, not a real IANA zone
	if err != location.ErrInvalidTimezone {
		t.Fatalf("expected ErrInvalidTimezone for a bogus zone name, got %v", err)
	}
}

func TestService_SetTimezone_AcceptsRealZone_AndSyncNeverOverwritesIt(t *testing.T) {
	conn := connectOrSkip(t)
	repo := location.NewRepository(conn)
	svc := location.NewService(repo)
	tenantID := uint64(time.Now().UnixNano())

	// Simulate a first sync run creating the location with the UTC placeholder.
	created, err := repo.Upsert(tenantID, "flowpos-loc-1", "Main Store", "")
	if err != nil {
		t.Fatalf("Upsert (initial sync): %v", err)
	}
	if created.Timezone != location.DefaultTimezone || created.TimezoneConfirmed {
		t.Fatalf("expected an unconfirmed UTC placeholder after first sync, got %+v", created)
	}

	// Admin explicitly sets the real timezone.
	updated, err := svc.SetTimezone(created.ID, "Europe/London")
	if err != nil {
		t.Fatalf("SetTimezone: %v", err)
	}
	if updated.Timezone != "Europe/London" || !updated.TimezoneConfirmed {
		t.Fatalf("expected timezone_confirmed=true and Europe/London, got %+v", updated)
	}

	// A later sync run (FlowPOS now supplying its own guess, or just
	// re-upserting the same placeholder) must NOT overwrite the admin's
	// manual value — "a manual value always wins."
	afterResync, err := repo.Upsert(tenantID, "flowpos-loc-1", "Main Store", "America/New_York")
	if err != nil {
		t.Fatalf("Upsert (later sync): %v", err)
	}
	if afterResync.Timezone != "Europe/London" || !afterResync.TimezoneConfirmed {
		t.Fatalf("sync must never overwrite a manually-confirmed timezone, got %+v", afterResync)
	}
}
