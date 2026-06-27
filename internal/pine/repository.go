package pine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// QueryRower abstracts pgx query methods for testability.
type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Repository handles CRUD operations for pine_strategies.
type Repository struct {
	db QueryRower
}

// Query defines filters for listing pine strategies.
type Query struct {
	Limit int
}

// NewRepository creates a new Pine strategy repository.
func NewRepository(db QueryRower) *Repository {
	return &Repository{db: db}
}

// Save inserts or updates a Pine strategy by name.
func (r *Repository) Save(ctx context.Context, strategy PineStrategy) (PineStrategy, error) {
	configJSON, err := json.Marshal(strategy.ConvertedConfig)
	if err != nil {
		return PineStrategy{}, fmt.Errorf("marshal converted config: %w", err)
	}
	query := `
INSERT INTO pine_strategies (name, pine_code, converted_config)
VALUES ($1, $2, $3)
ON CONFLICT (name)
DO UPDATE SET
    pine_code = EXCLUDED.pine_code,
    converted_config = EXCLUDED.converted_config,
    updated_at = NOW()
RETURNING id, name, pine_code, converted_config, created_at, updated_at`

	var saved PineStrategy
	var rawConfig []byte
	if err := r.db.QueryRow(ctx, query,
		strings.TrimSpace(strategy.Name),
		strategy.PineCode,
		configJSON,
	).Scan(
		&saved.ID,
		&saved.Name,
		&saved.PineCode,
		&rawConfig,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	); err != nil {
		return PineStrategy{}, fmt.Errorf("upsert pine strategy: %w", err)
	}
	if err := json.Unmarshal(rawConfig, &saved.ConvertedConfig); err != nil {
		return PineStrategy{}, fmt.Errorf("unmarshal config: %w", err)
	}
	return saved, nil
}

// Get retrieves a Pine strategy by ID.
func (r *Repository) Get(ctx context.Context, id string) (PineStrategy, error) {
	query := `
SELECT id, name, pine_code, converted_config, created_at, updated_at
FROM pine_strategies
WHERE id = $1`

	var strategy PineStrategy
	var rawConfig []byte
	if err := r.db.QueryRow(ctx, query, id).Scan(
		&strategy.ID,
		&strategy.Name,
		&strategy.PineCode,
		&rawConfig,
		&strategy.CreatedAt,
		&strategy.UpdatedAt,
	); err != nil {
		return PineStrategy{}, fmt.Errorf("get pine strategy: %w", err)
	}
	if err := json.Unmarshal(rawConfig, &strategy.ConvertedConfig); err != nil {
		return PineStrategy{}, fmt.Errorf("unmarshal config: %w", err)
	}
	return strategy, nil
}

// List returns pine strategies ordered by creation date.
func (r *Repository) List(ctx context.Context, query Query) ([]PineStrategy, error) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	sql := `
SELECT id, name, pine_code, converted_config, created_at, updated_at
FROM pine_strategies
ORDER BY created_at DESC
LIMIT $1`

	rows, err := r.db.Query(ctx, sql, limit)
	if err != nil {
		return nil, fmt.Errorf("list pine strategies: %w", err)
	}
	defer rows.Close()

	var strategies []PineStrategy
	for rows.Next() {
		var strategy PineStrategy
		var rawConfig []byte
		if err := rows.Scan(
			&strategy.ID,
			&strategy.Name,
			&strategy.PineCode,
			&rawConfig,
			&strategy.CreatedAt,
			&strategy.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pine strategy: %w", err)
		}
		if err := json.Unmarshal(rawConfig, &strategy.ConvertedConfig); err != nil {
			return nil, fmt.Errorf("unmarshal config: %w", err)
		}
		strategies = append(strategies, strategy)
	}
	return strategies, rows.Err()
}
