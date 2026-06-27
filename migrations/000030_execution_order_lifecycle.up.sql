ALTER TABLE paper_orders
    DROP CONSTRAINT IF EXISTS ck_paper_orders_status;

ALTER TABLE paper_orders
    ADD COLUMN IF NOT EXISTS client_order_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS exchange_order_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS strategy_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS requested_quantity DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS filled_quantity DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS average_fill_price DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS failure_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE paper_orders
SET requested_quantity = CASE WHEN requested_quantity = 0 THEN quantity ELSE requested_quantity END,
    filled_quantity = CASE WHEN filled_quantity = 0 THEN quantity ELSE filled_quantity END,
    average_fill_price = CASE WHEN average_fill_price = 0 THEN price ELSE average_fill_price END,
    updated_at = COALESCE(updated_at, created_at);

ALTER TABLE paper_orders
    ADD CONSTRAINT ck_paper_orders_status
        CHECK (status IN ('created', 'submitted', 'partially_filled', 'filled', 'cancelled', 'failed'));

CREATE TABLE paper_order_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES paper_orders(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_paper_order_events_status CHECK (status IN ('created', 'submitted', 'partially_filled', 'filled', 'cancelled', 'failed'))
);

CREATE INDEX idx_paper_order_events_order_created_at
    ON paper_order_events (order_id, created_at);
