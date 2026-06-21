package schema

import "testing"

func TestDetectClassifiesKinds(t *testing.T) {
	items := []map[string]any{
		{"title": "Inception", "plot": "A thief who enters dreams to steal corporate secrets is asked to plant an idea instead.", "genre": []any{"Action", "Sci-Fi"}, "year": 2010.0, "rating": "PG-13"},
		{"title": "Tron Legacy", "plot": "A young man is pulled into a digital world created by his missing father and must escape it.", "genre": []any{"Action", "Sci-Fi"}, "year": 2010.0, "rating": "PG-13"},
		{"title": "Memento", "plot": "A man with no short-term memory hunts the person he believes killed his wife using notes and tattoos.", "genre": []any{"Thriller"}, "year": 2000.0, "rating": "R"},
		{"title": "Sicario", "plot": "An idealistic FBI agent is enlisted by a task force to bring down a Mexican cartel boss.", "genre": []any{"Thriller"}, "year": 2015.0, "rating": "R"},
		{"title": "Arrival", "plot": "A linguist works to communicate with extraterrestrial visitors before global tensions ignite war.", "genre": []any{"Sci-Fi"}, "year": 2016.0, "rating": "PG-13"},
		{"title": "Logan", "plot": "An aging Wolverine protects a young mutant while his own powers fade in a bleak near future.", "genre": []any{"Action"}, "year": 2017.0, "rating": "R"},
	}

	s := Detect(items)

	if s["genre"].Kind != List {
		t.Errorf("genre kind = %q, want list", s["genre"].Kind)
	}
	if s["year"].Kind != Number {
		t.Errorf("year kind = %q, want number", s["year"].Kind)
	}
	if s["plot"].Kind != Text {
		t.Errorf("plot kind = %q, want text", s["plot"].Kind)
	}
	if s["title"].Kind != Text {
		t.Errorf("title kind = %q, want text", s["title"].Kind)
	}
	if s["rating"].Kind != Keyword {
		t.Errorf("rating kind = %q, want keyword (short, low-cardinality)", s["rating"].Kind)
	}
	if s["year"].Min == nil || *s["year"].Min != 2000 || s["year"].Max == nil || *s["year"].Max != 2017 {
		t.Errorf("year range = [%v,%v], want [2000,2017]", deref(s["year"].Min), deref(s["year"].Max))
	}
}

func TestEmbeddingTextUsesOnlyTextFields(t *testing.T) {
	s := Schema{
		"title": {Kind: Text},
		"plot":  {Kind: Text},
		"genre": {Kind: List},
		"year":  {Kind: Number},
	}
	fields := map[string]any{
		"title": "Inception",
		"plot":  "dreams and heists",
		"genre": []any{"Action"},
		"year":  2010.0,
	}
	got := EmbeddingText(fields, s)
	want := "plot: dreams and heists\ntitle: Inception"
	if got != want {
		t.Errorf("EmbeddingText() = %q, want %q (text fields only, sorted)", got, want)
	}
}

func TestEmbeddingTextFallsBackWhenNoTextFields(t *testing.T) {
	// Empty schema → fall back to rendering all fields, so we never embed "".
	got := EmbeddingText(map[string]any{"name": "Aero Lamp"}, Schema{})
	if got == "" {
		t.Fatal("EmbeddingText() returned empty string with no schema; want fallback rendering")
	}
}

func deref(p *float64) float64 {
	if p == nil {
		return -1
	}
	return *p
}
