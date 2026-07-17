package installation

import "time"

// Installation tracks a single tenant's install of this app. One row per
// tenant_id, created on first /install and never deleted — /uninstall just
// flips Installed back to false so re-installing is a cheap update.
type Installation struct {
	ID        uint64    `json:"id"`
	TenantID  uint64    `json:"tenant_id"`
	APIKey    string    `json:"-"`
	Installed bool      `json:"installed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
