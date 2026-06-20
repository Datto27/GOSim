// Package store contains all SQL for vecsim. No SQL lives anywhere else.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/Datto27/vecsim/internal/adapters"
)

const dbTimeout = 5 * time.Second

// ErrNotFound is returned when a requested item does not exist.
var ErrNotFound = errors.New("store: not found")

// ErrNotEmbedded is returned when an item exists but has no stored embedding.
var ErrNotEmbedded = errors.New("store: item not yet embedded (run 'vecsim index')")

// Store wraps a pgxpool.Pool and provides all database access for vecsim.
type Store struct {
	pool *pgxpool.Pool
}

// New returns a Store backed by pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// ─── Meta / profile validation ──────────────────────────────────────────────

// GetMeta fetches the value stored under key in vecsim_meta. It returns
// ErrNotFound if the key is absent.
func (s *Store) GetMeta(ctx context.Context, key string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	var value string
	err := s.pool.QueryRow(ctx, `SELECT value FROM vecsim_meta WHERE key = $1`, key).Scan(&value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("store: get meta: %w", err)
	}
	return value, nil
}

// SetMeta upserts key → value in vecsim_meta.
func (s *Store) SetMeta(ctx context.Context, key, value string) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	_, err := s.pool.Exec(ctx, `
		INSERT INTO vecsim_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, key, value)
	if err != nil {
		return fmt.Errorf("store: set meta: %w", err)
	}
	return nil
}

// isUndefinedTable reports whether err represents a "table does not exist"
// Postgres error (SQLSTATE 42P01), which occurs before the first migration.
func isUndefinedTable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}

// ValidateProfile checks that the profile stored in vecsim_meta matches
// the configured profile. It returns nil if the meta table is missing or
// the profile key is absent (interpreted as "not yet migrated").
func (s *Store) ValidateProfile(ctx context.Context, profile string) error {
	stored, err := s.GetMeta(ctx, "profile")
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		if isUndefinedTable(err) {
			return nil
		}
		return fmt.Errorf("store: validate profile: %w", err)
	}
	if stored != profile {
		return fmt.Errorf(
			"store: validate profile: config says %q but database was migrated with profile %q — re-run 'vecsim migrate' after changing profiles",
			profile, stored,
		)
	}
	return nil
}

// ─── Items CRUD ──────────────────────────────────────────────────────────────

// scanItem reads a row of (id, label, type, fields, tags, embedded, created_at)
// into item.
func scanItem(row pgx.Row, item *adapters.Item) error {
	return row.Scan(
		&item.ID, &item.Label, &item.Type,
		&item.Fields, &item.Tags,
		&item.Embedded, &item.CreatedAt,
	)
}

// InsertItem inserts item, skipping silently if it already exists.
// It returns true when a new row was inserted.
func (s *Store) InsertItem(ctx context.Context, item adapters.Item) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	tag, err := s.pool.Exec(ctx, `
		INSERT INTO items (id, label, type, fields, tags)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`, item.ID, item.Label, item.Type, item.Fields, item.Tags)
	if err != nil {
		return false, fmt.Errorf("store: insert item: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// GetItem fetches a single item by its primary key.
func (s *Store) GetItem(ctx context.Context, id string) (*adapters.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	var item adapters.Item
	row := s.pool.QueryRow(ctx, `
		SELECT id, label, type, fields, tags, (embedding IS NOT NULL) AS embedded, created_at
		FROM items
		WHERE id = $1
	`, id)
	if err := scanItem(row, &item); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: get item: %w", err)
	}
	return &item, nil
}

// GetItemByLabel looks up an item by its display label, optionally filtered
// by type. If typ is empty, all types are searched. If the label is
// ambiguous (multiple types match), an error listing the candidates is
// returned so the caller can ask the user to specify --type.
func (s *Store) GetItemByLabel(ctx context.Context, label, typ string) (*adapters.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT id, label, type, fields, tags, (embedding IS NOT NULL) AS embedded, created_at
		FROM items
		WHERE label = $1 AND ($2 = '' OR type = $2)
		ORDER BY type, id
	`, label, typ)
	if err != nil {
		return nil, fmt.Errorf("store: get item by label: %w", err)
	}
	defer rows.Close()

	var items []adapters.Item
	for rows.Next() {
		var item adapters.Item
		if err := scanItem(rows, &item); err != nil {
			return nil, fmt.Errorf("store: get item by label: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: get item by label: %w", err)
	}

	switch len(items) {
	case 0:
		return nil, ErrNotFound
	case 1:
		return &items[0], nil
	default:
		candidates := make([]string, len(items))
		for i, it := range items {
			candidates[i] = fmt.Sprintf("%s (%s)", it.ID, it.Type)
		}
		return nil, fmt.Errorf(
			"store: get item by label: %q matches multiple items: %s — add --type to disambiguate",
			label, strings.Join(candidates, ", "),
		)
	}
}

// ListItems returns a page of items and the total count matching the optional
// type filter. Pass typ="" for all types.
func (s *Store) ListItems(ctx context.Context, typ string, limit, offset int) ([]adapters.Item, int, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM items WHERE ($1 = '' OR type = $1)`, typ,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: list items: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, label, type, fields, tags, (embedding IS NOT NULL) AS embedded, created_at
		FROM items
		WHERE ($1 = '' OR type = $1)
		ORDER BY type, created_at, id
		LIMIT $2 OFFSET $3
	`, typ, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list items: %w", err)
	}
	defer rows.Close()

	var items []adapters.Item
	for rows.Next() {
		var item adapters.Item
		if err := scanItem(rows, &item); err != nil {
			return nil, 0, fmt.Errorf("store: list items: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: list items: %w", err)
	}

	return items, total, nil
}

// DeleteItem removes an item by id. It returns false when no row was deleted.
func (s *Store) DeleteItem(ctx context.Context, id string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	tag, err := s.pool.Exec(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("store: delete item: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ─── Embedding management ─────────────────────────────────────────────────

// SetEmbedding stores vec as the embedding for the item identified by id.
func (s *Store) SetEmbedding(ctx context.Context, id string, vec pgvector.Vector) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	tag, err := s.pool.Exec(ctx, `UPDATE items SET embedding = $1 WHERE id = $2`, vec, id)
	if err != nil {
		return fmt.Errorf("store: set embedding: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store: set embedding: %w", ErrNotFound)
	}
	return nil
}

// ItemsMissingEmbedding returns up to limit items whose embedding is NULL.
// Pass typ="" to query across all types.
func (s *Store) ItemsMissingEmbedding(ctx context.Context, typ string, limit int) ([]adapters.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT id, label, type, fields, tags, false AS embedded, created_at
		FROM items
		WHERE embedding IS NULL AND ($1 = '' OR type = $1)
		ORDER BY id
		LIMIT $2
	`, typ, limit)
	if err != nil {
		return nil, fmt.Errorf("store: items missing embedding: %w", err)
	}
	defer rows.Close()

	var items []adapters.Item
	for rows.Next() {
		var item adapters.Item
		if err := scanItem(rows, &item); err != nil {
			return nil, fmt.Errorf("store: items missing embedding: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: items missing embedding: %w", err)
	}

	return items, nil
}

// GetEmbedding returns the stored embedding for item id. It returns
// ErrNotFound when the item does not exist, and ErrNotEmbedded when the item
// exists but has not been indexed yet.
func (s *Store) GetEmbedding(ctx context.Context, id string) (pgvector.Vector, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	// Only match rows that have a non-NULL embedding to avoid scanning NULL
	// into pgvector.Vector.
	var vec pgvector.Vector
	err := s.pool.QueryRow(ctx, `
		SELECT embedding FROM items WHERE id = $1 AND embedding IS NOT NULL
	`, id).Scan(&vec)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Distinguish "item not found" from "not yet embedded".
			var exists bool
			if err2 := s.pool.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM items WHERE id = $1)`, id,
			).Scan(&exists); err2 != nil {
				return pgvector.Vector{}, fmt.Errorf("store: get embedding: %w", err2)
			}
			if !exists {
				return pgvector.Vector{}, ErrNotFound
			}
			return pgvector.Vector{}, ErrNotEmbedded
		}
		return pgvector.Vector{}, fmt.Errorf("store: get embedding: %w", err)
	}
	return vec, nil
}

// ─── Search ───────────────────────────────────────────────────────────────

// SearchResult pairs an item with its cosine similarity score (1 = identical,
// higher is more similar).
type SearchResult struct {
	Item  adapters.Item `json:"item"`
	Score float64       `json:"score"`
}

// SearchByVector finds the top limit items nearest to vec by cosine
// similarity. Pass typ="" for cross-type search. Pass excludeID="" when the
// query is not a stored item (e.g. POST /search/embed).
func (s *Store) SearchByVector(ctx context.Context, vec pgvector.Vector, typ, excludeID string, limit int) ([]SearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT id, label, type, fields, tags, true AS embedded, created_at,
		       1 - (embedding <=> $1) AS score
		FROM items
		WHERE embedding IS NOT NULL
		  AND ($2 = '' OR id != $2)
		  AND ($3 = '' OR type = $3)
		ORDER BY embedding <=> $1
		LIMIT $4
	`, vec, excludeID, typ, limit)
	if err != nil {
		return nil, fmt.Errorf("store: search by vector: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(
			&r.Item.ID, &r.Item.Label, &r.Item.Type,
			&r.Item.Fields, &r.Item.Tags,
			&r.Item.Embedded, &r.Item.CreatedAt,
			&r.Score,
		); err != nil {
			return nil, fmt.Errorf("store: search by vector: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: search by vector: %w", err)
	}

	return results, nil
}

// ─── Stats ────────────────────────────────────────────────────────────────

// TypeStats holds item and embedding counts for one content type.
type TypeStats struct {
	Count    int `json:"count"`
	Embedded int `json:"embedded"`
}

// StatsResult holds per-type stats and an overall total.
type StatsResult struct {
	Total    int                  `json:"total"`
	Embedded int                  `json:"embedded"`
	ByType   map[string]TypeStats `json:"by_type"`
}

// Stats returns per-type item counts and embedding coverage.
func (s *Store) Stats(ctx context.Context) (StatsResult, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT type,
		       count(*)                                       AS total,
		       count(*) FILTER (WHERE embedding IS NOT NULL) AS embedded
		FROM items
		GROUP BY type
		ORDER BY type
	`)
	if err != nil {
		return StatsResult{}, fmt.Errorf("store: stats: %w", err)
	}
	defer rows.Close()

	result := StatsResult{ByType: make(map[string]TypeStats)}
	for rows.Next() {
		var typ string
		var ts TypeStats
		if err := rows.Scan(&typ, &ts.Count, &ts.Embedded); err != nil {
			return StatsResult{}, fmt.Errorf("store: stats: %w", err)
		}
		result.ByType[typ] = ts
		result.Total += ts.Count
		result.Embedded += ts.Embedded
	}
	if err := rows.Err(); err != nil {
		return StatsResult{}, fmt.Errorf("store: stats: %w", err)
	}

	return result, nil
}
