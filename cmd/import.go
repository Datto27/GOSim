package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Datto27/GOSim/internal/indexer"
	"github.com/Datto27/GOSim/internal/ingest"
	"github.com/Datto27/GOSim/internal/schema"
)

var (
	importType       string
	importLabelField string
	importIDField    string
	importTagsField  string
	importNoIndex    bool
)

var importCmd = &cobra.Command{
	Use:   "import <file.json>",
	Short: "Import a JSON array of objects as a collection",
	Long: `Reads a JSON file containing an array of objects and stores each one as an
item. Every object is embedded automatically from its own fields — no per-type
code and no required envelope.

The collection name defaults to the file name; override it with --type. The
display label is auto-detected (label/title/name/heading) unless --label-field
is given. An "id" field is used as the primary key when present, otherwise an id
is generated from the label.

Examples:
  gosim import movies.json --type movie
  gosim import products.json --type product --label-field name
  gosim import people.json --type person --label-field full_name --tags-field roles

Already-existing ids are skipped. Items are embedded immediately unless
--no-index is passed (run 'gosim index' later to embed them).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport(cmd.Context(), args[0])
	},
}

func init() {
	importCmd.Flags().StringVar(&importType, "type", "", "Collection name (default: file name without extension)")
	importCmd.Flags().StringVar(&importLabelField, "label-field", "", "Object field to use as the display label (default: auto-detect)")
	importCmd.Flags().StringVar(&importIDField, "id-field", "id", "Object field to use as the primary key")
	importCmd.Flags().StringVar(&importTagsField, "tags-field", "tags", "Object field holding a string array of tags")
	importCmd.Flags().BoolVar(&importNoIndex, "no-index", false, "Insert without generating embeddings")
}

func runImport(ctx context.Context, path string) error {
	st := storeFromCtx(ctx)
	if st == nil {
		return fmt.Errorf("import: no store in context")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("import: read file: %w", err)
	}

	var raw []map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("import: %s must be a JSON array of objects: %w", filepath.Base(path), err)
	}
	if len(raw) == 0 {
		return fmt.Errorf("import: %s contains no objects", filepath.Base(path))
	}

	typ := importType
	if typ == "" {
		typ = collectionFromFilename(path)
	}

	items, err := ingest.Normalize(raw, ingest.Options{
		Type:       typ,
		LabelField: importLabelField,
		IDField:    importIDField,
		TagsField:  importTagsField,
	})
	if err != nil {
		return err
	}

	// Detect the collection's structure from the items' stored fields and
	// persist it (merged with any prior schema). Preserve user-set weights;
	// initialize defaults only for a brand-new collection.
	fieldMaps := make([]map[string]any, len(items))
	for i, item := range items {
		fieldMaps[i] = item.Fields
	}
	detected := schema.Detect(fieldMaps)
	existingSchema, weights, err := st.GetCollection(ctx, typ)
	if err != nil {
		return err
	}
	merged := schema.Merge(existingSchema, detected)
	weights = schema.EnsureWeights(merged, weights) // fill defaults for any unset parameter
	if err := st.UpsertCollection(ctx, typ, merged, weights); err != nil {
		return err
	}

	inserted, skipped := 0, 0
	for _, item := range items {
		ok, err := st.InsertItem(ctx, item)
		if err != nil {
			return fmt.Errorf("import: insert %s: %w", item.ID, err)
		}
		if ok {
			inserted++
		} else {
			skipped++
		}
	}

	fmt.Printf("Imported into %q: %d inserted, %d skipped (already existed)\n", typ, inserted, skipped)

	if importNoIndex {
		fmt.Printf("Skipped embedding (--no-index). Run 'gosim index --type %s' to embed.\n", typ)
		return nil
	}

	e := embedderFromCtx(ctx)
	if e == nil {
		return fmt.Errorf("import: no embedder in context")
	}

	fmt.Println("Generating embeddings…")
	indexedAny := false
	err = indexer.Run(ctx, st, e, typ, 20, false, func(p indexer.Progress) {
		indexedAny = true
		fmt.Printf("\rIndexing: %d/%d", p.Done, p.Total)
		if p.Done == p.Total {
			fmt.Println()
		}
	})
	if err != nil {
		return err
	}
	if !indexedAny {
		fmt.Println("Nothing to embed — all imported items were already indexed.")
	}
	return nil
}

// collectionFromFilename derives a default collection name from a file path,
// e.g. "/data/Movies.JSON" → "movies".
func collectionFromFilename(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if s := ingest.Slug(base); s != "" {
		return s
	}
	return "items"
}
