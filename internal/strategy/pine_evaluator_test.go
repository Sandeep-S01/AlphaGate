package strategy

import (
	"encoding/json"
	"testing"
	"time"

	"sentra/internal/marketdata"
	"sentra/internal/pine"
)

func TestPineEvaluatorBasic(t *testing.T) {
	// A simple EMA crossover strategy IR
	irJSON := `{
		"indicators": {
			"emaFast": { "type": "ema", "source": "close", "params": [2] },
			"emaSlow": { "type": "ema", "source": "close", "params": [4] }
		},
		"conditions": {
			"buy": { "op": "crossover", "args": [ { "op": "ref", "val": "emaFast" }, { "op": "ref", "val": "emaSlow" } ] },
			"sell": { "op": "crossunder", "args": [ { "op": "ref", "val": "emaFast" }, { "op": "ref", "val": "emaSlow" } ] }
		},
		"rules": [
			{ "condition": "buy", "action": "entry", "id": "LONG", "direction": "long" },
			{ "condition": "sell", "action": "close", "id": "LONG", "direction": "long" }
		]
	}`

	var cfg pine.IRConfig
	if err := json.Unmarshal([]byte(irJSON), &cfg); err != nil {
		t.Fatalf("unmarshal IRConfig: %v", err)
	}

	evaluator := NewPineEvaluator("CustomPine", "v1", "BTCUSDT", "1m", cfg)

	// Create candles to trigger a crossover
	// Fast period 2, slow period 4.
	// We want emaFast to cross above emaSlow.
	candles := []marketdata.Candle{
		{Open: "100", High: "100", Low: "100", Close: "100", Volume: "10", CloseTime: time.Now().Add(-5 * time.Minute)},
		{Open: "98", High: "98", Low: "98", Close: "98", Volume: "10", CloseTime: time.Now().Add(-4 * time.Minute)},
		{Open: "96", High: "96", Low: "96", Close: "96", Volume: "10", CloseTime: time.Now().Add(-3 * time.Minute)},
		{Open: "95", High: "95", Low: "95", Close: "95", Volume: "10", CloseTime: time.Now().Add(-2 * time.Minute)},
		{Open: "120", High: "120", Low: "120", Close: "120", Volume: "10", CloseTime: time.Now().Add(-1 * time.Minute)}, // Large jump to trigger crossover
	}

	sig, err := evaluator.Evaluate(candles)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Should trigger a buy signal
	if sig.Side != SideBuy {
		t.Errorf("expected SideBuy, got %s (Reason: %s)", sig.Side, sig.Reason)
	}
}

func TestPineEvaluatorCrossoverGapFilter(t *testing.T) {
	// A strategy where buy = crossover(emaFast, emaSlow)
	irJSON := `{
		"indicators": {
			"emaFast": { "type": "ema", "source": "close", "params": [2] },
			"emaSlow": { "type": "ema", "source": "close", "params": [4] }
		},
		"conditions": {
			"buy": { "op": "crossover", "args": [ { "op": "ref", "val": "emaFast" }, { "op": "ref", "val": "emaSlow" } ] }
		},
		"rules": [
			{ "condition": "buy", "action": "entry", "id": "LONG", "direction": "long" }
		]
	}`

	var cfg pine.IRConfig
	if err := json.Unmarshal([]byte(irJSON), &cfg); err != nil {
		t.Fatalf("unmarshal IRConfig: %v", err)
	}

	evaluator := NewPineEvaluator("CustomPine", "v1", "BTCUSDT", "1m", cfg)

	// Case 1: emaFast crosses above emaSlow but the gap is extremely tiny (less than 0.01% of emaSlow)
	// We want to verify it does NOT trigger SideBuy (whipsaw filter).
	// Let's create mock candles where they cross but are almost equal.
	candlesTinyGap := []marketdata.Candle{
		{Open: "100.0", High: "100.0", Low: "100.0", Close: "100.0", Volume: "10"},
		{Open: "100.0", High: "100.0", Low: "100.0", Close: "100.0", Volume: "10"},
		{Open: "100.0", High: "100.0", Low: "100.0", Close: "100.0", Volume: "10"},
		{Open: "100.0", High: "100.0", Low: "100.0", Close: "99.9", Volume: "10"},
		{Open: "100.0", High: "100.0", Low: "100.0", Close: "100.02", Volume: "10"},
	}

	sig, err := evaluator.Evaluate(candlesTinyGap)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if sig.Side == SideBuy {
		t.Errorf("expected whipsaw filter to prevent buy signal, but got SideBuy")
	}
}
