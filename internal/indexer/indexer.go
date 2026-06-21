// Package indexer provides the shared embed-batch-store loop used by both
// cmd/index.go and the POST /index HTTP handler.
package indexer

import (
	"context"
	"fmt"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/Datto27/GOSim/internal/adapters"
	"github.com/Datto27/GOSim/internal/embeddings"
	"github.com/Datto27/GOSim/internal/schema"
	"github.com/Datto27/GOSim/internal/store"
)

// Progress reports indexing progress: Done out of Total items embedded so far.
type Progress struct {
	Done  int
	Total int
}

// Run embeds items via embedder and persists the vectors. With force=false it
// embeds only items lacking an embedding; with force=true it recomputes every
// item (needed after the embedding text changes). Pass typ="" (or "all") to
// cover every collection. The embedding text for each item is built from its
// collection's detected text fields (schema.EmbeddingText), so the vector
// carries meaning rather than every flattened field. onProgress may be nil.
func Run(
	ctx context.Context,
	s *store.Store,
	e *embeddings.OllamaEmbedder,
	typ string,
	batchSize int,
	force bool,
	onProgress func(Progress),
) error {
	if typ == "all" {
		typ = ""
	}

	var (
		items []adapters.Item
		err   error
	)
	if force {
		items, err = s.ItemsForReembed(ctx, typ)
	} else {
		items, err = s.ItemsMissingEmbedding(ctx, typ, 1_000_000)
	}
	if err != nil {
		return fmt.Errorf("indexer: load items: %w", err)
	}

	total := len(items)
	if total == 0 {
		return nil
	}

	// Cache each collection's schema so embedding text uses only its text fields.
	schemaCache := map[string]schema.Schema{}
	schemaFor := func(t string) schema.Schema {
		if sc, ok := schemaCache[t]; ok {
			return sc
		}
		sc, _, err := s.GetCollection(ctx, t)
		if err != nil {
			sc = schema.Schema{}
		}
		schemaCache[t] = sc
		return sc
	}

	done := 0
	for start := 0; start < total; start += batchSize {
		end := start + batchSize
		if end > total {
			end = total
		}
		batch := items[start:end]

		texts := make([]string, len(batch))
		for i, item := range batch {
			texts[i] = schema.EmbeddingText(item.Fields, schemaFor(item.Type))
		}

		vecs, err := e.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("indexer: embed batch: %w", err)
		}

		for i, item := range batch {
			if err := s.SetEmbedding(ctx, item.ID, pgvector.NewVector(vecs[i])); err != nil {
				return fmt.Errorf("indexer: save embedding for %s: %w", item.ID, err)
			}
			done++
			if onProgress != nil {
				onProgress(Progress{Done: done, Total: total})
			}
		}
	}

	return nil
}
