CREATE TABLE risk_settings (
    id INTEGER PRIMARY KEY DEFAULT 1,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    max_signal_strength NUMERIC(18, 8) NOT NULL DEFAULT 100,
    min_signal_strength NUMERIC(18, 8) NOT NULL DEFAULT 0,
    max_quote_amount NUMERIC(30, 12) NOT NULL DEFAULT 0,
    max_daily_loss NUMERIC(30, 12) NOT NULL DEFAULT 0,
    max_daily_trades INTEGER NOT NULL DEFAULT 0,
    allow_buy BOOLEAN NOT NULL DEFAULT TRUE,
    allow_sell BOOLEAN NOT NULL DEFAULT TRUE,
    cooldown_seconds BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_risk_settings_singleton CHECK (id = 1),
    CONSTRAINT ck_risk_settings_strength CHECK (max_signal_strength >= 0 AND min_signal_strength >= 0),
    CONSTRAINT ck_risk_settings_amounts CHECK (max_quote_amount >= 0 AND max_daily_loss >= 0),
    CONSTRAINT ck_risk_settings_counts CHECK (max_daily_trades >= 0 AND cooldown_seconds >= 0)
);

INSERT INTO risk_settings (
    id,
    enabled,
    max_signal_strength,
    min_signal_strength,
    max_quote_amount,
    max_daily_loss,
    max_daily_trades,
    allow_buy,
    allow_sell,
    cooldown_seconds
) VALUES (
    1,
    TRUE,
    100,
    0,
    0,
    0,
    0,
    TRUE,
    TRUE,
    0
);
