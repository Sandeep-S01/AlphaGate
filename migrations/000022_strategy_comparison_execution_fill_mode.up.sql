ALTER TABLE strategy_comparisons
    ADD COLUMN execution_fill_mode TEXT NOT NULL DEFAULT 'same_close';

ALTER TABLE strategy_comparison_results
    ADD COLUMN execution_fill_mode TEXT NOT NULL DEFAULT 'same_close';
