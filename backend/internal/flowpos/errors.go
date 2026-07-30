package flowpos

import "errors"

var (
	// ErrUpstreamRejected means FlowPOS returned 401/403 — the stored
	// installation api_key is invalid/expired. Distinct from ErrInvalidInput
	// so the caller can tell "our request was malformed" from "FlowPOS
	// doesn't trust us anymore."
	ErrUpstreamRejected = errors.New("flowpos rejected the request (invalid or expired api key)")
	// ErrInvalidInput covers any other 4xx — a validation failure or bad
	// request on our side.
	ErrInvalidInput = errors.New("flowpos rejected the request as invalid")
	// ErrEndpointNotFound means FlowPOS returned 404 for the endpoint itself
	// (not "resource not found" — this client never requests a single
	// resource by id). Used specifically to detect "this FlowPOS install
	// doesn't expose a locations list at all" and fall back to single-
	// location mode (see modules/sync/service.go) — no other error triggers
	// that fallback.
	ErrEndpointNotFound = errors.New("flowpos endpoint not found")
	// ErrUpstreamUnavailable means the problem is on FlowPOS's side, not
	// ours or the tenant's: a 5xx response, a network-level failure
	// (timeout, connection refused, connection dropped mid-read), or a
	// non-JSON/malformed response body. Distinct from ErrUpstreamRejected
	// (bad api_key) and ErrInvalidInput (bad request) so callers can show
	// "FlowPOS is having trouble, try again shortly" instead of a generic
	// error — there's nothing a retry of the SAME request fixes here, but
	// it isn't a sign of a bug on this end either.
	ErrUpstreamUnavailable = errors.New("flowpos is currently unavailable")
)
