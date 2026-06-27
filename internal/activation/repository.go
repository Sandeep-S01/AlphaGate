package activation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Query struct {
	StrategyName string
	Symbol       string
	Interval     string
	Limit        int
}

type Repository struct {
	db QueryRower
}

func NewRepository(db QueryRower) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, record Record) (Record, error) {
	query, args, err := BuildInsertActivationSQL(record)
	if err != nil {
		return Record{}, fmt.Errorf("build insert activation SQL: %w", err)
	}
	var settingsJSON string
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&record.ID,
		&record.ComparisonID,
		&record.ComparisonResultID,
		&record.StrategyName,
		&record.Actor,
		&settingsJSON,
		&record.ComparisonReturn,
		&record.ComparisonDrawdown,
		&record.ComparisonWinRate,
		&record.ComparisonTotalTrades,
		&record.CreatedAt,
	); err != nil {
		return Record{}, fmt.Errorf("insert strategy activation: %w", err)
	}
	if err := json.Unmarshal([]byte(settingsJSON), &record.ActivatedSettings); err != nil {
		return Record{}, fmt.Errorf("decode activated settings: %w", err)
	}
	return record, nil
}

func (r *Repository) List(ctx context.Context, query Query) ([]Record, error) {
	sql, args := BuildListActivationsSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query strategy activations: %w", err)
	}
	defer rows.Close()
	records := []Record{}
	for rows.Next() {
		record, err := scanActivation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan activation row: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating activation rows: %w", err)
	}
	return records, nil
}

func (r *Repository) SaveLifecycle(ctx context.Context, record LifecycleRecord) (LifecycleRecord, error) {
	sql, args := BuildUpsertLifecycleSQL(record)
	if err := r.db.QueryRow(ctx, sql, args...).Scan(
		&record.ID,
		&record.StrategyName,
		&record.Symbol,
		&record.Interval,
		&record.State,
		&record.Reason,
		&record.UpdatedBy,
		&record.UpdatedAt,
	); err != nil {
		return LifecycleRecord{}, fmt.Errorf("upsert strategy lifecycle: %w", err)
	}
	return record, nil
}

func (r *Repository) ListLifecycles(ctx context.Context, query Query) ([]LifecycleRecord, error) {
	sql, args := BuildListLifecycleSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query strategy lifecycle: %w", err)
	}
	defer rows.Close()
	records := []LifecycleRecord{}
	for rows.Next() {
		var record LifecycleRecord
		if err := rows.Scan(&record.ID, &record.StrategyName, &record.Symbol, &record.Interval, &record.State, &record.Reason, &record.UpdatedBy, &record.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan strategy lifecycle row: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating strategy lifecycle rows: %w", err)
	}
	return records, nil
}

func (r *Repository) AdvanceLifecycle(ctx context.Context, id string, state LifecycleState, reason string, actor string) (LifecycleRecord, error) {
	var record LifecycleRecord
	if err := r.db.QueryRow(ctx, `
UPDATE strategy_lifecycle
SET state = $2,
    reason = $3,
    updated_by = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING id, strategy_name, symbol, interval, state, reason, updated_by, updated_at`, id, string(state), reason, actor).Scan(
		&record.ID,
		&record.StrategyName,
		&record.Symbol,
		&record.Interval,
		&record.State,
		&record.Reason,
		&record.UpdatedBy,
		&record.UpdatedAt,
	); err != nil {
		return LifecycleRecord{}, fmt.Errorf("advance strategy lifecycle: %w", err)
	}
	return record, nil
}

func BuildInsertActivationSQL(record Record) (string, []any, error) {
	settingsJSON, err := json.Marshal(record.ActivatedSettings)
	if err != nil {
		return "", nil, fmt.Errorf("encode activated settings: %w", err)
	}
	return `
INSERT INTO strategy_activations (
    comparison_id,
    comparison_result_id,
    strategy_name,
    actor,
    activated_settings,
    comparison_return,
    comparison_drawdown,
    comparison_win_rate,
    comparison_total_trades
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, comparison_id, comparison_result_id, strategy_name, actor, activated_settings::text,
          comparison_return, comparison_drawdown, comparison_win_rate, comparison_total_trades, created_at`, []any{
			record.ComparisonID,
			record.ComparisonResultID,
			record.StrategyName,
			record.Actor,
			string(settingsJSON),
			record.ComparisonReturn,
			record.ComparisonDrawdown,
			record.ComparisonWinRate,
			record.ComparisonTotalTrades,
		}, nil
}

func BuildListActivationsSQL(query Query) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var builder strings.Builder
	builder.WriteString(activationSelectSQL())
	builder.WriteString(" WHERE 1 = 1")
	args := []any{}
	if query.StrategyName != "" {
		args = append(args, query.StrategyName)
		builder.WriteString(fmt.Sprintf(" AND strategy_name = $%d", len(args)))
	}
	args = append(args, limit)
	builder.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args)))
	return builder.String(), args
}

func BuildUpsertLifecycleSQL(record LifecycleRecord) (string, []any) {
	return `
INSERT INTO strategy_lifecycle (
    strategy_name,
    symbol,
    interval,
    state,
    reason,
    updated_by,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (strategy_name, symbol, interval)
DO UPDATE SET
    state = EXCLUDED.state,
    reason = EXCLUDED.reason,
    updated_by = EXCLUDED.updated_by,
    updated_at = EXCLUDED.updated_at
RETURNING id, strategy_name, symbol, interval, state, reason, updated_by, updated_at`, []any{
			record.StrategyName,
			record.Symbol,
			record.Interval,
			string(record.State),
			record.Reason,
			record.UpdatedBy,
			record.UpdatedAt,
		}
}

func BuildListLifecycleSQL(query Query) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var builder strings.Builder
	builder.WriteString(`
SELECT id, strategy_name, symbol, interval, state, reason, updated_by, updated_at
FROM strategy_lifecycle
WHERE 1 = 1`)
	args := []any{}
	if query.StrategyName != "" {
		args = append(args, query.StrategyName)
		builder.WriteString(fmt.Sprintf(" AND strategy_name = $%d", len(args)))
	}
	if query.Symbol != "" {
		args = append(args, query.Symbol)
		builder.WriteString(fmt.Sprintf(" AND symbol = $%d", len(args)))
	}
	if query.Interval != "" {
		args = append(args, query.Interval)
		builder.WriteString(fmt.Sprintf(" AND interval = $%d", len(args)))
	}
	args = append(args, limit)
	builder.WriteString(fmt.Sprintf(" ORDER BY updated_at DESC LIMIT $%d", len(args)))
	return builder.String(), args
}

type scanner interface {
	Scan(dest ...any) error
}

func scanActivation(row scanner) (Record, error) {
	var record Record
	var settingsJSON string
	if err := row.Scan(
		&record.ID,
		&record.ComparisonID,
		&record.ComparisonResultID,
		&record.StrategyName,
		&record.Actor,
		&settingsJSON,
		&record.ComparisonReturn,
		&record.ComparisonDrawdown,
		&record.ComparisonWinRate,
		&record.ComparisonTotalTrades,
		&record.CreatedAt,
	); err != nil {
		return Record{}, fmt.Errorf("scan strategy activation row: %w", err)
	}
	if err := json.Unmarshal([]byte(settingsJSON), &record.ActivatedSettings); err != nil {
		return Record{}, fmt.Errorf("decode activated settings: %w", err)
	}
	return record, nil
}

func activationSelectSQL() string {
	return `
SELECT id, comparison_id, comparison_result_id, strategy_name, actor, activated_settings::text,
       comparison_return, comparison_drawdown, comparison_win_rate, comparison_total_trades, created_at
FROM strategy_activations`
}
