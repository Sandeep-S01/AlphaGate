DROP TABLE IF EXISTS paper_order_events;

ALTER TABLE paper_orders
    DROP CONSTRAINT IF EXISTS ck_paper_orders_status;

ALTER TABLE paper_orders
    ADD CONSTRAINT ck_paper_orders_status CHECK (status IN ('filled'));

ALTER TABLE paper_orders
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS submitted_at,
    DROP COLUMN IF EXISTS failure_reason,
    DROP COLUMN IF EXISTS average_fill_price,
    DROP COLUMN IF EXISTS filled_quantity,
    DROP COLUMN IF EXISTS requested_quantity,
    DROP COLUMN IF EXISTS strategy_name,
    DROP COLUMN IF EXISTS exchange_order_id,
    DROP COLUMN IF EXISTS client_order_id;
