CREATE TABLE strategy_activations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    comparison_id UUID NOT NULL REFERENCES strategy_comparisons(id) ON DELETE RESTRICT,
    comparison_result_id UUID NOT NULL REFERENCES strategy_comparison_results(id) ON DELETE RESTRICT,
    strategy_name TEXT NOT NULL,
    actor TEXT NOT NULL,
    activated_settings JSONB NOT NULL,
    comparison_return NUMERIC(30, 12) NOT NULL,
    comparison_drawdown NUMERIC(30, 12) NOT NULL,
    comparison_win_rate NUMERIC(30, 12) NOT NULL,
    comparison_total_trades INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_strategy_activations_created_at
    ON strategy_activations (created_at DESC);

CREATE INDEX idx_strategy_activations_strategy_created_at
    ON strategy_activations (strategy_name, created_at DESC);
