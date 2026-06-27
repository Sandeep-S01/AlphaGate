package strategy

import (
	"fmt"
	"math"
	"time"

	"sentra/internal/marketdata"
)

type RSIConfig struct {
	Name       string
	Version    string
	Symbol     string
	Interval   string
	Period     int
	Oversold   float64
	Overbought float64
}

type RSIMeanReversion struct {
	cfg RSIConfig
}

func NewRSIMeanReversion(cfg RSIConfig) *RSIMeanReversion {
	return &RSIMeanReversion{cfg: cfg}
}

func (r *RSIMeanReversion) Evaluate(candles []marketdata.Candle) (Signal, error) {
	if r.cfg.Period <= 0 {
		return Signal{}, fmt.Errorf("RSI period must be positive")
	}
	if r.cfg.Oversold <= 0 || r.cfg.Overbought <= r.cfg.Oversold || r.cfg.Overbought >= 100 {
		return Signal{}, fmt.Errorf("RSI thresholds are invalid")
	}
	required := r.cfg.Period + 1
	if len(candles) < required {
		return Signal{}, fmt.Errorf("insufficient candles: need %d, got %d", required, len(candles))
	}
	rsiWindow := candles
	if len(candles) > 250 {
		rsiWindow = candles[len(candles)-250:]
	}
	value, err := rsi(rsiWindow, r.cfg.Period)
	if err != nil {
		return Signal{}, err
	}
	side := SideHold
	reason := "RSI is neutral"
	if value <= r.cfg.Oversold {
		side = SideBuy
		reason = "RSI is oversold"
	}
	if value >= r.cfg.Overbought {
		side = SideSell
		reason = "RSI is overbought"
	}
	generatedAt := time.Now().UTC()
	if len(candles) > 0 {
		generatedAt = candles[len(candles)-1].CloseTime
	}
	return Signal{
		StrategyName: r.cfg.Name,
		Version:      r.cfg.Version,
		Symbol:       r.cfg.Symbol,
		Interval:     r.cfg.Interval,
		Side:         side,
		Strength:     math.Abs(value - 50),
		Reason:       reason,
		GeneratedAt:  generatedAt,
	}, nil
}

// rsi calculates RSI using Wilder's exponential smoothing, which is the
// industry standard used by TradingView and other charting tools. The previous
// simple-average implementation produced values that diverged from reference
// tools, causing signals to fire at unexpected levels.
//
// Algorithm:
//  1. Compute initial average gain/loss from the first `period` price changes.
//  2. Apply Wilder's smoothing for each subsequent change:
//     avgGain = (prevAvgGain * (period-1) + currentGain) / period
//     avgLoss = (prevAvgLoss * (period-1) + currentLoss) / period
//  3. RS = avgGain / avgLoss, RSI = 100 - 100/(1+RS)
func rsi(candles []marketdata.Candle, period int) (float64, error) {
	if period <= 0 {
		return 0, fmt.Errorf("RSI period must be positive")
	}
	if len(candles) < period+1 {
		return 0, fmt.Errorf("insufficient candles for RSI: need %d, got %d", period+1, len(candles))
	}

	changes := make([]float64, 0, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		prev, err := closeFloat(candles[i-1])
		if err != nil {
			return 0, err
		}
		cur, err := closeFloat(candles[i])
		if err != nil {
			return 0, err
		}
		changes = append(changes, cur-prev)
	}

	// Step 1: initial SMA over the first `period` changes.
	avgGain := 0.0
	avgLoss := 0.0
	for _, c := range changes[:period] {
		if c > 0 {
			avgGain += c
		} else if c < 0 {
			avgLoss += -c
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Step 2: Wilder's smoothing for remaining changes.
	for _, c := range changes[period:] {
		currentGain := 0.0
		currentLoss := 0.0
		if c > 0 {
			currentGain = c
		} else if c < 0 {
			currentLoss = -c
		}
		avgGain = (avgGain*float64(period-1) + currentGain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + currentLoss) / float64(period)
	}

	if avgLoss == 0 {
		if avgGain == 0 {
			return 50, nil // no movement → neutral
		}
		return 100, nil
	}
	if avgGain == 0 {
		return 0, nil
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs)), nil
}
