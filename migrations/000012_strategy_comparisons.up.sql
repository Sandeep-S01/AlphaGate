CREATE TABLE strategy_comparisons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    from_time TIMESTAMPTZ NOT NULL,
    to_time TIMESTAMPTZ NOT NULL,
    starting_balance NUMERIC(30, 12) NOT NULL CHECK (starting_balance > 0),
    fee_rate NUMERIC(18, 12) NOT NULL CHECK (fee_rate >= 0),
    winner_strategy TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_strategy_comparisons_symbol_created_at
    ON strategy_comparisons (symbol, created_at DESC);

CREATE TABLE strategy_comparison_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    comparison_id UUID NOT NULL REFERENCES strategy_comparisons(id) ON DELETE CASCADE,
    rank INTEGER NOT NULL CHECK (rank > 0),
    strategy_name TEXT NOT NULL,
    version TEXT NOT NULL,
    fast_period INTEGER NOT NULL CHECK (fast_period > 0),
    slow_period INTEGER NOT NULL CHECK (slow_period > fast_period),
    rsi_period INTEGER NOT NULL CHECK (rsi_period > 0),
    rsi_oversold DOUBLE PRECISION NOT NULL CHECK (rsi_oversold > 0 AND rsi_oversold < 100),
    rsi_overbought DOUBLE PRECISION NOT NULL CHECK (rsi_overbought > rsi_oversold AND rsi_overbought < 100),
    ending_balance NUMERIC(30, 12) NOT NULL,
    profit_loss NUMERIC(30, 12) NOT NULL,
    return_percent NUMERIC(30, 12) NOT NULL,
    win_rate NUMERIC(30, 12) NOT NULL,
    max_drawdown NUMERIC(30, 12) NOT NULL,
    total_trades INTEGER NOT NULL,
    buy_count INTEGER NOT NULL,
    sell_count INTEGER NOT NULL,
    best_trade NUMERIC(30, 12) NOT NULL,
    worst_trade NUMERIC(30, 12) NOT NULL,
    average_win NUMERIC(30, 12) NOT NULL,
    average_loss NUMERIC(30, 12) NOT NULL,
    open_position BOOLEAN NOT NULL
);

CREATE UNIQUE INDEX idx_strategy_comparison_results_rank
    ON strategy_comparison_results (comparison_id, rank);
