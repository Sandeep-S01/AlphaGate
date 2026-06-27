package safety

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	db QueryRower
}

func NewRepository(db QueryRower) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context) (Status, error) {
	query := `
SELECT kill_switch_active, reason, updated_by, updated_at
FROM safety_status
WHERE id = 1`
	var status Status
	if err := r.db.QueryRow(ctx, query).Scan(&status.KillSwitchActive, &status.Reason, &status.UpdatedBy, &status.UpdatedAt); err != nil {
		return Status{}, fmt.Errorf("query safety status: %w", err)
	}
	return status, nil
}

func (r *Repository) Save(ctx context.Context, status Status) (Status, error) {
	status = status.Normalize()
	query, args := BuildUpsertStatusSQL(status)
	var saved Status
	if err := r.db.QueryRow(ctx, query, args...).Scan(&saved.KillSwitchActive, &saved.Reason, &saved.UpdatedBy, &saved.UpdatedAt); err != nil {
		return Status{}, fmt.Errorf("save safety status: %w", err)
	}
	return saved, nil
}

func (r *Repository) IsKillSwitchActive(ctx context.Context) (bool, error) {
	status, err := r.Get(ctx)
	if err != nil {
		return false, err
	}
	return status.KillSwitchActive, nil
}

func (s Status) Normalize() Status {
	s.Reason = strings.TrimSpace(s.Reason)
	s.UpdatedBy = strings.TrimSpace(s.UpdatedBy)
	if s.UpdatedBy == "" {
		s.UpdatedBy = "operator"
	}
	return s
}

func BuildUpsertStatusSQL(status Status) (string, []any) {
	status = status.Normalize()
	return `
INSERT INTO safety_status (
    id,
    kill_switch_active,
    reason,
    updated_by,
    updated_at
) VALUES (1, $1, $2, $3, NOW())
ON CONFLICT (id)
DO UPDATE SET
    kill_switch_active = EXCLUDED.kill_switch_active,
    reason = EXCLUDED.reason,
    updated_by = EXCLUDED.updated_by,
    updated_at = NOW()
RETURNING kill_switch_active, reason, updated_by, updated_at`, []any{
			status.KillSwitchActive,
			status.Reason,
			status.UpdatedBy,
		}
}
