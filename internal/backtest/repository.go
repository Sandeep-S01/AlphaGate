package backtest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Repository struct {
	db QueryRower
}

type SaveOptions struct {
	SaveEquityCurve bool
}

func NewRepository(db QueryRower) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, run Run, trades []Trade) (Run, error) {
	return r.SaveWithOptions(ctx, run, trades, SaveOptions{SaveEquityCurve: true})
}

func (r *Repository) SaveWithOptions(ctx context.Context, run Run, trades []Trade, options SaveOptions) (Run, error) {
	if beginner, ok := r.db.(txBeginner); ok {
		tx, err := beginner.Begin(ctx)
		if err != nil {
			return Run{}, fmt.Errorf("begin backtest save transaction: %w", err)
		}
		defer tx.Rollback(ctx)
		saved, err := saveWithOptions(ctx, tx, run, trades, options)
		if err != nil {
			return Run{}, fmt.Errorf("save with options: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return Run{}, fmt.Errorf("commit backtest save transaction: %w", err)
		}
		return saved, nil
	}
	return saveWithOptions(ctx, r.db, run, trades, options)
}

func saveWithOptions(ctx context.Context, db QueryRower, run Run, trades []Trade, options SaveOptions) (Run, error) {
	query, args := BuildInsertRunSQL(run)
	if err := db.QueryRow(ctx, query, args...).Scan(&run.ID, &run.CreatedAt); err != nil {
		return Run{}, fmt.Errorf("insert backtest run: %w", err)
	}
	for _, trade := range trades {
		trade.RunID = run.ID
		query, args := BuildInsertTradeSQL(trade)
		if err := db.QueryRow(ctx, query, args...).Scan(&trade.ID); err != nil {
			return Run{}, fmt.Errorf("insert backtest trade: %w", err)
		}
	}
	for index := range run.RoundTrips {
		run.RoundTrips[index].RunID = run.ID
		query, args := BuildInsertRoundTripSQL(run.RoundTrips[index])
		if err := db.QueryRow(ctx, query, args...).Scan(&run.RoundTrips[index].ID); err != nil {
			return Run{}, fmt.Errorf("insert backtest round trip: %w", err)
		}
	}
	points := equityPointsForSave(run, options)
	for index := range points {
		points[index].RunID = run.ID
		query, args := BuildInsertEquityPointSQL(points[index])
		if err := db.QueryRow(ctx, query, args...).Scan(&points[index].ID); err != nil {
			return Run{}, fmt.Errorf("insert backtest equity point: %w", err)
		}
	}
	if options.SaveEquityCurve {
		run.EquityCurve = points
	} else {
		run.EquityCurve = nil
	}
	return run, nil
}

func equityPointsForSave(run Run, options SaveOptions) []EquityPoint {
	if !options.SaveEquityCurve {
		return nil
	}
	return run.EquityCurve
}

func (r *Repository) List(ctx context.Context, query Query) ([]Run, error) {
	sql, args := BuildListRunsSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query backtest runs: %w", err)
	}
	defer rows.Close()

	runs := []Run{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan backtest run: %w", err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating backtest runs: %w", err)
	}
	return runs, nil
}

func (r *Repository) Get(ctx context.Context, id string) (Run, []Trade, error) {
	runRows, err := r.db.Query(ctx, BuildGetRunSQL(), id)
	if err != nil {
		return Run{}, nil, fmt.Errorf("query backtest run: %w", err)
	}
	defer runRows.Close()
	if !runRows.Next() {
		return Run{}, nil, pgx.ErrNoRows
	}
	run, err := scanRun(runRows)
	if err != nil {
		return Run{}, nil, fmt.Errorf("scan backtest run: %w", err)
	}

	tradeRows, err := r.db.Query(ctx, BuildListTradesSQL(), id)
	if err != nil {
		return Run{}, nil, fmt.Errorf("query backtest trades: %w", err)
	}
	defer tradeRows.Close()

	trades := []Trade{}
	for tradeRows.Next() {
		var trade Trade
		var side string
		if err := tradeRows.Scan(&trade.ID, &trade.RunID, &trade.Symbol, &side, &trade.Quantity, &trade.Price, &trade.QuoteAmount, &trade.Fee, &trade.Equity, &trade.ExecutedAt); err != nil {
			return Run{}, nil, fmt.Errorf("scan backtest trade: %w", err)
		}
		trade.Side = Side(side)
		trades = append(trades, trade)
	}
	if err := tradeRows.Err(); err != nil {
		return Run{}, nil, fmt.Errorf("error iterating backtest trades: %w", err)
	}

	roundTripRows, err := r.db.Query(ctx, BuildListRoundTripsSQL(), id)
	if err != nil {
		return Run{}, nil, fmt.Errorf("query backtest round trips: %w", err)
	}
	defer roundTripRows.Close()
	for roundTripRows.Next() {
		var roundTrip RoundTrip
		if err := roundTripRows.Scan(
			&roundTrip.ID,
			&roundTrip.RunID,
			&roundTrip.Symbol,
			&roundTrip.EntryTime,
			&roundTrip.ExitTime,
			&roundTrip.EntryPrice,
			&roundTrip.ExitPrice,
			&roundTrip.Quantity,
			&roundTrip.GrossProfitLoss,
			&roundTrip.Fees,
			&roundTrip.NetProfitLoss,
			&roundTrip.ProfitPercent,
			&roundTrip.HoldingSeconds,
			&roundTrip.EntryReason,
			&roundTrip.ExitReason,
		); err != nil {
			return Run{}, nil, fmt.Errorf("scan backtest round trip: %w", err)
		}
		run.RoundTrips = append(run.RoundTrips, roundTrip)
	}
	if err := roundTripRows.Err(); err != nil {
		return Run{}, nil, fmt.Errorf("error iterating backtest round trips: %w", err)
	}

	equityRows, err := r.db.Query(ctx, BuildListEquityPointsSQL(), id)
	if err != nil {
		return Run{}, nil, fmt.Errorf("query backtest equity points: %w", err)
	}
	defer equityRows.Close()
	for equityRows.Next() {
		var point EquityPoint
		if err := equityRows.Scan(&point.ID, &point.RunID, &point.Time, &point.Equity, &point.DrawdownPercent); err != nil {
			return Run{}, nil, fmt.Errorf("scan backtest equity point: %w", err)
		}
		run.EquityCurve = append(run.EquityCurve, point)
	}
	if err := equityRows.Err(); err != nil {
		return Run{}, nil, fmt.Errorf("error iterating backtest equity points: %w", err)
	}
	return run, trades, nil
}

type runScanner interface {
	Scan(dest ...any) error
}

func scanRun(row runScanner) (Run, error) {
	var run Run
	if err := row.Scan(
		&run.ID,
		&run.StrategyName,
		&run.Version,
		&run.Symbol,
		&run.Interval,
		&run.From,
		&run.To,
		&run.FastPeriod,
		&run.SlowPeriod,
		&run.RSIPeriod,
		&run.RSIOversold,
		&run.RSIOverbought,
		&run.StartingBalance,
		&run.EndingBalance,
		&run.ProfitLoss,
		&run.GrossProfitLoss,
		&run.TotalFees,
		&run.EstimatedSlippageCost,
		&run.RoundTripCostPercent,
		&run.BreakEvenMovePercent,
		&run.ReturnPercent,
		&run.WinRate,
		&run.MaxDrawdown,
		&run.TotalTrades,
		&run.BuyCount,
		&run.SellCount,
		&run.BestTrade,
		&run.WorstTrade,
		&run.AverageWin,
		&run.AverageLoss,
		&run.OpenPosition,
		&run.FeeRate,
		&run.SlippageRate,
		&run.PositionSizingMode,
		&run.PositionSizeValue,
		&run.TrendFilterEnabled,
		&run.TrendPeriod,
		&run.CooldownBars,
		&run.MinHoldingBars,
		&run.ATRExitEnabled,
		&run.ATRPeriod,
		&run.ATRStopMultiplier,
		&run.ATRTakeProfitMultiplier,
		&run.RegimeFilterEnabled,
		&run.RegimeFilterPeriod,
		&run.RegimeMinATRPercent,
		&run.RegimeMaxATRPercent,
		&run.ShortingEnabled,
		&run.WinningTrades,
		&run.LosingTrades,
		&run.ProfitFactor,
		&run.AverageTrade,
		&run.AverageHoldingSeconds,
		&run.Expectancy,
		&run.TradesPerDay,
		&run.ChurnRatio,
		&run.SharpeRatio,
		&run.SortinoRatio,
		&run.BenchmarkEndingBalance,
		&run.BenchmarkProfitLoss,
		&run.BenchmarkReturnPercent,
		&run.ExcessReturnPercent,
		&run.ValidationStatus,
		&run.ValidationReason,
		&run.ExecutionFillMode,
		&run.CreatedAt,
		&run.PineStrategyID,
		&run.PineConfig,
	); err != nil {
		return Run{}, fmt.Errorf("scan backtest run: %w", err)
	}
	return run, nil
}

func BuildInsertRunSQL(run Run) (string, []any) {
	return `
INSERT INTO backtest_runs (
    strategy_name,
    version,
    symbol,
    interval,
    from_time,
    to_time,
    fast_period,
    slow_period,
    rsi_period,
    rsi_oversold,
    rsi_overbought,
    starting_balance,
    ending_balance,
    profit_loss,
    gross_profit_loss,
    total_fees,
    estimated_slippage_cost,
    round_trip_cost_percent,
    break_even_move_percent,
    return_percent,
    win_rate,
    max_drawdown,
    total_trades,
    buy_count,
    sell_count,
    best_trade,
    worst_trade,
    average_win,
    average_loss,
    open_position,
    fee_rate,
    slippage_rate,
    position_sizing_mode,
    position_size_value,
    trend_filter_enabled,
    trend_period,
    cooldown_bars,
    min_holding_bars,
    atr_exit_enabled,
    atr_period,
    atr_stop_multiplier,
    atr_take_profit_multiplier,
    regime_filter_enabled,
    regime_filter_period,
    regime_min_atr_percent,
    regime_max_atr_percent,
    shorting_enabled,
    winning_trades,
    losing_trades,
    profit_factor,
    average_trade,
    average_holding_seconds,
    expectancy,
    trades_per_day,
    churn_ratio,
    sharpe_ratio,
    sortino_ratio,
    benchmark_ending_balance,
    benchmark_profit_loss,
    benchmark_return_percent,
    excess_return_percent,
    validation_status,
    validation_reason,
    execution_fill_mode,
    created_at,
    request_snapshot,
    pine_strategy_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40, $41, $42, $43, $44, $45, $46, $47, $48, $49, $50, $51, $52, $53, $54, $55, $56, $57, $58, $59, $60, $61, $62, $63, $64, $65, $66, $67)
RETURNING id, created_at`, []any{
			run.StrategyName,
			run.Version,
			run.Symbol,
			run.Interval,
			run.From,
			run.To,
			run.FastPeriod,
			run.SlowPeriod,
			run.RSIPeriod,
			run.RSIOversold,
			run.RSIOverbought,
			run.StartingBalance,
			run.EndingBalance,
			run.ProfitLoss,
			run.GrossProfitLoss,
			run.TotalFees,
			run.EstimatedSlippageCost,
			run.RoundTripCostPercent,
			run.BreakEvenMovePercent,
			run.ReturnPercent,
			run.WinRate,
			run.MaxDrawdown,
			run.TotalTrades,
			run.BuyCount,
			run.SellCount,
			run.BestTrade,
			run.WorstTrade,
			run.AverageWin,
			run.AverageLoss,
			run.OpenPosition,
			run.FeeRate,
			run.SlippageRate,
			run.PositionSizingMode,
			run.PositionSizeValue,
			run.TrendFilterEnabled,
			run.TrendPeriod,
			run.CooldownBars,
			run.MinHoldingBars,
			run.ATRExitEnabled,
			run.ATRPeriod,
			run.ATRStopMultiplier,
			run.ATRTakeProfitMultiplier,
			run.RegimeFilterEnabled,
			run.RegimeFilterPeriod,
			run.RegimeMinATRPercent,
			run.RegimeMaxATRPercent,
			run.ShortingEnabled,
			run.WinningTrades,
			run.LosingTrades,
			run.ProfitFactor,
			run.AverageTrade,
			run.AverageHoldingSeconds,
			run.Expectancy,
			run.TradesPerDay,
			run.ChurnRatio,
			run.SharpeRatio,
			run.SortinoRatio,
			run.BenchmarkEndingBalance,
			run.BenchmarkProfitLoss,
			run.BenchmarkReturnPercent,
			run.ExcessReturnPercent,
			run.ValidationStatus,
			run.ValidationReason,
			defaultExecutionFillMode(run.ExecutionFillMode),
			defaultCreatedAt(run.CreatedAt),
			"{}",
			run.PineStrategyID,
		}
}

func BuildInsertTradeSQL(trade Trade) (string, []any) {
	return `
INSERT INTO backtest_trades (
    run_id,
    symbol,
    side,
    quantity,
    price,
    quote_amount,
    fee,
    equity,
    executed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id`, []any{
			trade.RunID,
			trade.Symbol,
			string(trade.Side),
			trade.Quantity,
			trade.Price,
			trade.QuoteAmount,
			trade.Fee,
			trade.Equity,
			trade.ExecutedAt,
		}
}

func BuildInsertRoundTripSQL(roundTrip RoundTrip) (string, []any) {
	return `
INSERT INTO backtest_round_trips (
    run_id,
    symbol,
    entry_time,
    exit_time,
    entry_price,
    exit_price,
    quantity,
    gross_profit_loss,
    fees,
    net_profit_loss,
    profit_percent,
    holding_seconds,
    entry_reason,
    exit_reason
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING id`, []any{
			roundTrip.RunID,
			roundTrip.Symbol,
			roundTrip.EntryTime,
			roundTrip.ExitTime,
			roundTrip.EntryPrice,
			roundTrip.ExitPrice,
			roundTrip.Quantity,
			roundTrip.GrossProfitLoss,
			roundTrip.Fees,
			roundTrip.NetProfitLoss,
			roundTrip.ProfitPercent,
			roundTrip.HoldingSeconds,
			roundTrip.EntryReason,
			roundTrip.ExitReason,
		}
}

func BuildInsertEquityPointSQL(point EquityPoint) (string, []any) {
	return `
INSERT INTO backtest_equity_points (
    run_id,
    time,
    equity,
    drawdown_percent
) VALUES ($1, $2, $3, $4)
RETURNING id`, []any{
			point.RunID,
			point.Time,
			point.Equity,
			point.DrawdownPercent,
		}
}

func BuildListRunsSQL(query Query) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var builder strings.Builder
	builder.WriteString(runSelectSQL())
	builder.WriteString(" WHERE 1 = 1")
	args := []any{}
	if query.Symbol != "" {
		args = append(args, query.Symbol)
		builder.WriteString(fmt.Sprintf(" AND symbol = $%d", len(args)))
	}
	args = append(args, limit)
	builder.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args)))
	return builder.String(), args
}

func BuildGetRunSQL() string {
	return runSelectSQL() + " WHERE r.id = $1"
}

func BuildListTradesSQL() string {
	return `
SELECT id, run_id, symbol, side, quantity, price, quote_amount, fee, equity, executed_at
FROM backtest_trades
WHERE run_id = $1
ORDER BY executed_at ASC`
}

func BuildListRoundTripsSQL() string {
	return `
SELECT id, run_id, symbol, entry_time, exit_time, entry_price, exit_price, quantity,
       gross_profit_loss, fees, net_profit_loss, profit_percent, holding_seconds,
       entry_reason, exit_reason
FROM backtest_round_trips
WHERE run_id = $1
ORDER BY entry_time ASC`
}

func BuildListEquityPointsSQL() string {
	return `
SELECT id, run_id, time, equity, drawdown_percent
FROM backtest_equity_points
WHERE run_id = $1
ORDER BY time ASC`
}

func runSelectSQL() string {
	return `
SELECT r.id, r.strategy_name, r.version, r.symbol, r.interval, r.from_time, r.to_time, r.fast_period, r.slow_period,
       r.rsi_period, r.rsi_oversold, r.rsi_overbought,
       r.starting_balance, r.ending_balance, r.profit_loss,
       r.gross_profit_loss, r.total_fees, r.estimated_slippage_cost,
       r.round_trip_cost_percent, r.break_even_move_percent,
       r.return_percent, r.win_rate, r.max_drawdown,
       r.total_trades, r.buy_count, r.sell_count, r.best_trade, r.worst_trade, r.average_win, r.average_loss,
       r.open_position, r.fee_rate, r.slippage_rate, r.position_sizing_mode, r.position_size_value,
       r.trend_filter_enabled, r.trend_period, r.cooldown_bars, r.min_holding_bars,
       r.atr_exit_enabled, r.atr_period, r.atr_stop_multiplier, r.atr_take_profit_multiplier,
       r.regime_filter_enabled, r.regime_filter_period, r.regime_min_atr_percent, r.regime_max_atr_percent,
       r.shorting_enabled,
       r.winning_trades, r.losing_trades, r.profit_factor, r.average_trade, r.average_holding_seconds,
       r.expectancy, r.trades_per_day, r.churn_ratio, r.sharpe_ratio, r.sortino_ratio,
       r.benchmark_ending_balance, r.benchmark_profit_loss, r.benchmark_return_percent, r.excess_return_percent,
       r.validation_status, r.validation_reason, r.execution_fill_mode, r.created_at,
       r.pine_strategy_id, p.converted_config::text as pine_config
FROM backtest_runs r
LEFT JOIN pine_strategies p ON r.pine_strategy_id = p.id`
}

func defaultExecutionFillMode(mode string) string {
	if mode == "" {
		return ExecutionFillModeSameClose
	}
	return mode
}

func defaultCreatedAt(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
}
