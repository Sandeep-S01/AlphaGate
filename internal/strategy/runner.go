package strategy

import (
	"context"
	"fmt"

	"sentra/internal/marketdata"
)

type CandleReader interface {
	List(ctx context.Context, query marketdata.CandleQuery) ([]marketdata.Candle, error)
}

type SignalStore interface {
	Save(ctx context.Context, signal Signal) (string, error)
}

type SignalPublisher interface {
	PublishSignal(ctx context.Context, stream string, signal Signal) error
}

type RunnerConfig struct {
	Symbol        string
	Interval      string
	LookbackLimit int
	SignalStream  string
	Evaluator     Evaluator
}

type Runner struct {
	candles   CandleReader
	store     SignalStore
	publisher SignalPublisher
	cfg       RunnerConfig
}

func NewRunner(candles CandleReader, store SignalStore, publisher SignalPublisher, cfg RunnerConfig) *Runner {
	return &Runner{
		candles:   candles,
		store:     store,
		publisher: publisher,
		cfg:       cfg,
	}
}

func (r *Runner) RunOnce(ctx context.Context) (Signal, error) {
	limit := r.cfg.LookbackLimit
	if limit <= 0 {
		limit = 100
	}

	candles, err := r.candles.List(ctx, marketdata.CandleQuery{
		Symbol:   r.cfg.Symbol,
		Interval: r.cfg.Interval,
		Limit:    limit,
	})
	if err != nil {
		return Signal{}, fmt.Errorf("list candles: %w", err)
	}

	signal, err := r.cfg.Evaluator.Evaluate(candles)
	if err != nil {
		return Signal{}, fmt.Errorf("evaluate strategy: %w", err)
	}

	id, err := r.store.Save(ctx, signal)
	if err != nil {
		return Signal{}, fmt.Errorf("save signal: %w", err)
	}
	signal.ID = id

	if r.publisher != nil && r.cfg.SignalStream != "" {
		if err := r.publisher.PublishSignal(ctx, r.cfg.SignalStream, signal); err != nil {
			return Signal{}, fmt.Errorf("publish signal: %w", err)
		}
	}

	return signal, nil
}
