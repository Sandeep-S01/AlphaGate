ALTER TABLE backtest_runs
    DROP COLUMN IF EXISTS rsi_overbought,
    DROP COLUMN IF EXISTS rsi_oversold,
    DROP COLUMN IF EXISTS rsi_period;

DELETE FROM strategy_settings
WHERE strategy_name = 'rsi-mean-reversion';

ALTER TABLE strategy_settings
    DROP COLUMN IF EXISTS rsi_overbought,
    DROP COLUMN IF EXISTS rsi_oversold,
    DROP COLUMN IF EXISTS rsi_period;
