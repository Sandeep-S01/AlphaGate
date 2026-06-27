CREATE TABLE pine_strategies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    pine_code TEXT NOT NULL,
    converted_config JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pine_strategies_name ON pine_strategies (name);

ALTER TABLE backtest_runs ADD COLUMN pine_strategy_id UUID REFERENCES pine_strategies(id) ON DELETE SET NULL;
