package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Datto27/vecsim/internal/indexer"
)

var indexType string

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Generate embeddings for un-indexed items",
	Long:  `Finds items with no stored embedding, calls Ollama in batches of 20, and saves the results. Safe to re-run.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runIndex(cmd.Context())
	},
}

func init() {
	indexCmd.Flags().StringVar(&indexType, "type", "all", "Content type to index (movie|music|book|all)")
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

	lastType := ""
	return indexer.Run(ctx, st, e, indexType, 20, func(p indexer.Progress) {
		if p.Type != lastType {
			if lastType != "" {
				fmt.Println()
			}
			lastType = p.Type
		}
		fmt.Printf("\rIndexing %s: %d/%d", p.Type, p.Done, p.Total)
		if p.Done == p.Total {
			fmt.Println()
		}
	})
}
