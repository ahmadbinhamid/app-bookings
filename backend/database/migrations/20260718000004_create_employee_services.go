package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: "20260718000004_create_employee_services",
		Up:   Up_20260718000004,
		Down: Down_20260718000004,
	})
}

// uniq_employee_service doubles as the target of BOOKING_SEGMENT's composite
// foreign key (employee_id, service_id) added in the booking_segments
// migration — that's how "a segment's employee must actually be qualified
// for its service" gets enforced at the DB level (design doc, Constraints &
// validation table).
func Up_20260718000004(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS employee_services (
		  id           VARCHAR(36)  NOT NULL PRIMARY KEY,
		  employee_id  VARCHAR(36)  NOT NULL,
		  service_id   VARCHAR(36)  NOT NULL,
		  created_at   DATETIME     NOT NULL,
		  UNIQUE KEY uniq_employee_service (employee_id, service_id),
		  INDEX idx_service (service_id),
		  CONSTRAINT fk_empsvc_employee FOREIGN KEY (employee_id) REFERENCES employees (id),
		  CONSTRAINT fk_empsvc_service FOREIGN KEY (service_id) REFERENCES services (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	return err
}

func Down_20260718000004(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS employee_services`)
	return err
}
