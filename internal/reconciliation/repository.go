package reconciliation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Repository struct {
	db QueryRower
}

func NewRepository(db QueryRower) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, run Run) (Run, error) {
	mismatchesJSON, _ := json.Marshal(run.Mismatches)
	var decoded string
	if err := r.db.QueryRow(ctx, BuildInsertRunSQL(), string(run.Status), string(mismatchesJSON), run.CreatedAt).Scan(
		&run.ID,
		&run.Status,
		&decoded,
		&run.CreatedAt,
	); err != nil {
		return Run{}, fmt.Errorf("insert reconciliation run: %w", err)
	}
	_ = json.Unmarshal([]byte(decoded), &run.Mismatches)
	return run, nil
}

func (r *Repository) List(ctx context.Context, query Query) ([]Run, error) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
SELECT id, status, mismatches_json::text, created_at
FROM reconciliation_runs
ORDER BY created_at DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("query reconciliation runs: %w", err)
	}
	defer rows.Close()
	runs := []Run{}
	for rows.Next() {
		var run Run
		var mismatchesJSON string
		if err := rows.Scan(&run.ID, &run.Status, &mismatchesJSON, &run.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan reconciliation run: %w", err)
		}
		_ = json.Unmarshal([]byte(mismatchesJSON), &run.Mismatches)
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id string) (Run, error) {
	var run Run
	var mismatchesJSON string
	if err := r.db.QueryRow(ctx, `
SELECT id, status, mismatches_json::text, created_at
FROM reconciliation_runs
WHERE id = $1`, id).Scan(&run.ID, &run.Status, &mismatchesJSON, &run.CreatedAt); err != nil {
		return Run{}, fmt.Errorf("query reconciliation run: %w", err)
	}
	_ = json.Unmarshal([]byte(mismatchesJSON), &run.Mismatches)
	return run, nil
}

func BuildInsertRunSQL() string {
	return `
INSERT INTO reconciliation_runs (
    status,
    mismatches_json,
    created_at
) VALUES ($1, $2::jsonb, $3)
RETURNING id, status, mismatches_json::text, created_at`
}
