// Command server runs the Library HTTP API.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kerblif/Library/internal/config"
	"github.com/Kerblif/Library/internal/rest"
	"github.com/Kerblif/Library/internal/server"
)

func main() {
	cfg := config.Load()

	srv := server.New(cfg, rest.New().ServerInterface())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("server listening", "addr", cfg.HTTPAddr)
	if err := srv.Run(ctx); err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
