package cmd

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/Datto27/vecsim/internal/config"
	"github.com/Datto27/vecsim/internal/server"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server on localhost",
	Long:  `Starts the vecsim HTTP API on localhost:<port>. Handles graceful shutdown on SIGINT/SIGTERM.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.FromContext(cmd.Context())
		port := servePort
		if port == 0 && cfg != nil {
			port = cfg.APIPort
		}
		if port == 0 {
			port = 7700
		}

		st := storeFromCtx(cmd.Context())
		e := embedderFromCtx(cmd.Context())
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		h := server.NewHandlers(st, e, cfg)
		srv := server.New(h, port, logger)

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		return srv.Run(ctx)
	},
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 0, "Port to listen on (default: value from config)")
}
