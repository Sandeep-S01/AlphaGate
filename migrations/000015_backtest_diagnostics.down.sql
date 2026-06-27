ALTER TABLE strategy_comparisons
    DROP COLUMN IF EXISTS slippage_rate;

ALTER TABLE strategy_comparison_results
    DROP COLUMN IF EXISTS validation_reason,
    DROP COLUMN IF EXISTS validation_status,
    DROP COLUMN IF EXISTS average_holding_seconds,
    DROP COLUMN IF EXISTS average_trade,
    DROP COLUMN IF EXISTS profit_factor;

DROP INDEX IF EXISTS idx_backtest_equity_points_run_time;
DROP TABLE IF EXISTS backtest_equity_points;

DROP INDEX IF EXISTS idx_backtest_round_trips_run_entry_time;
DROP TABLE IF EXISTS backtest_round_trips;

ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS validation_reason,
    DROP COLUMN IF EXISTS validation_status,
    DROP COLUMN IF EXISTS average_holding_seconds,
    DROP COLUMN IF EXISTS average_trade,
    DROP COLUMN IF EXISTS profit_factor,
    DROP COLUMN IF EXISTS losing_trades,
    DROP COLUMN IF EXISTS winning_trades,
    DROP COLUMN IF EXISTS slippage_rate;
