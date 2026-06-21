// Package schema infers the structure of a collection from its items and
// derives the two things hybrid search needs: which fields carry free text
// (and therefore belong in the embedding) and how every other field should be
// compared (set overlap, numeric distance, exact keyword).
//
// It is fully domain-agnostic — nothing here knows about movies, products, or
// any particular field name. Kinds are inferred from the data.
package schema

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

// Kind classifies how a field is compared and whether it is embedded.
type Kind string

const (
	// Text is free-form natural language (plot, description, title): embedded
	// into the semantic vector, compared by cosine similarity.
	Text Kind = "text"
	// List is a set of discrete tokens (genre, cast, tags): compared by overlap.
	List Kind = "list"
	// Number is a scalar (year, price): compared by range-normalized distance.
	Number Kind = "number"
	// Keyword is a low-cardinality short string (category, status): exact match.
	Keyword Kind = "keyword"
)

// Field describes one detected parameter.
type Field struct {
	Kind Kind     `json:"kind"`
	Min  *float64 `json:"min,omitempty"`
	Max  *float64 `json:"max,omitempty"`
}

// Schema maps a field name to its detected description.
type Schema map[string]Field

// detection thresholds for distinguishing free text from short keywords.
const (
	keywordMaxAvgLen        = 30
	keywordMaxDistinctRatio = 0.5
)

// fieldAcc accumulates per-field statistics while scanning a collection.
type fieldAcc struct {
	nList, nNum, nStr int
	min, max          float64
	haveNum           bool
	sumLen            int
	distinct          map[string]struct{}
}

func (a *fieldAcc) observeNum(f float64) {
	if !a.haveNum || f < a.min {
		a.min = f
	}
	if !a.haveNum || f > a.max {
		a.max = f
	}
	a.haveNum = true
}

func (a *fieldAcc) observeStr(s string) {
	a.sumLen += len(s)
	a.distinct[strings.ToLower(strings.TrimSpace(s))] = struct{}{}
}

// stringKind classifies a string field as Text or Keyword based on average
// length and value cardinality: short, frequently-repeated strings are
// keywords (category, status); everything else is free text.
func (a *fieldAcc) stringKind() Kind {
	if a.nStr == 0 {
		return Text
	}
	avgLen := float64(a.sumLen) / float64(a.nStr)
	distinctRatio := float64(len(a.distinct)) / float64(a.nStr)
	if avgLen < keywordMaxAvgLen && distinctRatio < keywordMaxDistinctRatio {
		return Keyword
	}
	return Text
}

// Detect infers a Schema from the field maps of a collection's items.
func Detect(items []map[string]any) Schema {
	stats := map[string]*fieldAcc{}

	for _, fields := range items {
		for name, v := range fields {
			if v == nil {
				continue
			}
			a := stats[name]
			if a == nil {
				a = &fieldAcc{distinct: map[string]struct{}{}}
				stats[name] = a
			}
			switch vv := v.(type) {
			case []any, []string:
				a.nList++
			case float64:
				a.nNum++
				a.observeNum(vv)
			case int:
				a.nNum++
				a.observeNum(float64(vv))
			case bool:
				a.nStr++ // treat booleans as keywords
				a.observeStr(strconv.FormatBool(vv))
			case string:
				a.nStr++
				a.observeStr(vv)
			default:
				// nested object / unknown — ignore for schema purposes.
			}
		}
	}

	out := Schema{}
	for name, a := range stats {
		switch {
		case a.nList >= a.nNum && a.nList >= a.nStr && a.nList > 0:
			out[name] = Field{Kind: List}
		case a.nNum >= a.nStr && a.nNum > 0:
			mn, mx := a.min, a.max
			out[name] = Field{Kind: Number, Min: &mn, Max: &mx}
		case a.nStr > 0:
			out[name] = Field{Kind: a.stringKind()}
		}
	}
	return out
}

// Merge returns the union of two schemas: existing fields keep their kind
// (stable across re-imports) but widen numeric ranges; new fields are added.
func Merge(old, latest Schema) Schema {
	out := Schema{}
	for k, v := range old {
		out[k] = v
	}
	for k, nv := range latest {
		ov, ok := out[k]
		if !ok {
			out[k] = nv
			continue
		}
		if ov.Kind == Number && nv.Kind == Number {
			ov.Min = minPtr(ov.Min, nv.Min)
			ov.Max = maxPtr(ov.Max, nv.Max)
			out[k] = ov
		}
	}
	return out
}

// TextFields returns the names of all text-kind fields, sorted.
func (s Schema) TextFields() []string {
	var out []string
	for name, f := range s {
		if f.Kind == Text {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// EmbeddingText assembles the string fed to the embedding model: only the
// text-kind fields, label-prefixed and sorted for determinism. If the schema
// has no text fields (e.g. before detection), it falls back to rendering every
// field, so an item is never embedded as an empty string.
func EmbeddingText(fields map[string]any, s Schema) string {
	textFields := s.TextFields()
	if len(textFields) == 0 {
		return renderAll(fields)
	}
	var parts []string
	for _, name := range textFields {
		if v, ok := fields[name]; ok {
			if str := AsString(v); str != "" {
				parts = append(parts, name+": "+str)
			}
		}
	}
	if len(parts) == 0 {
		return renderAll(fields)
	}
	return strings.Join(parts, "\n")
}

// renderAll is the schema-less fallback: every field as "key: value", sorted.
func renderAll(fields map[string]any) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		if v := AsString(fields[k]); v != "" {
			parts = append(parts, k+": "+v)
		}
	}
	return strings.Join(parts, "\n")
}

// defaultWeightFor returns the starting weight for a field by kind: list and
// keyword fields participate (1), numbers stay opt-in (0), text folds into the
// semantic signal (no own weight). Nothing here is tied to a domain.
func defaultWeightFor(k Kind) (float64, bool) {
	switch k {
	case List, Keyword:
		return 1, true
	case Number:
		return 0, true
	default: // Text
		return 0, false
	}
}

// EnsureWeights returns a weight map in which every parameter has a value:
// existing (user-set) weights are preserved, and a sensible default is filled
// in for "semantic" and for any schema field — including newly-detected ones —
// that the user has not weighted yet. This guarantees no parameter is ever left
// without a default, even after a re-import adds fields.
func EnsureWeights(s Schema, existing map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for k, v := range existing {
		out[k] = v
	}
	if _, ok := out["semantic"]; !ok {
		out["semantic"] = 1
	}
	for name, f := range s {
		if _, ok := out[name]; ok {
			continue
		}
		if def, weighted := defaultWeightFor(f.Kind); weighted {
			out[name] = def
		}
	}
	return out
}

// ─── value helpers (shared with rerank) ──────────────────────────────────────

// AsString renders a JSON-decoded value as a plain string for embedding.
func AsString(v any) string {
	switch vv := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(vv)
	case bool:
		return strconv.FormatBool(vv)
	case float64:
		return strconv.FormatFloat(vv, 'g', -1, 64)
	case int:
		return strconv.Itoa(vv)
	case []string:
		return strings.Join(vv, ", ")
	case []any:
		parts := make([]string, 0, len(vv))
		for _, e := range vv {
			if s := AsString(e); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		b, err := json.Marshal(vv)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// AsStringSlice extracts a []string from a list value (native or JSON-decoded).
func AsStringSlice(v any) ([]string, bool) {
	switch vv := v.(type) {
	case []string:
		return vv, true
	case []any:
		out := make([]string, 0, len(vv))
		for _, e := range vv {
			if s, ok := e.(string); ok {
				out = append(out, s)
			} else {
				out = append(out, AsString(e))
			}
		}
		return out, true
	default:
		return nil, false
	}
}

// AsFloat extracts a float64 from a numeric value (native or JSON-decoded).
func AsFloat(v any) (float64, bool) {
	switch vv := v.(type) {
	case float64:
		return vv, true
	case int:
		return float64(vv), true
	default:
		return 0, false
	}
}

func minPtr(a, b *float64) *float64 {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if *b < *a {
		return b
	}
	return a
}

func maxPtr(a, b *float64) *float64 {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if *b > *a {
		return b
	}
	return a
}
