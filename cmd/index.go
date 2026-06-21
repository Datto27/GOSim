package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Datto27/GOSim/internal/indexer"
)

var (
	indexType  string
	indexForce bool
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Generate embeddings for un-indexed items",
	Long: `Finds items with no stored embedding, calls Ollama in batches of 20, and saves
the results. Safe to re-run. Use --force to recompute embeddings for every item
(needed after changing a collection's schema or text fields).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runIndex(cmd.Context())
	},
}

func init() {
	indexCmd.Flags().StringVar(&indexType, "type", "all", "Collection to index, or \"all\" for every collection")
	indexCmd.Flags().BoolVar(&indexForce, "force", false, "Recompute embeddings for all items, not just un-indexed ones")
}

func runIndex(ctx context.Context) error {
	st := storeFromCtx(ctx)
	if st == nil {
		return fmt.Errorf("index: no store in context")
	}
	e := embedderFromCtx(ctx)
	if e == nil {
		return fmt.Errorf("index: no embedder in context")
	}

	indexedAny := false
	err := indexer.Run(ctx, st, e, indexType, 20, indexForce, func(p indexer.Progress) {
		indexedAny = true
		fmt.Printf("\rIndexing: %d/%d", p.Done, p.Total)
		if p.Done == p.Total {
			fmt.Println()
		}
	})
	if err != nil {
		return err
	}
	if !indexedAny {
		fmt.Println("Nothing to index — all items already embedded.")
	}
	return nil
}
