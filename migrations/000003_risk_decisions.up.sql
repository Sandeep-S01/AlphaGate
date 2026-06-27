DROP TABLE IF EXISTS risk_decisions;

CREATE TABLE risk_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id UUID NOT NULL REFERENCES strategy_signals(id),
    symbol TEXT NOT NULL,
    signal_side TEXT NOT NULL,
    decision TEXT NOT NULL,
    reason TEXT NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL,
    risk_snapshot_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_risk_decisions_signal_side CHECK (signal_side IN ('buy', 'sell', 'hold')),
    CONSTRAINT ck_risk_decisions_decision CHECK (decision IN ('approved', 'rejected'))
);

CREATE INDEX idx_risk_decisions_symbol_evaluated_at_desc
    ON risk_decisions (symbol, evaluated_at DESC);

CREATE INDEX idx_risk_decisions_signal_id
    ON risk_decisions (signal_id);
