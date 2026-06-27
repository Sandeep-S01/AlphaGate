CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_users_email UNIQUE (email)
);

CREATE TABLE exchange_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    exchange_name TEXT NOT NULL,
    account_label TEXT NOT NULL,
    api_key_ref TEXT,
    api_secret_ref TEXT,
    mode TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'inactive',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_exchange_accounts_mode CHECK (mode IN ('paper', 'live'))
);

CREATE TABLE symbols (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange TEXT NOT NULL,
    base_asset TEXT NOT NULL,
    quote_asset TEXT NOT NULL,
    symbol TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    price_precision INTEGER NOT NULL DEFAULT 0,
    quantity_precision INTEGER NOT NULL DEFAULT 0,
    min_notional NUMERIC(28, 12) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_symbols_exchange_symbol UNIQUE (exchange, symbol)
);

CREATE TABLE candles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    open_time TIMESTAMPTZ NOT NULL,
    close_time TIMESTAMPTZ NOT NULL,
    open NUMERIC(28, 12) NOT NULL,
    high NUMERIC(28, 12) NOT NULL,
    low NUMERIC(28, 12) NOT NULL,
    close NUMERIC(28, 12) NOT NULL,
    volume NUMERIC(28, 12) NOT NULL,
    is_closed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_candles_exchange_symbol_interval_open_time UNIQUE (exchange, symbol, interval, open_time)
);

CREATE INDEX idx_candles_symbol_interval_open_time_desc
    ON candles (symbol, interval, open_time DESC);

CREATE TABLE strategies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'disabled',
    parameters_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id UUID NOT NULL REFERENCES strategies(id),
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id),
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    strength NUMERIC(10, 6),
    reason TEXT NOT NULL,
    input_snapshot_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'generated',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_signals_side CHECK (side IN ('buy', 'sell', 'hold'))
);

CREATE INDEX idx_signals_strategy_created_at_desc ON signals (strategy_id, created_at DESC);
CREATE INDEX idx_signals_symbol_created_at_desc ON signals (symbol, created_at DESC);

CREATE TABLE risk_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id UUID NOT NULL REFERENCES signals(id),
    decision TEXT NOT NULL,
    reason TEXT NOT NULL,
    risk_snapshot_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_risk_decisions_decision CHECK (decision IN ('approved', 'rejected'))
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id),
    signal_id UUID REFERENCES signals(id),
    client_order_id TEXT NOT NULL,
    exchange_order_id TEXT,
    mode TEXT NOT NULL,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    type TEXT NOT NULL,
    quantity NUMERIC(28, 12) NOT NULL,
    price NUMERIC(28, 12),
    status TEXT NOT NULL DEFAULT 'created',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at TIMESTAMPTZ,
    filled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_orders_client_order_id UNIQUE (client_order_id),
    CONSTRAINT ck_orders_mode CHECK (mode IN ('paper', 'live')),
    CONSTRAINT ck_orders_side CHECK (side IN ('buy', 'sell'))
);

CREATE INDEX idx_orders_exchange_order_id ON orders (exchange_order_id);
CREATE INDEX idx_orders_exchange_account_created_at_desc ON orders (exchange_account_id, created_at DESC);
CREATE INDEX idx_orders_symbol_created_at_desc ON orders (symbol, created_at DESC);

CREATE TABLE trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    exchange_trade_id TEXT,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    quantity NUMERIC(28, 12) NOT NULL,
    price NUMERIC(28, 12) NOT NULL,
    fee NUMERIC(28, 12) NOT NULL DEFAULT 0,
    fee_asset TEXT,
    executed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_trades_side CHECK (side IN ('buy', 'sell'))
);

CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id),
    symbol TEXT NOT NULL,
    quantity NUMERIC(28, 12) NOT NULL DEFAULT 0,
    average_entry_price NUMERIC(28, 12) NOT NULL DEFAULT 0,
    realized_pnl NUMERIC(28, 12) NOT NULL DEFAULT 0,
    unrealized_pnl NUMERIC(28, 12) NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'open',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_positions_account_symbol UNIQUE (exchange_account_id, symbol)
);

CREATE TABLE balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id),
    asset TEXT NOT NULL,
    available NUMERIC(28, 12) NOT NULL DEFAULT 0,
    locked NUMERIC(28, 12) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_balances_account_asset UNIQUE (exchange_account_id, asset)
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    entity_type TEXT,
    entity_id UUID,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_created_at_desc ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_logs_action_created_at_desc ON audit_logs (action, created_at DESC);

CREATE TABLE system_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    source_module TEXT NOT NULL,
    correlation_id UUID,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_system_events_type_created_at_desc ON system_events (event_type, created_at DESC);
CREATE INDEX idx_system_events_correlation_id ON system_events (correlation_id);
