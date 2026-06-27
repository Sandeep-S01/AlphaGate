package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"sentra/internal/config"
	"sentra/internal/exchange/binance"
	"sentra/internal/execution"
	"sentra/internal/marketdata"
	"sentra/internal/observability"
	"sentra/internal/orchestration"
	"sentra/internal/platform/logger"
	"sentra/internal/platform/postgres"
	"sentra/internal/platform/redisclient"
	"sentra/internal/platform/streams"
	"sentra/internal/risk"
	"sentra/internal/safety"
	"sentra/internal/strategy"
)

func main() {
	if err := run(); err != nil {
		slog.Error("worker stopped with error", "error", err)
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
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			log.Warn("redis close failed", "error", closeErr)
		}
	}()

	log.Info("worker started", "env", cfg.App.Env)
	if cfg.MarketData.Enabled {
		binanceClient := binance.NewFromConfig(cfg.Binance)
		publisher := marketdata.NewRedisPublisher(redisClient)
		collector := marketdata.NewCollector(binanceClient, publisher, marketdata.CollectorConfig{
			Symbol:         cfg.MarketData.Symbol,
			Interval:       cfg.MarketData.Interval,
			RedisStream:    cfg.MarketData.RedisStream,
			MaxReconnects:  cfg.MarketData.MaxReconnects,
			ReconnectDelay: cfg.MarketData.ReconnectDelay,
		})

		go func() {
			log.Info("market data collector started", "exchange", "binance", "symbol", cfg.MarketData.Symbol, "interval", cfg.MarketData.Interval)
			if err := collector.Run(ctx); err != nil && ctx.Err() == nil {
				log.Error("market data collector stopped with error", "error", err)
				stop()
			}
		}()
	}
	if cfg.MarketData.PersistenceEnabled {
		repository := marketdata.NewCandleRepository(db)
		persister := marketdata.NewPersister(repository)
		consumer := streams.NewConsumer(redisClient, streams.ConsumerConfig{
			Stream:   cfg.MarketData.RedisStream,
			Group:    cfg.MarketData.ConsumerGroup,
			Consumer: cfg.MarketData.ConsumerName,
		})

		go func() {
			log.Info("market data persistence consumer started", "stream", cfg.MarketData.RedisStream, "group", cfg.MarketData.ConsumerGroup)
			if err := consumer.Run(ctx, persister.Handle); err != nil && ctx.Err() == nil {
				log.Error("market data persistence consumer stopped with error", "error", err)
				stop()
			}
		}()
	}
	if cfg.Orchestration.Enabled {
		candleRepository := marketdata.NewCandleRepository(db)
		executionRepository := execution.NewRepository(db)
		strategySettings := strategy.Settings{
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
		if settings, settingsErr := strategy.NewSettingsRepository(db).Get(ctx); settingsErr == nil {
			strategySettings = settings.Normalized()
		} else {
			log.Warn("strategy settings unavailable, using environment strategy config", "error", settingsErr)
		}
		strategyEvaluator, err := strategy.NewEvaluatorFromSettings(strategySettings)
		if err != nil {
			return err
		}
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
		orchestrator := orchestration.NewOrchestrator(orchestration.Dependencies{
			Idempotency:       orchestration.NewIdempotencyRepository(db),
			CandleStore:       candleRepository,
			CandleReader:      candleRepository,
			StrategyEvaluator: strategyEvaluator,
			SignalStore:       strategy.NewSignalRepository(db),
			RiskEvaluator:     risk.NewEvaluator(riskConfig),
			DecisionStore:     risk.NewDecisionRepository(db),
			PriceReader:       execution.NewMarketPriceReader(candleRepository),
			AccountStore:      execution.NewAccountRepository(db),
			ExecutionStore:    executionRepository,
			ExecutionStats:    executionRepository,
			Publisher:         orchestration.NewRedisPublisher(redisClient),
			Safety:            safety.NewRepository(db),
			Metrics:           metrics,
			ExecutionEngine: execution.NewPaperEngine(execution.Config{
				Enabled:          cfg.Execution.Enabled,
				Symbol:           cfg.Execution.Symbol,
				BaseAsset:        cfg.Execution.BaseAsset,
				QuoteAsset:       cfg.Execution.QuoteAsset,
				QuoteOrderAmount: cfg.Execution.QuoteOrderAmount,
				FeeRate:          cfg.Execution.FeeRate,
			}),
		}, orchestration.Config{
			Symbol:           cfg.Execution.Symbol,
			Interval:         cfg.Execution.Interval,
			LookbackLimit:    strategySettings.LookbackLimit,
			SignalStream:     cfg.Strategy.SignalStream,
			RiskStream:       cfg.Risk.DecisionStream,
			ExecutionStream:  cfg.Execution.Stream,
			QuoteOrderAmount: cfg.Execution.QuoteOrderAmount,
		})
		consumer := streams.NewConsumer(redisClient, streams.ConsumerConfig{
			Stream:   cfg.MarketData.RedisStream,
			Group:    cfg.Orchestration.ConsumerGroup,
			Consumer: cfg.Orchestration.ConsumerName,
		})

		go func() {
			log.Info("paper orchestration consumer started", "stream", cfg.MarketData.RedisStream, "group", cfg.Orchestration.ConsumerGroup)
			if err := consumer.Run(ctx, orchestrator.Handle); err != nil && ctx.Err() == nil {
				log.Error("paper orchestration consumer stopped with error", "error", err)
				stop()
			}
		}()
	}

	<-ctx.Done()
	log.Info("worker stopped")
	return nil
}
