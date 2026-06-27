CREATE TABLE strategy_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_name TEXT NOT NULL,
    version TEXT NOT NULL,
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    side TEXT NOT NULL,
    strength DOUBLE PRECISION NOT NULL DEFAULT 0,
    reason TEXT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_strategy_signals_side CHECK (side IN ('buy', 'sell', 'hold'))
);

CREATE INDEX idx_strategy_signals_symbol_interval_generated_at_desc
    ON strategy_signals (symbol, interval, generated_at DESC);

CREATE INDEX idx_strategy_signals_strategy_generated_at_desc
    ON strategy_signals (strategy_name, version, generated_at DESC);
