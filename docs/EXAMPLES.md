# GOSim — Use cases & full walkthrough

See [README.md](../README.md) for setup, CLI/API reference, and architecture.

GOSim is domain-agnostic: you import a JSON array of like-shaped objects as a
*collection*, and it detects each field's kind, embeds the text fields, and lets
you weight every parameter. The examples below use the `testdata/movies.json`
dataset included in the repo, but every command works the same way for products,
tickets, papers, or any other collection.

---

## Use cases

### "More like this" recommendations

You have an item a user just engaged with and want to suggest what's next. Feed
its label (or id) into `POST /search` and get a ranked list of semantically
similar items — no manual tagging or genre matching required. Add
`"cross_type": true` to draw from every collection at once.

```bash
gosim search "Inception" --limit 5
```

### Search from provided information (no stored item needed)

`POST /search/embed` embeds an ad-hoc query and returns the closest siblings.
The query can be free text…

```bash
curl -s localhost:7700/search/embed -d '{
  "text": "something melancholic and introspective, like a quiet rainy afternoon",
  "cross_type": true,
  "limit": 8
}'
```

…or a raw object that was never stored — GOSim embeds it on the fly and finds
its nearest neighbours:

```bash
curl -s localhost:7700/search/embed -d '{
  "object": { "title": "Unstored Film", "year": 2025, "genre": ["Sci-Fi"], "plot": "a lone pilot drifts past the edge of the solar system" },
  "type": "movie",
  "limit": 5
}'
```

### Any domain, zero code

Adding a new content domain requires **no Go code** — just import a file with a
new `--type`. A products catalogue and a movie list are handled by the exact
same pipeline:

```bash
gosim import movies.json   --type movie
gosim import products.json --type product --label-field name
gosim import tickets.json  --type ticket  --label-field subject
```

### Near-duplicate detection

Because identical content embeds to (near-)identical vectors, searching for an
item and inspecting the top scores surfaces duplicates and near-duplicates —
useful for deduplicating catalogues, merging records, or flagging plagiarism.

### Backend similarity microservice

Any other application — a web app, a recommendation engine, a data pipeline —
can call GOSim's HTTP API over localhost without knowing anything about
embeddings or Postgres. The caller sends a label, object, or free-text query and
receives ranked results with scores. GOSim acts as a self-contained semantic
search sidecar.

### Local, private search — no API keys or cloud dependency

All embedding and search happens on your own machine after the initial
`ollama pull`. No data leaves your environment, no rate limits, no per-request
cost. Suitable for personal libraries, private research collections, or
air-gapped environments.

---

## Full walkthrough

A complete session from a fresh checkout to a working search, using the
**optimized** profile (recommended for most machines).

### 1 — Setup

```
$ go run . setup --pull

╔══════════════════════════════════════════╗
║         GOSim — first-run setup         ║
╚══════════════════════════════════════════╝

Choose an embedding profile:

  [1] max       — qwen3-embedding:8b  (4096 dims)
  [2] optimized — nomic-embed-text    (768 dims)   ← Recommended

  Profile (1 or 2) [2]: 2
  → optimized selected (nomic-embed-text, 768 dims)

Connection details:
  Ollama URL [http://localhost:11434]:
  Database URL [postgres://gosim:gosim@localhost:5432/gosim?sslmode=disable]:
  API port [7700]:

Checking connectivity…
  ✓ Ollama at http://localhost:11434 — reachable
  ✓ Postgres — connected

  Config saved to /home/user/.config/gosim/config.json

Pulling nomic-embed-text…
  ✓ Model ready
```

### 2 — Start Postgres, migrate, and verify

```
$ docker compose up -d
$ go run . migrate
Migrations applied (profile: optimized, dims: 768)
Profile recorded in gosim_meta

$ go run . doctor

GOSim doctor

  ✓ Config file            /home/user/.config/gosim/config.json
  ✓ Ollama                 http://localhost:11434 reachable
  ✓ Embedding model        nomic-embed-text installed
  ✓ Postgres               connected
  ✓ Migrations             applied, profile matches

All checks passed — GOSim is ready. Import data with 'gosim import <file>'.
```

### 3 — Import a collection

Each object is embedded as it loads. `--type` names the collection; the label is
auto-detected from the `title` field.

```
$ go run . import testdata/movies.json --type movie
Imported into "movie": 100 inserted, 0 skipped (already existed)
Generating embeddings…
Indexing: 100/100

$ go run . stats
TYPE    TOTAL  EMBEDDED  COVERAGE
────    ─────  ────────  ────────
movie   100    100       100.0%
────    ─────  ────────  ────────
TOTAL   100    100       100.0%
```

### 4 — Search

```
$ go run . search "Avengers: Endgame" --limit 5
Similar to "Avengers: Endgame" [movie]:

RANK  TYPE   LABEL                       SCORE
────  ────   ─────                       ─────
1     movie  Avengers: Age of Ultron     0.9867
2     movie  The Avengers                0.9809
3     movie  Avengers: Infinity War      0.8993
4     movie  Captain America: Civil War  0.7647
5     movie  Iron Man 2                  0.7142
```

Ranking uses the collection's weights (step 5), so a Marvel film surfaces its
franchise siblings via shared cast and genre — not just films with similar plot
wording. Add `--cross-type` once you've imported more than one collection to
search across all of them at once. A query that doesn't match any title is
treated as free text:

```
$ go run . search "a heist that takes place inside someone's dream" --limit 3
```

### 5 — Inspect the schema and set per-parameter weights

GOSim detected your collection's structure on import. View it:

```
$ go run . schema --type movie
PARAMETER  KIND    WEIGHT  NOTES
semantic   text    1       embedding of text fields
cast       list    3
genre      list    2
plot       text    0       folded into semantic
title      text    0       folded into semantic
year       number  0       range 2010–2019
```

Tune what "similar" means — these persist and apply to every search of the
collection:

```
# Recommend by shared lead actors: Inception → Shutter Island, Django, etc.
$ go run . weights --type movie cast=10 genre=3 semantic=1

# Recommend by genre instead: Inception → Tron Legacy and other sci-fi.
$ go run . weights --type movie genre=10 cast=1 semantic=1
```

`list` fields (cast, genre) score by overlap where any shared value counts
strongly; `number` fields by range-normalized closeness; `keyword` fields by
exact match; `text` fields are embedded into the semantic vector. Override the
saved weights for a single query with `--weight cast=5,genre=2` (or
`--weight semantic=1` for pure semantic). Defaults are assigned to every
parameter, so search works before you tune anything.

### 6 — HTTP API

```
$ go run . serve &

$ curl -s localhost:7700/health | jq
{ "profile": "optimized", "model": "nomic-embed-text", "dimensions": 768, "ollama_ok": true, "db_ok": true }

$ curl -s localhost:7700/search \
    -H 'Content-Type: application/json' \
    -d '{"label":"Inception","limit":3}' \
  | jq '.results[] | {label:.item.label, score}'
```

### 7 — Search by a provided object

Find the closest siblings to an object that isn't stored — its structured fields
also influence weighted re-ranking:

```
$ curl -s localhost:7700/search/embed \
    -H 'Content-Type: application/json' \
    -d '{
      "object": {"title":"Echoes of Tomorrow","year":2021,"genre":["Sci-Fi"],"plot":"a memory archivist uncovers a forgotten war"},
      "type": "movie",
      "limit": 3
    }' | jq '.results[] | {label:.item.label, score}'
```

### 8 — Bulk import via the API

```
$ curl -s 'localhost:7700/import?type=movie' \
    -H 'Content-Type: application/json' \
    -d '[
      {"title":"New Arrival","year":2024,"genre":["Drama"],"plot":"..."},
      {"title":"Second Feature","year":2023,"genre":["Comedy"],"plot":"..."}
    ]' | jq
{ "type": "movie", "inserted": 2, "skipped": 0, "status": "indexing started" }
```

The items are inserted and embedded in the background; `GET /stats` reflects the
new count once indexing completes.

### 9 — Add a single item

```
$ curl -s localhost:7700/items \
    -H 'Content-Type: application/json' \
    -d '{
      "label": "Annihilation",
      "type": "movie",
      "fields": {"title":"Annihilation","year":2018,"genre":["Sci-Fi","Horror"],"plot":"a biologist enters a zone where nature breaks down"},
      "tags": ["sci-fi","horror"]
    }' | jq '{id, label, embedded}'
{ "id": "movie:annihilation", "label": "Annihilation", "embedded": true }
```

The item is inserted and embedded in a single request — immediately searchable.
