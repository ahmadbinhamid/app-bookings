// Package flowpos calls the core FlowPOS API on a tenant's behalf, using
// that tenant's installation api_key (never the frontend's session JWT —
// this is a server-to-server call, the same pattern quotes/internal/flowpos
// uses for its own catalog proxy). Response envelope is always
// {"data": {...}, "status": true}, matching quotes' client.
package flowpos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: 15 * time.Second}}
}

// do mirrors quotes/internal/flowpos/client.go's do(): unmarshal the
// response body, verify the HTTP status, decode envelope.Data into out.
func (c *Client) do(ctx context.Context, method, path, apiKey string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		// Network-level failure (timeout, connection refused, DNS) — FlowPOS
		// itself is unreachable, not a problem with this request.
		return fmt.Errorf("%s %s: %w: %w", method, path, ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		// Connection dropped mid-read — same "FlowPOS's side" bucket as above.
		return fmt.Errorf("%s %s: read response: %w: %w", method, path, ErrUpstreamUnavailable, err)
	}

	switch {
	case resp.StatusCode == http.StatusNotFound:
		// Deliberately its own case, not folded into the general 4xx branch
		// below: this is the one signal modules/sync/service.go uses to
		// decide "this install has no locations endpoint at all" and fall
		// back to single-location mode. Any other error must NOT trigger
		// that fallback.
		return fmt.Errorf("%s %s: %w", method, path, ErrEndpointNotFound)
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%s %s: %w: %s", method, path, ErrUpstreamRejected, respBody)
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return fmt.Errorf("%s %s: %w: %s", method, path, ErrInvalidInput, respBody)
	case resp.StatusCode >= 500:
		// A real outage/bug on FlowPOS's own side, not something a
		// differently-shaped request from us would fix.
		return fmt.Errorf("%s %s: %w: status %d: %s", method, path, ErrUpstreamUnavailable, resp.StatusCode, respBody)
	case resp.StatusCode >= 300:
		return fmt.Errorf("%s %s: unexpected status %d: %s", method, path, resp.StatusCode, respBody)
	}

	if out == nil {
		return nil
	}
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respBody, &env); err != nil {
		// FlowPOS returned 2xx but a non-JSON/malformed body — still a
		// problem on its side, not a request we sent wrong.
		return fmt.Errorf("%s %s: decode envelope: %w: %w", method, path, ErrUpstreamUnavailable, err)
	}
	if len(env.Data) == 0 {
		return nil
	}
	return json.Unmarshal(env.Data, out)
}

// Location is this app's own shape for a FlowPOS store/location — NOT
// confirmed against a real FlowPOS response (quotes' own equivalent proxies
// raw JSON straight through to its frontend precisely because the shape was
// never pinned down server-side either; see quotes/internal/service/catalog.go
// ListLocations). There's no Timezone field: FlowPOS's payload has never been
// confirmed to carry one, so every synced location is deliberately given
// app-bookings' own UTC default instead (see sync.Service.syncLocations) —
// an admin sets the real one manually.
type Location struct {
	FlowposID string `json:"id"`
	Name      string `json:"name"`
}

// UnmarshalJSON tolerates FlowPOS returning a numeric id (as its other list
// endpoints do — see quotes' CreateOrderInput.LocationID *uint64), a string
// id, an empty string, or no id field at all, since none of that is
// confirmed either way. A malformed/unusable id becomes FlowposID == "" —
// callers (sync.Service) treat that as "skip this row," never a parse
// error that would fail the whole list.
func (l *Location) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID   json.RawMessage `json:"id"`
		Name string          `json:"name"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	l.FlowposID = parseFlexibleID(raw.ID)
	l.Name = raw.Name
	return nil
}

// parseFlexibleID tolerates a FlowPOS id arriving as a JSON number, a JSON
// string (including an empty one), or being absent entirely — this never
// errors, it returns "" for anything it can't make sense of, so one
// malformed row's id never fails decoding the rest of the list.
func parseFlexibleID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil && n != "" {
		return n.String()
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return ""
}

// ListLocations proxies GET /locations (locations.view) — confirmed to
// exist and work as a plain (non-paginated) array by quotes' own client, but
// the exact fields on each element are not confirmed here (quotes never
// parses them into a Go struct, only passes them through as raw JSON to its
// frontend). Returns ErrEndpointNotFound if this FlowPOS install doesn't
// expose the endpoint at all — callers use that specifically to switch to
// single-location fallback mode.
func (c *Client) ListLocations(ctx context.Context, apiKey string) ([]Location, error) {
	var out struct {
		Locations []Location `json:"locations"`
	}
	if err := c.do(ctx, http.MethodGet, "/locations", apiKey, &out); err != nil {
		return nil, fmt.Errorf("list locations: %w", err)
	}
	return out.Locations, nil
}

// Employee, its UnmarshalJSON, and ListEmployees all moved to
// employees_unconfirmed.go — see that file's header for why it's isolated
// on its own, and read it before touching anything employee-sync-related.
