package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: "20260717000000_create_installations",
		Up:   Up_20260717000000,
		Down: Down_20260717000000,
	})
}

func Up_20260717000000(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS installations (
		  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		  tenant_id   BIGINT UNSIGNED NOT NULL,
		  api_key     VARCHAR(255)    NOT NULL,
		  installed   BOOLEAN         NOT NULL DEFAULT TRUE,
		  created_at  DATETIME        NOT NULL,
		  updated_at  DATETIME        NOT NULL,
		  UNIQUE KEY uniq_tenant (tenant_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	return err
}

func Down_20260717000000(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS installations`)
	return err
}
