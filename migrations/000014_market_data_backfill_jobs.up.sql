ALTER TABLE candles
    ADD COLUMN IF NOT EXISTS quote_volume NUMERIC(28, 12) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS trade_count BIGINT NOT NULL DEFAULT 0;

CREATE TABLE market_data_backfill_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol TEXT NOT NULL,
    base_interval TEXT NOT NULL,
    from_time TIMESTAMPTZ NOT NULL,
    to_time TIMESTAMPTZ NOT NULL,
    next_open_time TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    candles_inserted INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_market_data_backfill_jobs_status CHECK (status IN ('pending', 'running', 'failed', 'completed')),
    CONSTRAINT ck_market_data_backfill_jobs_range CHECK (from_time < to_time)
);

CREATE INDEX idx_market_data_backfill_jobs_symbol_created_at_desc
    ON market_data_backfill_jobs (symbol, created_at DESC);

CREATE INDEX idx_market_data_backfill_jobs_status_updated_at
    ON market_data_backfill_jobs (status, updated_at);
