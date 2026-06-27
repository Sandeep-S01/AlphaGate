package orchestration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Run struct {
	Key         string     `json:"key"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Error       string     `json:"error,omitempty"`
}

type RunQuery struct {
	Status string
	Limit  int
}

type IdempotencyRepository struct {
	db QueryRower
}

func NewIdempotencyRepository(db QueryRower) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

func (r *IdempotencyRepository) Begin(ctx context.Context, key string) (bool, error) {
	var started bool
	err := r.db.QueryRow(ctx, BuildBeginSQL(), key, time.Now().UTC()).Scan(&started)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("begin pipeline run: %w", err)
	}
	return started, nil
}

func (r *IdempotencyRepository) Complete(ctx context.Context, key string) error {
	var completed bool
	err := r.db.QueryRow(ctx, `
UPDATE pipeline_runs
SET status = 'completed',
    completed_at = $2,
    updated_at = $2,
    error = NULL
WHERE key = $1
RETURNING true`, key, time.Now().UTC()).Scan(&completed)
	if err != nil {
		return fmt.Errorf("complete pipeline run: %w", err)
	}
	return nil
}

func (r *IdempotencyRepository) Fail(ctx context.Context, key string, reason string) error {
	var failed bool
	err := r.db.QueryRow(ctx, `
UPDATE pipeline_runs
SET status = 'failed',
    error = $2,
    updated_at = $3
WHERE key = $1
RETURNING true`, key, reason, time.Now().UTC()).Scan(&failed)
	if err != nil {
		return fmt.Errorf("fail pipeline run: %w", err)
	}
	return nil
}

func BuildBeginSQL() string {
	return `
INSERT INTO pipeline_runs (key, status, started_at, updated_at)
VALUES ($1, 'processing', $2, $2)
ON CONFLICT (key)
DO UPDATE SET
    status = 'processing',
    started_at = EXCLUDED.started_at,
    updated_at = EXCLUDED.updated_at,
    completed_at = NULL,
    error = NULL
WHERE pipeline_runs.status = 'failed'
RETURNING true`
}

func (r *IdempotencyRepository) ListRuns(ctx context.Context, query RunQuery) ([]Run, error) {
	sql, args := BuildListRunsSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query pipeline runs: %w", err)
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var run Run
		if err := rows.Scan(&run.Key, &run.Status, &run.StartedAt, &run.CompletedAt, &run.UpdatedAt, &run.Error); err != nil {
			return nil, fmt.Errorf("scan pipeline run: %w", err)
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func BuildListRunsSQL(query RunQuery) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	sql := `
SELECT key, status, started_at, completed_at, updated_at, COALESCE(error, '')
FROM pipeline_runs
WHERE 1 = 1`
	args := []any{}
	if query.Status != "" {
		args = append(args, query.Status)
		sql += fmt.Sprintf(" AND status = $%d", len(args))
	}
	args = append(args, limit)
	sql += fmt.Sprintf(" ORDER BY updated_at DESC LIMIT $%d", len(args))
	return sql, args
}
