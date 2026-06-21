# GOSim

A generic semantic similarity search engine written in Go. **Import any homogeneous collection of JSON objects** — movies, products, support tickets, people, papers — embed them as vectors, and retrieve the closest siblings to any item, object, or free-text query. There is no per-domain code: GOSim detects each field's kind, embeds the text fields, and lets you weight every parameter to control what "similar" means.

Runs entirely offline after initial setup. Uses **Ollama** for local embedding generation and **PostgreSQL + pgvector** for vector storage and cosine-similarity search. Exposes both a CLI and a local HTTP API on `localhost:7700`.

---

## How it works

GOSim is a **schema-aware hybrid search engine**. When you import a collection, it inspects the objects and detects each parameter's kind — `text`, `list`, `number`, or `keyword` — with no hardcoded field names. From that schema it ranks by combining:

- **text** fields (title, plot, description) → embedded by Ollama into one vector and compared by cosine similarity (pgvector `<=>` + an HNSW index);
- **list** fields (cast, genre, tags) → set overlap, where *any* shared value counts strongly;
- **number** fields (year, price) → closeness, normalized by the field's range;
- **keyword** fields (category, status) → exact match.

Each parameter has a **weight you control per collection** (`gosim weights`), so you decide what "similar" means — e.g. weight `cast` highest and *Inception* surfaces *Shutter Island* (shared Leonardo DiCaprio); weight `genre` highest and it surfaces *Tron Legacy* (both sci-fi). Defaults are assigned to every parameter, so search works before you tune anything.

A "collection" is just the `type` you assign at import. Because text fields share one embedding space, **cross-type search works out of the box**: pass `--cross-type` to surface related items across collections.

---

Use cases (recommendations, deduplication, mood-based discovery, importing new domains, etc.) and a full step-by-step walkthrough with example output live in **[docs/EXAMPLES.md](docs/EXAMPLES.md)**.

---

## Embedding profiles

Chosen once during `gosim setup`. Governs the entire installation.

Both options are **embedding models**, not chat/generative LLMs — they only convert text into vectors for similarity search, so there is no text generation or chat behavior to configure.

| Profile | Ollama model | Dims | RAM | Download |
|---|---|---|---|---|
| `optimized` (recommended default) | [`nomic-embed-text`](https://ollama.com/library/nomic-embed-text) | 768 | ~600 MB, any modern laptop | ~274 MB |
| `max` | [`qwen3-embedding:8b`](https://ollama.com/library/qwen3-embedding) | 4096 | 6–8 GB, GPU recommended | ~4.7 GB |

- **`nomic-embed-text`** — small and fast, runs comfortably on any modern laptop with no GPU. Great quality for most use cases; this is the default `gosim setup` suggests.
- **`qwen3-embedding:8b`** — much larger model producing higher-dimensional, higher-fidelity embeddings at the cost of RAM/GPU requirements and a ~17x bigger download.

`gosim setup --pull` runs `ollama pull <model>` for whichever profile you choose, with live download progress. The chosen profile is stored in both `~/.config/gosim/config.json` and the `gosim_meta` Postgres table; the two are validated against each other on every startup to prevent dimension mismatches.

---

## Prerequisites

- **Go 1.22+**
- **Ollama** running locally — [ollama.com](https://ollama.com)
- **Docker + Docker Compose** (for the Postgres/pgvector container)

---

## Quick start

```bash
# 1. Clone and enter the project
git clone https://github.com/Datto27/GOSim
cd gosim

# 2. Run the interactive setup wizard
#    --pull downloads the embedding model automatically
go run . setup --pull

# 3. Start Postgres with the pgvector extension
docker compose up -d

# 4. Create tables and HNSW index
go run . migrate

# 5. Verify everything is ready (config, Ollama, model, Postgres, migrations)
go run . doctor

# 6. Import your data — schema is detected and each object embedded automatically.
#    The example dataset ships in testdata/; --type names the collection.
go run . import testdata/movies.json --type movie

# 7. See the detected parameters and tune what "similar" means (optional —
#    defaults are applied if you skip this).
go run . schema  --type movie
go run . weights --type movie cast=3 genre=2 year=0

# 8. Search!
go run . search "Inception" --limit 10

# 9. Start the HTTP API on localhost:7700
go run . serve
```

`import` embeds as it loads. To import without embedding, pass `--no-index` and run `go run . index` later.

Or build the binary once:

```bash
go build -o gosim .
./gosim setup --pull
```

---

## CLI reference

### `gosim setup [--pull]`

Interactive first-run wizard. Presents both embedding profiles with hardware requirements, prompts for Ollama URL, database URL, and API port (all with sensible defaults), checks connectivity, and writes `~/.config/gosim/config.json`.

`--pull` streams `ollama pull <model>` live with download progress.

### `gosim doctor`

Read-only diagnostics. Checks that the config file exists, Ollama is reachable, the embedding model is pulled, Postgres is reachable, and migrations are applied — printing ✓/✗ for each and the exact command to fix anything that isn't ready. Safe to run before setup is complete.

### `gosim migrate`

Runs Goose database migrations. Creates the `items` table with a `vector(N)` column sized for the active profile, the `gosim_meta` table, a B-tree index on `type`, and an HNSW cosine-distance index on `embedding`. Records the active profile in `gosim_meta`.

Safe to re-run — Goose is idempotent.

### `gosim import <file.json> [--type NAME] [--label-field F] [--id-field F] [--tags-field F] [--no-index]`

Imports a JSON **array of objects** as a collection and embeds each one. This is the primary way to get data into GOSim — no envelope and no per-type code required.

```bash
gosim import movies.json   --type movie
gosim import products.json --type product --label-field name
gosim import people.json   --type person  --label-field full_name --tags-field roles
```

- **`--type`** names the collection. Defaults to the file name (e.g. `movies.json` → `movies`).
- **`--label-field`** picks the human-readable label. If omitted, GOSim auto-detects `label`/`title`/`name`/`heading`.
- **`--id-field`** is the primary key (default `id`); when absent from an object, an id is generated from the label (`movie:hollow-horizon`).
- **`--tags-field`** is a string array used for tag-based re-ranking (default `tags`).
- **`--no-index`** inserts without embedding; run `gosim index` later.

Already-existing ids are skipped, so re-importing is safe. If an object happens to carry a nested `fields` object (the shape GOSim itself emits), that nested object is used as the fields.

### `gosim index [--type NAME|all] [--force]`

Finds all items where `embedding IS NULL`, calls Ollama in batches of 20, and stores the resulting vectors. Pass `--type` to scope to one collection, or `all` (default) for every collection. Safe to re-run — already-indexed items are never re-embedded. Use **`--force`** to recompute every embedding (needed after a collection's schema or text fields change).

### `gosim reset [--type NAME] [--yes]`

Deletes stored items so you can re-import fresh data. By default it removes every item; pass `--type` to clear one collection. The schema, weights, and HNSW index are kept. Destructive, so it asks for confirmation unless `--yes` is given. (No Docker needed — handy when your user isn't in the `docker` group.)

### `gosim schema [--type NAME]`

Shows the parameters GOSim detected for a collection — each one's kind (`text`/`list`/`number`/`keyword`) and current weight — so you know what you can tune. Without `--type`, lists all collections.

### `gosim weights --type NAME <param>=<n> ...`

Sets the per-parameter ranking weights for a collection, persisted and applied to every search of it. Values merge with the existing weights, so you can adjust one at a time. With no arguments, prints the current weights.

```bash
gosim weights --type movie   cast=5 genre=3 year=0
gosim weights --type product category=4 price=2
```

`semantic` weights the text embedding; `tags` weights tag overlap; any detected field name weights that parameter. Weights are non-negative; `0` excludes a parameter.

### `gosim search <query> [--type NAME] [--cross-type] [--limit N] [--weight key=value,...]`

If `<query>` exactly matches a stored item's label, returns the items most similar to it, ranked by the collection's **per-parameter weights** (set with `gosim weights`, defaults otherwise). If it matches nothing, `<query>` is treated as **free text**, embedded on the fly, and the closest items are returned — so a search always finds something. Use `--cross-type` to search across all collections; `--type` to disambiguate a label.

Use `--weight` (repeatable or comma-separated) to override the saved weights for one query:

```bash
gosim search "Inception" --weight cast=5,genre=2     # this query only
gosim search "Inception" --weight semantic=1         # pure semantic, ignore fields
gosim search "a heist inside a dream" --limit 5      # free-text query
```

When structured weights are active, GOSim re-ranks the whole collection (not just the semantic top-N), so a film sharing only an actor still surfaces.

### `gosim serve [--port N]`

Starts the HTTP API server on `localhost:<port>` (default: value from config, fallback `7700`). Handles graceful shutdown on SIGINT/SIGTERM with a 10-second drain window.

### `gosim list [--type] [--limit N] [--offset N]`

Paginated table of items showing ID, type, label, tags, and whether embedding has been generated.

### `gosim stats`

Per-collection item counts and embedding coverage percentage, plus overall totals. Collections appear automatically as you import them.

### `gosim info`

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

`POST /items` adds a single item to a collection. `type` and `label` are required; `fields` is any JSON object and `id`/`tags` are optional:
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

The server generates an ID from the label (`movie:dune-part-two`) when one isn't supplied and immediately attempts to embed the item. The response includes `"embedded": true|false`. Any non-empty `type` is accepted — collections are created implicitly.

### Search

```
POST /search
```
```json
{
  "label": "Inception",
  "cross_type": true,
  "limit": 10,
  "weights": { "genre": 2.0, "year": 0.5, "cast": 0 }
}
```

Use `"id"` instead of `"label"` for an unambiguous lookup. `"type"` scopes the label resolution when a label matches multiple types.

`"weights"` is optional. When omitted, the collection's **saved per-parameter weights** are used (see `PUT /collections/{type}/weights` and `gosim weights`). When supplied, they override the saved weights for that request. Recognized keys are `"semantic"`, `"tags"`, and any detected field name; weights must be non-negative (a `400` is returned otherwise); `0` drops a signal. Pass `{"semantic": 1}` for pure semantic ranking.

```
POST /search/embed
```

Retrieve the closest siblings to **provided information** that isn't a stored item — either free text or a raw object:
```json
{
  "text": "a mind-bending thriller about memory and identity",
  "cross_type": true,
  "limit": 10
}
```
```json
{
  "object": { "title": "Unstored Film", "year": 2025, "genre": ["Sci-Fi"], "plot": "..." },
  "type": "movie",
  "limit": 10
}
```

Provide exactly one of `"text"` or `"object"`. With an `"object"`, its structured fields also participate in weighted re-ranking; with free `"text"` there is no structured query, so only the `"semantic"` weight key has any effect (other keys are accepted but ignored).

Response for both search endpoints:
```json
{
  "query": { "id": "movie:inception", "label": "Inception", ... },
  "results": [
    { "item": { "id": "movie:tron-legacy", "label": "Tron Legacy", ... }, "score": 0.6547 },
    { "item": { "id": "movie:the-avengers", "label": "The Avengers", ... }, "score": 0.61 }
  ]
}
```

### Import, stats, and background indexing

```
POST /import   → bulk-import a collection, then index it in the background; returns 202
GET  /stats    → per-collection counts and embedding coverage
POST /index    → triggers background indexing of all unembedded items; returns 202 immediately
```

`POST /import` accepts either a bare JSON array (with `?type=` in the query string) or an object body:
```json
{
  "type": "movie",
  "label_field": "title",
  "items": [
    { "title": "Inception",    "year": 2010, "genre": ["Action","Science Fiction"], "plot": "..." },
    { "title": "Interstellar", "year": 2014, "genre": ["Science Fiction"],          "plot": "..." }
  ]
}
```
Response: `{ "type": "movie", "inserted": 2, "skipped": 0, "status": "indexing started" }`.

### Collections (schema & weights)

```
GET /collections                  → list collection names
GET /collections/{type}           → detected schema + current weights
PUT /collections/{type}/weights   → set weights (merged with existing)
```

`GET /collections/movie`:
```json
{
  "type": "movie",
  "schema": { "cast": {"kind":"list"}, "genre": {"kind":"list"}, "plot": {"kind":"text"}, "year": {"kind":"number","min":2010,"max":2019} },
  "weights": { "semantic": 1, "cast": 3, "genre": 2, "year": 0 }
}
```

`PUT /collections/movie/weights` with `{"weights": {"cast": 5, "genre": 3}}` → merges and returns the full weight map.

---

## Bring your own data

GOSim ships no built-in dataset — you import your own. Any JSON array of like-shaped objects works; GOSim detects each field's kind, embeds the text fields, and exposes the rest as tunable ranking signals — so a movies file and a products file are handled by exactly the same code path.

```bash
# example dataset included in the repo
gosim import testdata/movies.json --type movie

# a completely different shape — no code changes needed
gosim import products.json --type product --label-field name
```

A movie object and a product object need nothing in common:

```json
// movies.json
{ "title": "Inception", "year": 2010, "genre": ["Action","Science Fiction"], "cast": ["Leonardo DiCaprio","..."], "plot": "..." }

// products.json
{ "name": "Aero Desk Lamp", "price": 49.99, "category": "lighting", "description": "..." }
```

Mix collections in one database and pass `--cross-type` to search across all of them at once.

---

## Architecture

```
gosim/
├── main.go
├── cmd/                        Cobra CLI commands
│   ├── root.go                 PersistentPreRunE: loads config, connects DB/pool, creates store+embedder
│   ├── setup.go                Interactive first-run wizard
│   ├── doctor.go               Read-only prerequisite diagnostics
│   ├── migrate.go              Goose runner with dimension env-var injection
│   ├── import.go               Import a JSON array, detect schema, then embed
│   ├── index.go                Batch embedding with live progress (--force)
│   ├── schema.go               Show detected parameters + weights
│   ├── weights.go              Set per-parameter ranking weights
│   ├── search.go               Title or free-text search, hybrid-ranked
│   ├── reset.go                Delete items to reload
│   ├── list.go                 Paginated item listing
│   ├── stats.go                Per-collection coverage table
│   ├── info.go                 Connectivity and config status
│   └── serve.go                Starts HTTP server with graceful shutdown
├── internal/
│   ├── config/config.go        Profile definitions, JSON load/save, context helpers
│   ├── db/connect.go           pgxpool + pgvector type registration
│   ├── embeddings/ollama.go    Ollama HTTP client (embed / health / pull)
│   ├── schema/schema.go        Field-kind detection + embedding text + default weights
│   ├── store/items.go          ALL SQL — items, collections (schema/weights), search
│   ├── indexer/indexer.go      Schema-aware embed-batch-store loop (CLI + HTTP)
│   ├── ingest/ingest.go        Normalizes raw JSON objects → items (CLI + HTTP)
│   ├── rerank/rerank.go        Hybrid scoring: semantic + weighted field signals
│   ├── adapters/adapter.go     Item type (the stored row shape)
│   └── server/
│       ├── server.go           ServeMux wiring + graceful shutdown
│       ├── handlers.go         All HTTP route handlers
│       ├── middleware.go       RequestID, Logging, CORS, Recovery
│       └── response.go         WriteJSON / WriteError helpers
├── migrations/
│   ├── 001_init.sql            items + gosim_meta; ${GOSIM_DIMENSIONS} envsub placeholder
│   ├── 002_collections.sql     per-collection schema + weights
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
- **Adding a new content domain** (podcasts, recipes, products) requires **no code** — `gosim import <file> --type <name>`. `internal/schema` detects each field's kind, the embedding uses only text fields, and `internal/rerank` scores list/number/keyword fields by kind. No field name is special anywhere in the engine.
- **Ranking is hybrid and weight-driven.** When structured weights are active, the whole collection is re-ranked (not just the semantic top-N), so an item matching only on a list/number field still surfaces.
- **Cross-type search** is a single SQL query with a `($1 = '' OR type = $1)` parameterized filter; passing `""` queries all collections simultaneously.
- The migration SQL uses **Goose's `envsub`** to substitute `${GOSIM_DIMENSIONS}` at runtime from the active profile, creating the correctly-sized `vector(N)` column and HNSW index.

---

## Data model

```sql
-- items: every collection lives in one table
CREATE TABLE items (
    id         TEXT PRIMARY KEY,         -- "movie:hollow-horizon", "product:aero-desk-lamp"
    label      TEXT        NOT NULL,     -- display name
    type       TEXT        NOT NULL,     -- collection name: "movie", "product", ...
    fields     JSONB       NOT NULL,     -- the imported object, verbatim
    tags       TEXT[]      NOT NULL,     -- optional, used for tag-based re-ranking
    embedding  vector(768),              -- or vector(4096) for max profile
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- HNSW index for fast approximate nearest-neighbour search
CREATE INDEX items_embedding_hnsw_idx ON items USING hnsw (embedding vector_cosine_ops);

-- gosim_meta: installation metadata
CREATE TABLE gosim_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL                  -- "profile" → "optimized" | "max"
);

-- collections: detected schema + ranking weights, one row per type
CREATE TABLE collections (
    type    TEXT PRIMARY KEY,            -- "movie", "product", ...
    schema  JSONB NOT NULL,              -- {"cast":{"kind":"list"},"year":{"kind":"number","min":2010,"max":2019}, ...}
    weights JSONB NOT NULL               -- {"semantic":1,"cast":3,"genre":2,"year":0}
);
```

**`fields` is whatever you import** — there is no fixed schema. GOSim detects each field's kind and embeds only the text fields, so each collection can have a completely different shape:

```json
// type=movie
{ "title": "Inception", "year": 2010, "genre": ["Action","Science Fiction"], "cast": ["Leonardo DiCaprio","..."], "plot": "..." }

// type=product
{ "name": "Aero Desk Lamp", "price": 49.99, "category": "lighting", "description": "..." }

// type=ticket
{ "subject": "Login fails on SSO", "priority": "high", "body": "...", "status": "open" }
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
