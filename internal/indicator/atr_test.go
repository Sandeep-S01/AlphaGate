package indicator

import (
	"testing"
	"time"

	"sentra/internal/marketdata"
)

func TestAverageTrueRangeUsesHighLowAndPreviousClose(t *testing.T) {
	candles := []marketdata.Candle{
		testCandle(0, "10", "10", "10"),
		testCandle(1, "13", "9", "12"),
		testCandle(2, "15", "11", "14"),
		testCandle(3, "14", "10", "11"),
	}

	value, err := AverageTrueRange(candles, 3)
	if err != nil {
		t.Fatalf("AverageTrueRange returned error: %v", err)
	}

	if value != 4 {
		t.Fatalf("expected ATR 4, got %f", value)
	}
}

func testCandle(index int, high string, low string, close string) marketdata.Candle {
	openTime := time.Unix(1000, 0).UTC().Add(time.Duration(index) * time.Minute)
	return marketdata.Candle{
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		OpenTime:  openTime,
		CloseTime: openTime.Add(time.Minute),
		High:      high,
		Low:       low,
		Close:     close,
		IsClosed:  true,
	}
}
