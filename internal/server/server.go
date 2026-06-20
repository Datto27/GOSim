// Package server wires together vecsim's HTTP API.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Server wraps an http.Server with graceful-shutdown support.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// New builds the ServeMux, wraps it with middleware, and returns a Server
// ready to run on localhost:<port>.
func New(h *Handlers, port int, logger *slog.Logger) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /items", h.ListItems)
	mux.HandleFunc("GET /items/{id}", h.GetItem)
	mux.HandleFunc("POST /items", h.CreateItem)
	mux.HandleFunc("DELETE /items/{id}", h.DeleteItem)
	mux.HandleFunc("POST /search", h.Search)
	mux.HandleFunc("POST /search/embed", h.SearchByText)
	mux.HandleFunc("GET /stats", h.Stats)
	mux.HandleFunc("POST /index", h.Index)

	handler := Chain(mux,
		CORS,
		RequestID,
		Logging(logger),
		Recovery(logger),
	)

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf("localhost:%d", port),
			Handler: handler,
		},
		logger: logger,
	}
}

// Run starts the server, blocks until ctx is cancelled, then drains
// in-flight requests for up to 10 seconds before force-closing.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	s.logger.Info("vecsim HTTP API started", "addr", s.httpServer.Addr)

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.logger.Info("shutting down gracefully…")
	return s.httpServer.Shutdown(shutdownCtx)
}
