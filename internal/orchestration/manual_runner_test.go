package orchestration

import (
	"context"
	"testing"
	"time"

	"sentra/internal/execution"
	"sentra/internal/marketdata"
	"sentra/internal/risk"
	"sentra/internal/strategy"
)

func TestManualRunnerRunsOnePaperCycle(t *testing.T) {
	fixture := newFixture()
	fixture.candleReader.candles = manualRunnerCandles()
	settings := strategy.DefaultSettings()
	settings.FastPeriod = 2
	settings.SlowPeriod = 4
	settings.LookbackLimit = 6
	runner := NewManualRunner(ManualRunnerDependencies{
		CandleReader:     fixture.candleReader,
		StrategySettings: fixtureStrategySettings{settings: settings},
		SignalStore:      fixture.signals,
		RiskSettings:     fixtureRiskSettings{settings: risk.DefaultSettings()},
		DecisionStore:    fixture.decisions,
		PriceReader:      fixture.prices,
		AccountStore:     fixture.accounts,
		ExecutionStore:   fixture.executions,
		ExecutionStats:   fixture.executions,
		Publisher:        fixture.publisher,
		ExecutionEngine: execution.NewPaperEngine(execution.Config{
			Enabled:          true,
			Symbol:           "BTCUSDT",
			BaseAsset:        "BTC",
			QuoteAsset:       "USDT",
			QuoteOrderAmount: 100,
			FeeRate:          0.001,
		}),
	}, Config{
		SignalStream:     "stream:strategy-signals",
		RiskStream:       "stream:risk-decisions",
		ExecutionStream:  "stream:execution-results",
		QuoteOrderAmount: 100,
	})

	result, err := runner.RunOnce(context.Background(), ManualRunRequest{Symbol: "BTCUSDT", Interval: "1m"})
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}
	if result.Status != "executed" {
		t.Fatalf("expected executed status, got %q", result.Status)
	}
	if result.Signal.ID == "" || result.Decision.ID == "" || result.Execution == nil {
		t.Fatalf("expected signal, decision, and execution, got %+v", result)
	}
	if len(fixture.executions.saved) != 1 {
		t.Fatalf("expected one saved paper execution, got %d", len(fixture.executions.saved))
	}
	if !fixture.candleReader.query.Desc {
		t.Fatalf("expected manual runner to request newest candles first, got %+v", fixture.candleReader.query)
	}
	wantGeneratedAt := manualRunnerCandles()[len(manualRunnerCandles())-1].CloseTime
	if !result.Signal.GeneratedAt.Equal(wantGeneratedAt) {
		t.Fatalf("expected signal from newest candle %s, got %s", wantGeneratedAt, result.Signal.GeneratedAt)
	}
}

func TestManualRunnerBlocksExecutionWhenKillSwitchActive(t *testing.T) {
	fixture := newFixture()
	fixture.candleReader.candles = manualRunnerCandles()
	settings := strategy.DefaultSettings()
	settings.FastPeriod = 2
	settings.SlowPeriod = 4
	settings.LookbackLimit = 6
	runner := NewManualRunner(ManualRunnerDependencies{
		CandleReader:     fixture.candleReader,
		StrategySettings: fixtureStrategySettings{settings: settings},
		SignalStore:      fixture.signals,
		RiskSettings:     fixtureRiskSettings{settings: risk.DefaultSettings()},
		DecisionStore:    fixture.decisions,
		PriceReader:      fixture.prices,
		AccountStore:     fixture.accounts,
		ExecutionStore:   fixture.executions,
		ExecutionStats:   fixture.executions,
		Publisher:        fixture.publisher,
		Safety:           fixtureSafety{active: true},
		ExecutionEngine: execution.NewPaperEngine(execution.Config{
			Enabled:          true,
			Symbol:           "BTCUSDT",
			BaseAsset:        "BTC",
			QuoteAsset:       "USDT",
			QuoteOrderAmount: 100,
			FeeRate:          0.001,
		}),
	}, Config{QuoteOrderAmount: 100})

	result, err := runner.RunOnce(context.Background(), ManualRunRequest{Symbol: "BTCUSDT", Interval: "1m"})
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}
	if result.Status != "safety_blocked" {
		t.Fatalf("expected safety_blocked, got %q", result.Status)
	}
	if len(fixture.executions.saved) != 0 {
		t.Fatalf("expected no paper execution, got %d", len(fixture.executions.saved))
	}
}

func TestManualRunnerRejectsSellBeforeExecutionWhenBaseBalanceIsInsufficient(t *testing.T) {
	fixture := newFixture()
	fixture.candleReader.candles = manualRunnerRSIOverboughtCandles()
	fixture.accounts.account = execution.Account{BaseBalance: 0.001, QuoteBalance: 1000}
	settings := strategy.DefaultSettings()
	settings.StrategyName = strategy.StrategyRSIMeanReversion
	settings.RSIPeriod = 3
	settings.RSIOversold = 30
	settings.RSIOverbought = 70
	settings.LookbackLimit = 5
	runner := NewManualRunner(ManualRunnerDependencies{
		CandleReader:     fixture.candleReader,
		StrategySettings: fixtureStrategySettings{settings: settings},
		SignalStore:      fixture.signals,
		RiskSettings:     fixtureRiskSettings{settings: risk.DefaultSettings()},
		DecisionStore:    fixture.decisions,
		PriceReader:      fixture.prices,
		AccountStore:     fixture.accounts,
		ExecutionStore:   fixture.executions,
		ExecutionStats:   fixture.executions,
		Publisher:        fixture.publisher,
		ExecutionEngine: execution.NewPaperEngine(execution.Config{
			Enabled:          true,
			Symbol:           "BTCUSDT",
			BaseAsset:        "BTC",
			QuoteAsset:       "USDT",
			QuoteOrderAmount: 100,
			FeeRate:          0.001,
		}),
	}, Config{QuoteOrderAmount: 100})

	result, err := runner.RunOnce(context.Background(), ManualRunRequest{Symbol: "BTCUSDT", Interval: "1m"})
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}
	if result.Status != "risk_rejected" {
		t.Fatalf("expected risk_rejected, got %+v", result)
	}
	if result.Decision.Reason != "base balance 0.001000 is below required 0.002000" {
		t.Fatalf("expected insufficient base balance reason, got %+v", result.Decision)
	}
	if len(fixture.executions.saved) != 0 {
		t.Fatalf("expected no execution after risk rejection, got %d", len(fixture.executions.saved))
	}
}

func TestManualRunnerRejectsBuyBeforeExecutionWhenQuoteBalanceIsInsufficient(t *testing.T) {
	fixture := newFixture()
	fixture.candleReader.candles = manualRunnerCandles()
	fixture.accounts.account = execution.Account{BaseBalance: 0, QuoteBalance: 50}
	settings := strategy.DefaultSettings()
	settings.FastPeriod = 2
	settings.SlowPeriod = 4
	settings.LookbackLimit = 6
	runner := NewManualRunner(ManualRunnerDependencies{
		CandleReader:     fixture.candleReader,
		StrategySettings: fixtureStrategySettings{settings: settings},
		SignalStore:      fixture.signals,
		RiskSettings:     fixtureRiskSettings{settings: risk.DefaultSettings()},
		DecisionStore:    fixture.decisions,
		PriceReader:      fixture.prices,
		AccountStore:     fixture.accounts,
		ExecutionStore:   fixture.executions,
		ExecutionStats:   fixture.executions,
		Publisher:        fixture.publisher,
		ExecutionEngine: execution.NewPaperEngine(execution.Config{
			Enabled:          true,
			Symbol:           "BTCUSDT",
			BaseAsset:        "BTC",
			QuoteAsset:       "USDT",
			QuoteOrderAmount: 100,
			FeeRate:          0.001,
		}),
	}, Config{QuoteOrderAmount: 100})

	result, err := runner.RunOnce(context.Background(), ManualRunRequest{Symbol: "BTCUSDT", Interval: "1m"})
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}
	if result.Status != "risk_rejected" {
		t.Fatalf("expected risk_rejected, got %+v", result)
	}
	if result.Decision.Reason != "quote balance 50.000000 is below required 100.000000" {
		t.Fatalf("expected insufficient quote balance reason, got %+v", result.Decision)
	}
	if len(fixture.executions.saved) != 0 {
		t.Fatalf("expected no execution after risk rejection, got %d", len(fixture.executions.saved))
	}
}

type fixtureStrategySettings struct {
	settings strategy.Settings
}

func (f fixtureStrategySettings) Get(ctx context.Context) (strategy.Settings, error) {
	return f.settings, nil
}

type fixtureRiskSettings struct {
	settings risk.Settings
}

func (f fixtureRiskSettings) Get(ctx context.Context) (risk.Settings, error) {
	return f.settings, nil
}

type fixtureSafety struct {
	active bool
}

func (f fixtureSafety) IsKillSwitchActive(ctx context.Context) (bool, error) {
	return f.active, nil
}

func (f *fakeExecutionStore) DailyStats(ctx context.Context, symbol string, day time.Time) (execution.DailyStats, error) {
	return execution.DailyStats{}, nil
}

func manualRunnerCandles() []marketdata.Candle {
	base := time.Unix(100, 0).UTC()
	closes := []string{"100", "100", "100", "100", "80", "130"}
	return manualRunnerCandlesWithCloses(base, closes)
}

func manualRunnerRSIOverboughtCandles() []marketdata.Candle {
	base := time.Unix(200, 0).UTC()
	return manualRunnerCandlesWithCloses(base, []string{"80", "85", "90", "95", "100"})
}

func manualRunnerCandlesWithCloses(base time.Time, closes []string) []marketdata.Candle {
	candles := make([]marketdata.Candle, 0, len(closes))
	for index, closeValue := range closes {
		openTime := base.Add(time.Duration(index) * time.Minute)
		candles = append(candles, marketdata.Candle{
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Close:     closeValue,
			IsClosed:  true,
		})
	}
	return candles
}
