package strategy

import (
	"testing"
	"time"

	"sentra/internal/marketdata"
)

func TestSMACrossoverGeneratesBuyWhenFastCrossesAboveSlow(t *testing.T) {
	candles := candlesWithCloses("100", "100", "100", "100", "80", "130")
	evaluator := NewSMACrossover(SMAConfig{
		Name:       "sma-crossover",
		Version:    "v1",
		Symbol:     "BTCUSDT",
		Interval:   "1m",
		FastPeriod: 2,
		SlowPeriod: 4,
	})

	signal, err := evaluator.Evaluate(candles)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if signal.Side != SideBuy {
		t.Fatalf("expected buy, got %q", signal.Side)
	}
	if signal.Strength <= 0 {
		t.Fatalf("expected positive strength, got %f", signal.Strength)
	}
	if signal.Reason == "" {
		t.Fatal("expected signal reason")
	}
}

func TestSMACrossoverGeneratesSellWhenFastCrossesBelowSlow(t *testing.T) {
	candles := candlesWithCloses("100", "100", "100", "100", "120", "70")
	evaluator := NewSMACrossover(SMAConfig{
		Name:       "sma-crossover",
		Version:    "v1",
		Symbol:     "BTCUSDT",
		Interval:   "1m",
		FastPeriod: 2,
		SlowPeriod: 4,
	})

	signal, err := evaluator.Evaluate(candles)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if signal.Side != SideSell {
		t.Fatalf("expected sell, got %q", signal.Side)
	}
}

func TestSMACrossoverHoldsWithoutCross(t *testing.T) {
	candles := candlesWithCloses("100", "101", "102", "103", "104", "105")
	evaluator := NewSMACrossover(SMAConfig{
		Name:       "sma-crossover",
		Version:    "v1",
		Symbol:     "BTCUSDT",
		Interval:   "1m",
		FastPeriod: 2,
		SlowPeriod: 4,
	})

	signal, err := evaluator.Evaluate(candles)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if signal.Side != SideHold {
		t.Fatalf("expected hold, got %q", signal.Side)
	}
}

func TestSMACrossoverRejectsInsufficientCandles(t *testing.T) {
	evaluator := NewSMACrossover(SMAConfig{
		Name:       "sma-crossover",
		Version:    "v1",
		Symbol:     "BTCUSDT",
		Interval:   "1m",
		FastPeriod: 2,
		SlowPeriod: 4,
	})

	_, err := evaluator.Evaluate(candlesWithCloses("100", "101", "102", "103"))
	if err == nil {
		t.Fatal("expected insufficient candles to fail")
	}
}

func candlesWithCloses(values ...string) []marketdata.Candle {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	candles := make([]marketdata.Candle, 0, len(values))
	for index, value := range values {
		openTime := start.Add(time.Duration(index) * time.Minute)
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Close:     value,
			IsClosed:  true,
		})
	}
	return candles
}
