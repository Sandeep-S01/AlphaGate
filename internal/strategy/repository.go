package strategy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type SignalRepository struct {
	db QueryRower
}

type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type SignalQuery struct {
	Symbol string
	From   time.Time
	To     time.Time
	Limit  int
}

func NewSignalRepository(db QueryRower) *SignalRepository {
	return &SignalRepository{db: db}
}

func (r *SignalRepository) Save(ctx context.Context, signal Signal) (string, error) {
	query, args := BuildInsertSignalSQL(signal)
	var id string
	if err := r.db.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		return "", fmt.Errorf("save signal: %w", err)
	}
	return id, nil
}

func (r *SignalRepository) Latest(ctx context.Context, symbol string) (Signal, error) {
	query := `
SELECT id, strategy_name, version, symbol, interval, side, strength, reason, generated_at
FROM strategy_signals
WHERE symbol = $1
ORDER BY generated_at DESC
LIMIT 1`

	var signal Signal
	var side string
	if err := r.db.QueryRow(ctx, query, symbol).Scan(
		&signal.ID,
		&signal.StrategyName,
		&signal.Version,
		&signal.Symbol,
		&signal.Interval,
		&side,
		&signal.Strength,
		&signal.Reason,
		&signal.GeneratedAt,
	); err != nil {
		return Signal{}, fmt.Errorf("query latest signal: %w", err)
	}
	signal.Side = Side(side)
	return signal, nil
}

func (r *SignalRepository) List(ctx context.Context, query SignalQuery) ([]Signal, error) {
	sql, args := BuildListSignalsSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query signals: %w", err)
	}
	defer rows.Close()

	var signals []Signal
	for rows.Next() {
		var signal Signal
		var side string
		if err := rows.Scan(
			&signal.ID,
			&signal.StrategyName,
			&signal.Version,
			&signal.Symbol,
			&signal.Interval,
			&side,
			&signal.Strength,
			&signal.Reason,
			&signal.GeneratedAt,
		); err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}
		signal.Side = Side(side)
		signals = append(signals, signal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating signals: %w", err)
	}
	return signals, nil
}

func BuildInsertSignalSQL(signal Signal) (string, []any) {
	return `
INSERT INTO strategy_signals (
    strategy_name,
    version,
    symbol,
    interval,
    side,
    strength,
    reason,
    generated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id`, []any{
			signal.StrategyName,
			signal.Version,
			signal.Symbol,
			signal.Interval,
			string(signal.Side),
			signal.Strength,
			signal.Reason,
			signal.GeneratedAt,
		}
}

func BuildListSignalsSQL(query SignalQuery) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var builder strings.Builder
	builder.WriteString(`
SELECT id, strategy_name, version, symbol, interval, side, strength, reason, generated_at
FROM strategy_signals
WHERE 1 = 1`)
	args := []any{}
	if query.Symbol != "" {
		args = append(args, query.Symbol)
		builder.WriteString(fmt.Sprintf(" AND symbol = $%d", len(args)))
	}
	if !query.From.IsZero() {
		args = append(args, query.From)
		builder.WriteString(fmt.Sprintf(" AND generated_at >= $%d", len(args)))
	}
	if !query.To.IsZero() {
		args = append(args, query.To)
		builder.WriteString(fmt.Sprintf(" AND generated_at < $%d", len(args)))
	}
	args = append(args, limit)
	builder.WriteString(fmt.Sprintf(" ORDER BY generated_at DESC LIMIT $%d", len(args)))
	return builder.String(), args
}
