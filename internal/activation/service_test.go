package activation

import (
	"context"
	"testing"
	"time"

	"sentra/internal/backtest"
	"sentra/internal/strategy"
)

func TestServiceActivatesWinnerFromRecentComparison(t *testing.T) {
	comparison := backtest.Comparison{
		ID:             "comparison-1",
		Symbol:         "BTCUSDT",
		Interval:       "1m",
		WinnerStrategy: strategy.StrategyRSIMeanReversion,
		CreatedAt:      time.Now().UTC(),
		Results: []backtest.ComparisonResult{
			{
				ID:                          "result-1",
				Rank:                        1,
				StrategyName:                strategy.StrategyRSIMeanReversion,
				Version:                     "v1",
				FastPeriod:                  9,
				SlowPeriod:                  21,
				RSIPeriod:                   14,
				RSIOversold:                 30,
				RSIOverbought:               70,
				MaxDrawdown:                 12,
				TotalTrades:                 140,
				ProfitFactor:                1.4,
				Expectancy:                  0.2,
				ExcessReturnPercent:         3.2,
				ValidationStatus:            "candidate",
				ExecutionFillMode:           backtest.ExecutionFillModeNextOpen,
				TrainValidationStatus:       "candidate",
				TestValidationStatus:        "candidate",
				WalkForwardFolds:            4,
				WalkForwardPasses:           4,
				WalkForwardValidationStatus: "candidate",
			},
		},
	}
	settingsStore := &fakeSettingsStore{}
	activationStore := &fakeActivationStore{}
	service := NewService(Dependencies{
		Comparisons: &fakeComparisonReader{comparison: comparison},
		Settings:    settingsStore,
		Activations: activationStore,
		Now:         func() time.Time { return time.Now().UTC() },
	})

	activation, err := service.Activate(context.Background(), Request{
		ComparisonID: "comparison-1",
		Actor:        "operator",
	})
	if err != nil {
		t.Fatalf("Activate returned error: %v", err)
	}

	if settingsStore.saved.StrategyName != strategy.StrategyRSIMeanReversion {
		t.Fatalf("expected RSI settings saved, got %+v", settingsStore.saved)
	}
	if activation.ComparisonID != "comparison-1" || activation.StrategyName != strategy.StrategyRSIMeanReversion {
		t.Fatalf("unexpected activation: %+v", activation)
	}
	if len(activationStore.saved) != 1 {
		t.Fatalf("expected one activation saved, got %d", len(activationStore.saved))
	}
}

func TestServiceCreatesValidatedLifecycleRecordOnActivation(t *testing.T) {
	lifecycleStore := &fakeLifecycleStore{}
	service := NewService(Dependencies{
		Comparisons: &fakeComparisonReader{comparison: activationComparison(backtest.ComparisonResult{
			ID:                          "result-1",
			StrategyName:                strategy.StrategyRSIMeanReversion,
			Version:                     "v1",
			FastPeriod:                  9,
			SlowPeriod:                  21,
			RSIPeriod:                   14,
			RSIOversold:                 30,
			RSIOverbought:               70,
			MaxDrawdown:                 12,
			TotalTrades:                 140,
			ProfitFactor:                1.4,
			Expectancy:                  0.2,
			ExcessReturnPercent:         3.2,
			ValidationStatus:            "candidate",
			ExecutionFillMode:           backtest.ExecutionFillModeNextOpen,
			TrainValidationStatus:       "candidate",
			TestValidationStatus:        "candidate",
			WalkForwardFolds:            4,
			WalkForwardPasses:           4,
			WalkForwardValidationStatus: "candidate",
		})},
		Settings:    &fakeSettingsStore{},
		Activations: &fakeActivationStore{},
		Lifecycles:  lifecycleStore,
		Now:         func() time.Time { return time.Now().UTC() },
	})

	if _, err := service.Activate(context.Background(), Request{ComparisonID: "comparison-1", Actor: "operator"}); err != nil {
		t.Fatalf("Activate returned error: %v", err)
	}

	if len(lifecycleStore.saved) != 1 {
		t.Fatalf("expected one lifecycle record, got %d", len(lifecycleStore.saved))
	}
	if lifecycleStore.saved[0].State != StateValidated {
		t.Fatalf("expected VALIDATED lifecycle state, got %+v", lifecycleStore.saved[0])
	}
	if lifecycleStore.saved[0].StrategyName != strategy.StrategyRSIMeanReversion || lifecycleStore.saved[0].Symbol != "BTCUSDT" {
		t.Fatalf("unexpected lifecycle record: %+v", lifecycleStore.saved[0])
	}
}

func TestServiceRejectsCandidateWithTooFewTrades(t *testing.T) {
	service := NewService(Dependencies{
		Comparisons: &fakeComparisonReader{comparison: activationComparison(backtest.ComparisonResult{
			ID:                          "result-1",
			StrategyName:                strategy.StrategyRSIMeanReversion,
			Version:                     "v1",
			MaxDrawdown:                 12,
			TotalTrades:                 99,
			ProfitFactor:                1.4,
			Expectancy:                  0.2,
			ExcessReturnPercent:         3.2,
			ValidationStatus:            "candidate",
			ExecutionFillMode:           backtest.ExecutionFillModeNextOpen,
			TrainValidationStatus:       "candidate",
			TestValidationStatus:        "candidate",
			WalkForwardFolds:            4,
			WalkForwardValidationStatus: "candidate",
		})},
		Settings: &fakeSettingsStore{},
		Now:      func() time.Time { return time.Now().UTC() },
	})

	if _, err := service.Activate(context.Background(), Request{ComparisonID: "comparison-1"}); err == nil {
		t.Fatal("expected minimum trade count rejection")
	}
}

func TestServiceRejectsCandidateWithWeakProfitFactorOrExpectancy(t *testing.T) {
	for name, result := range map[string]backtest.ComparisonResult{
		"weak_profit_factor": {
			ID:                          "result-1",
			StrategyName:                strategy.StrategyRSIMeanReversion,
			Version:                     "v1",
			MaxDrawdown:                 12,
			TotalTrades:                 140,
			ProfitFactor:                1.29,
			Expectancy:                  0.2,
			ExcessReturnPercent:         3.2,
			ValidationStatus:            "candidate",
			ExecutionFillMode:           backtest.ExecutionFillModeNextOpen,
			TrainValidationStatus:       "candidate",
			TestValidationStatus:        "candidate",
			WalkForwardFolds:            4,
			WalkForwardValidationStatus: "candidate",
		},
		"negative_expectancy": {
			ID:                          "result-1",
			StrategyName:                strategy.StrategyRSIMeanReversion,
			Version:                     "v1",
			MaxDrawdown:                 12,
			TotalTrades:                 140,
			ProfitFactor:                1.4,
			Expectancy:                  -0.01,
			ExcessReturnPercent:         3.2,
			ValidationStatus:            "candidate",
			ExecutionFillMode:           backtest.ExecutionFillModeNextOpen,
			TrainValidationStatus:       "candidate",
			TestValidationStatus:        "candidate",
			WalkForwardFolds:            4,
			WalkForwardValidationStatus: "candidate",
		},
	} {
		service := NewService(Dependencies{
			Comparisons: &fakeComparisonReader{comparison: activationComparison(result)},
			Settings:    &fakeSettingsStore{},
			Now:         func() time.Time { return time.Now().UTC() },
		})

		if _, err := service.Activate(context.Background(), Request{ComparisonID: "comparison-1"}); err == nil {
			t.Fatalf("expected rejection for %s", name)
		}
	}
}

func TestServiceRejectsCandidateWithoutTrainTestEvidence(t *testing.T) {
	service := NewService(Dependencies{
		Comparisons: &fakeComparisonReader{comparison: activationComparison(backtest.ComparisonResult{
			ID:                          "result-1",
			StrategyName:                strategy.StrategyRSIMeanReversion,
			Version:                     "v1",
			FastPeriod:                  9,
			SlowPeriod:                  21,
			RSIPeriod:                   14,
			RSIOversold:                 30,
			RSIOverbought:               70,
			MaxDrawdown:                 12,
			TotalTrades:                 140,
			ProfitFactor:                1.4,
			ExcessReturnPercent:         3.2,
			ValidationStatus:            "candidate",
			ExecutionFillMode:           backtest.ExecutionFillModeNextOpen,
			WalkForwardFolds:            4,
			WalkForwardPasses:           4,
			WalkForwardValidationStatus: "candidate",
		})},
		Settings:    &fakeSettingsStore{},
		Activations: &fakeActivationStore{},
		Now:         func() time.Time { return time.Now().UTC() },
	})

	if _, err := service.Activate(context.Background(), Request{ComparisonID: "comparison-1"}); err == nil {
		t.Fatal("expected missing train/test evidence rejection")
	}
}

func TestServiceRejectsCandidateWithoutWalkForwardEvidence(t *testing.T) {
	service := NewService(Dependencies{
		Comparisons: &fakeComparisonReader{comparison: activationComparison(backtest.ComparisonResult{
			ID:                    "result-1",
			StrategyName:          strategy.StrategyRSIMeanReversion,
			Version:               "v1",
			FastPeriod:            9,
			SlowPeriod:            21,
			RSIPeriod:             14,
			RSIOversold:           30,
			RSIOverbought:         70,
			MaxDrawdown:           12,
			TotalTrades:           140,
			ProfitFactor:          1.4,
			ExcessReturnPercent:   3.2,
			ValidationStatus:      "candidate",
			ExecutionFillMode:     backtest.ExecutionFillModeNextOpen,
			TrainValidationStatus: "candidate",
			TestValidationStatus:  "candidate",
		})},
		Settings:    &fakeSettingsStore{},
		Activations: &fakeActivationStore{},
		Now:         func() time.Time { return time.Now().UTC() },
	})

	if _, err := service.Activate(context.Background(), Request{ComparisonID: "comparison-1"}); err == nil {
		t.Fatal("expected missing walk-forward evidence rejection")
	}
}

func TestServiceRejectsNonCandidateComparisonWinner(t *testing.T) {
	settingsStore := &fakeSettingsStore{}
	activationStore := &fakeActivationStore{}
	service := NewService(Dependencies{
		Comparisons: &fakeComparisonReader{comparison: activationComparison(backtest.ComparisonResult{
			ID:                  "result-1",
			StrategyName:        strategy.StrategyRSIMeanReversion,
			Version:             "v1",
			FastPeriod:          9,
			SlowPeriod:          21,
			RSIPeriod:           14,
			RSIOversold:         30,
			RSIOverbought:       70,
			MaxDrawdown:         12,
			TotalTrades:         140,
			ProfitFactor:        0.8,
			ExcessReturnPercent: 3.2,
			ValidationStatus:    "weak_profit_factor",
			ValidationReason:    "profit factor must be greater than 1.2",
			ExecutionFillMode:   backtest.ExecutionFillModeNextOpen,
		})},
		Settings:    settingsStore,
		Activations: activationStore,
		Now:         func() time.Time { return time.Now().UTC() },
	})

	_, err := service.Activate(context.Background(), Request{ComparisonID: "comparison-1", Actor: "operator"})
	if err == nil {
		t.Fatal("expected activation gate rejection")
	}
	if settingsStore.saved.StrategyName != "" {
		t.Fatalf("expected settings not to be saved, got %+v", settingsStore.saved)
	}
	if len(activationStore.saved) != 0 {
		t.Fatalf("expected no activation history, got %d", len(activationStore.saved))
	}
}

func TestServiceRejectsCandidateThatUnderperformsBenchmark(t *testing.T) {
	service := NewService(Dependencies{
		Comparisons: &fakeComparisonReader{comparison: activationComparison(backtest.ComparisonResult{
			ID:                  "result-1",
			StrategyName:        strategy.StrategyRSIMeanReversion,
			Version:             "v1",
			FastPeriod:          9,
			SlowPeriod:          21,
			RSIPeriod:           14,
			RSIOversold:         30,
			RSIOverbought:       70,
			MaxDrawdown:         12,
			TotalTrades:         140,
			ProfitFactor:        1.4,
			ExcessReturnPercent: -0.1,
			ValidationStatus:    "candidate",
			ExecutionFillMode:   backtest.ExecutionFillModeNextOpen,
		})},
		Settings:    &fakeSettingsStore{},
		Activations: &fakeActivationStore{},
		Now:         func() time.Time { return time.Now().UTC() },
	})

	if _, err := service.Activate(context.Background(), Request{ComparisonID: "comparison-1"}); err == nil {
		t.Fatal("expected benchmark underperformance rejection")
	}
}

func TestServiceRejectsCandidateWithFailedTestSplit(t *testing.T) {
	service := NewService(Dependencies{
		Comparisons: &fakeComparisonReader{comparison: activationComparison(backtest.ComparisonResult{
			ID:                    "result-1",
			StrategyName:          strategy.StrategyRSIMeanReversion,
			Version:               "v1",
			FastPeriod:            9,
			SlowPeriod:            21,
			RSIPeriod:             14,
			RSIOversold:           30,
			RSIOverbought:         70,
			MaxDrawdown:           12,
			TotalTrades:           140,
			ProfitFactor:          1.4,
			ExcessReturnPercent:   3.2,
			ValidationStatus:      "candidate",
			ExecutionFillMode:     backtest.ExecutionFillModeNextOpen,
			TrainValidationStatus: "candidate",
			TestValidationStatus:  "underperforms_benchmark",
			TestValidationReason:  "excess return must be positive",
		})},
		Settings:    &fakeSettingsStore{},
		Activations: &fakeActivationStore{},
		Now:         func() time.Time { return time.Now().UTC() },
	})

	if _, err := service.Activate(context.Background(), Request{ComparisonID: "comparison-1"}); err == nil {
		t.Fatal("expected failed test split rejection")
	}
}

func TestServiceRejectsCandidateWithFailedWalkForward(t *testing.T) {
	service := NewService(Dependencies{
		Comparisons: &fakeComparisonReader{comparison: activationComparison(backtest.ComparisonResult{
			ID:                          "result-1",
			StrategyName:                strategy.StrategyRSIMeanReversion,
			Version:                     "v1",
			FastPeriod:                  9,
			SlowPeriod:                  21,
			RSIPeriod:                   14,
			RSIOversold:                 30,
			RSIOverbought:               70,
			MaxDrawdown:                 12,
			TotalTrades:                 140,
			ProfitFactor:                1.4,
			ExcessReturnPercent:         3.2,
			ValidationStatus:            "candidate",
			ExecutionFillMode:           backtest.ExecutionFillModeNextOpen,
			WalkForwardFolds:            4,
			WalkForwardPasses:           2,
			WalkForwardAverageExcess:    1.5,
			WalkForwardValidationStatus: "unstable_walk_forward",
			WalkForwardValidationReason: "not all walk-forward folds passed validation",
		})},
		Settings:    &fakeSettingsStore{},
		Activations: &fakeActivationStore{},
		Now:         func() time.Time { return time.Now().UTC() },
	})

	if _, err := service.Activate(context.Background(), Request{ComparisonID: "comparison-1"}); err == nil {
		t.Fatal("expected failed walk-forward rejection")
	}
}

func TestServiceRejectsSameCloseComparisonEvidence(t *testing.T) {
	service := NewService(Dependencies{
		Comparisons: &fakeComparisonReader{comparison: activationComparison(backtest.ComparisonResult{
			ID:                  "result-1",
			StrategyName:        strategy.StrategyRSIMeanReversion,
			Version:             "v1",
			FastPeriod:          9,
			SlowPeriod:          21,
			RSIPeriod:           14,
			RSIOversold:         30,
			RSIOverbought:       70,
			MaxDrawdown:         12,
			TotalTrades:         140,
			ProfitFactor:        1.4,
			ExcessReturnPercent: 3.2,
			ValidationStatus:    "candidate",
			ExecutionFillMode:   backtest.ExecutionFillModeSameClose,
		})},
		Settings:    &fakeSettingsStore{},
		Activations: &fakeActivationStore{},
		Now:         func() time.Time { return time.Now().UTC() },
	})

	if _, err := service.Activate(context.Background(), Request{ComparisonID: "comparison-1"}); err == nil {
		t.Fatal("expected same-close evidence rejection")
	}
}

func TestServiceRejectsStaleComparisonEvidence(t *testing.T) {
	service := NewService(Dependencies{
		Comparisons: &fakeComparisonReader{comparison: backtest.Comparison{
			ID:             "comparison-1",
			WinnerStrategy: strategy.StrategySMACrossover,
			CreatedAt:      time.Now().UTC().Add(-8 * 24 * time.Hour),
			Results: []backtest.ComparisonResult{
				{ID: "result-1", StrategyName: strategy.StrategySMACrossover, Version: "v1", FastPeriod: 9, SlowPeriod: 21},
			},
		}},
		Settings:    &fakeSettingsStore{},
		Activations: &fakeActivationStore{},
		Now:         func() time.Time { return time.Now().UTC() },
	})

	if _, err := service.Activate(context.Background(), Request{ComparisonID: "comparison-1"}); err == nil {
		t.Fatal("expected stale comparison rejection")
	}
}

func activationComparison(result backtest.ComparisonResult) backtest.Comparison {
	return backtest.Comparison{
		ID:             "comparison-1",
		Symbol:         "BTCUSDT",
		Interval:       "1m",
		WinnerStrategy: result.StrategyName,
		CreatedAt:      time.Now().UTC(),
		Results:        []backtest.ComparisonResult{result},
	}
}

type fakeComparisonReader struct {
	comparison backtest.Comparison
}

func (f *fakeComparisonReader) Get(ctx context.Context, id string) (backtest.Comparison, error) {
	return f.comparison, nil
}

type fakeSettingsStore struct {
	saved strategy.Settings
}

func (f *fakeSettingsStore) Save(ctx context.Context, settings strategy.Settings) (strategy.Settings, error) {
	f.saved = settings
	return settings, nil
}

type fakeActivationStore struct {
	saved []Record
}

func (f *fakeActivationStore) Save(ctx context.Context, record Record) (Record, error) {
	record.ID = "activation-1"
	f.saved = append(f.saved, record)
	return record, nil
}

type fakeLifecycleStore struct {
	saved []LifecycleRecord
}

func (f *fakeLifecycleStore) SaveLifecycle(ctx context.Context, record LifecycleRecord) (LifecycleRecord, error) {
	record.ID = "lifecycle-1"
	f.saved = append(f.saved, record)
	return record, nil
}
