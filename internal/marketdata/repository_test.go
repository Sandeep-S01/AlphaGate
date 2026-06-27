package marketdata

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestBuildUpsertCandleSQLUsesConflictKey(t *testing.T) {
	candle := Candle{
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		OpenTime:  time.Unix(10, 0).UTC(),
		CloseTime: time.Unix(20, 0).UTC(),
		Open:      "100",
		High:      "110",
		Low:       "90",
		Close:     "105",
		Volume:    "12.5",
		IsClosed:  true,
	}

	query, args := BuildUpsertCandleSQL(candle)

	if !strings.Contains(query, "ON CONFLICT (exchange, symbol, interval, open_time)") {
		t.Fatalf("expected upsert conflict key in query: %s", query)
	}
	if len(args) != 13 {
		t.Fatalf("expected 13 args, got %d", len(args))
	}
	if args[0] != "binance" || args[1] != "BTCUSDT" || args[2] != "1m" {
		t.Fatalf("unexpected leading args: %+v", args[:3])
	}
}

func TestBuildListCandlesSQLUsesTimeRangeAndLimit(t *testing.T) {
	from := time.Unix(10, 0).UTC()
	to := time.Unix(20, 0).UTC()

	query, args := BuildListCandlesSQL(CandleQuery{
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     from,
		To:       to,
		Limit:    100,
	})

	if !strings.Contains(query, "WHERE symbol = $1 AND interval = $2") {
		t.Fatalf("expected symbol and interval filters: %s", query)
	}
	if !strings.Contains(query, "open_time >= $3") || !strings.Contains(query, "open_time < $4") {
		t.Fatalf("expected time range filters: %s", query)
	}
	if !strings.Contains(query, "LIMIT $5") {
		t.Fatalf("expected limit placeholder: %s", query)
	}
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d", len(args))
	}
}

func TestBuildListCandlesSQLCanOrderNewestFirst(t *testing.T) {
	query, _ := BuildListCandlesSQL(CandleQuery{
		Symbol:   "BTCUSDT",
		Interval: "1m",
		Limit:    1,
		Desc:     true,
	})

	if !strings.Contains(query, "ORDER BY open_time DESC") {
		t.Fatalf("expected descending order: %s", query)
	}
}

func TestBuildCoverageSQLUsesMarketAndTimeRange(t *testing.T) {
	from := time.Unix(10, 0).UTC()
	to := time.Unix(20, 0).UTC()

	query, args := BuildCoverageSQL(CandleQuery{
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     from,
		To:       to,
	})

	if !strings.Contains(query, "COUNT(*)") || !strings.Contains(query, "MIN(open_time)") || !strings.Contains(query, "MAX(open_time)") {
		t.Fatalf("expected coverage aggregates, got %s", query)
	}
	if !strings.Contains(query, "symbol = $1 AND interval = $2") {
		t.Fatalf("expected market filters, got %s", query)
	}
	if !strings.Contains(query, "open_time >= $3") || !strings.Contains(query, "open_time < $4") {
		t.Fatalf("expected time range filters, got %s", query)
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}
}

func TestCandleRepositoryUpsertBatchChunksBelowPostgresParameterLimit(t *testing.T) {
	db := &fakeCandleExecutor{}
	repository := NewCandleRepository(db)
	candles := make([]Candle, 6000)
	for index := range candles {
		openTime := time.Unix(int64(index*60), 0).UTC()
		candles[index] = Candle{
			Exchange:    "binance",
			Symbol:      "BTCUSDT",
			Interval:    "5m",
			OpenTime:    openTime,
			CloseTime:   openTime.Add(5 * time.Minute),
			Open:        "100",
			High:        "101",
			Low:         "99",
			Close:       "100",
			Volume:      "1",
			QuoteVolume: "100",
			TradeCount:  10,
			IsClosed:    true,
		}
	}

	if err := repository.UpsertBatch(context.Background(), candles); err != nil {
		t.Fatalf("UpsertBatch returned error: %v", err)
	}

	if len(db.execArgCounts) != 2 {
		t.Fatalf("expected 2 chunked exec calls, got %d", len(db.execArgCounts))
	}
	for _, count := range db.execArgCounts {
		if count > postgresMaxParameters {
			t.Fatalf("expected chunk under parameter limit, got %d", count)
		}
	}
}

func TestCandleRepositoryWrapsPostgresDisconnectOnWrites(t *testing.T) {
	dbErr := errors.New("postgres disconnect")
	repository := NewCandleRepository(&fakeCandleExecutor{execErr: dbErr})
	candle := Candle{
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		OpenTime:  time.Unix(10, 0).UTC(),
		CloseTime: time.Unix(70, 0).UTC(),
		Open:      "100",
		High:      "101",
		Low:       "99",
		Close:     "100",
		IsClosed:  true,
	}

	if err := repository.Upsert(context.Background(), candle); err == nil || !strings.Contains(err.Error(), "upsert candle") || !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped upsert disconnect, got %v", err)
	}
	if err := repository.UpsertBatch(context.Background(), []Candle{candle}); err == nil || !strings.Contains(err.Error(), "upsert candle batch") || !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped batch disconnect, got %v", err)
	}
	if err := repository.DeleteRange(context.Background(), CandleQuery{Symbol: "BTCUSDT", Interval: "1m"}); err == nil || !strings.Contains(err.Error(), "delete candle range") || !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped delete disconnect, got %v", err)
	}
}

func TestCandleRepositoryWrapsPostgresDisconnectOnReads(t *testing.T) {
	dbErr := errors.New("postgres disconnect")
	repository := NewCandleRepository(&fakeCandleExecutor{queryErr: dbErr})

	if _, err := repository.List(context.Background(), CandleQuery{Symbol: "BTCUSDT", Interval: "1m"}); err == nil || !strings.Contains(err.Error(), "query candles") || !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped list disconnect, got %v", err)
	}
	if _, err := repository.Coverage(context.Background(), CandleQuery{Symbol: "BTCUSDT", Interval: "1m"}); err == nil || !strings.Contains(err.Error(), "query candle coverage") || !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped coverage disconnect, got %v", err)
	}
	if _, err := repository.OpenTimes(context.Background(), CandleQuery{Symbol: "BTCUSDT", Interval: "1m"}); err == nil || !strings.Contains(err.Error(), "query candle open times") || !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped open times disconnect, got %v", err)
	}
}

func TestBuildDeleteCandlesRangeSQLUsesMarketAndTimeRange(t *testing.T) {
	from := time.Unix(10, 0).UTC()
	to := time.Unix(20, 0).UTC()

	query, args := BuildDeleteCandlesRangeSQL(CandleQuery{
		Symbol:   "BTCUSDT",
		Interval: "15m",
		From:     from,
		To:       to,
	})

	if !strings.Contains(query, "DELETE FROM candles") {
		t.Fatalf("expected delete query, got %s", query)
	}
	if !strings.Contains(query, "symbol = $1") || !strings.Contains(query, "interval = $2") {
		t.Fatalf("expected market filters, got %s", query)
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}
}

type fakeCandleExecutor struct {
	execArgCounts []int
	execErr       error
	queryErr      error
}

func (f *fakeCandleExecutor) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	f.execArgCounts = append(f.execArgCounts, len(arguments))
	if f.execErr != nil {
		return pgconn.CommandTag{}, f.execErr
	}
	return pgconn.CommandTag{}, nil
}

func (f *fakeCandleExecutor) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return nil, nil
}
