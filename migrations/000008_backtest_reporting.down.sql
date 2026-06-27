ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS open_position,
    DROP COLUMN IF EXISTS average_loss,
    DROP COLUMN IF EXISTS average_win,
    DROP COLUMN IF EXISTS worst_trade,
    DROP COLUMN IF EXISTS best_trade,
    DROP COLUMN IF EXISTS sell_count,
    DROP COLUMN IF EXISTS buy_count;
