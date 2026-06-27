package strategy

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"sentra/internal/marketdata"
)

type Evaluator interface {
	Evaluate(candles []marketdata.Candle) (Signal, error)
}

type SMAConfig struct {
	Name       string
	Version    string
	Symbol     string
	Interval   string
	FastPeriod int
	SlowPeriod int
}

type SMACrossover struct {
	cfg SMAConfig
}

func NewSMACrossover(cfg SMAConfig) *SMACrossover {
	return &SMACrossover{cfg: cfg}
}

func (s *SMACrossover) Evaluate(candles []marketdata.Candle) (Signal, error) {
	if s.cfg.FastPeriod <= 0 || s.cfg.SlowPeriod <= 0 {
		return Signal{}, fmt.Errorf("SMA periods must be positive")
	}
	if s.cfg.FastPeriod >= s.cfg.SlowPeriod {
		return Signal{}, fmt.Errorf("fast period must be less than slow period")
	}
	required := s.cfg.SlowPeriod + 1
	if len(candles) < required {
		return Signal{}, fmt.Errorf("insufficient candles: need %d, got %d", required, len(candles))
	}

	previousFast, err := averageClose(candles[len(candles)-s.cfg.FastPeriod-1 : len(candles)-1])
	if err != nil {
		return Signal{}, err
	}
	previousSlow, err := averageClose(candles[len(candles)-s.cfg.SlowPeriod-1 : len(candles)-1])
	if err != nil {
		return Signal{}, err
	}
	currentFast, err := averageClose(candles[len(candles)-s.cfg.FastPeriod:])
	if err != nil {
		return Signal{}, err
	}
	currentSlow, err := averageClose(candles[len(candles)-s.cfg.SlowPeriod:])
	if err != nil {
		return Signal{}, err
	}

	side := SideHold
	reason := "fast SMA did not cross slow SMA"

	// Minimum gap threshold: the crossover must exceed 0.01% of the slow SMA
	// to filter out noise-driven whipsaw signals in choppy/sideways markets.
	minGap := currentSlow * 0.0001

	if previousFast < previousSlow && currentFast > currentSlow && (currentFast-currentSlow) > minGap {
		side = SideBuy
		reason = "fast SMA crossed above slow SMA"
	}
	if previousFast > previousSlow && currentFast < currentSlow && (currentSlow-currentFast) > minGap {
		side = SideSell
		reason = "fast SMA crossed below slow SMA"
	}

	generatedAt := time.Now().UTC()
	if len(candles) > 0 {
		generatedAt = candles[len(candles)-1].CloseTime
	}

	return Signal{
		StrategyName: s.cfg.Name,
		Version:      s.cfg.Version,
		Symbol:       s.cfg.Symbol,
		Interval:     s.cfg.Interval,
		Side:         side,
		Strength:     smaStrengthPercent(currentFast, currentSlow),
		Reason:       reason,
		GeneratedAt:  generatedAt,
	}, nil
}

func closeFloat(candle marketdata.Candle) (float64, error) {
	return strconv.ParseFloat(candle.Close, 64)
}

func averageClose(candles []marketdata.Candle) (float64, error) {
	if len(candles) == 0 {
		return 0, fmt.Errorf("no candles to average")
	}

	var total float64
	for _, candle := range candles {
		closeValue, err := closeFloat(candle)
		if err != nil {
			return 0, fmt.Errorf("parse candle close %q: %w", candle.Close, err)
		}
		total += closeValue
	}
	return total / float64(len(candles)), nil
}

// smaStrengthPercent returns the absolute gap between fast and slow SMA
// as a percentage of the slow SMA. This normalizes strength across assets
// with different price scales (e.g., BTC at $100K vs ETH at $3K).
func smaStrengthPercent(fast float64, slow float64) float64 {
	if slow == 0 {
		return 0
	}
	return math.Abs(fast-slow) / slow * 100
}
