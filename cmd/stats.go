package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Datto27/vecsim/internal/adapters"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show item counts and embedding coverage",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStats(cmd.Context())
	},
}

func runStats(ctx context.Context) error {
	st := storeFromCtx(ctx)
	if st == nil {
		return fmt.Errorf("stats: no store in context")
	}

	result, err := st.Stats(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TYPE\tTOTAL\tEMBEDDED\tCOVERAGE")
	fmt.Fprintln(w, "────\t─────\t────────\t────────")

	for _, t := range adapters.Types() {
		ts, ok := result.ByType[t]
		if !ok {
			continue
		}
		pct := 0.0
		if ts.Count > 0 {
			pct = 100.0 * float64(ts.Embedded) / float64(ts.Count)
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%.1f%%\n", t, ts.Count, ts.Embedded, pct)
	}

	totalPct := 0.0
	if result.Total > 0 {
		totalPct = 100.0 * float64(result.Embedded) / float64(result.Total)
	}
	fmt.Fprintln(w, "────\t─────\t────────\t────────")
	fmt.Fprintf(w, "TOTAL\t%d\t%d\t%.1f%%\n", result.Total, result.Embedded, totalPct)
	w.Flush()

	return nil
}
