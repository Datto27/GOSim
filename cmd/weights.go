package cmd

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Datto27/GOSim/internal/schema"
	"github.com/Datto27/GOSim/internal/store"
)

var weightsType string

var weightsCmd = &cobra.Command{
	Use:   "weights <param>=<n> ...",
	Short: "Set per-parameter ranking weights for a collection",
	Long: `Sets how much each detected parameter influences similarity for a collection.
Run 'gosim schema --type <name>' first to see the available parameters.

  gosim weights --type movie cast=5 genre=3 year=0
  gosim weights --type product category=4 price=2 semantic=1

Weights are non-negative; 0 excludes a parameter. "semantic" weights the text
embedding; "tags" weights tag overlap. Values are merged with existing weights,
so you can adjust one at a time. With no arguments, prints current weights.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWeights(cmd.Context(), args)
	},
}

func init() {
	weightsCmd.Flags().StringVar(&weightsType, "type", "", "Collection to configure (required)")
}

func runWeights(ctx context.Context, args []string) error {
	st := storeFromCtx(ctx)
	if st == nil {
		return fmt.Errorf("weights: no store in context")
	}
	if weightsType == "" {
		return fmt.Errorf("weights: --type is required")
	}

	sch, weights, err := st.GetCollection(ctx, weightsType)
	if err != nil {
		return err
	}
	if len(sch) == 0 {
		return fmt.Errorf("weights: no collection %q (import it first)", weightsType)
	}

	// No args → just show current weights.
	if len(args) == 0 {
		printWeights(weights)
		fmt.Printf("\nSet with:  gosim weights --type %s <param>=<n> ...\n", weightsType)
		return nil
	}

	updates, err := parseKeyValueWeights(args)
	if err != nil {
		return err
	}
	if weights == nil {
		weights = map[string]float64{}
	}
	for k, v := range updates {
		if !knownWeightKey(k, sch) {
			fmt.Printf("warning: %q is not a detected parameter of %q (allowed: semantic, tags, %s)\n",
				k, weightsType, strings.Join(paramNames(sch), ", "))
		}
		weights[k] = v
	}

	if err := st.SetCollectionWeights(ctx, weightsType, weights); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("weights: no collection %q (import it first)", weightsType)
		}
		return err
	}

	fmt.Printf("Updated weights for %q:\n", weightsType)
	printWeights(weights)
	fmt.Println("\nThese apply to every search of this collection (override per query with --weight).")
	return nil
}

func parseKeyValueWeights(args []string) (map[string]float64, error) {
	out := make(map[string]float64, len(args))
	for _, a := range args {
		k, v, ok := strings.Cut(a, "=")
		if !ok {
			return nil, fmt.Errorf("weights: %q must be in key=value form", a)
		}
		k = strings.TrimSpace(k)
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return nil, fmt.Errorf("weights: %q is not a number for %q", v, k)
		}
		if f < 0 {
			return nil, fmt.Errorf("weights: %q must not be negative", k)
		}
		out[k] = f
	}
	return out, nil
}

// knownWeightKey reports whether key is a valid weight target: the special
// "semantic"/"tags" signals or a detected schema field.
func knownWeightKey(key string, sch schema.Schema) bool {
	if key == "semantic" || key == "tags" {
		return true
	}
	_, ok := sch[key]
	return ok
}

func printWeights(weights map[string]float64) {
	keys := make([]string, 0, len(weights))
	for k := range weights {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-14s %g\n", k, weights[k])
	}
}

// paramNames returns the schema's field names, sorted.
func paramNames(sch schema.Schema) []string {
	names := make([]string, 0, len(sch))
	for name := range sch {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
