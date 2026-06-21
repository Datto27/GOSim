package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
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

	types := make([]string, 0, len(result.ByType))
	for t := range result.ByType {
		types = append(types, t)
	}
	sort.Strings(types)

	for _, t := range types {
		ts := result.ByType[t]
		pct := 0.0
		if ts.Count > 0 {
			pct = 100.0 * float64(ts.Embedded) / float64(ts.Count)
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%.1f%%\n", t, ts.Count, ts.Embedded, pct)
	}

	if len(types) == 0 {
		fmt.Fprintln(w, "(no collections yet — import data with 'gosim import <file>')\t\t\t")
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
