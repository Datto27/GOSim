package rerank

import (
	"testing"

	"github.com/Datto27/vecsim/internal/adapters"
	"github.com/Datto27/vecsim/internal/store"
)

func TestScore_NoWeightsEqualsSemanticAverage(t *testing.T) {
	query := adapters.Item{
		Tags:   []string{"a", "b"},
		Fields: map[string]any{"year": 2010.0},
	}
	candidate := adapters.Item{
		Tags:   []string{"a", "b"},
		Fields: map[string]any{"year": 2010.0},
	}

	// All comparable fields are identical (tags Jaccard=1, year sim=1) and
	// semantic=0.8, all with implicit weight 1, so the average must be 1.
	got := Score(query, candidate, 0.8, nil)
	want := (0.8 + 1.0 + 1.0) / 3.0
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("Score() = %v, want %v", got, want)
	}
}

func TestScore_ZeroedFieldIsExcluded(t *testing.T) {
	query := adapters.Item{
		Fields: map[string]any{"genre": []string{"scifi"}, "cast": []string{"alice"}},
	}
	candidate := adapters.Item{
		Fields: map[string]any{"genre": []string{"drama"}, "cast": []string{"alice"}},
	}

	weights := map[string]float64{"cast": 0, "genre": 1}
	got := Score(query, candidate, 0.5, weights)
	// genre sim=0 (no overlap), cast weight=0 so excluded, semantic weight=1
	// (default) => (1*0.5 + 1*0) / (1+1) = 0.25
	want := 0.25
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("Score() = %v, want %v", got, want)
	}
}

func TestScore_AllWeightsZeroFallsBackToSemantic(t *testing.T) {
	query := adapters.Item{Fields: map[string]any{"genre": []string{"scifi"}}}
	candidate := adapters.Item{Fields: map[string]any{"genre": []string{"drama"}}}

	weights := map[string]float64{"semantic": 0, "genre": 0}
	got := Score(query, candidate, 0.42, weights)
	if got != 0.42 {
		t.Fatalf("Score() = %v, want semanticScore 0.42", got)
	}
}

func TestScore_MismatchedTypeFieldSkipped(t *testing.T) {
	query := adapters.Item{Fields: map[string]any{"year": 2010.0}}
	candidate := adapters.Item{Fields: map[string]any{"year": "not-a-number"}}

	// year is incomparable (type mismatch), so only semantic remains.
	got := Score(query, candidate, 0.9, nil)
	if got != 0.9 {
		t.Fatalf("Score() = %v, want 0.9 (semantic only)", got)
	}
}

func TestRerank_NoWeightsIsPassthroughTruncated(t *testing.T) {
	query := adapters.Item{}
	results := []store.SearchResult{
		{Item: adapters.Item{Label: "a"}, Score: 0.9},
		{Item: adapters.Item{Label: "b"}, Score: 0.8},
		{Item: adapters.Item{Label: "c"}, Score: 0.7},
	}

	got := Rerank(query, results, nil, 2)
	if len(got) != 2 || got[0].Item.Label != "a" || got[1].Item.Label != "b" {
		t.Fatalf("Rerank() = %+v, want first two results unchanged", got)
	}
}

func TestRerank_WeightedReordersResults(t *testing.T) {
	query := adapters.Item{Fields: map[string]any{"genre": []string{"scifi"}}}
	results := []store.SearchResult{
		{Item: adapters.Item{Label: "high-semantic-low-genre", Fields: map[string]any{"genre": []string{"drama"}}}, Score: 0.9},
		{Item: adapters.Item{Label: "low-semantic-high-genre", Fields: map[string]any{"genre": []string{"scifi"}}}, Score: 0.5},
	}

	weights := map[string]float64{"genre": 10, "semantic": 0}
	got := Rerank(query, results, weights, 2)
	if got[0].Item.Label != "low-semantic-high-genre" {
		t.Fatalf("Rerank() top result = %q, want genre match to outrank higher semantic score", got[0].Item.Label)
	}
}
