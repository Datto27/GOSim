package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Datto27/GOSim/internal/config"
	"github.com/Datto27/GOSim/internal/db"
	"github.com/Datto27/GOSim/internal/embeddings"
	"github.com/Datto27/GOSim/internal/store"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose configuration and connectivity",
	Long: `Runs read-only checks against every prerequisite GOSim needs — config
file, Ollama, the embedding model, Postgres, and applied migrations — and prints
the exact command to fix anything that is not ready.`,
	// Self-contained: must run before a config file or DB connection exists.
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		runDoctor(cmd.Context())
		return nil
	},
}

// check prints one diagnostic line and returns ok so the caller can tally
// failures. fix is printed (indented) only when ok is false.
func check(ok bool, name, detail, fix string) bool {
	mark := "✓"
	if !ok {
		mark = "✗"
	}
	fmt.Printf("  %s %-22s %s\n", mark, name, detail)
	if !ok && fix != "" {
		fmt.Printf("      ↳ fix: %s\n", fix)
	}
	return ok
}

func runDoctor(ctx context.Context) {
	fmt.Println()
	fmt.Println("GOSim doctor")
	fmt.Println()

	failures := 0
	fail := func(ok bool) {
		if !ok {
			failures++
		}
	}

	// ── Config file ─────────────────────────────────────────────────────────
	path := configPath
	if path == "" {
		if p, err := config.DefaultConfigPath(); err == nil {
			path = p
		}
	}
	cfg, err := config.Load(path)
	fail(check(err == nil, "Config file",
		statusOrErr(err == nil, path, "not found"),
		"gosim setup --pull"))
	if err != nil {
		fmt.Printf("\n%d check(s) failed. Run 'gosim setup' first.\n", failures)
		return
	}

	embedder := embeddings.New(cfg.OllamaURL, cfg.Model, cfg.Dimensions)

	// ── Ollama reachable ──────────────────────────────────────────────────────
	ollamaOK := embedder.Health(ctx) == nil
	fail(check(ollamaOK, "Ollama",
		statusOrErr(ollamaOK, cfg.OllamaURL+" reachable", cfg.OllamaURL+" unreachable"),
		"start Ollama (see https://ollama.com)"))

	// ── Embedding model present ───────────────────────────────────────────────
	if ollamaOK {
		present, _ := modelInstalled(ctx, cfg.OllamaURL, cfg.Model)
		fail(check(present, "Embedding model",
			statusOrErr(present, cfg.Model+" installed", cfg.Model+" not pulled"),
			"ollama pull "+cfg.Model))
	} else {
		fail(check(false, "Embedding model", "skipped (Ollama unreachable)", "ollama pull "+cfg.Model))
	}

	// ── Postgres reachable ────────────────────────────────────────────────────
	pool, dbErr := db.Connect(ctx, cfg.DatabaseURL)
	dbOK := dbErr == nil
	fail(check(dbOK, "Postgres",
		statusOrErr(dbOK, "connected", "not reachable"),
		"docker compose up -d"))
	if dbOK {
		defer pool.Close()

		// ── Migrations applied ───────────────────────────────────────────────
		st := store.New(pool)
		stored, metaErr := st.GetMeta(ctx, "profile")
		migrated := metaErr == nil
		switch {
		case migrated && stored == string(cfg.Profile):
			fail(check(true, "Migrations", "applied, profile matches", ""))
		case migrated:
			fail(check(false, "Migrations",
				fmt.Sprintf("profile mismatch: DB=%q config=%q", stored, cfg.Profile),
				"gosim migrate (after aligning profiles)"))
		case errors.Is(metaErr, store.ErrNotFound):
			fail(check(false, "Migrations", "tables present but profile not recorded", "gosim migrate"))
		default:
			fail(check(false, "Migrations", "not applied", "gosim migrate"))
		}
	} else {
		fail(check(false, "Migrations", "skipped (Postgres unreachable)", "docker compose up -d && gosim migrate"))
	}

	fmt.Println()
	if failures == 0 {
		fmt.Println("All checks passed — GOSim is ready. Import data with 'gosim import <file>'.")
	} else {
		fmt.Printf("%d check(s) failed. Fix the items marked ✗ above, then re-run 'gosim doctor'.\n", failures)
	}
}

func statusOrErr(ok bool, okMsg, failMsg string) string {
	if ok {
		return okMsg
	}
	return failMsg
}

// modelInstalled queries Ollama's /api/tags and reports whether model is among
// the installed models, tolerating the implicit ":latest" tag.
func modelInstalled(ctx context.Context, ollamaURL, model string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(ollamaURL, "/")+"/api/tags", nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("ollama returned %s", resp.Status)
	}

	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, err
	}
	for _, m := range body.Models {
		if m.Name == model || strings.HasPrefix(m.Name, model+":") {
			return true, nil
		}
	}
	return false, nil
}
