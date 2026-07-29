package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: "20260730000001_employees_tenant_scoped",
		Up:   Up_20260730000001,
		Down: Down_20260730000001,
	})
}

// FlowPOS turns out to have no employee-location relationship at all — just
// two flat, unrelated lists (employees, locations) per tenant. Employees can
// no longer be identified by (location_id, flowpos_employee_id): they're
// identified by (tenant_id, flowpos_employee_id), and location_id becomes a
// nullable field an admin sets by hand (see internal/modules/employee).
func Up_20260730000001(db *sql.DB) error {
	stmts := []string{
		`ALTER TABLE employees ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id`,
		`UPDATE employees e JOIN locations l ON l.id = e.location_id
			SET e.tenant_id = l.tenant_id WHERE e.tenant_id IS NULL`,
		`ALTER TABLE employees MODIFY COLUMN tenant_id BIGINT UNSIGNED NOT NULL`,
		`ALTER TABLE employees MODIFY COLUMN location_id VARCHAR(36) NULL`,
		`ALTER TABLE employees DROP INDEX uniq_location_flowpos_employee`,
		`ALTER TABLE employees ADD UNIQUE KEY uniq_tenant_flowpos_employee (tenant_id, flowpos_employee_id)`,
		`ALTER TABLE employees ADD INDEX idx_tenant (tenant_id)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func Down_20260730000001(db *sql.DB) error {
	stmts := []string{
		`ALTER TABLE employees DROP INDEX idx_tenant`,
		`ALTER TABLE employees DROP INDEX uniq_tenant_flowpos_employee`,
		`ALTER TABLE employees ADD UNIQUE KEY uniq_location_flowpos_employee (location_id, flowpos_employee_id)`,
		`ALTER TABLE employees MODIFY COLUMN location_id VARCHAR(36) NOT NULL`,
		`ALTER TABLE employees DROP COLUMN tenant_id`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
