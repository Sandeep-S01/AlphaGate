ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS regime_max_atr_percent,
    DROP COLUMN IF EXISTS regime_min_atr_percent,
    DROP COLUMN IF EXISTS regime_filter_period,
    DROP COLUMN IF EXISTS regime_filter_enabled;
