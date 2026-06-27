ALTER TABLE strategy_comparison_results
    DROP COLUMN IF EXISTS execution_fill_mode;

ALTER TABLE strategy_comparisons
    DROP COLUMN IF EXISTS execution_fill_mode;
