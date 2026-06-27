package marketdata

import "testing"

func TestParseBinanceKlineEventNormalizesClosedCandle(t *testing.T) {
	payload := []byte(`{
		"e": "kline",
		"E": 1672515782136,
		"s": "BTCUSDT",
		"k": {
			"t": 1672515780000,
			"T": 1672515839999,
			"s": "BTCUSDT",
			"i": "1m",
			"o": "16500.10",
			"c": "16510.20",
			"h": "16520.30",
			"l": "16490.40",
			"v": "12.345",
			"x": true
		}
	}`)

	candle, err := ParseBinanceKlineEvent(payload)
	if err != nil {
		t.Fatalf("ParseBinanceKlineEvent returned error: %v", err)
	}

	if candle.Exchange != "binance" {
		t.Fatalf("expected exchange binance, got %q", candle.Exchange)
	}
	if candle.Symbol != "BTCUSDT" || candle.Interval != "1m" {
		t.Fatalf("unexpected symbol/interval: %+v", candle)
	}
	if candle.Open != "16500.10" || candle.Close != "16510.20" || candle.High != "16520.30" || candle.Low != "16490.40" {
		t.Fatalf("unexpected OHLC values: %+v", candle)
	}
	if candle.Volume != "12.345" {
		t.Fatalf("unexpected volume %q", candle.Volume)
	}
	if !candle.IsClosed {
		t.Fatal("expected candle to be closed")
	}
	if candle.OpenTime.IsZero() || candle.CloseTime.IsZero() || candle.EventTime.IsZero() {
		t.Fatalf("expected timestamps to be populated: %+v", candle)
	}
}

func TestParseBinanceKlineEventRejectsNonKlineEvent(t *testing.T) {
	_, err := ParseBinanceKlineEvent([]byte(`{"e":"trade"}`))
	if err == nil {
		t.Fatal("expected non-kline event to fail")
	}
}
