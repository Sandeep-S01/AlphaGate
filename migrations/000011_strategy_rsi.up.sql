ALTER TABLE strategy_settings
    ADD COLUMN rsi_period INTEGER NOT NULL DEFAULT 14 CHECK (rsi_period > 0),
    ADD COLUMN rsi_oversold DOUBLE PRECISION NOT NULL DEFAULT 30 CHECK (rsi_oversold > 0 AND rsi_oversold < 100),
    ADD COLUMN rsi_overbought DOUBLE PRECISION NOT NULL DEFAULT 70 CHECK (rsi_overbought > rsi_oversold AND rsi_overbought < 100);

INSERT INTO strategy_settings (
    strategy_name,
    version,
    symbol,
    interval,
    fast_period,
    slow_period,
    lookback_limit,
    rsi_period,
    rsi_oversold,
    rsi_overbought
) VALUES (
    'rsi-mean-reversion',
    'v1',
    'BTCUSDT',
    '1m',
    9,
    21,
    100,
    14,
    30,
    70
) ON CONFLICT (strategy_name) DO NOTHING;

ALTER TABLE backtest_runs
    ADD COLUMN rsi_period INTEGER NOT NULL DEFAULT 14 CHECK (rsi_period > 0),
    ADD COLUMN rsi_oversold DOUBLE PRECISION NOT NULL DEFAULT 30 CHECK (rsi_oversold > 0 AND rsi_oversold < 100),
    ADD COLUMN rsi_overbought DOUBLE PRECISION NOT NULL DEFAULT 70 CHECK (rsi_overbought > rsi_oversold AND rsi_overbought < 100);
