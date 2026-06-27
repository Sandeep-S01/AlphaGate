CREATE TABLE safety_status (
    id INTEGER PRIMARY KEY DEFAULT 1,
    kill_switch_active BOOLEAN NOT NULL DEFAULT FALSE,
    reason TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_safety_status_singleton CHECK (id = 1)
);

INSERT INTO safety_status (id, kill_switch_active, reason, updated_by)
VALUES (1, FALSE, '', 'system')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    actor TEXT NOT NULL DEFAULT 'system',
    summary TEXT NOT NULL,
    details_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_created_at_desc
    ON audit_events (created_at DESC);

CREATE INDEX idx_audit_events_event_type_created_at_desc
    ON audit_events (event_type, created_at DESC);
