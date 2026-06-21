package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"

	// Registers the "pgx" database/sql driver used by goose.
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/Datto27/GOSim/internal/config"
	"github.com/Datto27/GOSim/internal/store"
	"github.com/Datto27/GOSim/migrations"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long: `Creates the items, gosim_meta, and collections tables plus the HNSW index in
Postgres. The vector column size is determined by the active embedding profile.

Safe to re-run — goose is idempotent.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMigrate(cmd.Context())
	},
}

func runMigrate(ctx context.Context) error {
	cfg, ok := config.FromContext(ctx)
	if !ok {
		return fmt.Errorf("migrate: no config in context")
	}

	// Inject the dimension so that the SQL template's ${GOSIM_DIMENSIONS}
	// placeholder is substituted by goose's envsub mechanism.
	os.Setenv("GOSIM_DIMENSIONS", strconv.Itoa(cfg.Dimensions))

	sqlDB, err := goose.OpenDBWithDriver("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("migrate: open db: %w", err)
	}
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.FS)
	defer goose.SetBaseFS(nil)

	goose.SetLogger(goose.NopLogger())

	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	fmt.Printf("Migrations applied (profile: %s, dims: %d)\n", cfg.Profile, cfg.Dimensions)

	// Record the active profile in the DB so future startups can validate
	// that the config and the migrated schema are consistent.
	pool := poolFromCtx(ctx)
	if pool != nil {
		st := store.New(pool)
		if err := st.SetMeta(ctx, "profile", string(cfg.Profile)); err != nil {
			return fmt.Errorf("migrate: record profile: %w", err)
		}
		fmt.Printf("Profile recorded in gosim_meta\n")
	}

	return nil
}
