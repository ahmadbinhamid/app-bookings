package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: "20260718000003_create_services",
		Up:   Up_20260718000003,
		Down: Down_20260718000003,
	})
}

func Up_20260718000003(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS services (
		  id               VARCHAR(36)     NOT NULL PRIMARY KEY,
		  location_id      VARCHAR(36)     NOT NULL,
		  name             VARCHAR(255)    NOT NULL,
		  description      TEXT            NULL,
		  duration_minutes INT UNSIGNED    NOT NULL,
		  buffer_minutes   INT UNSIGNED    NOT NULL DEFAULT 0,
		  price            DECIMAL(8,2)    NOT NULL,
		  active           BOOLEAN         NOT NULL DEFAULT TRUE,
		  created_at       DATETIME        NOT NULL,
		  updated_at       DATETIME        NOT NULL,
		  INDEX idx_location (location_id),
		  INDEX idx_name (name),
		  CONSTRAINT fk_services_location FOREIGN KEY (location_id) REFERENCES locations (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	return err
}

func Down_20260718000003(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS services`)
	return err
}
