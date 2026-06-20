package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"os"

	"github.com/spf13/cobra"

	"github.com/Datto27/vecsim/internal/adapters"
	"github.com/Datto27/vecsim/internal/config"
)

var seedType string

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Insert hardcoded seed items",
	Long:  `Inserts the 25 hardcoded seed items for a domain (or all domains). Already-existing items are skipped.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSeed(cmd.Context())
	},
}

func init() {
	seedCmd.Flags().StringVar(&seedType, "type", "all", "Content type to seed (movie|music|book|all)")
}

func runSeed(ctx context.Context) error {
	cfg, _ := config.FromContext(ctx)
	_ = cfg

	st := storeFromCtx(ctx)
	if st == nil {
		return fmt.Errorf("seed: no store in context")
	}

	types := resolveTypeArg(seedType)
	for _, t := range types {
		if _, ok := adapters.Get(t); !ok {
			return fmt.Errorf("seed: unknown type %q", t)
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TYPE\tID\tSTATUS")
	fmt.Fprintln(w, "────\t──\t──────")

	total, inserted := 0, 0
	for _, t := range types {
		ad, _ := adapters.Get(t)
		for _, seed := range ad.Seeds() {
			item := adapters.Item{
				ID:     seed.ID,
				Label:  seed.Label,
				Type:   t,
				Fields: seed.Fields,
				Tags:   seed.Tags,
			}
			ok, err := st.InsertItem(ctx, item)
			if err != nil {
				return fmt.Errorf("seed: %s: %w", seed.ID, err)
			}
			status := "skipped"
			if ok {
				status = "inserted"
				inserted++
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", t, seed.ID, status)
			total++
		}
	}
	w.Flush()

	fmt.Printf("\n%d items processed: %d inserted, %d skipped\n", total, inserted, total-inserted)
	return nil
}

// resolveTypeArg expands "all" to all registered types, otherwise returns a
// single-element slice.
func resolveTypeArg(typ string) []string {
	if typ == "all" || typ == "" {
		return adapters.Types()
	}
	return []string{typ}
}
