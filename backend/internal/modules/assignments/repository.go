package assignments

import (
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("assignment not found")
	// ErrDuplicate means this employee is already assigned to this service —
	// the DB's own UNIQUE(employee_id, service_id) is what actually catches
	// this (Phase 1); this just translates that into a domain error instead
	// of a raw driver error leaking out of the repository.
	ErrDuplicate = errors.New("employee is already assigned to this service")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(employeeID, serviceID string) (Assignment, error) {
	a := Assignment{ID: uuid.NewString(), EmployeeID: employeeID, ServiceID: serviceID, CreatedAt: time.Now().UTC()}
	_, err := r.db.Exec(`
		INSERT INTO employee_services (id, employee_id, service_id, created_at)
		VALUES (?, ?, ?, ?)
	`, a.ID, a.EmployeeID, a.ServiceID, a.CreatedAt)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return Assignment{}, ErrDuplicate
		}
		return Assignment{}, err
	}
	return a, nil
}

func (r *Repository) Delete(employeeID, serviceID string) error {
	res, err := r.db.Exec(`DELETE FROM employee_services WHERE employee_id = ? AND service_id = ?`, employeeID, serviceID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListEmployeeIDsForService returns the ids of every employee assigned to
// serviceID — used both by the "list assigned employees" endpoint and (via
// employee.Service, Phase 4) the solver to find qualified employees.
func (r *Repository) ListEmployeeIDsForService(serviceID string) ([]string, error) {
	rows, err := r.db.Query(`SELECT employee_id FROM employee_services WHERE service_id = ?`, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *Repository) ListServiceIDsForEmployee(employeeID string) ([]string, error) {
	rows, err := r.db.Query(`SELECT service_id FROM employee_services WHERE employee_id = ?`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
