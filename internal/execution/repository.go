package execution

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

type Query struct {
	Symbol string
	From   time.Time
	To     time.Time
	Limit  int
}

type Repository struct {
	db QueryRower
}

func NewRepository(db QueryRower) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, result Result) (string, string, error) {
	orderQuery, orderArgs := BuildInsertOrderSQL(result.Order)
	var orderID string
	if err := r.db.QueryRow(ctx, orderQuery, orderArgs...).Scan(&orderID); err != nil {
		return "", "", fmt.Errorf("insert paper order: %w", err)
	}

	result.Trade.OrderID = orderID
	tradeQuery, tradeArgs := BuildInsertTradeSQL(result.Trade)
	var tradeID string
	if err := r.db.QueryRow(ctx, tradeQuery, tradeArgs...).Scan(&tradeID); err != nil {
		return "", "", fmt.Errorf("insert paper trade: %w", err)
	}
	for _, event := range result.OrderEvents {
		event.OrderID = orderID
		eventQuery, eventArgs := BuildInsertOrderEventSQL(event)
		var eventID string
		if err := r.db.QueryRow(ctx, eventQuery, eventArgs...).Scan(&eventID); err != nil {
			return "", "", fmt.Errorf("insert paper order event: %w", err)
		}
	}
	return orderID, tradeID, nil
}

func (r *Repository) ListOrders(ctx context.Context, query Query) ([]Order, error) {
	sql, args := BuildListOrdersSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query paper orders: %w", err)
	}
	defer rows.Close()
	var orders []Order
	for rows.Next() {
		var order Order
		var side string
		var status string
		if err := rows.Scan(
			&order.ID,
			&order.RiskDecisionID,
			&order.ClientOrderID,
			&order.ExchangeOrderID,
			&order.StrategyName,
			&order.Symbol,
			&side,
			&order.Quantity,
			&order.RequestedQuantity,
			&order.FilledQuantity,
			&order.Price,
			&order.AverageFillPrice,
			&order.QuoteAmount,
			&order.Fee,
			&status,
			&order.FailureReason,
			&order.SubmittedAt,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan paper order: %w", err)
		}
		order.Side = strategy.Side(side)
		order.Status = OrderStatus(status)
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating paper orders: %w", err)
	}
	return orders, nil
}

func (r *Repository) ListTrades(ctx context.Context, query Query) ([]Trade, error) {
	sql, args := BuildListTradesSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query paper trades: %w", err)
	}
	defer rows.Close()
	var trades []Trade
	for rows.Next() {
		var trade Trade
		var side string
		if err := rows.Scan(&trade.ID, &trade.OrderID, &trade.Symbol, &side, &trade.Quantity, &trade.Price, &trade.Fee, &trade.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan paper trade: %w", err)
		}
		trade.Side = strategy.Side(side)
		trades = append(trades, trade)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating paper trades: %w", err)
	}
	return trades, nil
}

func (r *Repository) DailyStats(ctx context.Context, symbol string, day time.Time) (DailyStats, error) {
	start := time.Date(day.UTC().Year(), day.UTC().Month(), day.UTC().Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	var stats DailyStats
	var net float64
	if err := r.db.QueryRow(ctx, BuildDailyStatsSQL(), symbol, start, end).Scan(&stats.TradeCount, &net, &stats.LastTradeAt); err != nil {
		return DailyStats{}, fmt.Errorf("query daily paper stats: %w", err)
	}
	if net < 0 {
		stats.DailyLoss = -net
	}
	return stats, nil
}

func (r *Repository) DailyPnL(ctx context.Context, query Query) ([]DailyPnL, error) {
	sql, args := BuildDailyPnLSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query daily paper pnl: %w", err)
	}
	defer rows.Close()
	var summaries []DailyPnL
	for rows.Next() {
		var summary DailyPnL
		if err := rows.Scan(&summary.Day, &summary.NetPnL, &summary.TradeCount); err != nil {
			return nil, fmt.Errorf("scan daily paper pnl: %w", err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating daily paper pnl: %w", err)
	}
	return summaries, nil
}

func (r *Repository) TradeCounts(ctx context.Context, query Query) ([]DailyPnL, error) {
	return r.DailyPnL(ctx, query)
}

func BuildDailyStatsSQL() string {
	return `
SELECT COUNT(*),
       COALESCE(SUM(CASE
           WHEN side = 'buy' THEN -((quantity * price) + fee)
           WHEN side = 'sell' THEN ((quantity * price) - fee)
           ELSE 0
       END), 0),
       COALESCE(MAX(created_at), '0001-01-01T00:00:00Z'::timestamptz)
FROM paper_trades
WHERE symbol = $1 AND created_at >= $2 AND created_at < $3`
}

func BuildDailyPnLSQL(query Query) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 30
	}
	var builder strings.Builder
	builder.WriteString(`
SELECT DATE_TRUNC('day', created_at) AS day,
       COALESCE(SUM(CASE
           WHEN side = 'buy' THEN -((quantity * price) + fee)
           WHEN side = 'sell' THEN ((quantity * price) - fee)
           ELSE 0
       END), 0) AS net_pnl,
       COUNT(*) AS trade_count
FROM paper_trades
WHERE 1 = 1`)
	args := []any{}
	if query.Symbol != "" {
		args = append(args, query.Symbol)
		builder.WriteString(fmt.Sprintf(" AND symbol = $%d", len(args)))
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
	builder.WriteString(fmt.Sprintf(" GROUP BY day ORDER BY day DESC LIMIT $%d", len(args)))
	return builder.String(), args
}

func BuildInsertOrderSQL(order Order) (string, []any) {
	return `
INSERT INTO paper_orders (
    risk_decision_id,
    client_order_id,
    exchange_order_id,
    strategy_name,
    symbol,
    side,
    quantity,
    requested_quantity,
    filled_quantity,
    price,
    average_fill_price,
    quote_amount,
    fee,
    status,
    failure_reason,
    submitted_at,
    updated_at,
    created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
RETURNING id`, []any{
			order.RiskDecisionID,
			order.ClientOrderID,
			order.ExchangeOrderID,
			order.StrategyName,
			order.Symbol,
			string(order.Side),
			order.Quantity,
			order.RequestedQuantity,
			order.FilledQuantity,
			order.Price,
			order.AverageFillPrice,
			order.QuoteAmount,
			order.Fee,
			string(order.Status),
			order.FailureReason,
			order.SubmittedAt,
			order.UpdatedAt,
			order.CreatedAt,
		}
}

func BuildListOrdersSQL(query Query) (string, []any) {
	return buildExecutionListSQL(`
SELECT id, risk_decision_id, client_order_id, exchange_order_id, strategy_name, symbol, side,
       quantity, requested_quantity, filled_quantity, price, average_fill_price, quote_amount,
       fee, status, failure_reason, submitted_at, created_at, updated_at
FROM paper_orders`, query)
}

func BuildListTradesSQL(query Query) (string, []any) {
	return buildExecutionListSQL(`
SELECT id, order_id, symbol, side, quantity, price, fee, created_at
FROM paper_trades`, query)
}

func BuildInsertOrderEventSQL(event OrderEvent) (string, []any) {
	return `
INSERT INTO paper_order_events (
    order_id,
    status,
    reason,
    created_at
) VALUES ($1, $2, $3, $4)
RETURNING id`, []any{
			event.OrderID,
			string(event.Status),
			event.Reason,
			event.CreatedAt,
		}
}

func buildExecutionListSQL(prefix string, query Query) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var builder strings.Builder
	builder.WriteString(prefix)
	builder.WriteString("\nWHERE 1 = 1")
	args := []any{}
	if query.Symbol != "" {
		args = append(args, query.Symbol)
		builder.WriteString(fmt.Sprintf(" AND symbol = $%d", len(args)))
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

func BuildInsertTradeSQL(trade Trade) (string, []any) {
	return `
INSERT INTO paper_trades (
    order_id,
    symbol,
    side,
    quantity,
    price,
    fee,
    created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id`, []any{
			trade.OrderID,
			trade.Symbol,
			string(trade.Side),
			trade.Quantity,
			trade.Price,
			trade.Fee,
			trade.CreatedAt,
		}
}
