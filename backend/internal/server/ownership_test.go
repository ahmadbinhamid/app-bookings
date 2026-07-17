package server_test

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"app-booking/internal/auth"
	"app-booking/internal/config"
	"app-booking/internal/db"
	"app-booking/internal/server"

	_ "github.com/go-sql-driver/mysql"
)

// These tests exercise the real HTTP router built by server.New — the same
// one cmd/server/main.go runs — to prove RequireLocationOwnership actually
// blocks cross-tenant access end to end, not just at the unit level.

const testJWTSecret = "test-secret-for-ownership-tests"

func connectOrSkip(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST not set — skipping server ownership integration tests")
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

func seedLocationForTenant(t *testing.T, conn *sql.DB, tenantID uint64) string {
	t.Helper()
	id := newUUID()
	if _, err := conn.Exec(`
		INSERT INTO locations (id, tenant_id, name, timezone, timezone_confirmed, flowpos_location_id, created_at, updated_at)
		VALUES (?, ?, 'Test Location', 'UTC', FALSE, ?, NOW(), NOW())
	`, id, tenantID, newUUID()); err != nil {
		t.Fatalf("seed location: %v", err)
	}
	return id
}

func tokenFor(t *testing.T, tenantID uint64) string {
	t.Helper()
	tok, err := auth.Generate(auth.Claims{TenantID: tenantID, UserID: 1, UserEmail: "test@example.com"}, testJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return tok
}

func TestRequireLocationOwnership_CrossTenantAccess_Rejected(t *testing.T) {
	conn := connectOrSkip(t)

	cfg := config.Config{JWTSecret: testJWTSecret, SigningSecret: "irrelevant", FlowposAPIURL: "http://127.0.0.1:0", SyncInterval: time.Hour}
	srv := server.New(cfg, conn)

	tenantA := uint64(time.Now().UnixNano())
	tenantB := tenantA + 1
	locationOfB := seedLocationForTenant(t, conn, tenantB)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations/"+locationOfB+"/employees", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, tenantA))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when tenant A requests tenant B's location, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequireLocationOwnership_OwnLocation_Allowed(t *testing.T) {
	conn := connectOrSkip(t)

	cfg := config.Config{JWTSecret: testJWTSecret, SigningSecret: "irrelevant", FlowposAPIURL: "http://127.0.0.1:0", SyncInterval: time.Hour}
	srv := server.New(cfg, conn)

	tenantA := uint64(time.Now().UnixNano())
	locationOfA := seedLocationForTenant(t, conn, tenantA)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations/"+locationOfA+"/employees", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, tenantA))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a tenant requesting their own location, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Employees []any `json:"employees"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestRequireLocationOwnership_UnknownLocation_Rejected(t *testing.T) {
	conn := connectOrSkip(t)

	cfg := config.Config{JWTSecret: testJWTSecret, SigningSecret: "irrelevant", FlowposAPIURL: "http://127.0.0.1:0", SyncInterval: time.Hour}
	srv := server.New(cfg, conn)

	tenantA := uint64(time.Now().UnixNano())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/locations/"+newUUID()+"/employees", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, tenantA))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a location id that doesn't exist at all, got %d", rec.Code)
	}
}
