package indicator

import (
	"fmt"
	"math"
	"strconv"

	"sentra/internal/marketdata"
)

func AverageTrueRange(candles []marketdata.Candle, period int) (float64, error) {
	if period <= 0 {
		return 0, fmt.Errorf("period must be positive")
	}
	required := period + 1
	if len(candles) < required {
		return 0, fmt.Errorf("insufficient candles: need %d, got %d", required, len(candles))
	}

	window := candles[len(candles)-required:]
	total := 0.0
	for index := 1; index < len(window); index++ {
		high, err := candleValue(window[index].High, window[index].Close, "high")
		if err != nil {
			return 0, err
		}
		low, err := candleValue(window[index].Low, window[index].Close, "low")
		if err != nil {
			return 0, err
		}
		previousClose, err := candleValue(window[index-1].Close, "", "previous close")
		if err != nil {
			return 0, err
		}
		trueRange := math.Max(high-low, math.Max(math.Abs(high-previousClose), math.Abs(low-previousClose)))
		total += trueRange
	}
	return total / float64(period), nil
}

func candleValue(value string, fallback string, name string) (float64, error) {
	if value == "" {
		value = fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse candle %s %q: %w", name, value, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("candle %s must be positive", name)
	}
	return parsed, nil
}
