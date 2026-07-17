package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: "20260718000008_create_booking_segments",
		Up:   Up_20260718000008,
		Down: Down_20260718000008,
	})
}

// No DB-level exclusion constraint here (MySQL has nothing equivalent to
// Postgres's `EXCLUDE USING gist` range-overlap constraint) — the "any other
// DB" branch of the design doc's Concurrency section is what applies:
// confirmBooking/rescheduleBooking do `SELECT ... FOR UPDATE` against
// idx_employee_overlap below and re-check inside the transaction. That index
// exists specifically to make that query fast, not to enforce the rule
// itself — the rule is enforced in the service layer (Phase 5).
//
// (employee_id, service_id) is a composite FK into employee_services'
// unique key, not two separate FKs to employees/services — this is what
// enforces "a segment's employee must actually be qualified for its
// service" at the DB level (design doc, Constraints & validation table).
// It transitively guarantees both employee_id and service_id individually
// exist too, since employee_services itself FKs to both.
func Up_20260718000008(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS booking_segments (
		  id               VARCHAR(36)     NOT NULL PRIMARY KEY,
		  booking_id       VARCHAR(36)     NOT NULL,
		  employee_id      VARCHAR(36)     NOT NULL,
		  service_id       VARCHAR(36)     NOT NULL,
		  sequence_order   INT UNSIGNED    NOT NULL DEFAULT 0,
		  start_time       DATETIME        NOT NULL,
		  end_time         DATETIME        NOT NULL,
		  blocked_until    DATETIME        NOT NULL,
		  status           ENUM('booked','completed','cancelled') NOT NULL DEFAULT 'booked',
		  original_price   DECIMAL(8,2)    NOT NULL,
		  price            DECIMAL(8,2)    NOT NULL,
		  created_at       DATETIME        NOT NULL,
		  updated_at       DATETIME        NOT NULL,
		  INDEX idx_booking (booking_id),
		  INDEX idx_employee_overlap (employee_id, status, start_time, blocked_until),
		  CONSTRAINT fk_segment_booking FOREIGN KEY (booking_id) REFERENCES bookings (id),
		  CONSTRAINT fk_segment_employee_service FOREIGN KEY (employee_id, service_id)
		    REFERENCES employee_services (employee_id, service_id),
		  CONSTRAINT chk_segment_end_after_start CHECK (end_time > start_time),
		  CONSTRAINT chk_segment_blocked_after_end CHECK (blocked_until >= end_time)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	return err
}

func Down_20260718000008(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS booking_segments`)
	return err
}
