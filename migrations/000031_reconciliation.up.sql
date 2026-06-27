CREATE TABLE reconciliation_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status TEXT NOT NULL,
    mismatches_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_reconciliation_status CHECK (status IN ('matched', 'mismatch', 'failed'))
);

CREATE INDEX idx_reconciliation_runs_created_at
    ON reconciliation_runs (created_at DESC);
