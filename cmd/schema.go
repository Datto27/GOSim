package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Datto27/GOSim/internal/schema"
)

var schemaType string

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Show detected parameters and weights for a collection",
	Long: `Prints the fields GOSim detected for a collection — each one's kind
(text / list / number / keyword) and current ranking weight — so you know what
you can tune with 'gosim weights'. Without --type, lists all collections.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSchema(cmd.Context())
	},
}

func init() {
	schemaCmd.Flags().StringVar(&schemaType, "type", "", "Collection to inspect (default: list all)")
}

func runSchema(ctx context.Context) error {
	st := storeFromCtx(ctx)
	if st == nil {
		return fmt.Errorf("schema: no store in context")
	}

	if schemaType == "" {
		types, err := st.ListCollections(ctx)
		if err != nil {
			return err
		}
		if len(types) == 0 {
			fmt.Println("No collections yet. Import data with 'gosim import <file> --type <name>'.")
			return nil
		}
		fmt.Println("Collections (use --type <name> for detail):")
		for _, t := range types {
			fmt.Printf("  - %s\n", t)
		}
		return nil
	}

	sch, weights, err := st.GetCollection(ctx, schemaType)
	if err != nil {
		return err
	}
	if len(sch) == 0 {
		return fmt.Errorf("schema: no collection %q (import it first)", schemaType)
	}

	names := make([]string, 0, len(sch))
	for name := range sch {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Printf("Collection %q:\n\n", schemaType)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PARAMETER\tKIND\tWEIGHT\tNOTES")
	fmt.Fprintln(w, "─────────\t────\t──────\t─────")
	fmt.Fprintf(w, "semantic\ttext\t%s\tembedding of text fields\n", weightStr(weights, "semantic", 1))
	for _, name := range names {
		f := sch[name]
		notes := ""
		if f.Kind == schema.Number && f.Min != nil && f.Max != nil {
			notes = fmt.Sprintf("range %.0f–%.0f", *f.Min, *f.Max)
		}
		if f.Kind == schema.Text {
			notes = "folded into semantic"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, f.Kind, weightStr(weights, name, 0), notes)
	}
	w.Flush()

	fmt.Printf("\nTune with:  gosim weights --type %s <param>=<n> ...\n", schemaType)
	return nil
}

// weightStr formats a weight value, falling back to def when absent.
func weightStr(weights map[string]float64, key string, def float64) string {
	v, ok := weights[key]
	if !ok {
		v = def
	}
	return fmt.Sprintf("%g", v)
}
