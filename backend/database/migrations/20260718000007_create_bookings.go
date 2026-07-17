package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: "20260718000007_create_bookings",
		Up:   Up_20260718000007,
		Down: Down_20260718000007,
	})
}

// The design doc's `timestamptz` requirement (design doc, Constraints table)
// is satisfied here by MySQL DATETIME columns holding UTC instants by
// convention — the same convention internal/db.go already uses for this
// whole app (DSN sets loc=UTC). There's no MySQL type that carries a time
// zone the way Postgres's timestamptz does; storing everything in UTC and
// converting to LOCATION.timezone only in the application layer (never in
// the DB) is the equivalent guarantee.
//
// created_by_admin_id is NOT a foreign key: there is no local admin/user
// table in this app (same as tenant_id/user_id elsewhere) — it's an opaque
// id from the calling JWT claims, kept for audit only (closed decision:
// booking permissions, design doc Decisions table).
//
// customer_name/customer_phone/customer_email replace the design doc's
// original client_name/client_phone/client_email naming per this round's
// rename.
func Up_20260718000007(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS bookings (
		  id                  VARCHAR(36)     NOT NULL PRIMARY KEY,
		  location_id         VARCHAR(36)     NOT NULL,
		  created_by_admin_id BIGINT UNSIGNED NULL,
		  customer_name       VARCHAR(255)    NOT NULL,
		  customer_phone      VARCHAR(64)     NULL,
		  customer_email      VARCHAR(255)    NULL,
		  status              ENUM('pending','confirmed','completed','cancelled') NOT NULL DEFAULT 'pending',
		  window_start        DATETIME        NULL,
		  window_end          DATETIME        NULL,
		  total_price         DECIMAL(10,2)   NOT NULL DEFAULT 0,
		  created_at          DATETIME        NOT NULL,
		  updated_at          DATETIME        NOT NULL,
		  INDEX idx_location (location_id),
		  INDEX idx_status (status),
		  INDEX idx_window_start (window_start),
		  CONSTRAINT fk_bookings_location FOREIGN KEY (location_id) REFERENCES locations (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	return err
}

func Down_20260718000007(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS bookings`)
	return err
}
