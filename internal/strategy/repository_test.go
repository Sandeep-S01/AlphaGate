package strategy

import (
	"strings"
	"testing"
	"time"
)

func TestBuildInsertSignalSQLUsesSignalFields(t *testing.T) {
	signal := Signal{
		StrategyName: "sma-crossover",
		Version:      "v1",
		Symbol:       "BTCUSDT",
		Interval:     "1m",
		Side:         SideBuy,
		Strength:     1.25,
		Reason:       "fast SMA crossed above slow SMA",
		GeneratedAt:  time.Unix(10, 0).UTC(),
	}

	query, args := BuildInsertSignalSQL(signal)

	if !strings.Contains(query, "INSERT INTO strategy_signals") {
		t.Fatalf("expected strategy_signals insert, got %s", query)
	}
	if !strings.Contains(query, "RETURNING id") {
		t.Fatalf("expected RETURNING id, got %s", query)
	}
	if len(args) != 8 {
		t.Fatalf("expected 8 args, got %d", len(args))
	}
	if args[0] != "sma-crossover" || args[2] != "BTCUSDT" || args[4] != string(SideBuy) {
		t.Fatalf("unexpected args: %+v", args)
	}
}
