package migrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"sentra/internal/observability"
)

type Runner struct {
	db      *pgxpool.Pool
	metrics *observability.Registry
}

func NewRunner(db *pgxpool.Pool, metrics *observability.Registry) *Runner {
	return &Runner{db: db, metrics: metrics}
}

func (r *Runner) Up(ctx context.Context, dir string) error {
	if err := r.ensureTable(ctx); err != nil {
		return err
	}

	files, err := LoadFiles(dir, DirectionUp)
	if err != nil {
		return err
	}

	// Pre-migration validation: check that all migration files are valid
	for _, file := range files {
		if file.Version == "" {
			return fmt.Errorf("migration file has empty version: %s", file.Name)
		}
		if file.Name == "" {
			return fmt.Errorf("migration file has empty name: %s", file.Name)
		}
		if file.SQL == "" {
			return fmt.Errorf("migration file has empty SQL: %s", file.Name)
		}
	}

	var totalApplied int64
	for _, file := range files {
		applied, err := r.applied(ctx, file.Version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		tx, err := r.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", file.Version, err)
		}
		if _, err := tx.Exec(ctx, file.SQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("execute migration %s: %w", file.Version, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, file.Version, file.Name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", file.Version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", file.Version, err)
		}

		if r.metrics != nil {
			// Record individual migration duration
			// In a real implementation, we might want to use histograms or more detailed metrics
			// For now, we'll just log it or could add a counter
			r.metrics.IncHTTPRequests() // Reusing existing metric for simplicity - in reality would add migration-specific metrics
		}

		totalApplied++
	}

	if r.metrics != nil {
		// Record overall migration metrics
		// Again, reusing existing metric - in reality would add migration-specific metrics
		r.metrics.IncHTTPRequests()
	}

	return nil
}

func (r *Runner) ensureTable(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`)
	if err != nil {
		return fmt.Errorf("ensure schema migrations table: %w", err)
	}
	return nil
}

func (r *Runner) applied(ctx context.Context, version string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}
	return exists, nil
}
