// ============================================================================
// ⚠️  UNCONFIRMED — EVERY ASSUMPTION IN THIS FILE ABOUT THE FLOWPOS EMPLOYEE
// API IS A GUESS. NOTHING BELOW HAS BEEN CHECKED AGAINST A REAL RESPONSE.
// ============================================================================
//
// TODO(pre-release blocker): verify /employees against a real FlowPOS API
// response and correct whatever's wrong in this file. Also listed in the
// project README under "Pre-release blockers" — do not ship without doing
// this. Do not guess further in the meantime; wait for a real response.
//
// Every assumption this app makes about the FlowPOS employee payload is
// deliberately kept in this one file — not spread across client.go — so
// reconciling against the real API, once it's supplied, is a single-file
// change:
//   - the endpoint path ("/employees")
//   - the query param used to scope by store ("location_id")
//   - the response envelope key ("employees")
//   - every field name on an employee record (id, name, email, phone)
//   - the id's JSON type (number vs string — see parseFlexibleID in
//     client.go, which is shared with Location and already tolerates either)
//
// Contrast with Location in client.go: FlowPOS's /locations endpoint is
// confirmed to exist (quotes' own client already calls it), even though its
// exact field shape isn't pinned down either. /employees has no prior art
// anywhere in this codebase — quotes only ever proxies products, categories,
// customers, locations, and orders. This is invented from nothing.
package flowpos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Employee is this app's guessed shape for a FlowPOS staff member.
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

// ListEmployees proxies GET /employees?location_id=X — UNCONFIRMED, see this
// file's header. locationFlowposID is the FlowPOS-side id (i.e.
// Location.FlowposID from client.go's ListLocations), not this app's local
// locations.id. An empty locationFlowposID omits the query param entirely
// (used in single-location fallback mode, where there's no FlowPOS location
// to scope by).
func (c *Client) ListEmployees(ctx context.Context, apiKey, locationFlowposID string) ([]Employee, error) {
	q := url.Values{}
	if locationFlowposID != "" {
		q.Set("location_id", locationFlowposID)
	}
	path := "/employees"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out struct {
		Employees []Employee `json:"employees"`
	}
	if err := c.do(ctx, http.MethodGet, path, apiKey, &out); err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}
	return out.Employees, nil
}
