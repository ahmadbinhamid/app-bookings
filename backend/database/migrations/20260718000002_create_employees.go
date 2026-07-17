package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: "20260718000002_create_employees",
		Up:   Up_20260718000002,
		Down: Down_20260718000002,
	})
}

// Uniqueness is composite (location_id, flowpos_employee_id), not a global
// unique on flowpos_employee_id alone — FlowPOS may reuse employee ids
// across separate stores (closed decision, see design doc "Decisions" table,
// "Multi-location employees").
func Up_20260718000002(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS employees (
		  id                   VARCHAR(36)     NOT NULL PRIMARY KEY,
		  location_id          VARCHAR(36)     NOT NULL,
		  flowpos_employee_id  VARCHAR(255)    NOT NULL,
		  name                 VARCHAR(255)    NOT NULL,
		  email                VARCHAR(255)    NULL,
		  phone                VARCHAR(64)     NULL,
		  active               BOOLEAN         NOT NULL DEFAULT TRUE,
		  synced_at            DATETIME        NOT NULL,
		  created_at           DATETIME        NOT NULL,
		  updated_at           DATETIME        NOT NULL,
		  UNIQUE KEY uniq_location_flowpos_employee (location_id, flowpos_employee_id),
		  INDEX idx_location (location_id),
		  CONSTRAINT fk_employees_location FOREIGN KEY (location_id) REFERENCES locations (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	return err
}

func Down_20260718000002(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS employees`)
	return err
}
