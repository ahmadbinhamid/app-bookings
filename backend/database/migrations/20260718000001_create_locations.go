package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: "20260718000001_create_locations",
		Up:   Up_20260718000001,
		Down: Down_20260718000001,
	})
}

// LOCATION is synced/cached from FlowPOS (like EMPLOYEE) rather than a plain
// opaque location_id column the way appointments/quotes reference FlowPOS
// locations directly — this app's design needs a store IANA timezone
// (LOCATION.timezone) to convert EMPLOYEE_SCHEDULE's wall-clock times into
// absolute instants, which a bare id wouldn't carry.
//
// tenant_id is not part of the app-bookings design doc's ERD — it's added
// here to bridge the JWT's tenant_id claim (the only tenant signal this
// service receives) to the location-scoped domain model. Every other
// domain table cascades from location_id only, matching the design.
func Up_20260718000001(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS locations (
		  id                   VARCHAR(36)     NOT NULL PRIMARY KEY,
		  tenant_id            BIGINT UNSIGNED NOT NULL,
		  name                 VARCHAR(255)    NOT NULL,
		  timezone             VARCHAR(64)     NOT NULL,
		  flowpos_location_id  VARCHAR(255)    NOT NULL,
		  created_at           DATETIME        NOT NULL,
		  updated_at           DATETIME        NOT NULL,
		  UNIQUE KEY uniq_tenant_flowpos_location (tenant_id, flowpos_location_id),
		  INDEX idx_tenant (tenant_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	return err
}

func Down_20260718000001(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS locations`)
	return err
}
