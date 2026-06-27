package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sentra/internal/activation"
	"sentra/internal/api"
	"sentra/internal/audit"
	"sentra/internal/backtest"
	"sentra/internal/config"
	"sentra/internal/exchange/binance"
	"sentra/internal/execution"
	"sentra/internal/marketdata"
	"sentra/internal/observability"
	"sentra/internal/orchestration"
	"sentra/internal/pine"
	"sentra/internal/platform/logger"
	"sentra/internal/platform/postgres"
	"sentra/internal/platform/redisclient"
	"sentra/internal/reconciliation"
	"sentra/internal/risk"
	"sentra/internal/safety"
	"sentra/internal/strategy"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped with error", "error", err)
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

	candleRepository := marketdata.NewCandleRepository(db)
	backfillRepository := marketdata.NewBackfillRepository(db)
	backfillService := marketdata.NewBackfillService(
		binance.NewFromConfig(cfg.Binance),
		candleRepository,
		marketdata.WithBackfillJobs(backfillRepository),
	)
	signalRepository := strategy.NewSignalRepository(db)
	riskDecisionRepository := risk.NewDecisionRepository(db)
	executionRepository := execution.NewRepository(db)
	accountRepository := execution.NewAccountRepository(db)
	safetyRepository := safety.NewRepository(db)
	reconciliationRepository := reconciliation.NewRepository(db)
	paperSnapshotReader := reconciliation.NewPaperSnapshotReader(accountRepository, executionRepository, cfg.Execution.Symbol)
	reconciliationRunner := reconciliation.NewService(reconciliation.Dependencies{
		Internal: paperSnapshotReader,
		External: paperSnapshotReader,
		Store:    reconciliationRepository,
	})
	pineRepository := pine.NewRepository(db)
	redisPublisher := orchestration.NewRedisPublisher(redisClient)
	paperCycleRunner := orchestration.NewManualRunner(orchestration.ManualRunnerDependencies{
		CandleReader:     candleRepository,
		StrategySettings: strategy.NewSettingsRepository(db),
		SignalStore:      signalRepository,
		RiskSettings:     risk.NewSettingsRepository(db),
		DecisionStore:    riskDecisionRepository,
		PriceReader:      execution.NewMarketPriceReader(candleRepository),
		AccountStore:     accountRepository,
		ExecutionStore:   executionRepository,
		ExecutionStats:   executionRepository,
		Publisher:        redisPublisher,
		Safety:           safetyRepository,
		ExecutionEngine: execution.NewPaperEngine(execution.Config{
			Enabled:          cfg.Execution.Enabled,
			Symbol:           cfg.Execution.Symbol,
			BaseAsset:        cfg.Execution.BaseAsset,
			QuoteAsset:       cfg.Execution.QuoteAsset,
			QuoteOrderAmount: cfg.Execution.QuoteOrderAmount,
			FeeRate:          cfg.Execution.FeeRate,
		}),
	}, orchestration.Config{
		SignalStream:     cfg.Strategy.SignalStream,
		RiskStream:       cfg.Risk.DecisionStream,
		ExecutionStream:  cfg.Execution.Stream,
		QuoteOrderAmount: cfg.Execution.QuoteOrderAmount,
	})

	server := &http.Server{
		Addr: cfg.HTTP.Addr,
		Handler: api.NewRouter(api.Dependencies{
			Postgres:             db,
			Redis:                redisclient.NewPinger(redisClient),
			Candles:              candleRepository,
			Backfills:            backfillRepository,
			BackfillRunner:       backfillService,
			Aggregator:           marketdata.NewAggregationService(candleRepository),
			Signals:              signalRepository,
			SignalStore:          signalRepository,
			StrategySettings:     strategy.NewSettingsRepository(db),
			Backtests:            backtest.NewRepository(db),
			StrategyComparisons:  backtest.NewComparisonRepository(db),
			StrategyActivations:  activation.NewRepository(db),
			ReconciliationRuns:   reconciliationRepository,
			ReconciliationRunner: reconciliationRunner,
			RiskSettings:         risk.NewSettingsRepository(db),
			PineRepository:       pineRepository,
			RiskDecisions:        riskDecisionRepository,
			PaperAccount:         accountRepository,
			Orders:               executionRepository,
			Trades:               executionRepository,
			ExecutionStatus: execution.Status{
				Mode:               "paper",
				PaperEnabled:       cfg.Execution.Enabled,
				ExchangeAdapter:    "binance_disabled",
				LiveTradingEnabled: false,
				RetryAttempts:      3,
				Timeout:            "5s",
				LastError:          "binance live trading is disabled",
			},
			PaperCycleRunner: paperCycleRunner,
			Safety:           safetyRepository,
			Audit:            audit.NewRepository(db),
			Reports:          reportStore{execution: executionRepository, risk: riskDecisionRepository},
			Metrics:          metrics,
			PipelineRuns:     orchestration.NewIdempotencyRepository(db),
			Streams: observability.NewRedisStreamStatsReader(redisClient, []string{
				cfg.MarketData.RedisStream,
				cfg.Strategy.SignalStream,
				cfg.Risk.DecisionStream,
				cfg.Execution.Stream,
			}),
			Dashboard: http.Dir("web/dashboard"),
			Auth: api.AuthConfig{
				Enabled:     cfg.Auth.Enabled,
				AdminAPIKey: cfg.Auth.AdminAPIKey,
			},
			Security: api.SecurityConfig{
				MaxRequestBodyBytes:        cfg.Security.MaxRequestBodyBytes,
				RateLimitRequestsPerMinute: cfg.Security.RateLimitRequestsPerMinute,
			},
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("api listening", "addr", cfg.HTTP.Addr, "env", cfg.App.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

type reportStore struct {
	execution interface {
		DailyPnL(ctx context.Context, query execution.Query) ([]execution.DailyPnL, error)
		TradeCounts(ctx context.Context, query execution.Query) ([]execution.DailyPnL, error)
	}
	risk interface {
		RejectedReasons(ctx context.Context, query risk.DecisionQuery) ([]risk.RejectionSummary, error)
	}
}

func (s reportStore) DailyPnL(ctx context.Context, query execution.Query) ([]execution.DailyPnL, error) {
	return s.execution.DailyPnL(ctx, query)
}

func (s reportStore) TradeCounts(ctx context.Context, query execution.Query) ([]execution.DailyPnL, error) {
	return s.execution.TradeCounts(ctx, query)
}

func (s reportStore) RejectedReasons(ctx context.Context, query risk.DecisionQuery) ([]risk.RejectionSummary, error) {
	return s.risk.RejectedReasons(ctx, query)
}
