package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"sentra/internal/config"
	"sentra/internal/platform/logger"
	"sentra/internal/platform/postgres"
	"sentra/internal/platform/redisclient"
	"sentra/internal/risk"
	"sentra/internal/strategy"
	"sentra/internal/observability"
)

func main() {
	if err := run(); err != nil {
		slog.Error("risk runner failed", "error", err)
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

	redisClient, err := redisclient.Connect(ctx, cfg.Redis)
	if err != nil {
		return err
	}
	defer redisClient.Close()

	riskConfig := risk.Config{
		Enabled:           cfg.Risk.Enabled,
		MaxSignalStrength: cfg.Risk.MaxSignalStrength,
		AllowBuy:          cfg.Risk.AllowBuy,
		AllowSell:         cfg.Risk.AllowSell,
	}
	if settings, settingsErr := risk.NewSettingsRepository(db).Get(ctx); settingsErr == nil {
		riskConfig = settings.Config()
	} else {
		log.Warn("risk settings unavailable, using environment risk config", "error", settingsErr)
	}

	runner := risk.NewRunner(
		strategy.NewSignalRepository(db),
		risk.NewDecisionRepository(db),
		risk.NewRedisDecisionPublisher(redisClient),
		risk.RunnerConfig{
			Symbol:         cfg.Risk.Symbol,
			DecisionStream: cfg.Risk.DecisionStream,
			Evaluator:      risk.NewEvaluator(riskConfig),
		},
	)

	decision, err := runner.RunOnce(ctx)
	if err != nil {
		return err
	}

	log.Info("risk evaluated", "symbol", decision.Symbol, "side", decision.SignalSide, "decision", decision.Decision, "reason", decision.Reason)
	return nil
}
