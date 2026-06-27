package audit

import (
	"context"
	"fmt"
	"strings"

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

func (r *Repository) Save(ctx context.Context, event Event) (string, error) {
	event = event.Normalize()
	query, args := BuildInsertEventSQL(event)
	var id string
	if err := r.db.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		return "", fmt.Errorf("save audit event: %w", err)
	}
	return id, nil
}

func (r *Repository) List(ctx context.Context, query Query) ([]Event, error) {
	sql, args := BuildListEventsSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.EventType, &event.Actor, &event.Summary, &event.DetailsJSON, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (e Event) Normalize() Event {
	e.EventType = strings.TrimSpace(e.EventType)
	e.Actor = strings.TrimSpace(e.Actor)
	e.Summary = strings.TrimSpace(e.Summary)
	e.DetailsJSON = strings.TrimSpace(e.DetailsJSON)
	if e.Actor == "" {
		e.Actor = "system"
	}
	if e.DetailsJSON == "" {
		e.DetailsJSON = "{}"
	}
	return e
}

func BuildInsertEventSQL(event Event) (string, []any) {
	event = event.Normalize()
	return `
INSERT INTO audit_events (
    event_type,
    actor,
    summary,
    details_json,
    created_at
) VALUES ($1, $2, $3, $4::jsonb, NOW())
RETURNING id`, []any{
			event.EventType,
			event.Actor,
			event.Summary,
			event.DetailsJSON,
		}
}

func BuildListEventsSQL(query Query) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var builder strings.Builder
	builder.WriteString(`
SELECT id, event_type, actor, summary, details_json::text, created_at
FROM audit_events
WHERE 1 = 1`)
	args := []any{}
	if strings.TrimSpace(query.EventType) != "" {
		args = append(args, strings.TrimSpace(query.EventType))
		builder.WriteString(fmt.Sprintf(" AND event_type = $%d", len(args)))
	}
	if strings.TrimSpace(query.Actor) != "" {
		args = append(args, strings.TrimSpace(query.Actor))
		builder.WriteString(fmt.Sprintf(" AND actor = $%d", len(args)))
	}
	if !query.From.IsZero() {
		args = append(args, query.From)
		builder.WriteString(fmt.Sprintf(" AND created_at >= $%d", len(args)))
	}
	if !query.To.IsZero() {
		args = append(args, query.To)
		builder.WriteString(fmt.Sprintf(" AND created_at < $%d", len(args)))
	}
	args = append(args, limit)
	builder.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args)))
	return builder.String(), args
}
