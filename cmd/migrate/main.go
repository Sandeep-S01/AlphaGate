package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"sentra/internal/config"
	"sentra/internal/observability"
	"sentra/internal/platform/logger"
	"sentra/internal/platform/migrations"
	"sentra/internal/platform/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.Logging)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	metrics := observability.NewRegistry()
	db, err := postgres.Connect(ctx, cfg.Postgres, metrics)
	if err != nil {
		return err
	}
	defer db.Close()

	dir := "migrations"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	if err := migrations.NewRunner(db, metrics).Up(ctx, dir); err != nil {
		return err
	}

	log.Info("migrations applied", "dir", dir)
	return nil
}
