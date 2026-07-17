package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: "20260718000006_create_employee_time_off",
		Up:   Up_20260718000006,
		Down: Down_20260718000006,
	})
}

func Up_20260718000006(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS employee_time_off (
		  id              VARCHAR(36)  NOT NULL PRIMARY KEY,
		  employee_id     VARCHAR(36)  NOT NULL,
		  start_datetime  DATETIME     NOT NULL,
		  end_datetime    DATETIME     NOT NULL,
		  reason          VARCHAR(255) NULL,
		  created_at      DATETIME     NOT NULL,
		  INDEX idx_employee (employee_id),
		  INDEX idx_range (start_datetime, end_datetime),
		  CONSTRAINT fk_timeoff_employee FOREIGN KEY (employee_id) REFERENCES employees (id),
		  CONSTRAINT chk_timeoff_range CHECK (start_datetime < end_datetime)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	return err
}

func Down_20260718000006(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS employee_time_off`)
	return err
}
