// ============================================================================
// CONFIRMED against flowpos-backend's real EmployeeController — see
// app/Modules/Employee/Http/Controllers/EmployeeController.php (index) in the
// flowpos-backend repo, and tenant-dashboard's src/lib/api/employees.ts,
// which already calls this same endpoint successfully.
// ============================================================================
//
// Confirmed facts that corrected the original guesses in this file:
//   - The response is a Laravel paginator object under the "employees" key
//     (current_page/data/last_page/total), not a flat array. client.go's
//     do() already unwraps the outer {"data": ...} envelope; what's left is
//     {"employees": {"data": [...], ...}}.
//   - Employees are scoped to tenant only — the tenant_employee table has no
//     location_id/store_id column, and the controller never reads a
//     location_id (or similarly named) query param. There is no way to
//     filter employees by location server-side; every location for a tenant
//     sees the same full employee list (sync/service.go upserts that same
//     list once per local location row, matching its existing per-location
//     upsert design).
//   - Field names (id, name, email, phone) were already right — "name" is a
//     Laravel accessor ($appends) computed from first_name/last_name, and
//     Laravel includes appended accessors in the JSON response.
//   - Auth (X-API-Key, no TID header) was already right: flowpos-backend's
//     SetActorMiddleware resolves the tenant straight from the API key on
//     that path and never requires TID there.
package flowpos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Employee is this app's shape for a FlowPOS staff member.
type Employee struct {
	FlowposID string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

// UnmarshalJSON — see Location.UnmarshalJSON's doc comment in client.go: id
// may be a number, a string, empty, or absent; a malformed id becomes
// FlowposID == "" rather than a decode error, so one bad row doesn't fail
// the whole list.
func (e *Employee) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID    json.RawMessage `json:"id"`
		Name  string          `json:"name"`
		Email string          `json:"email"`
		Phone string          `json:"phone"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.FlowposID = parseFlexibleID(raw.ID)
	e.Name = raw.Name
	e.Email = raw.Email
	e.Phone = raw.Phone
	return nil
}

// employeesPageSize is the "limit" sent per page — the controller defaults
// to 15 if omitted; we ask for a larger page to keep round trips down for
// tenants with many staff.
const employeesPageSize = 100

// ListEmployees proxies GET /employees, paginating through every page (the
// endpoint always returns a Laravel paginator, never the full list at once).
// FlowPOS has no employee-to-location scoping at all, so this returns the
// tenant's complete employee list regardless of location.
func (c *Client) ListEmployees(ctx context.Context, apiKey string) ([]Employee, error) {
	var all []Employee
	for page := 1; ; page++ {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(employeesPageSize))
		path := "/employees?" + q.Encode()

		var out struct {
			Employees struct {
				Data        []Employee `json:"data"`
				CurrentPage int        `json:"current_page"`
				LastPage    int        `json:"last_page"`
			} `json:"employees"`
		}
		if err := c.do(ctx, http.MethodGet, path, apiKey, &out); err != nil {
			return nil, fmt.Errorf("list employees (page %d): %w", page, err)
		}

		all = append(all, out.Employees.Data...)
		if len(out.Employees.Data) == 0 || out.Employees.CurrentPage >= out.Employees.LastPage {
			break
		}
	}
	return all, nil
}
