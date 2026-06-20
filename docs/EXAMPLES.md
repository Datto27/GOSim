# vecsim — Use cases & full walkthrough

See [README.md](../README.md) for setup, CLI/API reference, and architecture.

---

## Use cases

### "More like this" recommendations

You have a movie, book, or album a user just finished and want to suggest what to read, watch, or listen to next. Feed the item's ID into `POST /search` with `"cross_type": true` and get a ranked list of semantically similar items across all domains — no manual tagging or genre matching required.

```bash
vecsim search "Interstellar" --cross-type --limit 5
# → The Three-Body Problem (book, score 0.91)
# → Interstellar Soundtrack (music, score 0.94)
# → Arrival (movie, score 0.89)
# → Project Hail Mary (book, score 0.87)
# → Origin of Symmetry (music, score 0.84)
```

### Mood-based discovery without keywords

A user describes what they're in the mood for in plain language. Use `POST /search/embed` to embed their description directly and search without needing a matching item in the database.

```bash
curl -s localhost:7700/search/embed -d '{
  "text": "something melancholic and introspective, like a quiet rainy afternoon",
  "cross_type": true,
  "limit": 8
}'
# → For Emma, Forever Ago, Norwegian Wood, A Moon Shaped Pool,
#    The Fault in Our Stars, In Rainbows, Eternal Sunshine …
```

### Backend similarity microservice

Any other application — a web app, a recommendation engine, a data pipeline — can call vecsim's HTTP API over localhost without knowing anything about embeddings or Postgres. The caller just sends a label or free-text query and receives ranked results with scores. vecsim acts as a self-contained semantic search sidecar.

### Extending to new content domains

Add podcasts, recipes, video games, or products by creating a single new file in `internal/adapters/` that implements the `Adapter` interface. The new type immediately participates in seeding, indexing, search, and cross-type queries — the rest of the codebase requires zero changes.

```go
// internal/adapters/podcast.go
func init() { Register(&PodcastAdapter{}) }

func (a *PodcastAdapter) Type() string { return "podcast" }
func (a *PodcastAdapter) Seeds() []SeedItem { /* 25 episodes */ }
func (a *PodcastAdapter) BuildText(fields map[string]any) string {
    return fmt.Sprintf("%s — %s. Topics: %s. %s",
        fields["title"], fields["host"],
        joinOr(stringSlice(fields["topics"]), ""),
        fields["description"])
}
```

### Local, private search — no API keys or cloud dependency

All embedding and search happens on your own machine after the initial `ollama pull`. No data leaves your environment, no rate limits, no per-request cost. Suitable for personal media libraries, private research collections, or air-gapped environments.

### Seeding and testing a vector search pipeline

vecsim's 75 curated seed items (with intentional thematic cross-type overlap) make it a useful fixture for testing embedding quality, tuning HNSW index parameters, or benchmarking cosine similarity performance without building your own dataset.

---

## Full walkthrough

A complete example session from a fresh checkout to a working cross-type search, using the **optimized** profile (recommended for most machines).

### 1 — Setup

```
$ go run . setup --pull

╔══════════════════════════════════════════╗
║         vecsim — first-run setup         ║
╚══════════════════════════════════════════╝

Choose an embedding profile:

  [1] max       — qwen3-embedding:8b  (4096 dims)
                  Best quality. 6-8 GB RAM, GPU recommended.
                  ~4.7 GB download.

  [2] optimized — nomic-embed-text    (768 dims)
                  Great quality. Runs on any modern laptop, ~600 MB RAM.
                  ~274 MB download.  ← Recommended for most users

  Profile (1 or 2) [2]: 2
  → optimized selected (nomic-embed-text, 768 dims)

Connection details:
  Ollama URL [http://localhost:11434]:
  Database URL [postgres://vecsim:vecsim@localhost:5432/vecsim?sslmode=disable]:
  API port [7700]:

Checking connectivity…
  ✓ Ollama at http://localhost:11434 — reachable
  ✓ Postgres — connected

  Config saved to /home/user/.config/vecsim/config.json

Pulling nomic-embed-text…
  pulling manifest: 100%

  ✓ Model ready
```

### 2 — Start Postgres, migrate, seed, index

```
$ docker compose up -d
$ go run . migrate
Migrations applied (profile: optimized, dims: 768)
Profile recorded in vecsim_meta

$ go run . seed --type all
75 items processed: 75 inserted, 0 skipped

$ go run . index --type all
Indexing movie: 25/25
Indexing music: 25/25
Indexing book: 25/25

$ go run . stats
TYPE    TOTAL  EMBEDDED  COVERAGE
────    ─────  ────────  ────────
book    25     25        100.0%
movie   25     25        100.0%
music   25     25        100.0%
────    ─────  ────────  ────────
TOTAL   75     75        100.0%
```

### 3 — Search within a type

```
$ go run . search "Inception" --limit 5
Similar to "Inception" [movie]:

RANK  TYPE   LABEL                                  SCORE
────  ────   ─────                                  ─────
1     movie  Memento                                0.8923
2     movie  Arrival                                0.8741
3     movie  The Matrix                             0.8634
4     movie  Eternal Sunshine of the Spotless Mind  0.8501
5     movie  The Truman Show                        0.8388
```

### 4 — Cross-type search

The same query, now searching across all three domains simultaneously:

```
$ go run . search "Inception" --cross-type --limit 10
Similar to "Inception" [movie] (cross-type):

RANK  TYPE   LABEL                                  SCORE
────  ────   ─────                                  ─────
1     movie  Memento                                0.8923
2     book   Recursion                              0.8811
3     music  Time                                   0.8794
4     movie  Arrival                                0.8741
5     book   Dark Matter                            0.8702
6     music  Interstellar (Soundtrack)              0.8659
7     movie  The Matrix                             0.8634
8     book   Kafka on the Shore                     0.8597
9     music  Lateralus                              0.8521
10    movie  Eternal Sunshine of the Spotless Mind  0.8501
```

Inception (2010 film) surfaces a Blake Crouch novel, a Hans Zimmer track, and a Tool album — not because they share genre tags, but because the embedding model encodes shared themes of memory, altered perception, and layered reality.

### 5 — HTTP API

```
$ go run . serve &

$ curl -s localhost:7700/health | jq
{ "profile": "optimized", "model": "nomic-embed-text", "dimensions": 768, "ollama_ok": true, "db_ok": true }

$ curl -s localhost:7700/search \
    -H 'Content-Type: application/json' \
    -d '{"label":"Inception","cross_type":true,"limit":3}' \
  | jq '.results[] | {label:.item.label, type:.item.type, score}'
{ "label": "Memento",   "type": "movie", "score": 0.8923 }
{ "label": "Recursion", "type": "book",  "score": 0.8811 }
{ "label": "Time",      "type": "music", "score": 0.8794 }
```

### 6 — Free-text search (no existing item needed)

```
$ curl -s localhost:7700/search/embed \
    -H 'Content-Type: application/json' \
    -d '{"text":"a melancholic coming-of-age story about friendship and nostalgia","cross_type":true,"limit":3}' \
  | jq '.results[] | {label:.item.label, type:.item.type, score}'
{ "label": "Stand By Me", "type": "movie", "score": 0.9102 }
{ "label": "The Body",    "type": "book",  "score": 0.9057 }
{ "label": "Stand By Me", "type": "music", "score": 0.8934 }
```

A free-text description matched the movie, the Stephen King novella it's based on, and Ben E. King's song — all from a single query with no item ID required.

### 7 — Add a new item via API

```
$ curl -s localhost:7700/items \
    -H 'Content-Type: application/json' \
    -d '{
      "label": "Annihilation",
      "type": "movie",
      "fields": {
        "title": "Annihilation", "year": 2018,
        "genre": ["Sci-Fi", "Horror"],
        "cast": ["Natalie Portman", "Jennifer Jason Leigh"],
        "plot": "A biologist signs up for a dangerous expedition into an environmental disaster zone where the laws of nature do not apply."
      },
      "tags": ["sci-fi", "horror", "mind-bending", "identity"]
    }' | jq '{id, label, embedded}'
{ "id": "movie:annihilation", "label": "Annihilation", "embedded": true }
```

The item is inserted and embedded in a single request — immediately searchable.
