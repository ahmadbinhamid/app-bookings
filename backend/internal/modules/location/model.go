package location

import "time"

// Location is synced/cached from FlowPOS — see design doc "Decisions" table
// (Multi-location employees) and the tenant/location resolution follow-up.
// tenant_id bridges the JWT's tenant_id claim (the only tenant signal this
// service receives) to this location-scoped domain; every other domain
// table cascades from location_id only, per the design.
type Location struct {
	ID       string `json:"id"`
	TenantID uint64 `json:"tenant_id"`
	Name     string `json:"name"`
	// Timezone is IANA (e.g. "Europe/London"). TimezoneConfirmed is false
	// until an admin has explicitly set it — until then, Timezone just holds
	// the DefaultTimezone placeholder ("UTC") and must be treated as
	// "unconfigured," not as truth, since the solver's schedule-to-instant
	// conversion depends on it (Phase 4).
	Timezone          string    `json:"timezone"`
	TimezoneConfirmed bool      `json:"timezone_confirmed"`
	FlowposLocationID string    `json:"flowpos_location_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// DefaultFlowposLocationID is the sentinel used when a FlowPOS install
// exposes no locations list at all (single implicit store per install) —
// see the design resolution: "fall back to creating/maintaining exactly one
// location per tenant from the install metadata." There is no real install
// metadata beyond tenant_id/api_key to name it from, so it's named
// generically until the real API confirms which mode is actually active.
const DefaultFlowposLocationID = "default"

// DefaultTimezone is the placeholder stored until an admin explicitly sets a
// real one (TimezoneConfirmed=true) — never treated as a confirmed value,
// even if FlowPOS's location payload happens to supply a timezone (see
// Upsert): a manual value always wins and sync never overwrites it.
const DefaultTimezone = "UTC"
