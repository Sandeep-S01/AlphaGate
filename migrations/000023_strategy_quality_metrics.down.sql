ALTER TABLE strategy_comparison_results
    DROP COLUMN IF EXISTS sortino_ratio,
    DROP COLUMN IF EXISTS sharpe_ratio,
    DROP COLUMN IF EXISTS churn_ratio,
    DROP COLUMN IF EXISTS trades_per_day,
    DROP COLUMN IF EXISTS expectancy;

ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS sortino_ratio,
    DROP COLUMN IF EXISTS sharpe_ratio,
    DROP COLUMN IF EXISTS churn_ratio,
    DROP COLUMN IF EXISTS trades_per_day,
    DROP COLUMN IF EXISTS expectancy;
