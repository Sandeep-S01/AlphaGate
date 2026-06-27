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
    'btc-trend-pullback',
    'v1',
    'BTCUSDT',
    '15m',
    21,
    200,
    201,
    14,
    30,
    70
) ON CONFLICT (strategy_name) DO NOTHING;
