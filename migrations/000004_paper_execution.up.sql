CREATE TABLE paper_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    base_asset TEXT NOT NULL,
    quote_asset TEXT NOT NULL,
    base_balance DOUBLE PRECISION NOT NULL DEFAULT 0,
    quote_balance DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO paper_accounts (base_asset, quote_asset, base_balance, quote_balance)
VALUES ('BTC', 'USDT', 0, 10000);

CREATE TABLE paper_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    risk_decision_id UUID NOT NULL REFERENCES risk_decisions(id),
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    quote_amount DOUBLE PRECISION NOT NULL,
    fee DOUBLE PRECISION NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_paper_orders_side CHECK (side IN ('buy', 'sell')),
    CONSTRAINT ck_paper_orders_status CHECK (status IN ('filled'))
);

CREATE INDEX idx_paper_orders_symbol_created_at_desc
    ON paper_orders (symbol, created_at DESC);

CREATE TABLE paper_trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES paper_orders(id),
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    fee DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_paper_trades_side CHECK (side IN ('buy', 'sell'))
);

CREATE INDEX idx_paper_trades_symbol_created_at_desc
    ON paper_trades (symbol, created_at DESC);
