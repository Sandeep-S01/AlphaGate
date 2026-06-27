ALTER TABLE risk_settings
    ADD COLUMN IF NOT EXISTS allowed_symbols JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS max_order_quote_amount NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_position_quote_amount NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_total_exposure_quote_amount NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_open_positions INTEGER NOT NULL DEFAULT 0;

ALTER TABLE risk_settings
    ADD CONSTRAINT ck_risk_settings_hardened_amounts
        CHECK (
            max_order_quote_amount >= 0
            AND max_position_quote_amount >= 0
            AND max_total_exposure_quote_amount >= 0
            AND max_open_positions >= 0
        );
