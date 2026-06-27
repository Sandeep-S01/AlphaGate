ALTER TABLE strategy_comparison_results
    DROP COLUMN IF EXISTS excess_return_percent,
    DROP COLUMN IF EXISTS benchmark_return_percent,
    DROP COLUMN IF EXISTS benchmark_profit_loss,
    DROP COLUMN IF EXISTS benchmark_ending_balance,
    DROP COLUMN IF EXISTS position_size_value,
    DROP COLUMN IF EXISTS position_sizing_mode;

ALTER TABLE strategy_comparisons
    DROP COLUMN IF EXISTS position_size_value,
    DROP COLUMN IF EXISTS position_sizing_mode;

ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS excess_return_percent,
    DROP COLUMN IF EXISTS benchmark_return_percent,
    DROP COLUMN IF EXISTS benchmark_profit_loss,
    DROP COLUMN IF EXISTS benchmark_ending_balance,
    DROP COLUMN IF EXISTS position_size_value,
    DROP COLUMN IF EXISTS position_sizing_mode;
