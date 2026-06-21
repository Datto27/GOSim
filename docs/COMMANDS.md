# GOSim — Command Reference

Every command the `gosim` binary supports, in one place. For explanations and
rationale see [README.md](../README.md); for end-to-end walkthroughs see
[EXAMPLES.md](EXAMPLES.md).

All commands accept a global `--config <path>` flag (default:
`~/.config/gosim/config.json`).

## CLI

### Setup & diagnostics

| Command | Flags | Purpose |
|---|---|---|
| `gosim setup` | `--pull` | Interactive first-run wizard: pick an embedding profile, set connection details, write the config file. `--pull` downloads the model via Ollama. |
| `gosim doctor` | — | Read-only checks: config file, Ollama, embedding model, Postgres, migrations. Prints the fix for anything not ready. Safe to run before setup is complete. |
| `gosim migrate` | — | Apply database migrations (items, gosim_meta, collections tables + HNSW index, sized for the active profile). Idempotent. |
| `gosim info` | — | Show active profile, model, dimensions, config path, API port, Ollama/DB connectivity. |

### Data

| Command | Flags | Purpose |
|---|---|---|
| `gosim import <file.json>` | `--type NAME`, `--label-field F`, `--id-field F` (default `id`), `--tags-field F` (default `tags`), `--no-index` | Import a JSON array of objects as a collection. Detects each field's kind, persists the schema, and embeds every item unless `--no-index`. |
| `gosim index` | `--type NAME\|all` (default `all`), `--force` | Generate embeddings for un-indexed items, in batches of 20. `--force` recomputes every item (needed after a schema/text-field change). |
| `gosim reset` | `--type NAME`, `--yes` | Delete stored items — one collection, or all if `--type` is omitted. Schema and weights are kept. Destructive; asks for confirmation unless `--yes`. |
| `gosim list` | `--type NAME`, `--limit N` (default `20`), `--offset N` (default `0`) | Paginated table of items: type, id, label, tags, embedded status. |
| `gosim stats` | — | Per-collection item counts and embedding coverage, plus totals. |

### Ranking

| Command | Flags | Purpose |
|---|---|---|
| `gosim schema` | `--type NAME` | Show detected parameters (kind + current weight) for a collection. Without `--type`, lists all collections. |
| `gosim weights --type NAME <param>=<n> ...` | `--type NAME` (required) | Set persisted per-parameter ranking weights for a collection, merged with existing values. No `<param>=<n>` args just prints current weights. |

### Search & serve

| Command | Flags | Purpose |
|---|---|---|
| `gosim search <query>` | `--type NAME`, `--cross-type`, `--limit N` (default `10`), `--weight key=value,...` | If `<query>` matches a stored label, rank similar items by the collection's weights. Otherwise treat `<query>` as free text and rank by semantic similarity. `--cross-type` searches every collection at once. |
| `gosim serve` | `--port N` (default: config value, fallback `7700`) | Start the HTTP API on `localhost:<port>`. Graceful shutdown on SIGINT/SIGTERM. |

### Built-in (Cobra)

| Command | Purpose |
|---|---|
| `gosim help [command]` | Show help for any command. |
| `gosim completion [bash\|zsh\|fish\|powershell]` | Generate a shell completion script. |

## HTTP API

Only reachable while `gosim serve` is running. Base URL `http://localhost:7700` (or `--port`). All responses are JSON.

| Method & path | Purpose |
|---|---|
| `GET /health` | Profile, model, dimensions, Ollama/DB reachability. |
| `GET /items?type=&limit=&offset=` | Paginated item list. |
| `GET /items/{id}` | Fetch one item. |
| `POST /items` | Add a single item; embeds immediately. |
| `DELETE /items/{id}` | Delete one item. |
| `POST /search` | Find items similar to a stored item, by `label` or `id`. Uses the collection's saved weights unless `weights` is supplied. |
| `POST /search/embed` | Find items similar to ad-hoc `text` or a raw `object` that was never stored. |
| `GET /stats` | Per-collection counts and embedding coverage. |
| `POST /index` | Trigger background indexing of all unembedded items; returns `202` immediately. |
| `POST /import` | Bulk-import a collection, then index it in the background; returns `202` immediately. |
| `GET /collections` | List collection names. |
| `GET /collections/{type}` | Detected schema + current weights for a collection. |
| `PUT /collections/{type}/weights` | Set weights for a collection (merged with existing). |
