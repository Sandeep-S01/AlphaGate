CREATE TABLE strategy_lifecycle (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    state TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_strategy_lifecycle_state CHECK (state IN ('DRAFT', 'BACKTESTING', 'VALIDATED', 'PAPER_TRADING', 'APPROVED', 'LIVE_ENABLED')),
    CONSTRAINT uq_strategy_lifecycle_key UNIQUE (strategy_name, symbol, interval)
);

CREATE INDEX idx_strategy_lifecycle_updated_at
    ON strategy_lifecycle (updated_at DESC);
