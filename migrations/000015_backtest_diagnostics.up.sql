ALTER TABLE backtest_runs
    ADD COLUMN slippage_rate NUMERIC(18, 12) NOT NULL DEFAULT 0,
    ADD COLUMN winning_trades INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN losing_trades INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN profit_factor NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN average_trade NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN average_holding_seconds NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN validation_status TEXT NOT NULL DEFAULT 'insufficient_sample',
    ADD COLUMN validation_reason TEXT NOT NULL DEFAULT 'completed trades below 100';

CREATE TABLE backtest_round_trips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    entry_time TIMESTAMPTZ NOT NULL,
    exit_time TIMESTAMPTZ NOT NULL,
    entry_price NUMERIC(30, 12) NOT NULL,
    exit_price NUMERIC(30, 12) NOT NULL,
    quantity NUMERIC(30, 12) NOT NULL,
    gross_profit_loss NUMERIC(30, 12) NOT NULL,
    fees NUMERIC(30, 12) NOT NULL,
    net_profit_loss NUMERIC(30, 12) NOT NULL,
    profit_percent NUMERIC(30, 12) NOT NULL,
    holding_seconds BIGINT NOT NULL,
    entry_reason TEXT NOT NULL,
    exit_reason TEXT NOT NULL,
    CONSTRAINT ck_backtest_round_trips_time CHECK (exit_time > entry_time)
);

CREATE INDEX idx_backtest_round_trips_run_entry_time
    ON backtest_round_trips (run_id, entry_time ASC);

CREATE TABLE backtest_equity_points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    time TIMESTAMPTZ NOT NULL,
    equity NUMERIC(30, 12) NOT NULL,
    drawdown_percent NUMERIC(30, 12) NOT NULL
);

CREATE INDEX idx_backtest_equity_points_run_time
    ON backtest_equity_points (run_id, time ASC);

ALTER TABLE strategy_comparison_results
    ADD COLUMN profit_factor NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN average_trade NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN average_holding_seconds NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN validation_status TEXT NOT NULL DEFAULT 'insufficient_sample',
    ADD COLUMN validation_reason TEXT NOT NULL DEFAULT 'completed trades below 100';

ALTER TABLE strategy_comparisons
    ADD COLUMN slippage_rate NUMERIC(18, 12) NOT NULL DEFAULT 0;
