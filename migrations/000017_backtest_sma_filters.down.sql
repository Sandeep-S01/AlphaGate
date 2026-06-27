ALTER TABLE strategy_comparison_results
    DROP COLUMN IF EXISTS min_holding_bars,
    DROP COLUMN IF EXISTS cooldown_bars,
    DROP COLUMN IF EXISTS trend_period,
    DROP COLUMN IF EXISTS trend_filter_enabled;

ALTER TABLE strategy_comparisons
    DROP COLUMN IF EXISTS min_holding_bars,
    DROP COLUMN IF EXISTS cooldown_bars,
    DROP COLUMN IF EXISTS trend_period,
    DROP COLUMN IF EXISTS trend_filter_enabled;

ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS min_holding_bars,
    DROP COLUMN IF EXISTS cooldown_bars,
    DROP COLUMN IF EXISTS trend_period,
    DROP COLUMN IF EXISTS trend_filter_enabled;
