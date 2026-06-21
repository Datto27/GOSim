// Package ingest normalizes arbitrary JSON objects into the adapters.Item
// shape GOSim stores. It is the shared logic behind the `gosim import` CLI
// command and the POST /import HTTP endpoint, so both accept exactly the same
// inputs.
//
// A "collection" is a homogeneous array of JSON objects. Each object is stored
// verbatim as the item's fields; an id, label, and tags are derived from it
// using either explicit field names (Options) or sensible auto-detection. No
// envelope is required — but if an object does carry a nested "fields" object
// (the shape GOSim itself emits), that nested object is used as the fields and
// the surrounding id/label/type/tags are read from the top level.
package ingest

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Datto27/GOSim/internal/adapters"
)

// Options controls how raw objects are mapped to items. Empty fields fall back
// to auto-detection (LabelField) or conventional defaults (IDField "id",
// TagsField "tags").
type Options struct {
	Type       string // collection name; required
	LabelField string // field to use as the display label; "" = auto-detect
	IDField    string // field to use as the id; "" = "id"
	TagsField  string // field holding a string array of tags; "" = "tags"
}

// labelCandidates is the auto-detection order for a display label when
// Options.LabelField is empty.
var labelCandidates = []string{"label", "title", "name", "heading"}

// Normalize converts raw decoded JSON objects into items ready for insertion.
// It returns an error if Type is empty or any object cannot yield an id.
func Normalize(raw []map[string]any, opts Options) ([]adapters.Item, error) {
	if strings.TrimSpace(opts.Type) == "" {
		return nil, fmt.Errorf("ingest: a collection type is required")
	}

	idField := opts.IDField
	if idField == "" {
		idField = "id"
	}
	tagsField := opts.TagsField
	if tagsField == "" {
		tagsField = "tags"
	}

	items := make([]adapters.Item, 0, len(raw))
	seen := make(map[string]int, len(raw))

	for i, obj := range raw {
		if obj == nil {
			return nil, fmt.Errorf("ingest: object %d is null", i+1)
		}

		// Unwrap a nested "fields" object if present (GOSim's own envelope);
		// otherwise the whole object is the fields payload.
		fields := obj
		if nested, ok := obj["fields"].(map[string]any); ok {
			fields = nested
		}

		label := detectLabel(obj, fields, opts.LabelField)

		id := asString(lookup(obj, fields, idField))
		if id == "" {
			if label != "" {
				id = opts.Type + ":" + Slug(label)
			} else {
				id = fmt.Sprintf("%s:item-%d", opts.Type, i+1)
			}
		}
		id = dedupe(id, seen)

		if label == "" {
			label = id
		}

		items = append(items, adapters.Item{
			ID:     id,
			Label:  label,
			Type:   opts.Type,
			Fields: fields,
			Tags:   asStringSlice(lookup(obj, fields, tagsField)),
		})
	}

	return items, nil
}

// detectLabel resolves a display label from an explicit field name, or by
// trying labelCandidates in order.
func detectLabel(obj, fields map[string]any, explicit string) string {
	if explicit != "" {
		return asString(lookup(obj, fields, explicit))
	}
	for _, key := range labelCandidates {
		if s := asString(lookup(obj, fields, key)); s != "" {
			return s
		}
	}
	return ""
}

// lookup returns obj[key] when present, otherwise fields[key]. (When no
// envelope was unwrapped, obj and fields are the same map.)
func lookup(obj, fields map[string]any, key string) any {
	if v, ok := obj[key]; ok {
		return v
	}
	return fields[key]
}

// dedupe ensures id is unique within a single import batch by appending a
// numeric suffix on collision.
func dedupe(id string, seen map[string]int) string {
	if _, exists := seen[id]; !exists {
		seen[id] = 1
		return id
	}
	for {
		seen[id]++
		candidate := fmt.Sprintf("%s-%d", id, seen[id])
		if _, exists := seen[candidate]; !exists {
			seen[candidate] = 1
			return candidate
		}
	}
}

var nonAlphanumRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slug converts a label into a URL-friendly lowercase slug.
func Slug(label string) string {
	s := strings.ToLower(label)
	s = nonAlphanumRE.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// asString coerces a JSON-decoded value to a trimmed string, returning "" for
// anything that isn't a string.
func asString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

// asStringSlice coerces a JSON-decoded value to []string, accepting a native
// []string or a []any of strings. Non-string elements are skipped; a
// non-array value yields an empty (non-nil) slice.
func asStringSlice(v any) []string {
	out := []string{}
	switch vv := v.(type) {
	case []string:
		return vv
	case []any:
		for _, e := range vv {
			if s, ok := e.(string); ok {
				if t := strings.TrimSpace(s); t != "" {
					out = append(out, t)
				}
			}
		}
	}
	return out
}
