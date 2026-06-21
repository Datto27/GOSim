// Package rerank implements GOSim's hybrid ranking: it blends the semantic
// (embedding) similarity with per-field structured similarity, weighted by a
// user-supplied, per-collection weight map and driven entirely by the detected
// schema. It hardcodes no field names — list, number and keyword fields are
// handled by their detected kind.
package rerank

import (
	"sort"
	"strings"

	"github.com/Datto27/GOSim/internal/adapters"
	"github.com/Datto27/GOSim/internal/schema"
	"github.com/Datto27/GOSim/internal/store"
)

// SemanticKey is the weight-map key for the embedding (cosine) similarity. It
// always participates, defaulting to weight 1 when unspecified.
const SemanticKey = "semantic"

// TagsKey is the weight-map key for overlap between the items' Tags slices
// (which live outside Fields, so they are not part of the detected schema).
const TagsKey = "tags"

// Rerank re-scores results with the hybrid score when any structured weight is
// active, then truncates to limit. With only the semantic weight (or none), it
// is a passthrough, preserving plain vector-search order.
func Rerank(query adapters.Item, results []store.SearchResult, sch schema.Schema, weights map[string]float64, limit int) []store.SearchResult {
	if HasStructuralWeight(weights) {
		for i := range results {
			results[i].Score = Score(query, results[i].Item, results[i].Score, sch, weights)
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

// HasStructuralWeight reports whether any weight other than "semantic" is
// positive — i.e. whether re-ranking depends on structured fields. When true,
// callers must rerank the whole collection rather than only the semantic
// top-N, since a structurally-similar item may not be semantically near.
func HasStructuralWeight(weights map[string]float64) bool {
	for k, w := range weights {
		if k != SemanticKey && w > 0 {
			return true
		}
	}
	return false
}

// Score blends the semantic similarity with per-field structured similarity.
// Each signal participates only if it has positive weight and is comparable
// between query and candidate. semantic defaults to weight 1. Field similarity
// is chosen by the field's detected kind:
//   - list    → strong overlap (any shared value counts; see overlapStrong)
//   - number  → range-normalized closeness
//   - keyword → exact (case-insensitive) match
//   - text    → ignored here (already captured by the semantic vector)
//
// The result is the weighted average of active signals, or semanticScore if no
// structured signal applies.
func Score(query, candidate adapters.Item, semanticScore float64, sch schema.Schema, weights map[string]float64) float64 {
	semW, ok := weights[SemanticKey]
	if !ok {
		semW = 1.0
	}
	weightedSum := semW * semanticScore
	totalWeight := semW

	for name, w := range weights {
		if name == SemanticKey || w <= 0 {
			continue
		}

		var (
			sim float64
			ok  bool
		)
		if name == TagsKey {
			sim, ok = overlapStrong(query.Tags, candidate.Tags)
		} else {
			sim, ok = fieldSimilarity(name, query, candidate, sch)
		}
		if !ok {
			continue
		}
		weightedSum += w * sim
		totalWeight += w
	}

	if totalWeight <= 0 {
		return semanticScore
	}
	return weightedSum / totalWeight
}

// fieldSimilarity computes the similarity of one schema field between query and
// candidate. ok is false when the field is absent from either item, is a text
// field (handled by the embedding), or is unknown to the schema.
func fieldSimilarity(name string, query, candidate adapters.Item, sch schema.Schema) (float64, bool) {
	f, known := sch[name]
	if !known {
		return 0, false
	}
	qv, qok := query.Fields[name]
	cv, cok := candidate.Fields[name]
	if !qok || !cok {
		return 0, false
	}

	switch f.Kind {
	case schema.List:
		qs, ok1 := schema.AsStringSlice(qv)
		cs, ok2 := schema.AsStringSlice(cv)
		if !ok1 || !ok2 {
			return 0, false
		}
		return overlapStrong(qs, cs)
	case schema.Number:
		qf, ok1 := schema.AsFloat(qv)
		cf, ok2 := schema.AsFloat(cv)
		if !ok1 || !ok2 {
			return 0, false
		}
		return numberSim(qf, cf, f.Min, f.Max), true
	case schema.Keyword:
		return keywordSim(qv, cv)
	default: // schema.Text — embedded, not compared here
		return 0, false
	}
}

// overlapStrong scores set overlap so that any shared value counts strongly:
// identical sets → 1, otherwise shared/(shared+1) (1 shared ≈ 0.5, 2 ≈ 0.67).
// ok is false when either side is empty (nothing to compare).
func overlapStrong(a, b []string) (float64, bool) {
	if len(a) == 0 || len(b) == 0 {
		return 0, false
	}
	aset := toLowerSet(a)
	bset := toLowerSet(b)

	shared := 0
	for v := range aset {
		if _, ok := bset[v]; ok {
			shared++
		}
	}
	if shared == len(aset) && shared == len(bset) {
		return 1.0, true
	}
	if shared == 0 {
		return 0.0, true
	}
	return float64(shared) / float64(shared+1), true
}

// numberSim returns 1 for equal values and decays linearly with the absolute
// difference, normalized by the field's observed range. Without a usable range
// it falls back to 1/(1+|Δ|).
func numberSim(a, b float64, min, max *float64) float64 {
	d := a - b
	if d < 0 {
		d = -d
	}
	if min != nil && max != nil && *max > *min {
		s := 1 - d/(*max-*min)
		if s < 0 {
			return 0
		}
		return s
	}
	return 1 / (1 + d)
}

// keywordSim returns 1 for a case-insensitive exact match, else 0. ok is false
// when the values aren't strings.
func keywordSim(a, b any) (float64, bool) {
	as, ok1 := a.(string)
	bs, ok2 := b.(string)
	if !ok1 || !ok2 {
		return 0, false
	}
	if strings.EqualFold(strings.TrimSpace(as), strings.TrimSpace(bs)) {
		return 1.0, true
	}
	return 0.0, true
}

func toLowerSet(xs []string) map[string]struct{} {
	out := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		out[strings.ToLower(strings.TrimSpace(x))] = struct{}{}
	}
	return out
}
