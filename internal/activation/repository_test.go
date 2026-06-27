package activation

import (
	"strings"
	"testing"

	"sentra/internal/strategy"
)

func TestBuildInsertActivationSQLUsesRecordFields(t *testing.T) {
	record := Record{
		ComparisonID:       "comparison-1",
		ComparisonResultID: "result-1",
		StrategyName:       strategy.StrategySMACrossover,
		Actor:              "operator",
		ActivatedSettings: strategy.Settings{
			StrategyName:  strategy.StrategySMACrossover,
			Version:       "v1",
			Symbol:        "BTCUSDT",
			Interval:      "1m",
			FastPeriod:    9,
			SlowPeriod:    21,
			LookbackLimit: 100,
			RSIPeriod:     14,
			RSIOversold:   30,
			RSIOverbought: 70,
		},
		ComparisonReturn:      1.2,
		ComparisonDrawdown:    0.5,
		ComparisonWinRate:     60,
		ComparisonTotalTrades: 4,
	}

	query, args, err := BuildInsertActivationSQL(record)
	if err != nil {
		t.Fatalf("BuildInsertActivationSQL returned error: %v", err)
	}

	if !strings.Contains(query, "INSERT INTO strategy_activations") {
		t.Fatalf("expected strategy_activations insert, got %s", query)
	}
	if len(args) != 9 {
		t.Fatalf("expected 9 args, got %d", len(args))
	}
	if args[0] != "comparison-1" || args[2] != strategy.StrategySMACrossover || args[3] != "operator" {
		t.Fatalf("unexpected args: %+v", args)
	}
}
