package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/Datto27/vecsim/internal/config"
	"github.com/Datto27/vecsim/internal/store"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show current configuration and connectivity status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInfo(cmd.Context())
	},
}

func runInfo(ctx context.Context) error {
	cfg, ok := config.FromContext(ctx)
	if !ok {
		return fmt.Errorf("info: no config in context")
	}
	e := embedderFromCtx(ctx)
	st := storeFromCtx(ctx)

	fmt.Printf("Profile:      %s (%s, %d dims)\n", cfg.Profile, cfg.Model, cfg.Dimensions)
	fmt.Printf("Config path:  %s\n", configPath)
	fmt.Printf("API port:     %d\n", cfg.APIPort)
	fmt.Println()

	// Ollama
	ollamaStatus := "unreachable"
	if e != nil {
		if err := e.Health(ctx); err == nil {
			ollamaStatus = "reachable"
		}
	}
	fmt.Printf("Ollama:       %s  [%s]\n", cfg.OllamaURL, ollamaStatus)

	// Database
	dbDisplay := maskPassword(cfg.DatabaseURL)
	dbStatus := "not connected"
	profileStatus := ""
	if st != nil {
		dbStatus = "connected"
		if stored, err := st.GetMeta(ctx, "profile"); err == nil {
			if stored == string(cfg.Profile) {
				profileStatus = ", profile matches"
			} else {
				profileStatus = fmt.Sprintf(", profile mismatch: DB=%q config=%q", stored, cfg.Profile)
			}
		} else if !errors.Is(err, store.ErrNotFound) {
			profileStatus = ", meta not readable"
		}
	}
	fmt.Printf("Database:     %s  [%s%s]\n", dbDisplay, dbStatus, profileStatus)

	return nil
}

// maskPassword replaces the password in a Postgres URL with "***".
func maskPassword(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if u.User != nil {
		if _, hasPass := u.User.Password(); hasPass {
			u.User = url.UserPassword(u.User.Username(), "***")
		}
	}
	return u.String()
}
