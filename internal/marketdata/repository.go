package marketdata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type CandleExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type CandleRepository struct {
	db CandleExecutor
}

const (
	postgresMaxParameters = 65535
	candleUpsertArgCount  = 13
)

type CandleQuery struct {
	Symbol   string
	Interval string
	From     time.Time
	To       time.Time
	Limit    int
	Desc     bool
}

type Coverage struct {
	Symbol    string     `json:"symbol"`
	Interval  string     `json:"interval"`
	Count     int64      `json:"count"`
	FirstTime *time.Time `json:"first_time,omitempty"`
	LastTime  *time.Time `json:"last_time,omitempty"`
}

func NewCandleRepository(db CandleExecutor) *CandleRepository {
	return &CandleRepository{db: db}
}

func (r *CandleRepository) Upsert(ctx context.Context, candle Candle) error {
	query, args := BuildUpsertCandleSQL(candle)
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert candle: %w", err)
	}
	return nil
}

func (r *CandleRepository) UpsertBatch(ctx context.Context, candles []Candle) error {
	if len(candles) == 0 {
		return nil
	}
	maxCandlesPerBatch := postgresMaxParameters / candleUpsertArgCount
	for start := 0; start < len(candles); start += maxCandlesPerBatch {
		end := start + maxCandlesPerBatch
		if end > len(candles) {
			end = len(candles)
		}
		query, args := BuildUpsertCandlesBatchSQL(candles[start:end])
		if _, err := r.db.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert candle batch: %w", err)
		}
	}
	return nil
}

func (r *CandleRepository) DeleteRange(ctx context.Context, query CandleQuery) error {
	sqlText, args := BuildDeleteCandlesRangeSQL(query)
	if _, err := r.db.Exec(ctx, sqlText, args...); err != nil {
		return fmt.Errorf("delete candle range: %w", err)
	}
	return nil
}

func (r *CandleRepository) List(ctx context.Context, query CandleQuery) ([]Candle, error) {
	sql, args := BuildListCandlesSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query candles: %w", err)
	}
	defer rows.Close()

	var candles []Candle
	for rows.Next() {
		var candle Candle
		if err := rows.Scan(
			&candle.Exchange,
			&candle.Symbol,
			&candle.Interval,
			&candle.OpenTime,
			&candle.CloseTime,
			&candle.Open,
			&candle.High,
			&candle.Low,
			&candle.Close,
			&candle.Volume,
			&candle.QuoteVolume,
			&candle.TradeCount,
			&candle.IsClosed,
		); err != nil {
			return nil, fmt.Errorf("scan candle: %w", err)
		}
		candles = append(candles, candle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate candles: %w", err)
	}
	return candles, nil
}

func (r *CandleRepository) Coverage(ctx context.Context, query CandleQuery) (Coverage, error) {
	sqlText, args := BuildCoverageSQL(query)
	rows, err := r.db.Query(ctx, sqlText, args...)
	if err != nil {
		return Coverage{}, fmt.Errorf("query candle coverage: %w", err)
	}
	defer rows.Close()

	coverage := Coverage{Symbol: query.Symbol, Interval: query.Interval}
	if rows.Next() {
		var first sql.NullTime
		var last sql.NullTime
		if err := rows.Scan(&coverage.Count, &first, &last); err != nil {
			return Coverage{}, fmt.Errorf("scan candle coverage: %w", err)
		}
		if first.Valid {
			value := first.Time
			coverage.FirstTime = &value
		}
		if last.Valid {
			value := last.Time
			coverage.LastTime = &value
		}
	}
	if err := rows.Err(); err != nil {
		return Coverage{}, fmt.Errorf("iterate candle coverage: %w", err)
	}
	return coverage, nil
}

func (r *CandleRepository) OpenTimes(ctx context.Context, query CandleQuery) ([]time.Time, error) {
	sql, args := BuildListOpenTimesSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query candle open times: %w", err)
	}
	defer rows.Close()

	var openTimes []time.Time
	for rows.Next() {
		var openTime time.Time
		if err := rows.Scan(&openTime); err != nil {
			return nil, fmt.Errorf("scan candle open time: %w", err)
		}
		openTimes = append(openTimes, openTime)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate candle open times: %w", err)
	}
	return openTimes, nil
}

func BuildUpsertCandleSQL(candle Candle) (string, []any) {
	return `
INSERT INTO candles (
    exchange,
    symbol,
    interval,
    open_time,
    close_time,
    open,
    high,
    low,
    close,
    volume,
    quote_volume,
    trade_count,
    is_closed
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (exchange, symbol, interval, open_time)
DO UPDATE SET
    close_time = EXCLUDED.close_time,
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume,
    quote_volume = EXCLUDED.quote_volume,
    trade_count = EXCLUDED.trade_count,
    is_closed = EXCLUDED.is_closed`, []any{
			candle.Exchange,
			candle.Symbol,
			candle.Interval,
			candle.OpenTime,
			candle.CloseTime,
			candle.Open,
			candle.High,
			candle.Low,
			candle.Close,
			candle.Volume,
			candle.QuoteVolume,
			candle.TradeCount,
			candle.IsClosed,
		}
}

func BuildUpsertCandlesBatchSQL(candles []Candle) (string, []any) {
	var builder strings.Builder
	builder.WriteString(`
INSERT INTO candles (
    exchange,
    symbol,
    interval,
    open_time,
    close_time,
    open,
    high,
    low,
    close,
    volume,
    quote_volume,
    trade_count,
    is_closed
) VALUES `)

	args := make([]any, 0, len(candles)*13)
	for index, candle := range candles {
		if index > 0 {
			builder.WriteString(", ")
		}
		base := len(args)
		builder.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9, base+10, base+11, base+12, base+13))
		args = append(args,
			candle.Exchange,
			candle.Symbol,
			candle.Interval,
			candle.OpenTime,
			candle.CloseTime,
			candle.Open,
			candle.High,
			candle.Low,
			candle.Close,
			candle.Volume,
			candle.QuoteVolume,
			candle.TradeCount,
			candle.IsClosed,
		)
	}
	builder.WriteString(`
ON CONFLICT (exchange, symbol, interval, open_time)
DO UPDATE SET
    close_time = EXCLUDED.close_time,
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume,
    quote_volume = EXCLUDED.quote_volume,
    trade_count = EXCLUDED.trade_count,
    is_closed = EXCLUDED.is_closed`)
	return builder.String(), args
}

func BuildListCandlesSQL(query CandleQuery) (string, []any) {
	limit := query.Limit
	if limit <= 0 {
		limit = 500
	}

	var builder strings.Builder
	builder.WriteString(`
SELECT exchange, symbol, interval, open_time, close_time, open, high, low, close, volume, quote_volume, trade_count, is_closed
FROM candles
WHERE symbol = $1 AND interval = $2`)

	args := []any{query.Symbol, query.Interval}
	if !query.From.IsZero() {
		args = append(args, query.From)
		builder.WriteString(fmt.Sprintf(" AND open_time >= $%d", len(args)))
	}
	if !query.To.IsZero() {
		args = append(args, query.To)
		builder.WriteString(fmt.Sprintf(" AND open_time < $%d", len(args)))
	}
	args = append(args, limit)
	if query.Desc {
		builder.WriteString(fmt.Sprintf(" ORDER BY open_time DESC LIMIT $%d", len(args)))
	} else {
		builder.WriteString(fmt.Sprintf(" ORDER BY open_time ASC LIMIT $%d", len(args)))
	}

	return builder.String(), args
}

func BuildDeleteCandlesRangeSQL(query CandleQuery) (string, []any) {
	return `
DELETE FROM candles
WHERE symbol = $1
  AND interval = $2
  AND open_time >= $3
  AND open_time < $4`, []any{
			query.Symbol,
			query.Interval,
			query.From,
			query.To,
		}
}

func BuildListOpenTimesSQL(query CandleQuery) (string, []any) {
	limit := query.Limit
	if limit <= 0 {
		limit = 100000
	}

	var builder strings.Builder
	builder.WriteString(`
SELECT open_time
FROM candles
WHERE symbol = $1 AND interval = $2`)

	args := []any{query.Symbol, query.Interval}
	if !query.From.IsZero() {
		args = append(args, query.From)
		builder.WriteString(fmt.Sprintf(" AND open_time >= $%d", len(args)))
	}
	if !query.To.IsZero() {
		args = append(args, query.To)
		builder.WriteString(fmt.Sprintf(" AND open_time < $%d", len(args)))
	}
	args = append(args, limit)
	if query.Desc {
		builder.WriteString(fmt.Sprintf(" ORDER BY open_time DESC LIMIT $%d", len(args)))
	} else {
		builder.WriteString(fmt.Sprintf(" ORDER BY open_time ASC LIMIT $%d", len(args)))
	}

	return builder.String(), args
}

func BuildCoverageSQL(query CandleQuery) (string, []any) {
	var builder strings.Builder
	builder.WriteString(`
SELECT COUNT(*), MIN(open_time), MAX(open_time)
FROM candles
WHERE symbol = $1 AND interval = $2`)

	args := []any{query.Symbol, query.Interval}
	if !query.From.IsZero() {
		args = append(args, query.From)
		builder.WriteString(fmt.Sprintf(" AND open_time >= $%d", len(args)))
	}
	if !query.To.IsZero() {
		args = append(args, query.To)
		builder.WriteString(fmt.Sprintf(" AND open_time < $%d", len(args)))
	}
	return builder.String(), args
}
