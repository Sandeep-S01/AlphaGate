package strategy

import "testing"

func TestRSIMeanReversionGeneratesBuyWhenOversold(t *testing.T) {
	evaluator := NewRSIMeanReversion(RSIConfig{
		Name:       StrategyRSIMeanReversion,
		Version:    "v1",
		Symbol:     "BTCUSDT",
		Interval:   "1m",
		Period:     3,
		Oversold:   30,
		Overbought: 70,
	})

	signal, err := evaluator.Evaluate(candlesWithCloses("100", "95", "90", "85", "80"))
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if signal.Side != SideBuy {
		t.Fatalf("expected buy, got %q", signal.Side)
	}
}

func TestRSIMeanReversionGeneratesSellWhenOverbought(t *testing.T) {
	evaluator := NewRSIMeanReversion(RSIConfig{
		Name:       StrategyRSIMeanReversion,
		Version:    "v1",
		Symbol:     "BTCUSDT",
		Interval:   "1m",
		Period:     3,
		Oversold:   30,
		Overbought: 70,
	})

	signal, err := evaluator.Evaluate(candlesWithCloses("80", "85", "90", "95", "100"))
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if signal.Side != SideSell {
		t.Fatalf("expected sell, got %q", signal.Side)
	}
}
