CREATE TABLE pipeline_runs (
    key TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    error TEXT,
    CONSTRAINT ck_pipeline_runs_status CHECK (status IN ('processing', 'completed', 'failed'))
);

CREATE INDEX idx_pipeline_runs_status_updated_at
    ON pipeline_runs (status, updated_at DESC);
