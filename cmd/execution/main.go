package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"sentra/internal/config"
	"sentra/internal/execution"
	"sentra/internal/marketdata"
	"sentra/internal/platform/logger"
	"sentra/internal/platform/postgres"
	"sentra/internal/platform/redisclient"
	"sentra/internal/risk"
	"sentra/internal/observability"
)

func main() {
	if err := run(); err != nil {
		slog.Error("paper execution failed", "error", err)
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

	candleRepo := marketdata.NewCandleRepository(db)
	runner := execution.NewRunner(
		risk.NewDecisionRepository(db),
		execution.NewMarketPriceReader(candleRepo),
		execution.NewAccountRepository(db),
		execution.NewRepository(db),
		execution.NewRedisPublisher(redisClient),
		execution.RunnerConfig{
			Symbol:          cfg.Execution.Symbol,
			Interval:        cfg.Execution.Interval,
			ExecutionStream: cfg.Execution.Stream,
			Engine: execution.NewPaperEngine(execution.Config{
				Enabled:          cfg.Execution.Enabled,
				Symbol:           cfg.Execution.Symbol,
				BaseAsset:        cfg.Execution.BaseAsset,
				QuoteAsset:       cfg.Execution.QuoteAsset,
				QuoteOrderAmount: cfg.Execution.QuoteOrderAmount,
				FeeRate:          cfg.Execution.FeeRate,
			}),
		},
	)

	result, err := runner.RunOnce(ctx)
	if err != nil {
		return err
	}

	log.Info("paper order filled", "symbol", result.Order.Symbol, "side", result.Order.Side, "quantity", result.Order.Quantity, "price", result.Order.Price)
	return nil
}
