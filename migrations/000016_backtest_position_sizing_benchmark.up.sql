ALTER TABLE backtest_runs
    ADD COLUMN position_sizing_mode TEXT NOT NULL DEFAULT 'all_in',
    ADD COLUMN position_size_value NUMERIC(30, 12) NOT NULL DEFAULT 100,
    ADD COLUMN benchmark_ending_balance NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN benchmark_profit_loss NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN benchmark_return_percent NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN excess_return_percent NUMERIC(30, 12) NOT NULL DEFAULT 0;

ALTER TABLE strategy_comparisons
    ADD COLUMN position_sizing_mode TEXT NOT NULL DEFAULT 'all_in',
    ADD COLUMN position_size_value NUMERIC(30, 12) NOT NULL DEFAULT 100;

ALTER TABLE strategy_comparison_results
    ADD COLUMN position_sizing_mode TEXT NOT NULL DEFAULT 'all_in',
    ADD COLUMN position_size_value NUMERIC(30, 12) NOT NULL DEFAULT 100,
    ADD COLUMN benchmark_ending_balance NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN benchmark_profit_loss NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN benchmark_return_percent NUMERIC(30, 12) NOT NULL DEFAULT 0,
    ADD COLUMN excess_return_percent NUMERIC(30, 12) NOT NULL DEFAULT 0;
