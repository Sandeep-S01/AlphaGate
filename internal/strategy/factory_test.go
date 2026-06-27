package strategy

import "testing"

func TestNewEvaluatorFromSettingsSelectsRSI(t *testing.T) {
	settings := DefaultSettings()
	settings.StrategyName = StrategyRSIMeanReversion
	settings.RSIPeriod = 14
	settings.RSIOversold = 30
	settings.RSIOverbought = 70

	evaluator, err := NewEvaluatorFromSettings(settings)
	if err != nil {
		t.Fatalf("NewEvaluatorFromSettings returned error: %v", err)
	}
	if _, ok := evaluator.(*RSIMeanReversion); !ok {
		t.Fatalf("expected RSI evaluator, got %T", evaluator)
	}
}

func TestNewEvaluatorFromSettingsSelectsBTCTrendPullback(t *testing.T) {
	settings := DefaultSettings()
	settings.StrategyName = StrategyBTCTrendPullback
	settings.FastPeriod = 21
	settings.SlowPeriod = 200
	settings.LookbackLimit = 201

	evaluator, err := NewEvaluatorFromSettings(settings)
	if err != nil {
		t.Fatalf("NewEvaluatorFromSettings returned error: %v", err)
	}
	trendPullback, ok := evaluator.(*BTCTrendPullback)
	if !ok {
		t.Fatalf("expected BTC trend pullback evaluator, got %T", evaluator)
	}
	if !trendPullback.cfg.VolatilityFilter {
		t.Fatalf("expected BTC trend pullback volatility filter to be enabled by default")
	}
	if trendPullback.cfg.ATRPeriod != DefaultTrendPullbackATRPeriod {
		t.Fatalf("expected default ATR period %d, got %+v", DefaultTrendPullbackATRPeriod, trendPullback.cfg)
	}
	if trendPullback.cfg.MinATRPercent != DefaultTrendPullbackMinATRPercent {
		t.Fatalf("expected default minimum ATR percent %f, got %+v", DefaultTrendPullbackMinATRPercent, trendPullback.cfg)
	}
	if trendPullback.cfg.MaxATRPercent != DefaultTrendPullbackMaxATRPercent {
		t.Fatalf("expected default maximum ATR percent %f, got %+v", DefaultTrendPullbackMaxATRPercent, trendPullback.cfg)
	}
}

func TestNewEvaluatorFromSettingsSelectsExecutableTemplatePine(t *testing.T) {
	settings := DefaultSettings()
	settings.StrategyName = StrategyMultiFactorMomentum
	settings.FastPeriod = 50
	settings.SlowPeriod = 200
	settings.LookbackLimit = 450

	evaluator, err := NewEvaluatorFromSettings(settings)
	if err != nil {
		t.Fatalf("NewEvaluatorFromSettings returned error: %v", err)
	}
	if _, ok := evaluator.(*PineEvaluator); !ok {
		t.Fatalf("expected Pine evaluator for executable template, got %T", evaluator)
	}
}

func TestNewEvaluatorFromSettingsRejectsDataBlockedTemplate(t *testing.T) {
	settings := DefaultSettings()
	settings.StrategyName = StrategyFundingRateArbitrage

	if _, err := NewEvaluatorFromSettings(settings); err == nil {
		t.Fatalf("expected data-blocked template to be rejected")
	}
}
