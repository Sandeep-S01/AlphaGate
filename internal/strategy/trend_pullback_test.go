package strategy

import (
	"testing"
	"time"

	"sentra/internal/marketdata"
)

func TestBTCTrendPullbackBuysOnRecoveryInUptrend(t *testing.T) {
	evaluator := NewBTCTrendPullback(TrendPullbackConfig{
		Name:           StrategyBTCTrendPullback,
		Version:        "v1",
		Symbol:         "BTCUSDT",
		Interval:       "15m",
		PullbackPeriod: 3,
		TrendPeriod:    5,
		RSIPeriod:      3,
	})

	signal, err := evaluator.Evaluate(candlesForTrendPullback([]string{"100", "104", "108", "112", "116", "110", "108", "117"}))
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if signal.Side != SideBuy {
		t.Fatalf("expected buy signal, got %+v", signal)
	}
	if signal.StrategyName != StrategyBTCTrendPullback {
		t.Fatalf("expected trend pullback strategy name, got %+v", signal)
	}
}

func TestBTCTrendPullbackStrengthIsPriceScaleInvariant(t *testing.T) {
	evaluator := NewBTCTrendPullback(TrendPullbackConfig{
		Name:           StrategyBTCTrendPullback,
		Version:        "v1",
		Symbol:         "BTCUSDT",
		Interval:       "15m",
		PullbackPeriod: 3,
		TrendPeriod:    5,
		RSIPeriod:      3,
	})

	baseSignal, err := evaluator.Evaluate(candlesForTrendPullback([]string{"100", "104", "108", "112", "116", "110", "108", "117"}))
	if err != nil {
		t.Fatalf("Evaluate returned error for base candles: %v", err)
	}
	scaledSignal, err := evaluator.Evaluate(candlesForTrendPullback([]string{"60000", "62400", "64800", "67200", "69600", "66000", "64800", "70200"}))
	if err != nil {
		t.Fatalf("Evaluate returned error for scaled candles: %v", err)
	}

	if baseSignal.Side != scaledSignal.Side {
		t.Fatalf("expected same side after price scaling, got base=%s scaled=%s", baseSignal.Side, scaledSignal.Side)
	}
	if diff := absFloat(baseSignal.Strength - scaledSignal.Strength); diff > 0.000001 {
		t.Fatalf("expected price-scale invariant strength, base=%f scaled=%f diff=%f", baseSignal.Strength, scaledSignal.Strength, diff)
	}
	if scaledSignal.Strength > 100 {
		t.Fatalf("expected scaled strength to remain compatible with 0-100 risk threshold, got %f", scaledSignal.Strength)
	}
}

func TestBTCTrendPullbackRequiresRSICrossAboveMidline(t *testing.T) {
	evaluator := NewBTCTrendPullback(TrendPullbackConfig{
		Name:           StrategyBTCTrendPullback,
		Version:        "v1",
		Symbol:         "BTCUSDT",
		Interval:       "15m",
		PullbackPeriod: 3,
		TrendPeriod:    5,
		RSIPeriod:      3,
	})

	signal, err := evaluator.Evaluate(candlesForTrendPullback([]string{"100", "102", "104", "106", "108", "105", "109"}))
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if signal.Side != SideHold {
		t.Fatalf("expected hold without fresh RSI midline cross, got %+v", signal)
	}
}

func TestBTCTrendPullbackBuysWhenVolatilityIsInRange(t *testing.T) {
	evaluator := NewBTCTrendPullback(TrendPullbackConfig{
		Name:             StrategyBTCTrendPullback,
		Version:          "v1",
		Symbol:           "BTCUSDT",
		Interval:         "15m",
		PullbackPeriod:   3,
		TrendPeriod:      5,
		RSIPeriod:        3,
		ATRPeriod:        3,
		MinATRPercent:    1,
		MaxATRPercent:    10,
		VolatilityFilter: true,
	})

	signal, err := evaluator.Evaluate(candlesForTrendPullback([]string{"100", "104", "108", "112", "116", "110", "108", "117"}))
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if signal.Side != SideBuy {
		t.Fatalf("expected buy signal with acceptable volatility, got %+v", signal)
	}
}

func TestBTCTrendPullbackHoldsWhenVolatilityIsTooLow(t *testing.T) {
	evaluator := NewBTCTrendPullback(TrendPullbackConfig{
		Name:             StrategyBTCTrendPullback,
		Version:          "v1",
		Symbol:           "BTCUSDT",
		Interval:         "15m",
		PullbackPeriod:   3,
		TrendPeriod:      5,
		RSIPeriod:        3,
		ATRPeriod:        3,
		MinATRPercent:    6,
		MaxATRPercent:    10,
		VolatilityFilter: true,
	})

	signal, err := evaluator.Evaluate(candlesForTrendPullback([]string{"100", "104", "108", "112", "116", "110", "108", "117"}))
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if signal.Side != SideHold {
		t.Fatalf("expected hold when ATR percent is below minimum, got %+v", signal)
	}
}

func TestBTCTrendPullbackHoldsWhenVolatilityIsTooHigh(t *testing.T) {
	evaluator := NewBTCTrendPullback(TrendPullbackConfig{
		Name:             StrategyBTCTrendPullback,
		Version:          "v1",
		Symbol:           "BTCUSDT",
		Interval:         "15m",
		PullbackPeriod:   3,
		TrendPeriod:      5,
		RSIPeriod:        3,
		ATRPeriod:        3,
		MinATRPercent:    1,
		MaxATRPercent:    3,
		VolatilityFilter: true,
	})

	signal, err := evaluator.Evaluate(candlesForTrendPullback([]string{"100", "104", "108", "112", "116", "110", "108", "117"}))
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if signal.Side != SideHold {
		t.Fatalf("expected hold when ATR percent is above maximum, got %+v", signal)
	}
}

func TestBTCTrendPullbackHoldsWithoutPullbackRecovery(t *testing.T) {
	evaluator := NewBTCTrendPullback(TrendPullbackConfig{
		Name:           StrategyBTCTrendPullback,
		Version:        "v1",
		Symbol:         "BTCUSDT",
		Interval:       "15m",
		PullbackPeriod: 3,
		TrendPeriod:    5,
		RSIPeriod:      3,
	})

	signal, err := evaluator.Evaluate(candlesForTrendPullback([]string{"100", "102", "104", "106", "108", "110", "112"}))
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if signal.Side != SideHold {
		t.Fatalf("expected hold signal without pullback recovery, got %+v", signal)
	}
}

func TestBTCTrendPullbackIgnoresCandlesOutsideLookback(t *testing.T) {
	evaluator := NewBTCTrendPullback(TrendPullbackConfig{
		Name:           StrategyBTCTrendPullback,
		Version:        "v1",
		Symbol:         "BTCUSDT",
		Interval:       "15m",
		PullbackPeriod: 3,
		TrendPeriod:    5,
		RSIPeriod:      3,
	})
	candles := append([]marketdata.Candle{
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "15m",
			OpenTime:  time.Unix(0, 0).UTC(),
			CloseTime: time.Unix(60, 0).UTC(),
			Close:     "not-a-price",
			High:      "not-a-price",
			Low:       "not-a-price",
			IsClosed:  true,
		},
	}, candlesForTrendPullback([]string{"100", "102", "104", "106", "108", "105", "109"})...)

	if _, err := evaluator.Evaluate(candles); err != nil {
		t.Fatalf("expected old out-of-window candle to be ignored, got error: %v", err)
	}
}

func candlesForTrendPullback(closes []string) []marketdata.Candle {
	base := time.Unix(1000, 0).UTC()
	candles := make([]marketdata.Candle, 0, len(closes))
	for index, closeValue := range closes {
		openTime := base.Add(time.Duration(index) * time.Minute)
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "15m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Close:     closeValue,
			High:      closeValue,
			Low:       closeValue,
			IsClosed:  true,
		})
	}
	return candles
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
