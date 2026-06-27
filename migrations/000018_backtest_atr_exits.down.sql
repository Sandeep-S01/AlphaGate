ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS atr_take_profit_multiplier,
    DROP COLUMN IF EXISTS atr_stop_multiplier,
    DROP COLUMN IF EXISTS atr_period,
    DROP COLUMN IF EXISTS atr_exit_enabled;
