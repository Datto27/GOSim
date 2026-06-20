package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Datto27/vecsim/internal/store"
)

var (
	searchType      string
	searchCrossType bool
	searchLimit     int
)

var searchCmd = &cobra.Command{
	Use:   "search <title>",
	Short: "Find similar items by title",
	Long: `Fetches the stored embedding for <title> and returns the top similar
items ranked by cosine similarity score (1.0 = identical).

Use --cross-type to search across all content types simultaneously.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSearch(cmd.Context(), args[0])
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchType, "type", "", "Restrict query item lookup to a specific type (movie|music|book)")
	searchCmd.Flags().BoolVar(&searchCrossType, "cross-type", false, "Search across all content types")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Number of results to return")
}

func runSearch(ctx context.Context, label string) error {
	st := storeFromCtx(ctx)
	if st == nil {
		return fmt.Errorf("search: no store in context")
	}

	item, err := st.GetItemByLabel(ctx, label, searchType)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("search: no item found with label %q (try 'vecsim list')", label)
		}
		return err
	}

	vec, err := st.GetEmbedding(ctx, item.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotEmbedded) {
			return fmt.Errorf("search: %q has not been indexed yet — run 'vecsim index --type %s'", label, item.Type)
		}
		return err
	}

	typeFilter := item.Type
	if searchCrossType {
		typeFilter = ""
	}

	results, err := st.SearchByVector(ctx, vec, typeFilter, item.ID, searchLimit)
	if err != nil {
		return err
	}

	crossLabel := ""
	if searchCrossType {
		crossLabel = " (cross-type)"
	}
	fmt.Printf("Similar to \"%s\" [%s]%s:\n\n", item.Label, item.Type, crossLabel)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RANK\tTYPE\tLABEL\tSCORE")
	fmt.Fprintln(w, "────\t────\t─────\t─────")
	for i, r := range results {
		fmt.Fprintf(w, "%d\t%s\t%s\t%.4f\n", i+1, r.Item.Type, r.Item.Label, r.Score)
	}
	w.Flush()

	if len(results) == 0 {
		fmt.Println("No results found. Ensure items are seeded and indexed.")
	}

	return nil
}
