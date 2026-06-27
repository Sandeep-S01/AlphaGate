package execution

import (
	"context"
	"testing"

	"sentra/internal/risk"
	"sentra/internal/strategy"
)

func TestRunnerExecutesLatestApprovedRiskDecision(t *testing.T) {
	decisions := &fakeDecisionReader{decision: approvedDecision(strategy.SideBuy)}
	prices := &fakePriceReader{price: 50000}
	accounts := &fakeAccountStore{account: Account{BaseBalance: 0, QuoteBalance: 1000}}
	store := &fakeExecutionStore{}
	publisher := &fakeExecutionPublisher{}
	runner := NewRunner(decisions, prices, accounts, store, publisher, RunnerConfig{
		Symbol:          "BTCUSDT",
		Interval:        "1m",
		ExecutionStream: "stream:execution-results",
		Engine: NewPaperEngine(Config{
			Enabled:          true,
			Symbol:           "BTCUSDT",
			BaseAsset:        "BTC",
			QuoteAsset:       "USDT",
			QuoteOrderAmount: 100,
			FeeRate:          0.001,
		}),
	})

	result, err := runner.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	if result.Order.Status != OrderStatusFilled {
		t.Fatalf("expected filled order, got %q", result.Order.Status)
	}
	if len(store.saved) != 1 {
		t.Fatalf("expected 1 saved execution, got %d", len(store.saved))
	}
	if accounts.saved.QuoteBalance != 900 {
		t.Fatalf("expected saved quote balance 900, got %.8f", accounts.saved.QuoteBalance)
	}
	if publisher.stream != "stream:execution-results" {
		t.Fatalf("expected execution stream, got %q", publisher.stream)
	}
}

type fakeDecisionReader struct {
	decision risk.Decision
}

func (f *fakeDecisionReader) LatestApproved(ctx context.Context, symbol string) (risk.Decision, error) {
	return f.decision, nil
}

type fakePriceReader struct {
	price float64
}

func (f *fakePriceReader) LatestPrice(ctx context.Context, symbol string, interval string) (float64, error) {
	return f.price, nil
}

type fakeAccountStore struct {
	account Account
	saved   Account
}

func (f *fakeAccountStore) Get(ctx context.Context) (Account, error) {
	return f.account, nil
}

func (f *fakeAccountStore) Save(ctx context.Context, account Account) error {
	f.saved = account
	return nil
}

type fakeExecutionStore struct {
	saved []Result
}

func (f *fakeExecutionStore) Save(ctx context.Context, result Result) (string, string, error) {
	f.saved = append(f.saved, result)
	return "order-1", "trade-1", nil
}

type fakeExecutionPublisher struct {
	stream string
}

func (f *fakeExecutionPublisher) PublishExecution(ctx context.Context, stream string, result Result) error {
	f.stream = stream
	return nil
}
