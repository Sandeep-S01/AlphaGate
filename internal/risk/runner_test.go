package risk

import (
	"context"
	"testing"

	"sentra/internal/strategy"
)

func TestRunnerEvaluatesLatestSignalAndPublishesDecision(t *testing.T) {
	signals := &fakeSignalReader{latest: signal(strategy.SideBuy, 2)}
	store := &fakeDecisionStore{}
	publisher := &fakeDecisionPublisher{}
	runner := NewRunner(signals, store, publisher, RunnerConfig{
		Symbol:         "BTCUSDT",
		DecisionStream: "stream:risk-decisions",
		Evaluator: NewEvaluator(Config{
			Enabled:           true,
			MaxSignalStrength: 10,
			AllowBuy:          true,
			AllowSell:         true,
		}),
	})

	decision, err := runner.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	if decision.Decision != DecisionApproved {
		t.Fatalf("expected approved, got %+v", decision)
	}
	if len(store.saved) != 1 {
		t.Fatalf("expected 1 saved decision, got %d", len(store.saved))
	}
	if publisher.stream != "stream:risk-decisions" {
		t.Fatalf("expected risk stream, got %q", publisher.stream)
	}
	if len(publisher.published) != 1 || publisher.published[0].Decision != DecisionApproved {
		t.Fatalf("unexpected published decisions: %+v", publisher.published)
	}
}

type fakeSignalReader struct {
	latest strategy.Signal
}

func (f *fakeSignalReader) Latest(ctx context.Context, symbol string) (strategy.Signal, error) {
	return f.latest, nil
}

type fakeDecisionStore struct {
	saved []Decision
}

func (f *fakeDecisionStore) Save(ctx context.Context, decision Decision) (string, error) {
	f.saved = append(f.saved, decision)
	return "risk-1", nil
}

type fakeDecisionPublisher struct {
	stream    string
	published []Decision
}

func (f *fakeDecisionPublisher) PublishDecision(ctx context.Context, stream string, decision Decision) error {
	f.stream = stream
	f.published = append(f.published, decision)
	return nil
}
