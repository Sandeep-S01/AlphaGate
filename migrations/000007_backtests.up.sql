CREATE TABLE backtest_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_name TEXT NOT NULL,
    version TEXT NOT NULL,
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    from_time TIMESTAMPTZ NOT NULL,
    to_time TIMESTAMPTZ NOT NULL,
    fast_period INTEGER NOT NULL CHECK (fast_period > 0),
    slow_period INTEGER NOT NULL CHECK (slow_period > fast_period),
    starting_balance NUMERIC(30, 12) NOT NULL CHECK (starting_balance > 0),
    ending_balance NUMERIC(30, 12) NOT NULL,
    profit_loss NUMERIC(30, 12) NOT NULL,
    return_percent NUMERIC(30, 12) NOT NULL,
    win_rate NUMERIC(30, 12) NOT NULL,
    max_drawdown NUMERIC(30, 12) NOT NULL,
    total_trades INTEGER NOT NULL,
    fee_rate NUMERIC(18, 12) NOT NULL CHECK (fee_rate >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    request_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX idx_backtest_runs_symbol_created_at ON backtest_runs (symbol, created_at DESC);

CREATE TABLE backtest_trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    quantity NUMERIC(30, 12) NOT NULL,
    price NUMERIC(30, 12) NOT NULL,
    quote_amount NUMERIC(30, 12) NOT NULL,
    fee NUMERIC(30, 12) NOT NULL,
    equity NUMERIC(30, 12) NOT NULL,
    executed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_backtest_trades_run_executed_at ON backtest_trades (run_id, executed_at ASC);
