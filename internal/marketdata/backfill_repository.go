package marketdata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type BackfillQueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type BackfillRepository struct {
	db BackfillQueryRower
}

func NewBackfillRepository(db BackfillQueryRower) *BackfillRepository {
	return &BackfillRepository{db: db}
}

func (r *BackfillRepository) Create(ctx context.Context, job BackfillJob) (BackfillJob, error) {
	query, args := BuildInsertBackfillJobSQL(job)
	job, err := scanBackfillJob(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		return BackfillJob{}, fmt.Errorf("insert backfill job: %w", err)
	}
	return job, nil
}

func (r *BackfillRepository) Get(ctx context.Context, id string) (BackfillJob, error) {
	query := backfillJobSelectSQL() + " WHERE id = $1"
	job, err := scanBackfillJob(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return BackfillJob{}, fmt.Errorf("get backfill job: %w", err)
	}
	return job, nil
}

func (r *BackfillRepository) List(ctx context.Context, query BackfillJobQuery) ([]BackfillJob, error) {
	sqlText, args := BuildListBackfillJobsSQL(query)
	rows, err := r.db.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("query backfill jobs: %w", err)
	}
	defer rows.Close()

	jobs := []BackfillJob{}
	for rows.Next() {
		job, err := scanBackfillJob(rows)
		if err != nil {
			return nil, fmt.Errorf("scan backfill job: %w", err)
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *BackfillRepository) Save(ctx context.Context, job BackfillJob) (BackfillJob, error) {
	query, args := BuildSaveBackfillJobSQL(job)
	job, err := scanBackfillJob(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		return BackfillJob{}, fmt.Errorf("save backfill job: %w", err)
	}
	return job, nil
}

func BuildInsertBackfillJobSQL(job BackfillJob) (string, []any) {
	return `
INSERT INTO market_data_backfill_jobs (
    symbol,
    base_interval,
    from_time,
    to_time,
    next_open_time,
    status
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING ` + backfillJobColumns(), []any{
			job.Symbol,
			job.BaseInterval,
			job.From,
			job.To,
			job.NextOpenTime,
			string(job.Status),
		}
}

func BuildSaveBackfillJobSQL(job BackfillJob) (string, []any) {
	return `
UPDATE market_data_backfill_jobs
SET next_open_time = $2,
    status = $3,
    candles_inserted = $4,
    last_error = $5,
    started_at = $6,
    completed_at = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING ` + backfillJobColumns(), []any{
			job.ID,
			job.NextOpenTime,
			string(job.Status),
			job.CandlesInserted,
			nullString(job.LastError),
			job.StartedAt,
			job.CompletedAt,
		}
}

func BuildListBackfillJobsSQL(query BackfillJobQuery) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var builder strings.Builder
	builder.WriteString(backfillJobSelectSQL())
	builder.WriteString(" WHERE 1 = 1")
	args := []any{}
	if query.Symbol != "" {
		args = append(args, strings.ToUpper(query.Symbol))
		builder.WriteString(fmt.Sprintf(" AND symbol = $%d", len(args)))
	}
	args = append(args, limit)
	builder.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args)))
	return builder.String(), args
}

func backfillJobSelectSQL() string {
	return "SELECT " + backfillJobColumns() + " FROM market_data_backfill_jobs"
}

func backfillJobColumns() string {
	return "id, symbol, base_interval, from_time, to_time, next_open_time, status, candles_inserted, last_error, started_at, completed_at, created_at, updated_at"
}

type backfillJobScanner interface {
	Scan(dest ...any) error
}

func scanBackfillJob(row backfillJobScanner) (BackfillJob, error) {
	var job BackfillJob
	var status string
	var lastError sql.NullString
	var startedAt sql.NullTime
	var completedAt sql.NullTime
	if err := row.Scan(
		&job.ID,
		&job.Symbol,
		&job.BaseInterval,
		&job.From,
		&job.To,
		&job.NextOpenTime,
		&status,
		&job.CandlesInserted,
		&lastError,
		&startedAt,
		&completedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return BackfillJob{}, err
	}
	job.Status = BackfillStatus(status)
	if lastError.Valid {
		job.LastError = lastError.String
	}
	if startedAt.Valid {
		value := startedAt.Time
		job.StartedAt = &value
	}
	if completedAt.Valid {
		value := completedAt.Time
		job.CompletedAt = &value
	}
	return job, nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
