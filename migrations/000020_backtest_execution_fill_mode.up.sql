ALTER TABLE backtest_runs
    ADD COLUMN execution_fill_mode TEXT NOT NULL DEFAULT 'same_close';
