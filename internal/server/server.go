// Package server assembles the HTTP router and runs the API server.
package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Kerblif/Library/internal/api"
	"github.com/Kerblif/Library/internal/config"
)

// Server wraps an *http.Server with graceful shutdown.
type Server struct {
	http *http.Server
}

// New builds the HTTP server: the REST API plus, when non-nil, the MCP handler at /mcp.
func New(cfg config.Config, si api.ServerInterface, mcpHandler http.Handler) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// MCP holds long-lived SSE streams, so keep it off the per-request timeout.
	if mcpHandler != nil {
		r.Handle("/mcp", mcpHandler)
		r.Handle("/mcp/*", mcpHandler)
	}

	r.Group(func(gr chi.Router) {
		gr.Use(middleware.Timeout(30 * time.Second))
		api.HandlerFromMux(si, gr)
	})

	return &Server{
		http: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           r,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Run serves until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	errc := make(chan error, 1)
	go func() {
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.http.Shutdown(shutdownCtx)
	}
}
