# vecsim

Universal semantic similarity search engine written in Go. Indexes movies, music, and books as vector embeddings and finds the most similar items to any query — including across content types, so a movie can match a book or a song that shares the same mood, themes, or narrative style.

Runs entirely offline after initial setup. Uses **Ollama** for local embedding generation and **PostgreSQL + pgvector** for vector storage and cosine-similarity search. Exposes both a CLI and a local HTTP API on `localhost:7700`.

---

## How it works

Each item's text fields are combined into a descriptive string and sent to Ollama's embedding API, which returns a `float32` vector. That vector is stored in Postgres as a `vector(N)` column. At search time, the query item's stored vector is compared against all others using cosine distance via pgvector's `<=>` operator and an HNSW index. The top K results are returned ranked by similarity score. No similarity math runs in Go — it all happens inside a single SQL query.

Because every item — movie, album, book — lives in the same embedding space, cross-type search works out of the box. Searching for *Inception* can return *Recursion* (book), *Time* by Hans Zimmer (music), and *Memento* (movie) in a single ranked list.

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

This is a complete example session from a fresh checkout to a working cross-type search, using the **optimized** profile (recommended for most machines).

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

Next steps:
  1. docker compose up -d   (if Postgres isn't running yet)
  2. vecsim migrate          (create tables & HNSW index)
  3. vecsim seed --type all  (load 75 seed items)
  4. vecsim index --type all (generate embeddings)
  5. vecsim search "Inception" --cross-type
     vecsim serve            (start the HTTP API on localhost:7700)
```

### 2 — Start Postgres

```
$ docker compose up -d
[+] Running 2/2
 ✔ Volume "vecsim_pgdata"  Created
 ✔ Container gosim-postgres-1  Started
```

### 3 — Migrate

```
$ go run . migrate
Migrations applied (profile: optimized, dims: 768)
Profile recorded in vecsim_meta
```

At this point the database has:
- `items` table with `embedding vector(768)` column
- HNSW cosine index (`items_embedding_hnsw_idx`)
- `vecsim_meta` table with `profile = optimized`

### 4 — Seed

```
$ go run . seed --type all
TYPE    ID                                    STATUS
────    ──                                    ──────
movie   movie:inception                       inserted
movie   movie:the-matrix                      inserted
movie   movie:interstellar                    inserted
...
music   music:time-hans-zimmer                inserted
music   music:ok-computer-radiohead           inserted
...
book    book:recursion                        inserted
book    book:dark-matter                      inserted
...

75 items processed: 75 inserted, 0 skipped
```

Re-running is safe — all 75 report `skipped`.

### 5 — Index

```
$ go run . index --type all
Indexing movie: 25/25
Indexing music: 25/25
Indexing book: 25/25
```

Ollama is called in batches of 20 items. Each batch makes one HTTP request to `/api/embed` and stores the resulting vectors. Total time depends on hardware — on a modern laptop with `nomic-embed-text` this takes roughly 10–30 seconds.

### 6 — Check stats

```
$ go run . stats
TYPE    TOTAL  EMBEDDED  COVERAGE
────    ─────  ────────  ────────
book    25     25        100.0%
movie   25     25        100.0%
music   25     25        100.0%
────    ─────  ────────  ────────
TOTAL   75     75        100.0%
```

### 7 — Search within a type

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

### 8 — Cross-type search

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

### 9 — Start the HTTP API

```
$ go run . serve
time=2026-06-16T22:00:00Z level=INFO msg="vecsim HTTP API started" addr=localhost:7700
```

In another terminal:

```
$ curl -s localhost:7700/health | jq
{
  "profile": "optimized",
  "model": "nomic-embed-text",
  "dimensions": 768,
  "ollama_ok": true,
  "db_ok": true
}

$ curl -s localhost:7700/stats | jq '.by_type'
{
  "book":  { "count": 25, "embedded": 25 },
  "movie": { "count": 25, "embedded": 25 },
  "music": { "count": 25, "embedded": 25 }
}

$ curl -s localhost:7700/search \
    -H 'Content-Type: application/json' \
    -d '{"label":"Inception","cross_type":true,"limit":3}' | jq '.results[] | {label:.item.label, type:.item.type, score}'
{
  "label": "Memento",
  "type": "movie",
  "score": 0.8923
}
{
  "label": "Recursion",
  "type": "book",
  "score": 0.8811
}
{
  "label": "Time",
  "type": "music",
  "score": 0.8794
}
```

### 10 — Free-text search (no existing item needed)

```
$ curl -s localhost:7700/search/embed \
    -H 'Content-Type: application/json' \
    -d '{
      "text": "a melancholic coming-of-age story about friendship and nostalgia",
      "cross_type": true,
      "limit": 5
    }' | jq '.results[] | {label:.item.label, type:.item.type, score}'
{
  "label": "Stand By Me",
  "type": "movie",
  "score": 0.9102
}
{
  "label": "The Body",
  "type": "book",
  "score": 0.9057
}
{
  "label": "Stand By Me",
  "type": "music",
  "score": 0.8934
}
{
  "label": "The Suburbs",
  "type": "music",
  "score": 0.8811
}
{
  "label": "Norwegian Wood",
  "type": "book",
  "score": 0.8743
}
```

A free-text description matched the movie, the Stephen King novella it's based on, Ben E. King's song, and two other coming-of-age items — all from a single query with no item ID required.

### 11 — Add a new item via API

```
$ curl -s localhost:7700/items \
    -H 'Content-Type: application/json' \
    -d '{
      "label": "Annihilation",
      "type": "movie",
      "fields": {
        "title": "Annihilation",
        "year": 2018,
        "genre": ["Sci-Fi", "Horror"],
        "cast": ["Natalie Portman", "Jennifer Jason Leigh"],
        "plot": "A biologist signs up for a dangerous expedition into an environmental disaster zone where the laws of nature do not apply."
      },
      "tags": ["sci-fi", "horror", "mind-bending", "identity"]
    }' | jq '{id, label, embedded}'
{
  "id": "movie:annihilation",
  "label": "Annihilation",
  "embedded": true
}
```

The item is inserted and embedded in a single request. It is immediately searchable.

---

## Embedding profiles

Chosen once during `vecsim setup`. Governs the entire installation.

| Profile | Model | Dims | RAM | Download |
|---|---|---|---|---|
| `optimized` | `nomic-embed-text` | 768 | ~600 MB, any modern laptop | ~274 MB |
| `max` | `qwen3-embedding:8b` | 4096 | 6–8 GB, GPU recommended | ~4.7 GB |

The chosen profile is stored in both `~/.config/vecsim/config.json` and the `vecsim_meta` Postgres table. The two are validated on every startup to prevent dimension mismatches.

---

## Prerequisites

- **Go 1.22+**
- **Ollama** running locally — [ollama.com](https://ollama.com)
- **Docker + Docker Compose** (for the Postgres/pgvector container)

---

## Quick start

```bash
# 1. Clone and enter the project
git clone https://github.com/Datto27/vecsim
cd vecsim

# 2. Run the interactive setup wizard
#    --pull downloads the embedding model automatically
go run . setup --pull

# 3. Start Postgres with the pgvector extension
docker compose up -d

# 4. Create tables and HNSW index
go run . migrate

# 5. Load 75 curated seed items (25 movies, 25 albums, 25 books)
go run . seed --type all

# 6. Generate embeddings (calls Ollama in batches of 20)
go run . index --type all

# 7. Search!
go run . search "Inception" --cross-type --limit 10

# 8. Start the HTTP API on localhost:7700
go run . serve
```

Or build the binary once:

```bash
go build -o vecsim .
./vecsim setup --pull
```

---

## CLI reference

### `vecsim setup [--pull]`

Interactive first-run wizard. Presents both embedding profiles with hardware requirements, prompts for Ollama URL, database URL, and API port (all with sensible defaults), checks connectivity, and writes `~/.config/vecsim/config.json`.

`--pull` streams `ollama pull <model>` live with download progress.

### `vecsim migrate`

Runs Goose database migrations. Creates the `items` table with a `vector(N)` column sized for the active profile, the `vecsim_meta` table, a B-tree index on `type`, and an HNSW cosine-distance index on `embedding`. Records the active profile in `vecsim_meta`.

Safe to re-run — Goose is idempotent.

### `vecsim seed [--type movie|music|book|all]`

Inserts the 25 hardcoded seed items for a domain (default: all three). Already-existing IDs are skipped. Prints a status table.

### `vecsim index [--type movie|music|book|all]`

Finds all items where `embedding IS NULL`, calls Ollama in batches of 20, and stores the resulting vectors. Shows live progress per type. Safe to re-run — already-indexed items are never re-embedded.

### `vecsim search <title> [--type movie|music|book] [--cross-type] [--limit N]`

Fetches the stored embedding for `<title>` and returns the top N most similar items ranked by cosine similarity score (1.0 = identical). Use `--cross-type` to search across all domains simultaneously. Use `--type` to disambiguate if a label matches items in multiple types.

### `vecsim serve [--port N]`

Starts the HTTP API server on `localhost:<port>` (default: value from config, fallback `7700`). Handles graceful shutdown on SIGINT/SIGTERM with a 10-second drain window.

### `vecsim list [--type] [--limit N] [--offset N]`

Paginated table of items showing ID, type, label, tags, and whether embedding has been generated.

### `vecsim stats`

Per-type item counts and embedding coverage percentage, plus overall totals.

### `vecsim info`

Shows active profile, model, dimensions, config file path, Ollama reachability, database connectivity, profile match status, and API port.

---

## HTTP API

Base URL: `http://localhost:7700`

All responses are JSON. Errors return `{"error": "message"}` with an appropriate HTTP status. Every response includes an `X-Request-ID` header and `Access-Control-Allow-Origin: *`.

### Health

```
GET /health
```
```json
{
  "profile": "optimized",
  "model": "nomic-embed-text",
  "dimensions": 768,
  "ollama_ok": true,
  "db_ok": true
}
```

### Items

```
GET    /items          ?type=movie&limit=20&offset=0
GET    /items/{id}
POST   /items
DELETE /items/{id}
```

`POST /items` request body:
```json
{
  "label": "Dune: Part Two",
  "type": "movie",
  "fields": {
    "title": "Dune: Part Two",
    "year": 2024,
    "genre": ["Sci-Fi", "Adventure"],
    "cast": ["Timothee Chalamet", "Zendaya"],
    "plot": "Paul Atreides unites with the Fremen to wage war against the Harkonnens."
  },
  "tags": ["sci-fi", "epic", "desert"]
}
```

The server generates an ID from the label (`movie:dune-part-two`) and immediately attempts to embed the item. The response includes `"embedded": true|false`.

### Search

```
POST /search
```
```json
{
  "label": "Inception",
  "cross_type": true,
  "limit": 10
}
```

Use `"id"` instead of `"label"` for an unambiguous lookup. `"type"` scopes the label resolution when a label matches multiple types.

```
POST /search/embed
```

Search with free text instead of a stored item:
```json
{
  "text": "a mind-bending thriller about memory and identity",
  "cross_type": true,
  "limit": 10
}
```

Response for both search endpoints:
```json
{
  "query": { "id": "movie:inception", "label": "Inception", ... },
  "results": [
    { "item": { "id": "book:recursion", "label": "Recursion", ... }, "score": 0.9214 },
    { "item": { "id": "music:time-hans-zimmer", "label": "Time", ... }, "score": 0.8876 }
  ]
}
```

### Stats and background indexing

```
GET  /stats    → per-type counts and embedding coverage
POST /index    → triggers background indexing of all unembedded items; returns 202 immediately
```

---

## Seed data

75 hand-curated items designed so cross-type search produces compelling thematic clusters:

| Cluster | Movies | Music | Books |
|---|---|---|---|
| Mind-bending / memory | Inception, Memento, Arrival, Eternal Sunshine | Time (Zimmer), Interstellar OST, Lateralus | Recursion, Dark Matter, Kafka on the Shore |
| Dystopian | Blade Runner 2049, Children of Men, Mad Max | OK Computer, Kid A, Blade Runner 2049 OST | 1984, Brave New World, Never Let Me Go, Fahrenheit 451 |
| Epic adventure | Dune, LOTR: Fellowship, Interstellar | Dune OST, LOTR OST, Origin of Symmetry | Dune, LOTR: Fellowship, The Three-Body Problem |
| Coming-of-age / nostalgia | Stand By Me, Spirited Away, Perks of a Wallflower | Stand By Me (Ben E. King), The Suburbs, For Emma | The Body, Perks of a Wallflower, Norwegian Wood |
| Melancholic / romantic | La La Land, Her, Eternal Sunshine | La La Land OST, A Moon Shaped Pool, In Rainbows | The Time Traveler's Wife, The Fault in Our Stars |
| Identity / technology | The Matrix, The Truman Show, Her | Discovery, Ghost in the Shell OST | Ghost in the Shell, Cloud Atlas, The Midnight Library |

---

## Architecture

```
vecsim/
├── main.go
├── cmd/                        Cobra CLI commands
│   ├── root.go                 PersistentPreRunE: loads config, connects DB/pool, creates store+embedder
│   ├── setup.go                Interactive first-run wizard
│   ├── migrate.go              Goose runner with dimension env-var injection
│   ├── seed.go                 Inserts hardcoded seed items
│   ├── index.go                Batch embedding with live progress
│   ├── search.go               Cosine similarity search with tabwriter output
│   ├── list.go                 Paginated item listing
│   ├── stats.go                Per-type coverage table
│   ├── info.go                 Connectivity and config status
│   └── serve.go                Starts HTTP server with graceful shutdown
├── internal/
│   ├── config/config.go        Profile definitions, JSON load/save, context helpers
│   ├── db/connect.go           pgxpool + pgvector type registration
│   ├── embeddings/ollama.go    Ollama HTTP client (embed / health / pull)
│   ├── store/items.go          ALL SQL — CRUD, embedding management, cosine search
│   ├── indexer/indexer.go      Shared embed-batch-store loop (CLI + HTTP handler)
│   ├── adapters/
│   │   ├── adapter.go          Adapter interface + Item type + registry
│   │   ├── movie.go            MovieAdapter + 25 seed items
│   │   ├── music.go            MusicAdapter + 25 seed items
│   │   └── book.go             BookAdapter + 25 seed items
│   └── server/
│       ├── server.go           ServeMux wiring + graceful shutdown
│       ├── handlers.go         All HTTP route handlers
│       ├── middleware.go       RequestID, Logging, CORS, Recovery
│       └── response.go         WriteJSON / WriteError helpers
├── migrations/
│   ├── 001_init.sql            Schema with ${VECSIM_DIMENSIONS} envsub placeholder
│   └── embed.go                //go:embed *.sql for binary portability
├── docker-compose.yml          ankane/pgvector Postgres container
└── .env.example                Reference values for setup prompts
```

### Key architectural rules

- **All SQL lives in `internal/store/items.go`** — no SQL in cmd files or HTTP handlers.
- **No third-party logging, HTTP, or config libraries** — stdlib `log/slog`, `net/http`, `encoding/json`.
- **Every DB call has a 5-second timeout** applied internally in each `Store` method.
- **Every Ollama embed call has a 120-second timeout**; health checks have 10 seconds.
- **Errors are wrapped** as `fmt.Errorf("layer: operation: %w", err)` throughout.
- **Adding a new content domain** (podcasts, recipes, products) requires one new file in `internal/adapters/` implementing the `Adapter` interface — nothing else changes.
- **Cross-type search** is a single SQL query with a `($1 = '' OR type = $1)` parameterized filter; passing `""` queries all types simultaneously.
- The migration SQL uses **Goose's `envsub`** to substitute `${VECSIM_DIMENSIONS}` at runtime from the active profile, creating the correctly-sized `vector(N)` column and HNSW index.

---

## Data model

```sql
-- items: all content types in one table
CREATE TABLE items (
    id         TEXT PRIMARY KEY,         -- "movie:inception", "book:recursion"
    label      TEXT        NOT NULL,     -- display name
    type       TEXT        NOT NULL,     -- "movie", "music", "book"
    fields     JSONB       NOT NULL,     -- domain-specific metadata
    tags       TEXT[]      NOT NULL,     -- genres, moods, keywords
    embedding  vector(768),              -- or vector(4096) for max profile
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- HNSW index for fast approximate nearest-neighbour search
CREATE INDEX items_embedding_hnsw_idx ON items USING hnsw (embedding vector_cosine_ops);

-- vecsim_meta: installation metadata
CREATE TABLE vecsim_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL                  -- "profile" → "optimized" | "max"
);
```

**Fields shape per domain:**

```json
// movie
{ "title": "Inception", "year": 2010, "genre": ["Sci-Fi"], "cast": ["..."], "plot": "..." }

// music
{ "title": "Time", "artist": "Hans Zimmer", "genre": ["Soundtrack"], "mood": ["Contemplative"], "description": "..." }

// book
{ "title": "Recursion", "author": "Blake Crouch", "genre": ["Sci-Fi"], "synopsis": "..." }
```

---

## Tech stack

| Concern | Library |
|---|---|
| CLI | `github.com/spf13/cobra` |
| Database driver | `github.com/jackc/pgx/v5` + `pgxpool` |
| Vector type | `github.com/pgvector/pgvector-go` + `/pgx` subpackage |
| Migrations | `github.com/pressly/goose/v3` |
| HTTP server | stdlib `net/http` (Go 1.22 method-prefixed mux) |
| Logging | stdlib `log/slog` |
| Embeddings | Ollama HTTP API via stdlib `net/http` |
| Postgres container | `ankane/pgvector` Docker image |
