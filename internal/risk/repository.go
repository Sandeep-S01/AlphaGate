package risk

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"sentra/internal/strategy"
)

type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type DecisionQuery struct {
	Symbol string
	From   time.Time
	To     time.Time
	Limit  int
}

type DecisionRepository struct {
	db QueryRower
}

func NewDecisionRepository(db QueryRower) *DecisionRepository {
	return &DecisionRepository{db: db}
}

func (r *DecisionRepository) Save(ctx context.Context, decision Decision) (string, error) {
	query, args := BuildInsertDecisionSQL(decision)
	var id string
	if err := r.db.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *DecisionRepository) LatestApproved(ctx context.Context, symbol string) (Decision, error) {
	query := `
SELECT id, signal_id, symbol, signal_side, decision, reason, evaluated_at, risk_snapshot_json::text
FROM risk_decisions
WHERE symbol = $1 AND decision = 'approved'
ORDER BY evaluated_at DESC
LIMIT 1`
	return r.scanDecision(ctx, query, symbol)
}

func (r *DecisionRepository) Latest(ctx context.Context, symbol string) (Decision, error) {
	query := `
SELECT id, signal_id, symbol, signal_side, decision, reason, evaluated_at, risk_snapshot_json::text
FROM risk_decisions
WHERE symbol = $1
ORDER BY evaluated_at DESC
LIMIT 1`
	return r.scanDecision(ctx, query, symbol)
}

func (r *DecisionRepository) List(ctx context.Context, query DecisionQuery) ([]Decision, error) {
	sql, args := BuildListDecisionsSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query risk decisions: %w", err)
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		decision, err := scanDecisionRow(rows)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, decision)
	}
	return decisions, rows.Err()
}

func (r *DecisionRepository) RejectedReasons(ctx context.Context, query DecisionQuery) ([]RejectionSummary, error) {
	sql, args := BuildRejectedReasonsSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query rejected risk reasons: %w", err)
	}
	defer rows.Close()
	var summaries []RejectionSummary
	for rows.Next() {
		var summary RejectionSummary
		if err := rows.Scan(&summary.Reason, &summary.Count); err != nil {
			return nil, fmt.Errorf("scan rejected risk reason: %w", err)
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

func (r *DecisionRepository) scanDecision(ctx context.Context, query string, args ...any) (Decision, error) {
	return scanDecisionRow(r.db.QueryRow(ctx, query, args...))
}

func BuildInsertDecisionSQL(decision Decision) (string, []any) {
	return `
INSERT INTO risk_decisions (
    signal_id,
    symbol,
    signal_side,
    decision,
    reason,
    evaluated_at,
    risk_snapshot_json
) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
RETURNING id`, []any{
			decision.SignalID,
			decision.Symbol,
			string(decision.SignalSide),
			string(decision.Decision),
			decision.Reason,
			decision.EvaluatedAt,
			decision.RiskSnapshot,
		}
}

type decisionScanner interface {
	Scan(dest ...any) error
}

func scanDecisionRow(row decisionScanner) (Decision, error) {
	var decision Decision
	var side string
	var status string
	if err := row.Scan(
		&decision.ID,
		&decision.SignalID,
		&decision.Symbol,
		&side,
		&status,
		&decision.Reason,
		&decision.EvaluatedAt,
		&decision.RiskSnapshot,
	); err != nil {
		return Decision{}, err
	}
	decision.SignalSide = strategy.Side(side)
	decision.Decision = DecisionStatus(status)
	return decision, nil
}

func BuildListDecisionsSQL(query DecisionQuery) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var builder strings.Builder
	builder.WriteString(`
SELECT id, signal_id, symbol, signal_side, decision, reason, evaluated_at, risk_snapshot_json::text
FROM risk_decisions
WHERE 1 = 1`)
	args := []any{}
	if query.Symbol != "" {
		args = append(args, query.Symbol)
		builder.WriteString(fmt.Sprintf(" AND symbol = $%d", len(args)))
	}
	if !query.From.IsZero() {
		args = append(args, query.From)
		builder.WriteString(fmt.Sprintf(" AND evaluated_at >= $%d", len(args)))
	}
	if !query.To.IsZero() {
		args = append(args, query.To)
		builder.WriteString(fmt.Sprintf(" AND evaluated_at < $%d", len(args)))
	}
	args = append(args, limit)
	builder.WriteString(fmt.Sprintf(" ORDER BY evaluated_at DESC LIMIT $%d", len(args)))
	return builder.String(), args
}

func BuildRejectedReasonsSQL(query DecisionQuery) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var builder strings.Builder
	builder.WriteString(`
SELECT reason, COUNT(*)
FROM risk_decisions
WHERE decision = 'rejected'`)
	args := []any{}
	if query.Symbol != "" {
		args = append(args, query.Symbol)
		builder.WriteString(fmt.Sprintf(" AND symbol = $%d", len(args)))
	}
	if !query.From.IsZero() {
		args = append(args, query.From)
		builder.WriteString(fmt.Sprintf(" AND evaluated_at >= $%d", len(args)))
	}
	if !query.To.IsZero() {
		args = append(args, query.To)
		builder.WriteString(fmt.Sprintf(" AND evaluated_at < $%d", len(args)))
	}
	args = append(args, limit)
	builder.WriteString(fmt.Sprintf(" GROUP BY reason ORDER BY COUNT(*) DESC LIMIT $%d", len(args)))
	return builder.String(), args
}
