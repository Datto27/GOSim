// Package adapters defines the pluggable per-content-type behavior (movie,
// music, book, ...) that vecsim needs: how to build the text fed to the
// embedding model, and the hardcoded seed data for each domain.
//
// Adding a new domain means creating one new file in this package that
// implements Adapter and registers itself via Register in an init function
// — nothing else in the codebase needs to change.
package adapters

import "time"

// Item mirrors a row of the items table (the embedding column is handled
// separately by internal/store). Embedded is a derived field indicating
// whether the item's embedding has been computed.
type Item struct {
	ID        string         `json:"id"`
	Label     string         `json:"label"`
	Type      string         `json:"type"`
	Fields    map[string]any `json:"fields"`
	Tags      []string       `json:"tags"`
	Embedded  bool           `json:"embedded"`
	CreatedAt time.Time      `json:"created_at"`
}

// SeedItem is the hardcoded shape returned by an adapter's Seeds method.
type SeedItem struct {
	ID     string
	Label  string
	Fields map[string]any
	Tags   []string
}

// Adapter defines how a content domain is embedded and seeded.
type Adapter interface {
	// Type returns the domain's identifier, e.g. "movie".
	Type() string

	// Seeds returns the hardcoded seed items for this domain.
	Seeds() []SeedItem

	// BuildText assembles the descriptive string sent to the embedding
	// model from an item's fields.
	BuildText(fields map[string]any) string
}

var registry = map[string]Adapter{}

// order preserves a stable, deterministic ordering of registered adapters.
var order []string

// Register adds an adapter to the registry. It is called from each
// adapter implementation's init function.
func Register(a Adapter) {
	t := a.Type()
	if _, exists := registry[t]; !exists {
		order = append(order, t)
	}
	registry[t] = a
}

// Get returns the adapter registered for typ, if any.
func Get(typ string) (Adapter, bool) {
	a, ok := registry[typ]
	return a, ok
}

// All returns every registered adapter in registration order.
func All() []Adapter {
	out := make([]Adapter, 0, len(order))
	for _, t := range order {
		out = append(out, registry[t])
	}
	return out
}

// Types returns the type identifiers of every registered adapter, in
// registration order.
func Types() []string {
	out := make([]string, len(order))
	copy(out, order)
	return out
}

// stringSlice extracts a []string from a fields value that may be either a
// native []string (when built in-process from seed data) or a
// []interface{} of strings (when round-tripped through JSONB).
func stringSlice(v any) []string {
	switch vv := v.(type) {
	case []string:
		return vv
	case []any:
		out := make([]string, 0, len(vv))
		for _, e := range vv {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// stringField extracts a string from a fields value, returning "" if it is
// not a string.
func stringField(v any) string {
	s, _ := v.(string)
	return s
}

// intField extracts an int from a fields value that may be an int (native)
// or a float64 (when round-tripped through JSON/JSONB).
func intField(v any) int {
	switch vv := v.(type) {
	case int:
		return vv
	case float64:
		return int(vv)
	default:
		return 0
	}
}
