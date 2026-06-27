package marketdata

import (
	"context"
	"testing"
	"time"
)

func TestAggregationServiceDeletesTargetRangeBeforeWriting(t *testing.T) {
	start := time.Date(2026, 6, 16, 1, 47, 0, 0, time.UTC)
	store := &fakeAggregationStore{
		candles: []Candle{},
	}
	for index := 0; index < 45; index++ {
		openTime := start.Add(time.Duration(index) * time.Minute)
		store.candles = append(store.candles, testCandle(openTime, "100", "105", "99", "101", "10", "1000", 1))
	}
	service := NewAggregationService(store)

	result, err := service.Aggregate(context.Background(), AggregationRequest{
		Symbol:          "BTCUSDT",
		SourceInterval:  "1m",
		TargetIntervals: []string{"15m"},
		From:            start,
		To:              start.Add(45 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Aggregate returned error: %v", err)
	}

	if len(store.deleted) != 1 {
		t.Fatalf("expected one target range delete, got %d", len(store.deleted))
	}
	deleted := store.deleted[0]
	if deleted.Symbol != "BTCUSDT" || deleted.Interval != "15m" {
		t.Fatalf("unexpected delete query: %+v", deleted)
	}
	if !deleted.From.Equal(start.UTC()) || !deleted.To.Equal(start.Add(45*time.Minute).UTC()) {
		t.Fatalf("unexpected delete range: %+v", deleted)
	}
	if len(store.batches) != 1 || len(store.batches[0]) != result.TargetIntervals[0].Count {
		t.Fatalf("expected one write batch matching result count, got batches=%d result=%+v", len(store.batches), result)
	}
}

type fakeAggregationStore struct {
	candles []Candle
	deleted []CandleQuery
	batches [][]Candle
}

func (f *fakeAggregationStore) Upsert(ctx context.Context, candle Candle) error {
	f.candles = append(f.candles, candle)
	return nil
}

func (f *fakeAggregationStore) UpsertBatch(ctx context.Context, candles []Candle) error {
	f.batches = append(f.batches, append([]Candle(nil), candles...))
	return nil
}

func (f *fakeAggregationStore) DeleteRange(ctx context.Context, query CandleQuery) error {
	f.deleted = append(f.deleted, query)
	return nil
}

func (f *fakeAggregationStore) List(ctx context.Context, query CandleQuery) ([]Candle, error) {
	return f.candles, nil
}
