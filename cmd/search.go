package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	pgvector "github.com/pgvector/pgvector-go"
	"github.com/spf13/cobra"

	"github.com/Datto27/GOSim/internal/rerank"
	"github.com/Datto27/GOSim/internal/store"
)

// candidatePool bounds how many items are pulled from pgvector for hybrid
// re-ranking. When structured weights are active the ranking can be driven by
// fields (cast, genre) that are semantically distant, so we must consider the
// whole collection rather than only the semantic top-N; this cap keeps that
// bounded for very large collections.
const candidatePool = 10000

var (
	searchType      string
	searchCrossType bool
	searchLimit     int
	searchWeights   map[string]string
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Find similar items by title or free text",
	Long: `If <query> exactly matches a stored item's label, returns the items most
similar to it, ranked by the collection's per-parameter weights (set with
'gosim weights'; defaults otherwise). Otherwise <query> is treated as free text,
embedded on the fly, and the closest items are returned — so a search always
finds something.

Use --cross-type to search across all collections simultaneously.

Use --weight key=value (repeatable, or comma-separated) to set your own
emphasis, e.g. --weight genre=3,cast=2 or --weight semantic=1 for pure semantic.
Recognized keys are "semantic", "tags", and any field name present on your
imported objects. Only the signals you list (plus semantic) are blended.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSearch(cmd.Context(), args[0])
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchType, "type", "", "Restrict query item lookup to a specific collection")
	searchCmd.Flags().BoolVar(&searchCrossType, "cross-type", false, "Search across all collections")
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

func runSearch(ctx context.Context, query string) error {
	st := storeFromCtx(ctx)
	if st == nil {
		return fmt.Errorf("search: no store in context")
	}

	weights, err := parseWeights(searchWeights)
	if err != nil {
		return err
	}

	// Try to resolve the query as an exact stored item; fall back to treating
	// it as free text when there is no such item.
	item, err := st.GetItemByLabel(ctx, query, searchType)
	if errors.Is(err, store.ErrNotFound) {
		return runTextSearch(ctx, st, query)
	}
	if err != nil {
		return err // ambiguous label, etc. — surface the message
	}

	vec, err := st.GetEmbedding(ctx, item.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotEmbedded) {
			return fmt.Errorf("search: %q has not been indexed yet — run 'gosim index --type %s'", query, item.Type)
		}
		return err
	}

	// Use the collection's persisted per-parameter weights unless overridden.
	sch, persisted, err := st.GetCollection(ctx, item.Type)
	if err != nil {
		return err
	}
	if len(weights) == 0 {
		weights = persisted
	}

	typeFilter := item.Type
	if searchCrossType {
		typeFilter = ""
	}

	results, err := st.SearchByVector(ctx, vec, typeFilter, item.ID, fetchLimitFor(searchLimit, weights))
	if err != nil {
		return err
	}
	results = rerank.Rerank(*item, results, sch, weights, searchLimit)

	crossLabel := ""
	if searchCrossType {
		crossLabel = " (cross-type)"
	}
	fmt.Printf("Similar to \"%s\" [%s]%s:\n\n", item.Label, item.Type, crossLabel)
	printResults(results)
	return nil
}

// runTextSearch embeds an arbitrary query string and returns the closest items
// by semantic similarity. Field weights don't apply (there's no structured
// query item), so ranking is purely semantic.
func runTextSearch(ctx context.Context, st *store.Store, text string) error {
	e := embedderFromCtx(ctx)
	if e == nil {
		return fmt.Errorf("search: no embedder in context")
	}

	vecs, err := e.Embed(ctx, []string{text})
	if err != nil {
		return fmt.Errorf("search: embed query: %w", err)
	}

	typeFilter := searchType
	if searchCrossType {
		typeFilter = ""
	}

	results, err := st.SearchByVector(ctx, pgvector.NewVector(vecs[0]), typeFilter, "", searchLimit)
	if err != nil {
		return err
	}

	fmt.Printf("No exact title %q — closest matches by text:\n\n", text)
	printResults(results)
	return nil
}

// fetchLimitFor returns how many candidates to pull before re-ranking: the
// whole collection (capped) when structured weights are active, otherwise just
// the requested limit (pure semantic order needs no wider pool).
func fetchLimitFor(limit int, weights map[string]float64) int {
	if rerank.HasStructuralWeight(weights) {
		return candidatePool
	}
	return limit
}

// printResults renders search results as a ranked table.
func printResults(results []store.SearchResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RANK\tTYPE\tLABEL\tSCORE")
	fmt.Fprintln(w, "────\t────\t─────\t─────")
	for i, r := range results {
		fmt.Fprintf(w, "%d\t%s\t%s\t%.4f\n", i+1, r.Item.Type, r.Item.Label, r.Score)
	}
	w.Flush()

	if len(results) == 0 {
		fmt.Println("No results found. Ensure items are imported and indexed.")
	}
}
