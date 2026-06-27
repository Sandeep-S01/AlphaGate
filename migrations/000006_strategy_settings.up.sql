CREATE TABLE strategy_settings (
    strategy_name TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    fast_period INTEGER NOT NULL CHECK (fast_period > 0),
    slow_period INTEGER NOT NULL CHECK (slow_period > fast_period),
    lookback_limit INTEGER NOT NULL CHECK (lookback_limit >= slow_period),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO strategy_settings (
    strategy_name,
    version,
    symbol,
    interval,
    fast_period,
    slow_period,
    lookback_limit
) VALUES (
    'sma-crossover',
    'v1',
    'BTCUSDT',
    '1m',
    9,
    21,
    100
);
