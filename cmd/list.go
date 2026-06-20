package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	listType   string
	listLimit  int
	listOffset int
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List indexed items",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd.Context())
	},
}

func init() {
	listCmd.Flags().StringVar(&listType, "type", "", "Filter by content type (movie|music|book)")
	listCmd.Flags().IntVar(&listLimit, "limit", 20, "Maximum items to show")
	listCmd.Flags().IntVar(&listOffset, "offset", 0, "Offset for pagination")
}

func runList(ctx context.Context) error {
	st := storeFromCtx(ctx)
	if st == nil {
		return fmt.Errorf("list: no store in context")
	}

	items, total, err := st.ListItems(ctx, listType, listLimit, listOffset)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TYPE\tID\tLABEL\tTAGS\tEMBEDDED")
	fmt.Fprintln(w, "────\t──\t─────\t────\t────────")
	for _, item := range items {
		embedded := "no"
		if item.Embedded {
			embedded = "yes"
		}
		tags := strings.Join(item.Tags, ", ")
		if len(tags) > 40 {
			tags = tags[:37] + "…"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			item.Type, item.ID, item.Label, tags, embedded)
	}
	w.Flush()

	shown := listOffset + len(items)
	fmt.Printf("\nShowing %d–%d of %d items", listOffset+1, shown, total)
	if shown < total {
		fmt.Printf(" (use --offset %d to see more)", shown)
	}
	fmt.Println()

	return nil
}
