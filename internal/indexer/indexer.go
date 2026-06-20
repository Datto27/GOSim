// Package indexer provides the shared embed-batch-store loop used by both
// cmd/index.go and the POST /index HTTP handler.
package indexer

import (
	"context"
	"fmt"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/Datto27/vecsim/internal/adapters"
	"github.com/Datto27/vecsim/internal/embeddings"
	"github.com/Datto27/vecsim/internal/store"
)

// Progress reports indexing progress for a single type.
type Progress struct {
	Type  string
	Done  int
	Total int // items that still needed embedding when this type's run began
}

// Run fetches unembedded items for the given type (or all types when typ is
// "all" or ""), generates embeddings in batches of batchSize via embedder,
// and persists them via s. onProgress is called after each batch (may be nil).
func Run(
	ctx context.Context,
	s *store.Store,
	e *embeddings.OllamaEmbedder,
	typ string,
	batchSize int,
	onProgress func(Progress),
) error {
	types := resolveTypes(typ)

	for _, t := range types {
		if err := runForType(ctx, s, e, t, batchSize, onProgress); err != nil {
			return err
		}
	}
	return nil
}

func resolveTypes(typ string) []string {
	if typ == "" || typ == "all" {
		return adapters.Types()
	}
	return []string{typ}
}

func runForType(
	ctx context.Context,
	s *store.Store,
	e *embeddings.OllamaEmbedder,
	typ string,
	batchSize int,
	onProgress func(Progress),
) error {
	ad, ok := adapters.Get(typ)
	if !ok {
		return fmt.Errorf("indexer: unknown type %q", typ)
	}

	// Count how many items need embedding at the start of this run.
	pending, err := s.ItemsMissingEmbedding(ctx, typ, 100_000)
	if err != nil {
		return fmt.Errorf("indexer: %s: %w", typ, err)
	}
	total := len(pending)
	done := 0

	for {
		batch, err := s.ItemsMissingEmbedding(ctx, typ, batchSize)
		if err != nil {
			return fmt.Errorf("indexer: %s: fetch batch: %w", typ, err)
		}
		if len(batch) == 0 {
			break
		}

		texts := make([]string, len(batch))
		for i, item := range batch {
			texts[i] = ad.BuildText(item.Fields)
		}

		vecs, err := e.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("indexer: %s: embed batch: %w", typ, err)
		}

		for i, item := range batch {
			if err := s.SetEmbedding(ctx, item.ID, pgvector.NewVector(vecs[i])); err != nil {
				return fmt.Errorf("indexer: %s: save embedding for %s: %w", typ, item.ID, err)
			}
			done++
			if onProgress != nil {
				onProgress(Progress{Type: typ, Done: done, Total: total})
			}
		}
	}

	return nil
}
