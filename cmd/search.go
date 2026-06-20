package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Datto27/vecsim/internal/rerank"
	"github.com/Datto27/vecsim/internal/store"
)

// candidatePoolMultiplier and candidatePoolMax bound how many candidates are
// fetched from pgvector before re-ranking by weight, so the weighted-closest
// item isn't missed just because it wasn't in the top searchLimit by raw
// cosine similarity.
const (
	candidatePoolMultiplier = 5
	candidatePoolMax        = 200
)

var (
	searchType      string
	searchCrossType bool
	searchLimit     int
	searchWeights   map[string]string
)

var searchCmd = &cobra.Command{
	Use:   "search <title>",
	Short: "Find similar items by title",
	Long: `Fetches the stored embedding for <title> and returns the top similar
items ranked by cosine similarity score (1.0 = identical).

Use --cross-type to search across all content types simultaneously.

Use --weight key=value (repeatable, or comma-separated) to weight individual
parameters in the ranking, e.g. --weight genre=2,year=0.5,cast=0. Recognized
keys are "semantic" (the base similarity score), "tags", and any field name
from the item's domain (year, genre, cast, plot, artist, mood, author,
synopsis, ...). Fields not mentioned default to a weight of 1.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSearch(cmd.Context(), args[0])
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchType, "type", "", "Restrict query item lookup to a specific type (movie|music|book)")
	searchCmd.Flags().BoolVar(&searchCrossType, "cross-type", false, "Search across all content types")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Number of results to return")
	searchCmd.Flags().StringToStringVar(&searchWeights, "weight", nil, "Per-field weight as key=value (repeatable or comma-separated)")
}

// parseWeights converts the --weight flag's string values to float64,
// rejecting negative weights.
func parseWeights(raw map[string]string) (map[string]float64, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	weights := make(map[string]float64, len(raw))
	for k, v := range raw {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("search: invalid weight for %q: %q is not a number", k, v)
		}
		if f < 0 {
			return nil, fmt.Errorf("search: invalid weight for %q: weights must not be negative", k)
		}
		weights[k] = f
	}
	return weights, nil
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

	weights, err := parseWeights(searchWeights)
	if err != nil {
		return err
	}

	typeFilter := item.Type
	if searchCrossType {
		typeFilter = ""
	}

	fetchLimit := searchLimit
	if len(weights) > 0 {
		fetchLimit = min(searchLimit*candidatePoolMultiplier, candidatePoolMax)
	}

	results, err := st.SearchByVector(ctx, vec, typeFilter, item.ID, fetchLimit)
	if err != nil {
		return err
	}
	results = rerank.Rerank(*item, results, weights, searchLimit)

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
