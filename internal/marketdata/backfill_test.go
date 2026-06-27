package marketdata

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBackfillFetchesAndStoresCandles(t *testing.T) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Minute)
	fetcher := &fakeHistoricalFetcher{
		candles: []Candle{
			{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", OpenTime: start, CloseTime: start.Add(time.Minute), Close: "100"},
			{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", OpenTime: start.Add(time.Minute), CloseTime: end, Close: "101"},
		},
	}
	store := &fakeCandleStore{}
	service := NewBackfillService(fetcher, store)

	count, err := service.Backfill(context.Background(), BackfillRequest{
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     start,
		To:       end,
		Limit:    1000,
	})
	if err != nil {
		t.Fatalf("Backfill returned error: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
	if len(store.upserted) != 2 {
		t.Fatalf("expected 2 stored candles, got %d", len(store.upserted))
	}
	if fetcher.request.Symbol != "BTCUSDT" || fetcher.request.Interval != "1m" {
		t.Fatalf("unexpected fetch request: %+v", fetcher.request)
	}
}

func TestBackfillPagesUntilRangeCompleteAndStoresBatches(t *testing.T) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	fetcher := &fakeHistoricalFetcher{
		pages: [][]Candle{
			{
				{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", OpenTime: start, CloseTime: start.Add(time.Minute), Close: "100", IsClosed: true},
				{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", OpenTime: start.Add(time.Minute), CloseTime: start.Add(2 * time.Minute), Close: "101", IsClosed: true},
			},
			{
				{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", OpenTime: start.Add(2 * time.Minute), CloseTime: start.Add(3 * time.Minute), Close: "102", IsClosed: true},
			},
		},
	}
	store := &fakeCandleStore{}
	jobs := &fakeBackfillJobStore{}
	service := NewBackfillService(fetcher, store, WithBackfillJobs(jobs))

	result, err := service.Start(context.Background(), BackfillRequest{
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     start,
		To:       start.Add(3 * time.Minute),
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if result.CandlesInserted != 3 {
		t.Fatalf("expected 3 inserted candles, got %d", result.CandlesInserted)
	}
	if len(fetcher.requests) != 2 {
		t.Fatalf("expected 2 fetch requests, got %d", len(fetcher.requests))
	}
	if !fetcher.requests[1].StartTime.Equal(start.Add(2 * time.Minute)) {
		t.Fatalf("expected second fetch to start at third candle, got %s", fetcher.requests[1].StartTime)
	}
	if len(store.batches) != 2 || len(store.batches[0]) != 2 || len(store.batches[1]) != 1 {
		t.Fatalf("unexpected stored batches: %+v", store.batches)
	}
	if jobs.saved.Status != BackfillStatusCompleted {
		t.Fatalf("expected completed job, got %q", jobs.saved.Status)
	}
}

func TestBackfillResumeUsesPersistedNextOpenTime(t *testing.T) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	jobs := &fakeBackfillJobStore{saved: BackfillJob{
		ID:              "job-1",
		Symbol:          "BTCUSDT",
		BaseInterval:    "1m",
		From:            start,
		To:              start.Add(3 * time.Minute),
		NextOpenTime:    start.Add(2 * time.Minute),
		Status:          BackfillStatusFailed,
		CandlesInserted: 2,
	}}
	fetcher := &fakeHistoricalFetcher{pages: [][]Candle{{
		{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", OpenTime: start.Add(2 * time.Minute), CloseTime: start.Add(3 * time.Minute), Close: "102", IsClosed: true},
	}}}
	service := NewBackfillService(fetcher, &fakeCandleStore{}, WithBackfillJobs(jobs))

	result, err := service.Resume(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("Resume returned error: %v", err)
	}

	if len(fetcher.requests) != 1 {
		t.Fatalf("expected 1 fetch request, got %d", len(fetcher.requests))
	}
	if !fetcher.requests[0].StartTime.Equal(start.Add(2 * time.Minute)) {
		t.Fatalf("expected resume from persisted next open time, got %s", fetcher.requests[0].StartTime)
	}
	if result.CandlesInserted != 3 {
		t.Fatalf("expected cumulative inserted count 3, got %d", result.CandlesInserted)
	}
}

func TestBackfillMarksJobFailedWhenStoreFails(t *testing.T) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	jobs := &fakeBackfillJobStore{}
	storeErr := errors.New("database unavailable")
	service := NewBackfillService(
		&fakeHistoricalFetcher{pages: [][]Candle{{
			{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", OpenTime: start, CloseTime: start.Add(time.Minute), Close: "100", IsClosed: true},
		}}},
		&fakeCandleStore{err: storeErr},
		WithBackfillJobs(jobs),
	)

	_, err := service.Start(context.Background(), BackfillRequest{
		Symbol: "BTCUSDT", Interval: "1m", From: start, To: start.Add(time.Minute), Limit: 1000,
	})
	if err == nil {
		t.Fatal("expected backfill error")
	}
	if jobs.saved.Status != BackfillStatusFailed {
		t.Fatalf("expected failed job, got %q", jobs.saved.Status)
	}
	if jobs.saved.LastError == "" {
		t.Fatal("expected failed job to include last error")
	}
}

func TestBackfillRetriesTemporaryFetchFailure(t *testing.T) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	fetcher := &fakeHistoricalFetcher{
		errs: []error{errors.New("binance unavailable")},
		pages: [][]Candle{{
			{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", OpenTime: start, CloseTime: start.Add(time.Minute), Close: "100", IsClosed: true},
		}},
	}
	service := NewBackfillService(fetcher, &fakeCandleStore{}, WithBackfillRetryDelay(0))

	result, err := service.Start(context.Background(), BackfillRequest{
		Symbol: "BTCUSDT", Interval: "1m", From: start, To: start.Add(time.Minute), Limit: 1000,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if result.CandlesInserted != 1 {
		t.Fatalf("expected 1 inserted candle after retry, got %d", result.CandlesInserted)
	}
	if len(fetcher.requests) != 2 {
		t.Fatalf("expected original fetch plus retry, got %d requests", len(fetcher.requests))
	}
}

type fakeHistoricalFetcher struct {
	request  HistoricalKlineRequest
	requests []HistoricalKlineRequest
	candles  []Candle
	pages    [][]Candle
	errs     []error
}

func (f *fakeHistoricalFetcher) FetchKlines(ctx context.Context, request HistoricalKlineRequest) ([]Candle, error) {
	f.request = request
	f.requests = append(f.requests, request)
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		return nil, err
	}
	if len(f.pages) > 0 {
		page := f.pages[0]
		f.pages = f.pages[1:]
		return page, nil
	}
	return f.candles, nil
}

type fakeCandleStore struct {
	upserted []Candle
	batches  [][]Candle
	err      error
}

func (f *fakeCandleStore) Upsert(ctx context.Context, candle Candle) error {
	if f.err != nil {
		return f.err
	}
	f.upserted = append(f.upserted, candle)
	return nil
}

func (f *fakeCandleStore) UpsertBatch(ctx context.Context, candles []Candle) error {
	if f.err != nil {
		return f.err
	}
	copied := append([]Candle(nil), candles...)
	f.batches = append(f.batches, copied)
	f.upserted = append(f.upserted, candles...)
	return nil
}

type fakeBackfillJobStore struct {
	saved BackfillJob
}

func (f *fakeBackfillJobStore) Create(ctx context.Context, job BackfillJob) (BackfillJob, error) {
	if job.ID == "" {
		job.ID = "job-1"
	}
	f.saved = job
	return job, nil
}

func (f *fakeBackfillJobStore) Get(ctx context.Context, id string) (BackfillJob, error) {
	return f.saved, nil
}

func (f *fakeBackfillJobStore) List(ctx context.Context, query BackfillJobQuery) ([]BackfillJob, error) {
	return []BackfillJob{f.saved}, nil
}

func (f *fakeBackfillJobStore) Save(ctx context.Context, job BackfillJob) (BackfillJob, error) {
	f.saved = job
	return job, nil
}
