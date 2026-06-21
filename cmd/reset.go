package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	resetType string
	resetYes  bool
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Delete stored items (a collection, or everything)",
	Long: `Removes items from the database so you can re-import fresh data. By default
it deletes every item; pass --type to clear only one collection. The schema and
HNSW index are kept — re-importing repopulates them with fresh embeddings.

This is destructive and asks for confirmation unless --yes is given.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReset(cmd.Context())
	},
}

func init() {
	resetCmd.Flags().StringVar(&resetType, "type", "", "Only delete this collection (default: all)")
	resetCmd.Flags().BoolVar(&resetYes, "yes", false, "Skip the confirmation prompt")
}

func runReset(ctx context.Context) error {
	st := storeFromCtx(ctx)
	if st == nil {
		return fmt.Errorf("reset: no store in context")
	}

	target := "ALL collections"
	if resetType != "" {
		target = fmt.Sprintf("collection %q", resetType)
	}

	if !resetYes {
		fmt.Printf("This will permanently delete %s. Continue? [y/N]: ", target)
		sc := bufio.NewScanner(os.Stdin)
		sc.Scan()
		if ans := strings.ToLower(strings.TrimSpace(sc.Text())); ans != "y" && ans != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	n, err := st.DeleteItems(ctx, resetType)
	if err != nil {
		return err
	}

	fmt.Printf("Deleted %d item(s) from %s.\n", n, target)
	if resetType == "" {
		fmt.Println("Re-import with 'gosim import <file> --type <name>'.")
	} else {
		fmt.Printf("Re-import with 'gosim import <file> --type %s'.\n", resetType)
	}
	return nil
}
