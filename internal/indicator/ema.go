package indicator

import "fmt"

func ExponentialMovingAverage(values []float64, period int) (float64, error) {
	if period <= 0 {
		return 0, fmt.Errorf("period must be positive")
	}
	if len(values) < period {
		return 0, fmt.Errorf("insufficient values: need %d, got %d", period, len(values))
	}

	window := values[len(values)-period:]
	ema := window[0]
	multiplier := 2.0 / float64(period+1)
	for index := 1; index < len(window); index++ {
		ema = (window[index]-ema)*multiplier + ema
	}
	return ema, nil
}
