// Package adapters defines the item shape stored in the database.
//
// GOSim is domain-agnostic: there are no per-content-type adapters or seed
// data. Any homogeneous collection of JSON objects is imported as-is; the
// embedding text and ranking signals are derived from the detected schema
// (see internal/schema and internal/rerank), so no field name is special.
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
