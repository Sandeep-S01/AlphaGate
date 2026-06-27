package backtest

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type ComparisonRepository struct {
	db QueryRower
}

func NewComparisonRepository(db QueryRower) *ComparisonRepository {
	return &ComparisonRepository{db: db}
}

func (r *ComparisonRepository) Save(ctx context.Context, comparison Comparison) (Comparison, error) {
	query, args := BuildInsertComparisonSQL(comparison)
	if err := r.db.QueryRow(ctx, query, args...).Scan(&comparison.ID, &comparison.CreatedAt); err != nil {
		return Comparison{}, fmt.Errorf("insert strategy comparison: %w", err)
	}
	for index := range comparison.Results {
		comparison.Results[index].ComparisonID = comparison.ID
		query, args := BuildInsertComparisonResultSQL(comparison.Results[index])
		if err := r.db.QueryRow(ctx, query, args...).Scan(&comparison.Results[index].ID); err != nil {
			return Comparison{}, fmt.Errorf("insert strategy comparison result: %w", err)
		}
	}
	return comparison, nil
}

func (r *ComparisonRepository) List(ctx context.Context, query Query) ([]Comparison, error) {
	sql, args := BuildListComparisonsSQL(query)
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query strategy comparisons: %w", err)
	}
	defer rows.Close()

	comparisons := []Comparison{}
	for rows.Next() {
		comparison, err := scanComparison(rows)
		if err != nil {
			return nil, err
		}
		comparisons = append(comparisons, comparison)
	}
	return comparisons, rows.Err()
}

func (r *ComparisonRepository) Get(ctx context.Context, id string) (Comparison, error) {
	rows, err := r.db.Query(ctx, BuildGetComparisonSQL(), id)
	if err != nil {
		return Comparison{}, fmt.Errorf("query strategy comparison: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return Comparison{}, pgx.ErrNoRows
	}
	comparison, err := scanComparison(rows)
	if err != nil {
		return Comparison{}, err
	}

	resultRows, err := r.db.Query(ctx, BuildListComparisonResultsSQL(), id)
	if err != nil {
		return Comparison{}, fmt.Errorf("query strategy comparison results: %w", err)
	}
	defer resultRows.Close()
	for resultRows.Next() {
		result, err := scanComparisonResult(resultRows)
		if err != nil {
			return Comparison{}, err
		}
		comparison.Results = append(comparison.Results, result)
	}
	return comparison, resultRows.Err()
}

func BuildInsertComparisonSQL(comparison Comparison) (string, []any) {
	return `
INSERT INTO strategy_comparisons (
    symbol,
    interval,
    from_time,
    to_time,
    starting_balance,
    fee_rate,
    slippage_rate,
    execution_fill_mode,
    position_sizing_mode,
    position_size_value,
    trend_filter_enabled,
    trend_period,
    cooldown_bars,
    min_holding_bars,
    train_test_enabled,
    train_ratio,
    train_from,
    train_to,
    test_from,
    test_to,
    walk_forward_enabled,
    walk_forward_folds,
    winner_strategy
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
RETURNING id, created_at`, []any{
			comparison.Symbol,
			comparison.Interval,
			comparison.From,
			comparison.To,
			comparison.StartingBalance,
			comparison.FeeRate,
			comparison.SlippageRate,
			defaultExecutionFillMode(comparison.ExecutionFillMode),
			comparison.PositionSizingMode,
			comparison.PositionSizeValue,
			comparison.TrendFilterEnabled,
			comparison.TrendPeriod,
			comparison.CooldownBars,
			comparison.MinHoldingBars,
			comparison.TrainTestEnabled,
			comparison.TrainRatio,
			comparison.TrainFrom,
			comparison.TrainTo,
			comparison.TestFrom,
			comparison.TestTo,
			comparison.WalkForwardEnabled,
			comparison.WalkForwardFolds,
			comparison.WinnerStrategy,
		}
}

func BuildInsertComparisonResultSQL(result ComparisonResult) (string, []any) {
	return `
INSERT INTO strategy_comparison_results (
    comparison_id,
    rank,
    strategy_name,
    version,
    fast_period,
    slow_period,
    rsi_period,
    rsi_oversold,
    rsi_overbought,
    ending_balance,
    profit_loss,
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
    profit_factor,
    average_trade,
    average_holding_seconds,
    expectancy,
    trades_per_day,
    churn_ratio,
    sharpe_ratio,
    sortino_ratio,
    execution_fill_mode,
    position_sizing_mode,
    position_size_value,
    trend_filter_enabled,
    trend_period,
    cooldown_bars,
    min_holding_bars,
    benchmark_ending_balance,
    benchmark_profit_loss,
    benchmark_return_percent,
    excess_return_percent,
    validation_status,
    validation_reason,
    train_return_percent,
    train_excess_return_percent,
    train_profit_factor,
    train_max_drawdown,
    train_total_trades,
    train_validation_status,
    train_validation_reason,
    test_return_percent,
    test_excess_return_percent,
    test_profit_factor,
    test_max_drawdown,
    test_total_trades,
    test_validation_status,
    test_validation_reason,
    walk_forward_folds,
    walk_forward_passes,
    walk_forward_average_return,
    walk_forward_average_excess,
    walk_forward_worst_drawdown,
    walk_forward_validation_status,
    walk_forward_validation_reason,
    open_position
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40, $41, $42, $43, $44, $45, $46, $47, $48, $49, $50, $51, $52, $53, $54, $55, $56, $57, $58, $59, $60, $61, $62, $63, $64)
RETURNING id`, []any{
			result.ComparisonID,
			result.Rank,
			result.StrategyName,
			result.Version,
			result.FastPeriod,
			result.SlowPeriod,
			result.RSIPeriod,
			result.RSIOversold,
			result.RSIOverbought,
			result.EndingBalance,
			result.ProfitLoss,
			result.ReturnPercent,
			result.WinRate,
			result.MaxDrawdown,
			result.TotalTrades,
			result.BuyCount,
			result.SellCount,
			result.BestTrade,
			result.WorstTrade,
			result.AverageWin,
			result.AverageLoss,
			result.ProfitFactor,
			result.AverageTrade,
			result.AverageHoldingSeconds,
			result.Expectancy,
			result.TradesPerDay,
			result.ChurnRatio,
			result.SharpeRatio,
			result.SortinoRatio,
			defaultExecutionFillMode(result.ExecutionFillMode),
			result.PositionSizingMode,
			result.PositionSizeValue,
			result.TrendFilterEnabled,
			result.TrendPeriod,
			result.CooldownBars,
			result.MinHoldingBars,
			result.BenchmarkEndingBalance,
			result.BenchmarkProfitLoss,
			result.BenchmarkReturnPercent,
			result.ExcessReturnPercent,
			result.ValidationStatus,
			result.ValidationReason,
			result.TrainReturnPercent,
			result.TrainExcessReturn,
			result.TrainProfitFactor,
			result.TrainMaxDrawdown,
			result.TrainTotalTrades,
			result.TrainValidationStatus,
			result.TrainValidationReason,
			result.TestReturnPercent,
			result.TestExcessReturn,
			result.TestProfitFactor,
			result.TestMaxDrawdown,
			result.TestTotalTrades,
			result.TestValidationStatus,
			result.TestValidationReason,
			result.WalkForwardFolds,
			result.WalkForwardPasses,
			result.WalkForwardAverageReturn,
			result.WalkForwardAverageExcess,
			result.WalkForwardWorstDrawdown,
			result.WalkForwardValidationStatus,
			result.WalkForwardValidationReason,
			result.OpenPosition,
		}
}

func BuildListComparisonsSQL(query Query) (string, []any) {
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var builder strings.Builder
	builder.WriteString(comparisonSelectSQL())
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

func BuildGetComparisonSQL() string {
	return comparisonSelectSQL() + " WHERE id = $1"
}

func BuildListComparisonResultsSQL() string {
	return comparisonResultSelectSQL() + " WHERE comparison_id = $1 ORDER BY rank ASC"
}

func scanComparison(row runScanner) (Comparison, error) {
	var comparison Comparison
	if err := row.Scan(
		&comparison.ID,
		&comparison.Symbol,
		&comparison.Interval,
		&comparison.From,
		&comparison.To,
		&comparison.StartingBalance,
		&comparison.FeeRate,
		&comparison.SlippageRate,
		&comparison.ExecutionFillMode,
		&comparison.PositionSizingMode,
		&comparison.PositionSizeValue,
		&comparison.TrendFilterEnabled,
		&comparison.TrendPeriod,
		&comparison.CooldownBars,
		&comparison.MinHoldingBars,
		&comparison.TrainTestEnabled,
		&comparison.TrainRatio,
		&comparison.TrainFrom,
		&comparison.TrainTo,
		&comparison.TestFrom,
		&comparison.TestTo,
		&comparison.WalkForwardEnabled,
		&comparison.WalkForwardFolds,
		&comparison.WinnerStrategy,
		&comparison.CreatedAt,
	); err != nil {
		return Comparison{}, fmt.Errorf("scan strategy comparison: %w", err)
	}
	return comparison, nil
}

func scanComparisonResult(row runScanner) (ComparisonResult, error) {
	var result ComparisonResult
	if err := row.Scan(
		&result.ID,
		&result.ComparisonID,
		&result.Rank,
		&result.StrategyName,
		&result.Version,
		&result.FastPeriod,
		&result.SlowPeriod,
		&result.RSIPeriod,
		&result.RSIOversold,
		&result.RSIOverbought,
		&result.EndingBalance,
		&result.ProfitLoss,
		&result.ReturnPercent,
		&result.WinRate,
		&result.MaxDrawdown,
		&result.TotalTrades,
		&result.BuyCount,
		&result.SellCount,
		&result.BestTrade,
		&result.WorstTrade,
		&result.AverageWin,
		&result.AverageLoss,
		&result.ProfitFactor,
		&result.AverageTrade,
		&result.AverageHoldingSeconds,
		&result.Expectancy,
		&result.TradesPerDay,
		&result.ChurnRatio,
		&result.SharpeRatio,
		&result.SortinoRatio,
		&result.ExecutionFillMode,
		&result.PositionSizingMode,
		&result.PositionSizeValue,
		&result.TrendFilterEnabled,
		&result.TrendPeriod,
		&result.CooldownBars,
		&result.MinHoldingBars,
		&result.BenchmarkEndingBalance,
		&result.BenchmarkProfitLoss,
		&result.BenchmarkReturnPercent,
		&result.ExcessReturnPercent,
		&result.ValidationStatus,
		&result.ValidationReason,
		&result.TrainReturnPercent,
		&result.TrainExcessReturn,
		&result.TrainProfitFactor,
		&result.TrainMaxDrawdown,
		&result.TrainTotalTrades,
		&result.TrainValidationStatus,
		&result.TrainValidationReason,
		&result.TestReturnPercent,
		&result.TestExcessReturn,
		&result.TestProfitFactor,
		&result.TestMaxDrawdown,
		&result.TestTotalTrades,
		&result.TestValidationStatus,
		&result.TestValidationReason,
		&result.WalkForwardFolds,
		&result.WalkForwardPasses,
		&result.WalkForwardAverageReturn,
		&result.WalkForwardAverageExcess,
		&result.WalkForwardWorstDrawdown,
		&result.WalkForwardValidationStatus,
		&result.WalkForwardValidationReason,
		&result.OpenPosition,
	); err != nil {
		return ComparisonResult{}, fmt.Errorf("scan strategy comparison result: %w", err)
	}
	return result, nil
}

func comparisonSelectSQL() string {
	return `
SELECT id, symbol, interval, from_time, to_time, starting_balance, fee_rate, slippage_rate,
       execution_fill_mode, position_sizing_mode, position_size_value, trend_filter_enabled, trend_period,
       cooldown_bars, min_holding_bars, train_test_enabled, train_ratio, train_from, train_to,
       test_from, test_to, walk_forward_enabled, walk_forward_folds, winner_strategy, created_at
FROM strategy_comparisons`
}

func comparisonResultSelectSQL() string {
	return `
SELECT id, comparison_id, rank, strategy_name, version, fast_period, slow_period,
       rsi_period, rsi_oversold, rsi_overbought, ending_balance, profit_loss, return_percent,
       win_rate, max_drawdown, total_trades, buy_count, sell_count, best_trade, worst_trade,
       average_win, average_loss, profit_factor, average_trade, average_holding_seconds,
       expectancy, trades_per_day, churn_ratio, sharpe_ratio, sortino_ratio,
       execution_fill_mode, position_sizing_mode, position_size_value, trend_filter_enabled, trend_period, cooldown_bars,
       min_holding_bars, benchmark_ending_balance, benchmark_profit_loss,
       benchmark_return_percent, excess_return_percent, validation_status, validation_reason,
       train_return_percent, train_excess_return_percent, train_profit_factor, train_max_drawdown,
       train_total_trades, train_validation_status, train_validation_reason,
       test_return_percent, test_excess_return_percent, test_profit_factor, test_max_drawdown,
       test_total_trades, test_validation_status, test_validation_reason,
       walk_forward_folds, walk_forward_passes, walk_forward_average_return,
       walk_forward_average_excess, walk_forward_worst_drawdown,
       walk_forward_validation_status, walk_forward_validation_reason, open_position
FROM strategy_comparison_results`
}
