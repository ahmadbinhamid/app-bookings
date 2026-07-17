package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: "20260718000005_create_employee_schedules",
		Up:   Up_20260718000005,
		Down: Down_20260718000005,
	})
}

// Deliberately NO unique constraint on (employee_id, day_of_week): split
// shifts mean multiple rows per employee per day are valid (closed decision,
// design doc "Constraints & validation" table). Overlap between an
// employee's own rows for the same day is a service-layer check (Phase 3),
// not a DB constraint.
func Up_20260718000005(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS employee_schedules (
		  id           VARCHAR(36)  NOT NULL PRIMARY KEY,
		  employee_id  VARCHAR(36)  NOT NULL,
		  day_of_week  TINYINT UNSIGNED NOT NULL,
		  start_time   TIME         NOT NULL,
		  end_time     TIME         NOT NULL,
		  created_at   DATETIME     NOT NULL,
		  updated_at   DATETIME     NOT NULL,
		  INDEX idx_employee_day (employee_id, day_of_week),
		  CONSTRAINT fk_schedule_employee FOREIGN KEY (employee_id) REFERENCES employees (id),
		  CONSTRAINT chk_schedule_day_of_week CHECK (day_of_week BETWEEN 0 AND 6),
		  CONSTRAINT chk_schedule_time_range CHECK (start_time < end_time)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	return err
}

func Down_20260718000005(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS employee_schedules`)
	return err
}
