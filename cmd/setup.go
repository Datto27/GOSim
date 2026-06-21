package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Datto27/GOSim/internal/config"
	"github.com/Datto27/GOSim/internal/db"
	"github.com/Datto27/GOSim/internal/embeddings"
)

var setupPull bool

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run the interactive first-run wizard",
	Long: `Presents the two embedding profiles, prompts for connection details,
checks connectivity, and writes ~/.config/gosim/config.json.

Use --pull to automatically download the embedding model via Ollama.`,
	Annotations: map[string]string{"skipConfig": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetup(cmd.Context())
	},
}

func init() {
	setupCmd.Flags().BoolVar(&setupPull, "pull", false, "Pull the embedding model from Ollama after setup")
}

func runSetup(ctx context.Context) error {
	sc := bufio.NewScanner(os.Stdin)
	prompt := func(label, defaultVal string) string {
		if defaultVal != "" {
			fmt.Printf("  %s [%s]: ", label, defaultVal)
		} else {
			fmt.Printf("  %s: ", label)
		}
		sc.Scan()
		if v := strings.TrimSpace(sc.Text()); v != "" {
			return v
		}
		return defaultVal
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║         GOSim — first-run setup         ║")
	fmt.Println("╚══════════════════════════════════════════╝")

	// ── Profile selection ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("Choose an embedding profile:")
	fmt.Println()
	fmt.Println("  [1] max       — qwen3-embedding:8b  (4096 dims)")
	fmt.Println("                  Best quality. 6-8 GB RAM, GPU recommended.")
	fmt.Println("                  ~4.7 GB download.")
	fmt.Println()
	fmt.Println("  [2] optimized — nomic-embed-text    (768 dims)")
	fmt.Println("                  Great quality. Runs on any modern laptop, ~600 MB RAM.")
	fmt.Println("                  ~274 MB download.  ← Recommended for most users")
	fmt.Println()

	var selectedProfile config.Profile
	for {
		raw := prompt("Profile (1 or 2)", "2")
		switch raw {
		case "1", "max":
			selectedProfile = config.ProfileMax
		case "2", "optimized":
			selectedProfile = config.ProfileOptimized
		default:
			fmt.Println("  Please enter 1 (max) or 2 (optimized).")
			continue
		}
		break
	}
	pi := config.Profiles[selectedProfile]
	fmt.Printf("  → %s selected (%s, %d dims)\n", pi.Name, pi.Model, pi.Dimensions)

	// ── Connection details ─────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("Connection details:")
	ollamaURL := prompt("Ollama URL", "http://localhost:11434")
	databaseURL := prompt("Database URL", "postgres://gosim:gosim@localhost:5432/gosim?sslmode=disable")
	apiPortStr := prompt("API port", "7700")

	apiPort, err := strconv.Atoi(apiPortStr)
	if err != nil || apiPort < 1 || apiPort > 65535 {
		apiPort = 7700
	}

	// ── Connectivity checks ────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("Checking connectivity…")

	embedder := embeddings.New(ollamaURL, pi.Model, pi.Dimensions)
	if err := embedder.Health(ctx); err != nil {
		fmt.Printf("  ✗ Ollama at %s — %v\n", ollamaURL, err)
		fmt.Println("    (Ollama must be running before embedding; you can proceed and start it later.)")
	} else {
		fmt.Printf("  ✓ Ollama at %s — reachable\n", ollamaURL)
	}

	if _, dbErr := db.Connect(ctx, databaseURL); dbErr != nil {
		fmt.Printf("  ✗ Postgres — %v\n", dbErr)
		fmt.Println("    (Start Postgres with 'docker compose up -d' then run 'gosim migrate'.)")
	} else {
		fmt.Printf("  ✓ Postgres — connected\n")
	}

	// ── Save config ────────────────────────────────────────────────────────
	cfg := &config.Config{
		Profile:     selectedProfile,
		Model:       pi.Model,
		Dimensions:  pi.Dimensions,
		OllamaURL:   ollamaURL,
		DatabaseURL: databaseURL,
		APIPort:     apiPort,
	}

	if configPath == "" {
		p, err := config.DefaultConfigPath()
		if err != nil {
			return err
		}
		configPath = p
	}

	if err := cfg.Save(configPath); err != nil {
		return err
	}
	fmt.Printf("\n  Config saved to %s\n", configPath)

	// ── Optional model pull ────────────────────────────────────────────────
	if setupPull {
		if present, _ := modelInstalled(ctx, ollamaURL, pi.Model); present {
			fmt.Printf("\n  ✓ %s already installed — skipping pull\n", pi.Model)
		} else {
			fmt.Printf("\nPulling %s…\n", pi.Model)
			err := embedder.Pull(ctx, pi.Model, func(status string, completed, total int64) {
				if total > 0 {
					pct := 100 * completed / total
					fmt.Printf("\r  %s: %d%%  ", status, pct)
				} else {
					fmt.Printf("\r  %s…   ", status)
				}
			})
			fmt.Println()
			if err != nil {
				return fmt.Errorf("pull failed: %w", err)
			}
			fmt.Println("  ✓ Model ready")
		}
	}

	// ── Next steps ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. docker compose up -d              (if Postgres isn't running yet)")
	fmt.Println("  2. gosim migrate                     (create tables & HNSW index)")
	fmt.Println("  3. gosim doctor                      (verify everything is ready)")
	fmt.Println("  4. gosim import <file.json> --type <name>   (load & embed your data)")
	fmt.Println("  5. gosim search \"<a label>\" --cross-type")
	fmt.Println("     gosim serve                       (start the HTTP API on localhost:7700)")
	fmt.Println()

	return nil
}
