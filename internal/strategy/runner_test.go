package strategy

import (
	"context"
	"testing"

	"sentra/internal/marketdata"
)

func TestRunnerStoresAndPublishesGeneratedSignal(t *testing.T) {
	candleReader := &fakeCandleReader{candles: candlesWithCloses("100", "100", "100", "100", "80", "130")}
	store := &fakeSignalStore{}
	publisher := &fakeSignalPublisher{}
	runner := NewRunner(candleReader, store, publisher, RunnerConfig{
		Symbol:        "BTCUSDT",
		Interval:      "1m",
		LookbackLimit: 6,
		SignalStream:  "stream:strategy-signals",
		Evaluator: NewSMACrossover(SMAConfig{
			Name:       "sma-crossover",
			Version:    "v1",
			Symbol:     "BTCUSDT",
			Interval:   "1m",
			FastPeriod: 2,
			SlowPeriod: 4,
		}),
	})

	signal, err := runner.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	if signal.Side != SideBuy {
		t.Fatalf("expected buy signal, got %q", signal.Side)
	}
	if len(store.saved) != 1 {
		t.Fatalf("expected 1 saved signal, got %d", len(store.saved))
	}
	if publisher.stream != "stream:strategy-signals" {
		t.Fatalf("expected strategy stream, got %q", publisher.stream)
	}
	if len(publisher.published) != 1 || publisher.published[0].Side != SideBuy {
		t.Fatalf("unexpected published signals: %+v", publisher.published)
	}
	if candleReader.query.Limit != 6 {
		t.Fatalf("expected lookback limit 6, got %d", candleReader.query.Limit)
	}
}

type fakeCandleReader struct {
	query   marketdata.CandleQuery
	candles []marketdata.Candle
}

func (f *fakeCandleReader) List(ctx context.Context, query marketdata.CandleQuery) ([]marketdata.Candle, error) {
	f.query = query
	return f.candles, nil
}

type fakeSignalStore struct {
	saved []Signal
}

func (f *fakeSignalStore) Save(ctx context.Context, signal Signal) (string, error) {
	f.saved = append(f.saved, signal)
	return "signal-1", nil
}

type fakeSignalPublisher struct {
	stream    string
	published []Signal
}

func (f *fakeSignalPublisher) PublishSignal(ctx context.Context, stream string, signal Signal) error {
	f.stream = stream
	f.published = append(f.published, signal)
	return nil
}
