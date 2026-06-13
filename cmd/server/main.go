// Command server runs the Library HTTP API.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kerblif/Library/internal/config"
	"github.com/Kerblif/Library/internal/rest"
	"github.com/Kerblif/Library/internal/server"
	"github.com/Kerblif/Library/internal/store/postgres"
)

func main() {
	cfg := config.Load()

	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	repo, err := postgres.New(connectCtx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		slog.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	defer repo.Close()

	srv := server.New(cfg, rest.New(repo).ServerInterface())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("server listening", "addr", cfg.HTTPAddr)
	if err := srv.Run(ctx); err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
