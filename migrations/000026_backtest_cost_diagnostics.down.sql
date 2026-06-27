ALTER TABLE strategy_comparison_results
    DROP COLUMN IF EXISTS break_even_move_percent,
    DROP COLUMN IF EXISTS round_trip_cost_percent,
    DROP COLUMN IF EXISTS estimated_slippage_cost,
    DROP COLUMN IF EXISTS total_fees,
    DROP COLUMN IF EXISTS gross_profit_loss;

ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS break_even_move_percent,
    DROP COLUMN IF EXISTS round_trip_cost_percent,
    DROP COLUMN IF EXISTS estimated_slippage_cost,
    DROP COLUMN IF EXISTS total_fees,
    DROP COLUMN IF EXISTS gross_profit_loss;
