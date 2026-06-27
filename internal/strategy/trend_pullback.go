package strategy

import (
	"fmt"
	"math"
	"time"

	"sentra/internal/indicator"
	"sentra/internal/marketdata"
)

const (
	DefaultTrendPullbackATRPeriod     = 14
	DefaultTrendPullbackMinATRPercent = 0.15
	DefaultTrendPullbackMaxATRPercent = 8.0
)

type TrendPullbackConfig struct {
	Name             string
	Version          string
	Symbol           string
	Interval         string
	PullbackPeriod   int
	TrendPeriod      int
	RSIPeriod        int
	VolatilityFilter bool
	ATRPeriod        int
	MinATRPercent    float64
	MaxATRPercent    float64
}

type BTCTrendPullback struct {
	cfg TrendPullbackConfig
}

func NewBTCTrendPullback(cfg TrendPullbackConfig) *BTCTrendPullback {
	return &BTCTrendPullback{cfg: cfg}
}

func (b *BTCTrendPullback) Evaluate(candles []marketdata.Candle) (Signal, error) {
	if b.cfg.PullbackPeriod <= 0 || b.cfg.TrendPeriod <= 0 || b.cfg.RSIPeriod <= 0 {
		return Signal{}, fmt.Errorf("trend pullback periods must be positive")
	}
	if b.cfg.PullbackPeriod >= b.cfg.TrendPeriod {
		return Signal{}, fmt.Errorf("pullback period must be less than trend period")
	}
	if b.cfg.VolatilityFilter {
		if b.cfg.ATRPeriod <= 0 {
			return Signal{}, fmt.Errorf("ATR period must be positive when volatility filter is enabled")
		}
		if b.cfg.MinATRPercent < 0 || b.cfg.MaxATRPercent < 0 {
			return Signal{}, fmt.Errorf("ATR percent thresholds cannot be negative")
		}
		if b.cfg.MaxATRPercent > 0 && b.cfg.MinATRPercent > b.cfg.MaxATRPercent {
			return Signal{}, fmt.Errorf("min ATR percent must be less than or equal to max ATR percent")
		}
	}
	required := b.cfg.TrendPeriod + 1
	if b.cfg.RSIPeriod+2 > required {
		required = b.cfg.RSIPeriod + 2
	}
	if b.cfg.VolatilityFilter && b.cfg.ATRPeriod+1 > required {
		required = b.cfg.ATRPeriod + 1
	}
	if len(candles) < required {
		return Signal{}, fmt.Errorf("insufficient candles: need %d, got %d", required, len(candles))
	}

	window := candles[len(candles)-required:]
	closes, err := closeValues(window)
	if err != nil {
		return Signal{}, err
	}
	previousCloses := closes[:len(closes)-1]
	previousPullback, err := indicator.ExponentialMovingAverage(previousCloses, b.cfg.PullbackPeriod)
	if err != nil {
		return Signal{}, err
	}
	currentPullback, err := indicator.ExponentialMovingAverage(closes, b.cfg.PullbackPeriod)
	if err != nil {
		return Signal{}, err
	}
	currentTrend, err := indicator.ExponentialMovingAverage(closes, b.cfg.TrendPeriod)
	if err != nil {
		return Signal{}, err
	}
	currentRSI, err := rsi(window, b.cfg.RSIPeriod)
	if err != nil {
		return Signal{}, err
	}
	previousRSI, err := rsi(window[:len(window)-1], b.cfg.RSIPeriod)
	if err != nil {
		return Signal{}, err
	}

	previousClose := closes[len(closes)-2]
	currentClose := closes[len(closes)-1]
	volatilityOK, err := b.volatilityInRange(window, currentClose)
	if err != nil {
		return Signal{}, err
	}
	side := SideHold
	reason := "trend pullback setup not confirmed"
	if currentPullback > currentTrend &&
		currentClose > currentTrend &&
		previousClose <= previousPullback &&
		currentClose > currentPullback &&
		previousRSI <= 50 &&
		currentRSI > 50 &&
		volatilityOK {
		side = SideBuy
		reason = "price recovered above pullback EMA with RSI momentum confirmation"
	}
	// Sell requires BOTH conditions: price must break below the trend EMA AND RSI
	// must confirm weakness by dropping below 35. The previous OR condition
	// (currentClose < currentTrend || currentRSI < 45) was far too aggressive —
	// RSI < 45 fires during normal consolidation in uptrends, causing premature
	// exits that lock in fee losses on every round trip.
	if currentClose < currentTrend && currentRSI < 35 {
		side = SideSell
		reason = "trend pullback invalidated: price below trend EMA and RSI confirms weakness"
	}

	generatedAt := time.Now().UTC()
	if len(candles) > 0 {
		generatedAt = candles[len(candles)-1].CloseTime
	}
	return Signal{
		StrategyName: b.cfg.Name,
		Version:      b.cfg.Version,
		Symbol:       b.cfg.Symbol,
		Interval:     b.cfg.Interval,
		Side:         side,
		Strength:     math.Abs(currentClose-currentPullback) + math.Abs(currentRSI-50) + math.Max(0, currentRSI-previousRSI),
		Reason:       reason,
		GeneratedAt:  generatedAt,
	}, nil
}

func (b *BTCTrendPullback) volatilityInRange(candles []marketdata.Candle, currentClose float64) (bool, error) {
	if !b.cfg.VolatilityFilter {
		return true, nil
	}
	atrValue, err := indicator.AverageTrueRange(candles, b.cfg.ATRPeriod)
	if err != nil {
		return false, err
	}
	atrPercent := atrValue / currentClose * 100
	if b.cfg.MinATRPercent > 0 && atrPercent < b.cfg.MinATRPercent {
		return false, nil
	}
	if b.cfg.MaxATRPercent > 0 && atrPercent > b.cfg.MaxATRPercent {
		return false, nil
	}
	return true, nil
}

func closeValues(candles []marketdata.Candle) ([]float64, error) {
	values := make([]float64, 0, len(candles))
	for _, candle := range candles {
		value, err := closeFloat(candle)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
