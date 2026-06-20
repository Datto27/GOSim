// Package rerank adds optional per-field weighting on top of vecsim's
// cosine-similarity search. Given a wider candidate pool fetched by
// pgvector, it recomputes a weighted blend of the semantic score and
// structured-field similarity (read directly off adapters.Item.Fields and
// .Tags) and re-sorts. It has no dependency on any per-domain adapter code:
// fields are compared generically by their Go type after JSON decoding, so
// adding a new content domain requires no changes here.
package rerank

import (
	"sort"
	"strings"

	"github.com/Datto27/vecsim/internal/adapters"
	"github.com/Datto27/vecsim/internal/store"
)

// SemanticKey is the weight map key referring to the existing cosine
// similarity score (always present, even when query and candidate share no
// comparable structured fields).
const SemanticKey = "semantic"

// TagsKey is the weight map key referring to Jaccard similarity between the
// query's and candidate's Tags slices (always present and comparable across
// content types).
const TagsKey = "tags"

// Rerank returns the top limit results from results, re-scored by a
// weighted blend of similarity signals when weights is non-empty. When
// weights is empty, it is a no-op passthrough (besides truncating to
// limit), so callers that pass no weights see identical behavior to plain
// vector search.
func Rerank(query adapters.Item, results []store.SearchResult, weights map[string]float64, limit int) []store.SearchResult {
	if len(weights) > 0 {
		for i := range results {
			results[i].Score = Score(query, results[i].Item, results[i].Score, weights)
		}
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})
	}
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// Score combines semanticScore (the cosine similarity already computed by
// SearchByVector) with structured-field similarity between query and
// candidate, weighted by weights. A field missing from weights defaults to
// a weight of 1 (equal weighting), so omitting weights entirely reproduces
// plain semantic ranking. If every comparable field ends up with zero total
// weight, semanticScore is returned so results never go unordered.
func Score(query, candidate adapters.Item, semanticScore float64, weights map[string]float64) float64 {
	sims := map[string]float64{SemanticKey: semanticScore}
	if s, ok := jaccard(query.Tags, candidate.Tags); ok {
		sims[TagsKey] = s
	}
	for name, qv := range query.Fields {
		cv, present := candidate.Fields[name]
		if !present {
			continue
		}
		if s, ok := fieldSimilarity(qv, cv); ok {
			sims[name] = s
		}
	}

	var weightedSum, totalWeight float64
	for name, sim := range sims {
		w, ok := weights[name]
		if !ok {
			w = 1.0
		}
		weightedSum += w * sim
		totalWeight += w
	}
	if totalWeight <= 0 {
		return semanticScore
	}
	return weightedSum / totalWeight
}

// fieldSimilarity compares two field values by their Go type, returning
// ok=false when the types don't match or aren't a kind this package knows
// how to compare.
func fieldSimilarity(a, b any) (float64, bool) {
	if af, ok := asFloat(a); ok {
		if bf, ok := asFloat(b); ok {
			return 1 / (1 + abs(af-bf)), true
		}
		return 0, false
	}
	if as, ok := asStringSlice(a); ok {
		if bs, ok := asStringSlice(b); ok {
			return jaccard(as, bs)
		}
		return 0, false
	}
	if as, ok := a.(string); ok {
		if bs, ok := b.(string); ok {
			if strings.EqualFold(as, bs) {
				return 1.0, true
			}
			return 0.0, true
		}
		return 0, false
	}
	return 0, false
}

// jaccard returns the Jaccard similarity of two string sets. It returns
// ok=false only when both inputs are empty (nothing to compare).
func jaccard(a, b []string) (float64, bool) {
	if len(a) == 0 && len(b) == 0 {
		return 0, false
	}
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		set[strings.ToLower(v)] = struct{}{}
	}
	bSet := make(map[string]struct{}, len(b))
	for _, v := range b {
		bSet[strings.ToLower(v)] = struct{}{}
	}

	intersection := 0
	union := len(bSet)
	for v := range set {
		if _, ok := bSet[v]; ok {
			intersection++
		} else {
			union++
		}
	}
	if union == 0 {
		return 0, false
	}
	return float64(intersection) / float64(union), true
}

// asFloat extracts a float64 from a numeric field value, which after a JSON
// round trip through JSONB always arrives as float64, but may be a native
// int when built in-process (e.g. from seed data).
func asFloat(v any) (float64, bool) {
	switch vv := v.(type) {
	case float64:
		return vv, true
	case int:
		return float64(vv), true
	default:
		return 0, false
	}
}

// asStringSlice extracts a []string from a field value that may be either a
// native []string or a []interface{} of strings (round-tripped through
// JSONB).
func asStringSlice(v any) ([]string, bool) {
	switch vv := v.(type) {
	case []string:
		return vv, true
	case []any:
		out := make([]string, 0, len(vv))
		for _, e := range vv {
			s, ok := e.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
