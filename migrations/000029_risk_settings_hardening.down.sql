ALTER TABLE risk_settings
    DROP CONSTRAINT IF EXISTS ck_risk_settings_hardened_amounts;

ALTER TABLE risk_settings
    DROP COLUMN IF EXISTS max_open_positions,
    DROP COLUMN IF EXISTS max_total_exposure_quote_amount,
    DROP COLUMN IF EXISTS max_position_quote_amount,
    DROP COLUMN IF EXISTS max_order_quote_amount,
    DROP COLUMN IF EXISTS allowed_symbols;
