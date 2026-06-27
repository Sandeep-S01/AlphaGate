package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"sentra/internal/config"
	"sentra/internal/marketdata"
	"sentra/internal/platform/logger"
	"sentra/internal/platform/postgres"
	"sentra/internal/platform/redisclient"
	"sentra/internal/strategy"
	"sentra/internal/observability"
)

func main() {
	if err := run(); err != nil {
		slog.Error("strategy runner failed", "error", err)
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

	settings := strategy.Settings{
		StrategyName:  cfg.Strategy.Name,
		Version:       cfg.Strategy.Version,
		Symbol:        cfg.Strategy.Symbol,
		Interval:      cfg.Strategy.Interval,
		FastPeriod:    cfg.Strategy.FastPeriod,
		SlowPeriod:    cfg.Strategy.SlowPeriod,
		LookbackLimit: cfg.Strategy.LookbackLimit,
		RSIPeriod:     cfg.Strategy.RSIPeriod,
		RSIOversold:   cfg.Strategy.RSIOversold,
		RSIOverbought: cfg.Strategy.RSIOverbought,
	}
	evaluator, err := strategy.NewEvaluatorFromSettings(settings)
	if err != nil {
		return err
	}

	runner := strategy.NewRunner(
		marketdata.NewCandleRepository(db),
		strategy.NewSignalRepository(db),
		strategy.NewRedisSignalPublisher(redisClient),
		strategy.RunnerConfig{
			Symbol:        cfg.Strategy.Symbol,
			Interval:      cfg.Strategy.Interval,
			LookbackLimit: cfg.Strategy.LookbackLimit,
			SignalStream:  cfg.Strategy.SignalStream,
			Evaluator:     evaluator,
		},
	)

	signal, err := runner.RunOnce(ctx)
	if err != nil {
		return err
	}

	log.Info("strategy evaluated", "strategy", signal.StrategyName, "symbol", signal.Symbol, "interval", signal.Interval, "side", signal.Side, "strength", signal.Strength)
	return nil
}
