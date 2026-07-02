package featureflags

import "time"

// Flag is a single feature flag. Resolution precedence for a given customer:
//
//  1. CustomerOverrides[customerID] — explicit per-customer override.
//  2. EnabledGlobally               — global on/off switch.
//  3. DefaultValue                  — fallback when neither of the above applies.
//
// CustomerOverrides maps a customer UUID to a boolean. A missing key means "no
// override" (fall through to the global/default resolution), which is distinct
// from an explicit `false`.
type Flag struct {
	Key               string          `json:"key"`
	EnabledGlobally   bool            `json:"enabled_globally"`
	CustomerOverrides map[string]bool `json:"customer_overrides"`
	DefaultValue      bool            `json:"default_value"`
	Description       string          `json:"description,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// Resolve computes whether the flag is enabled for the given customer, applying
// override > global > default precedence. An empty customerID skips the
// per-customer override lookup.
func (f *Flag) Resolve(customerID string) bool {
	if customerID != "" && f.CustomerOverrides != nil {
		if v, ok := f.CustomerOverrides[customerID]; ok {
			return v
		}
	}
	if f.EnabledGlobally {
		return true
	}
	return f.DefaultValue
}

// UpdateInput carries the mutable fields of a flag for the admin toggle
// endpoint. Nil pointers leave the corresponding field unchanged (upsert also
// works: an unknown key is created).
type UpdateInput struct {
	EnabledGlobally   *bool           `json:"enabled_globally"`
	CustomerOverrides map[string]bool `json:"customer_overrides"`
	DefaultValue      *bool           `json:"default_value"`
	Description       *string         `json:"description"`
}
