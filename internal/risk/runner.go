package risk

import (
	"context"
	"fmt"

	"sentra/internal/strategy"
)

type SignalReader interface {
	Latest(ctx context.Context, symbol string) (strategy.Signal, error)
}

type DecisionStore interface {
	Save(ctx context.Context, decision Decision) (string, error)
}

type DecisionPublisher interface {
	PublishDecision(ctx context.Context, stream string, decision Decision) error
}

type RunnerConfig struct {
	Symbol         string
	DecisionStream string
	Evaluator      *Evaluator
}

type Runner struct {
	signals   SignalReader
	store     DecisionStore
	publisher DecisionPublisher
	cfg       RunnerConfig
}

func NewRunner(signals SignalReader, store DecisionStore, publisher DecisionPublisher, cfg RunnerConfig) *Runner {
	return &Runner{
		signals:   signals,
		store:     store,
		publisher: publisher,
		cfg:       cfg,
	}
}

func (r *Runner) RunOnce(ctx context.Context) (Decision, error) {
	signal, err := r.signals.Latest(ctx, r.cfg.Symbol)
	if err != nil {
		return Decision{}, fmt.Errorf("read latest signal: %w", err)
	}

	decision := r.cfg.Evaluator.Evaluate(signal)
	id, err := r.store.Save(ctx, decision)
	if err != nil {
		return Decision{}, fmt.Errorf("save risk decision: %w", err)
	}
	decision.ID = id

	if r.publisher != nil && r.cfg.DecisionStream != "" {
		if err := r.publisher.PublishDecision(ctx, r.cfg.DecisionStream, decision); err != nil {
			return Decision{}, fmt.Errorf("publish risk decision: %w", err)
		}
	}

	return decision, nil
}
