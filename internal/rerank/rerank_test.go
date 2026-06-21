package rerank

import (
	"testing"

	"github.com/Datto27/GOSim/internal/adapters"
	"github.com/Datto27/GOSim/internal/schema"
	"github.com/Datto27/GOSim/internal/store"
)

var movieSchema = schema.Schema{
	"genre": {Kind: schema.List},
	"cast":  {Kind: schema.List},
	"plot":  {Kind: schema.Text},
	"year":  {Kind: schema.Number, Min: f(2000), Max: f(2020)},
}

func TestScore_NoStructuralWeightIsPureSemantic(t *testing.T) {
	q := adapters.Item{Fields: map[string]any{"genre": []any{"Sci-Fi"}}}
	c := adapters.Item{Fields: map[string]any{"genre": []any{"Sci-Fi"}}}

	// Only "semantic" weighted → score equals the semantic score exactly.
	got := Score(q, c, 0.8, movieSchema, map[string]float64{"semantic": 1})
	if diff := got - 0.8; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("Score() = %v, want 0.8", got)
	}
}

func TestScore_SharedSingleCastMemberCountsStrongly(t *testing.T) {
	// The Inception → Shutter Island case: one shared actor (DiCaprio) out of
	// large casts must be a strong signal, not ~0.09 like Jaccard.
	q := adapters.Item{Fields: map[string]any{
		"cast": []any{"Leonardo DiCaprio", "Joseph Gordon-Levitt", "Elliot Page", "Tom Hardy", "Ken Watanabe", "Cillian Murphy"},
	}}
	c := adapters.Item{Fields: map[string]any{
		"cast": []any{"Leonardo DiCaprio", "Mark Ruffalo", "Ben Kingsley", "Michelle Williams"},
	}}

	got := Score(q, c, 0.0, movieSchema, map[string]float64{"semantic": 0, "cast": 1})
	if got < 0.49 || got > 0.51 {
		t.Fatalf("one shared cast member scored %v, want ~0.5 (shared/(shared+1))", got)
	}
}

func TestScore_GenreVsCastWeightingShiftsRanking(t *testing.T) {
	query := adapters.Item{Fields: map[string]any{
		"genre": []any{"Action", "Sci-Fi"},
		"cast":  []any{"Leonardo DiCaprio"},
	}}
	// sameGenre: perfect genre match, no shared cast (like Tron Legacy).
	sameGenre := adapters.Item{Fields: map[string]any{
		"genre": []any{"Action", "Sci-Fi"},
		"cast":  []any{"Garrett Hedlund"},
	}}
	// sameCast: shares the lead actor, different genre (like Shutter Island).
	sameCast := adapters.Item{Fields: map[string]any{
		"genre": []any{"Thriller"},
		"cast":  []any{"Leonardo DiCaprio"},
	}}

	results := func() []store.SearchResult {
		return []store.SearchResult{
			{Item: sameGenre, Score: 0.5},
			{Item: sameCast, Score: 0.5},
		}
	}

	// Weight cast heavily → the shared-actor film should win.
	castFirst := Rerank(query, results(), movieSchema, map[string]float64{"semantic": 0, "cast": 5, "genre": 1}, 2)
	if castFirst[0].Item.Fields["genre"].([]any)[0] != "Thriller" {
		t.Fatalf("with cast=5 the shared-actor film should rank first, got genre %v", castFirst[0].Item.Fields["genre"])
	}

	// Weight genre heavily → the same-genre film should win.
	genreFirst := Rerank(query, results(), movieSchema, map[string]float64{"semantic": 0, "cast": 1, "genre": 5}, 2)
	if genreFirst[0].Item.Fields["genre"].([]any)[0] != "Action" {
		t.Fatalf("with genre=5 the same-genre film should rank first, got genre %v", genreFirst[0].Item.Fields["genre"])
	}
}

func TestScore_NumberSimUsesRange(t *testing.T) {
	q := adapters.Item{Fields: map[string]any{"year": 2010.0}}
	c := adapters.Item{Fields: map[string]any{"year": 2015.0}}
	// range 2000..2020 (20); |Δ|=5 → 1 - 5/20 = 0.75.
	got := Score(q, c, 0.0, movieSchema, map[string]float64{"semantic": 0, "year": 1})
	if got < 0.74 || got > 0.76 {
		t.Fatalf("year sim = %v, want ~0.75", got)
	}
}

func f(v float64) *float64 { return &v }
