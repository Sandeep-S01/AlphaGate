package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sentra/internal/config"
	"sentra/internal/exchange/binance"
	"sentra/internal/marketdata"
	"sentra/internal/platform/logger"
	"sentra/internal/platform/postgres"
	"sentra/internal/observability"
)

func main() {
	if err := run(); err != nil {
		slog.Error("backfill failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	var symbol string
	var interval string
	var fromRaw string
	var toRaw string
	var limit int
	var resumeID string

	flag.StringVar(&symbol, "symbol", "BTCUSDT", "market symbol")
	flag.StringVar(&interval, "interval", "1m", "kline interval")
	flag.StringVar(&fromRaw, "from", "", "start time in RFC3339")
	flag.StringVar(&toRaw, "to", "", "end time in RFC3339")
	flag.IntVar(&limit, "limit", 1000, "Binance kline request limit")
	flag.StringVar(&resumeID, "resume", "", "existing backfill job id to resume")
	flag.Parse()

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

	backfillRepository := marketdata.NewBackfillRepository(db)
	service := marketdata.NewBackfillService(
		binance.NewFromConfig(cfg.Binance),
		marketdata.NewCandleRepository(db),
		marketdata.WithBackfillJobs(backfillRepository),
	)
	if resumeID != "" {
		result, err := service.Resume(ctx, resumeID)
		if err != nil {
			return err
		}
		log.Info("backfill resumed", "job_id", result.Job.ID, "symbol", result.Job.Symbol, "interval", result.Job.BaseInterval, "count", result.CandlesInserted, "status", result.Job.Status)
		return nil
	}

	from, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		return err
	}
	to, err := time.Parse(time.RFC3339, toRaw)
	if err != nil {
		return err
	}
	result, err := service.Start(ctx, marketdata.BackfillRequest{
		Symbol:   symbol,
		Interval: interval,
		From:     from,
		To:       to,
		Limit:    limit,
	})
	if err != nil {
		return err
	}

	log.Info("backfill completed", "job_id", result.Job.ID, "symbol", symbol, "interval", interval, "count", result.CandlesInserted)
	return nil
}
