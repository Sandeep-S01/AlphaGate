package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestKlineStreamURLBuildsLowercaseStreamName(t *testing.T) {
	client := NewClient(Config{
		RESTBaseURL: "https://api.binance.com",
		WSBaseURL:   "wss://stream.binance.com:9443/ws",
		HTTPClient:  http.DefaultClient,
	})

	got, err := client.KlineStreamURL("BTCUSDT", "1m")
	if err != nil {
		t.Fatalf("KlineStreamURL returned error: %v", err)
	}

	want := "wss://stream.binance.com:9443/ws/btcusdt@kline_1m"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestGetKlinesRequestsRESTAPIAndNormalizesCandles(t *testing.T) {
	var requestedPath string
	var requestedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([][]any{
			{
				float64(1672515780000),
				"16500.10",
				"16520.30",
				"16490.40",
				"16510.20",
				"12.345",
				float64(1672515839999),
				"203703.500",
				float64(100),
				"6.1",
				"100000.0",
				"0",
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		RESTBaseURL: server.URL,
		WSBaseURL:   "wss://stream.binance.com:9443/ws",
		HTTPClient:  server.Client(),
	})

	start := time.UnixMilli(1672515780000).UTC()
	end := time.UnixMilli(1672515840000).UTC()
	candles, err := client.GetKlines(context.Background(), KlineRequest{
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		StartTime: start,
		EndTime:   end,
		Limit:     1000,
	})
	if err != nil {
		t.Fatalf("GetKlines returned error: %v", err)
	}

	if requestedPath != "/api/v3/klines" {
		t.Fatalf("expected klines path, got %q", requestedPath)
	}
	if requestedQuery != "endTime=1672515840000&interval=1m&limit=1000&startTime=1672515780000&symbol=BTCUSDT" {
		t.Fatalf("unexpected query %q", requestedQuery)
	}
	if len(candles) != 1 {
		t.Fatalf("expected 1 candle, got %d", len(candles))
	}
	candle := candles[0]
	if candle.Exchange != "binance" || candle.Symbol != "BTCUSDT" || candle.Interval != "1m" {
		t.Fatalf("unexpected candle identity: %+v", candle)
	}
	if candle.Open != "16500.10" || candle.High != "16520.30" || candle.Low != "16490.40" || candle.Close != "16510.20" {
		t.Fatalf("unexpected OHLC: %+v", candle)
	}
	if !candle.IsClosed {
		t.Fatal("expected historical kline to be closed")
	}
}

func TestGetExchangeInfoRequestsSymbolFromRESTAPI(t *testing.T) {
	var requestedPath string
	var requestedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(exchangeInfoResponse{
			Symbols: []symbolResponse{
				{
					Symbol:              "BTCUSDT",
					Status:              "TRADING",
					BaseAsset:           "BTC",
					QuoteAsset:          "USDT",
					BaseAssetPrecision:  8,
					QuoteAssetPrecision: 8,
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		RESTBaseURL: server.URL,
		WSBaseURL:   "wss://stream.binance.com:9443/ws",
		HTTPClient:  server.Client(),
	})

	info, err := client.GetExchangeInfo(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("GetExchangeInfo returned error: %v", err)
	}

	if requestedPath != "/api/v3/exchangeInfo" {
		t.Fatalf("expected exchange info path, got %q", requestedPath)
	}
	if requestedQuery != "symbol=BTCUSDT" {
		t.Fatalf("expected symbol query, got %q", requestedQuery)
	}
	if len(info.Symbols) != 1 || info.Symbols[0].Symbol != "BTCUSDT" {
		t.Fatalf("unexpected exchange info response: %+v", info)
	}
}
