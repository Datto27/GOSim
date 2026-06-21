// Package config defines gosim's configuration: embedding profiles, the
// on-disk config file, and a context-based mechanism for passing the loaded
// config through Cobra commands.
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Profile identifies which embedding model/dimension pair the installation
// was set up with.
type Profile string

const (
	ProfileMax       Profile = "max"
	ProfileOptimized Profile = "optimized"
)

// ProfileInfo describes a selectable embedding profile.
type ProfileInfo struct {
	Name         Profile
	Model        string
	Dimensions   int
	RAMHint      string
	DownloadHint string
}

// Profiles enumerates the two embedding profiles a user can choose from
// during `gosim setup`.
var Profiles = map[Profile]ProfileInfo{
	ProfileMax: {
		Name:         ProfileMax,
		Model:        "qwen3-embedding:8b",
		Dimensions:   4096,
		RAMHint:      "6-8 GB RAM, GPU recommended",
		DownloadHint: "~4.7 GB download",
	},
	ProfileOptimized: {
		Name:         ProfileOptimized,
		Model:        "nomic-embed-text",
		Dimensions:   768,
		RAMHint:      "runs on any modern laptop, ~600 MB RAM",
		DownloadHint: "~274 MB download",
	},
}

// Config is the persisted gosim configuration, stored as JSON at
// DefaultConfigPath (or a user-specified path via --config).
type Config struct {
	Profile     Profile `json:"profile"`
	Model       string  `json:"model"`
	Dimensions  int     `json:"dimensions"`
	OllamaURL   string  `json:"ollama_url"`
	DatabaseURL string  `json:"database_url"`
	APIPort     int     `json:"api_port"`
}

// ProfileInfo returns the ProfileInfo entry matching c.Profile.
func (c *Config) ProfileInfo() ProfileInfo {
	return Profiles[c.Profile]
}

// DefaultConfigPath returns the standard location for gosim's config file:
// $XDG_CONFIG_HOME/gosim/config.json (or the OS equivalent).
func DefaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: default path: %w", err)
	}
	return filepath.Join(dir, "gosim", "config.json"), nil
}

// DefaultPIDPath returns the standard location for the pid file written by
// 'gosim start' and read by 'gosim stop'.
func DefaultPIDPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: default pid path: %w", err)
	}
	return filepath.Join(dir, "gosim", "gosim.pid"), nil
}

// DefaultLogPath returns the standard location 'gosim start' redirects the
// background server's stdout/stderr to.
func DefaultLogPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: default log path: %w", err)
	}
	return filepath.Join(dir, "gosim", "gosim.log"), nil
}

// Load reads and parses the config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: load: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: load: %w", err)
	}

	return &cfg, nil
}

// Save writes the config as JSON to path, creating the parent directory if
// necessary.
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: save: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: save: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("config: save: %w", err)
	}

	return nil
}

type ctxKey struct{}

// WithContext returns a copy of ctx carrying cfg.
func WithContext(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, ctxKey{}, cfg)
}

// FromContext retrieves the Config stored in ctx by WithContext.
func FromContext(ctx context.Context) (*Config, bool) {
	cfg, ok := ctx.Value(ctxKey{}).(*Config)
	return cfg, ok
}
