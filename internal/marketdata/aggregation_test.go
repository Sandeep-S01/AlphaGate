package marketdata

import (
	"testing"
	"time"
)

func TestAggregateCandlesBuildsCompleteBuckets(t *testing.T) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	candles := []Candle{
		testCandle(start, "100", "105", "99", "101", "10", "1000", 1),
		testCandle(start.Add(time.Minute), "101", "106", "100", "102", "11", "1100", 2),
		testCandle(start.Add(2*time.Minute), "102", "104", "97", "103", "12", "1200", 3),
		testCandle(start.Add(3*time.Minute), "103", "108", "102", "104", "13", "1300", 4),
		testCandle(start.Add(4*time.Minute), "104", "107", "101", "105", "14", "1400", 5),
		testCandle(start.Add(5*time.Minute), "105", "109", "104", "106", "15", "1500", 6),
	}

	aggregated, err := AggregateCandles(candles, "5m")
	if err != nil {
		t.Fatalf("AggregateCandles returned error: %v", err)
	}

	if len(aggregated) != 1 {
		t.Fatalf("expected only one complete 5m bucket, got %d", len(aggregated))
	}
	got := aggregated[0]
	if got.Interval != "5m" || !got.OpenTime.Equal(start) || !got.CloseTime.Equal(start.Add(5*time.Minute)) {
		t.Fatalf("unexpected aggregate time metadata: %+v", got)
	}
	if got.Open != "100" || got.High != "108" || got.Low != "97" || got.Close != "105" {
		t.Fatalf("unexpected OHLC: %+v", got)
	}
	if got.Volume != "60" || got.QuoteVolume != "6000" || got.TradeCount != 15 {
		t.Fatalf("unexpected aggregate sums: volume=%s quote=%s trades=%d", got.Volume, got.QuoteVolume, got.TradeCount)
	}
}

func TestAggregateCandlesAlignsBucketsToIntervalBoundaries(t *testing.T) {
	start := time.Date(2026, 6, 16, 1, 47, 0, 0, time.UTC)
	candles := make([]Candle, 0, 45)
	for index := 0; index < 45; index++ {
		openTime := start.Add(time.Duration(index) * time.Minute)
		candles = append(candles, testCandle(openTime, "100", "105", "99", "101", "10", "1000", 1))
	}

	aggregated, err := AggregateCandles(candles, "15m")
	if err != nil {
		t.Fatalf("AggregateCandles returned error: %v", err)
	}

	if len(aggregated) != 2 {
		t.Fatalf("expected two complete aligned 15m buckets, got %d", len(aggregated))
	}
	firstAligned := time.Date(2026, 6, 16, 2, 0, 0, 0, time.UTC)
	if !aggregated[0].OpenTime.Equal(firstAligned) {
		t.Fatalf("expected first aligned bucket at %s, got %s", firstAligned, aggregated[0].OpenTime)
	}
	if !aggregated[1].OpenTime.Equal(firstAligned.Add(15 * time.Minute)) {
		t.Fatalf("expected second aligned bucket at %s, got %s", firstAligned.Add(15*time.Minute), aggregated[1].OpenTime)
	}
}

func testCandle(openTime time.Time, open string, high string, low string, close string, volume string, quoteVolume string, tradeCount int64) Candle {
	return Candle{
		Exchange:    "binance",
		Symbol:      "BTCUSDT",
		Interval:    "1m",
		OpenTime:    openTime,
		CloseTime:   openTime.Add(time.Minute),
		Open:        open,
		High:        high,
		Low:         low,
		Close:       close,
		Volume:      volume,
		QuoteVolume: quoteVolume,
		TradeCount:  tradeCount,
		IsClosed:    true,
	}
}
