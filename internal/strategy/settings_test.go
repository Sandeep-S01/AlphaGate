package strategy

import (
	"strings"
	"testing"
)

func TestSettingsValidateRejectsInvalidPeriods(t *testing.T) {
	settings := Settings{
		StrategyName:  "sma-crossover",
		Version:       "v1",
		Symbol:        "BTCUSDT",
		Interval:      "1m",
		FastPeriod:    21,
		SlowPeriod:    9,
		LookbackLimit: 100,
	}

	if err := settings.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSettingsValidateRequiresLookbackToCoverSlowPeriod(t *testing.T) {
	settings := DefaultSettings()
	settings.SlowPeriod = 50
	settings.LookbackLimit = 20

	err := settings.Validate()
	if err == nil || !strings.Contains(err.Error(), "lookback_limit") {
		t.Fatalf("expected lookback validation error, got %v", err)
	}
}

func TestBuildUpsertSettingsSQLUsesSettingsFields(t *testing.T) {
	settings := DefaultSettings()
	settings.StrategyName = StrategyRSIMeanReversion
	settings.Symbol = "ETHUSDT"
	settings.RSIPeriod = 14
	settings.RSIOversold = 30
	settings.RSIOverbought = 70

	query, args := BuildUpsertSettingsSQL(settings)

	if !strings.Contains(query, "INSERT INTO strategy_settings") {
		t.Fatalf("expected strategy_settings insert, got %s", query)
	}
	if !strings.Contains(query, "ON CONFLICT") {
		t.Fatalf("expected upsert conflict handling, got %s", query)
	}
	if len(args) != 10 {
		t.Fatalf("expected 10 args, got %d", len(args))
	}
	if args[0] != StrategyRSIMeanReversion || args[2] != "ETHUSDT" || args[7] != 14 {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestSettingsValidateAllowsRSIStrategy(t *testing.T) {
	settings := DefaultSettings()
	settings.StrategyName = StrategyRSIMeanReversion
	settings.RSIPeriod = 14
	settings.RSIOversold = 30
	settings.RSIOverbought = 70
	settings.LookbackLimit = 20

	if err := settings.Validate(); err != nil {
		t.Fatalf("expected RSI settings valid, got %v", err)
	}
}

func TestSettingsRequiredCandlesForTrendPullbackIncludesPreviousRSI(t *testing.T) {
	settings := DefaultSettings()
	settings.StrategyName = StrategyBTCTrendPullback
	settings.FastPeriod = 2
	settings.SlowPeriod = 3
	settings.RSIPeriod = 14
	settings.LookbackLimit = 16

	if required := settings.RequiredCandles(); required != 16 {
		t.Fatalf("expected trend pullback to require previous and current RSI windows, got %d", required)
	}
}

func TestSettingsRequiredCandlesForTrendPullbackIncludesDefaultATRFilter(t *testing.T) {
	settings := DefaultSettings()
	settings.StrategyName = StrategyBTCTrendPullback
	settings.FastPeriod = 2
	settings.SlowPeriod = 3
	settings.RSIPeriod = 3
	settings.LookbackLimit = 15

	if required := settings.RequiredCandles(); required != 15 {
		t.Fatalf("expected trend pullback to require default ATR filter history, got %d", required)
	}
}
