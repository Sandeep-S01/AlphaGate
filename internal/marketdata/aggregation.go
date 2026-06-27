package marketdata

import (
	"fmt"
	"strconv"
	"time"
)

func AggregateCandles(candles []Candle, targetInterval string) ([]Candle, error) {
	targetStep, err := IntervalDuration(targetInterval)
	if err != nil {
		return nil, err
	}
	if len(candles) == 0 {
		return nil, nil
	}
	sourceStep, err := IntervalDuration(candles[0].Interval)
	if err != nil {
		return nil, err
	}
	if targetStep <= sourceStep || targetStep%sourceStep != 0 {
		return nil, fmt.Errorf("target interval must be a multiple of source interval")
	}

	bucketSize := int(targetStep / sourceStep)
	aggregated := make([]Candle, 0, len(candles)/bucketSize)
	byOpenTime := make(map[time.Time]Candle, len(candles))
	for _, candle := range candles {
		byOpenTime[candle.OpenTime.UTC()] = candle
	}
	firstOpen := candles[0].OpenTime.UTC()
	lastOpen := candles[len(candles)-1].OpenTime.UTC()
	for bucketStart := firstOpen.Truncate(targetStep); !bucketStart.After(lastOpen); bucketStart = bucketStart.Add(targetStep) {
		bucket := make([]Candle, 0, bucketSize)
		for offset := 0; offset < bucketSize; offset++ {
			openTime := bucketStart.Add(time.Duration(offset) * sourceStep)
			candle, ok := byOpenTime[openTime]
			if !ok {
				bucket = nil
				break
			}
			bucket = append(bucket, candle)
		}
		if len(bucket) != bucketSize || !isCompleteBucket(bucket, sourceStep) {
			continue
		}
		candle, err := aggregateBucket(bucket, targetInterval, targetStep)
		if err != nil {
			return nil, err
		}
		aggregated = append(aggregated, candle)
	}
	return aggregated, nil
}

func isCompleteBucket(candles []Candle, step time.Duration) bool {
	for index := 1; index < len(candles); index++ {
		if !candles[index].OpenTime.Equal(candles[index-1].OpenTime.Add(step)) {
			return false
		}
	}
	return true
}

func aggregateBucket(candles []Candle, interval string, targetStep time.Duration) (Candle, error) {
	first := candles[0]
	last := candles[len(candles)-1]
	high := first.High
	low := first.Low
	volume := 0.0
	quoteVolume := 0.0
	tradeCount := int64(0)

	for _, candle := range candles {
		if greaterNumeric(candle.High, high) {
			high = candle.High
		}
		if lessNumeric(candle.Low, low) {
			low = candle.Low
		}
		parsedVolume, err := strconv.ParseFloat(candle.Volume, 64)
		if err != nil {
			return Candle{}, fmt.Errorf("parse volume: %w", err)
		}
		parsedQuoteVolume := 0.0
		if candle.QuoteVolume != "" {
			parsedQuoteVolume, err = strconv.ParseFloat(candle.QuoteVolume, 64)
			if err != nil {
				return Candle{}, fmt.Errorf("parse quote volume: %w", err)
			}
		}
		volume += parsedVolume
		quoteVolume += parsedQuoteVolume
		tradeCount += candle.TradeCount
	}

	return Candle{
		Exchange:    first.Exchange,
		Symbol:      first.Symbol,
		Interval:    interval,
		OpenTime:    first.OpenTime,
		CloseTime:   first.OpenTime.Add(targetStep),
		EventTime:   last.EventTime,
		Open:        first.Open,
		High:        high,
		Low:         low,
		Close:       last.Close,
		Volume:      formatFloat(volume),
		QuoteVolume: formatFloat(quoteVolume),
		TradeCount:  tradeCount,
		IsClosed:    true,
	}, nil
}

func greaterNumeric(left string, right string) bool {
	leftValue, leftErr := strconv.ParseFloat(left, 64)
	rightValue, rightErr := strconv.ParseFloat(right, 64)
	return leftErr == nil && rightErr == nil && leftValue > rightValue
}

func lessNumeric(left string, right string) bool {
	leftValue, leftErr := strconv.ParseFloat(left, 64)
	rightValue, rightErr := strconv.ParseFloat(right, 64)
	return leftErr == nil && rightErr == nil && leftValue < rightValue
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
