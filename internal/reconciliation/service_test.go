package reconciliation

import (
	"context"
	"testing"
	"time"

	"sentra/internal/execution"
	"sentra/internal/strategy"
)

func TestServiceDetectsBalanceMismatch(t *testing.T) {
	store := &fakeStore{}
	service := NewService(Dependencies{
		Internal: fakeSnapshotReader{snapshot: Snapshot{
			Balances: []Balance{{Asset: "USDT", Free: 1000}},
		}},
		External: fakeSnapshotReader{snapshot: Snapshot{
			Balances: []Balance{{Asset: "USDT", Free: 998}},
		}},
		Store: store,
		Now:   func() time.Time { return time.Unix(100, 0).UTC() },
	})

	run, err := service.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if run.Status != StatusMismatch {
		t.Fatalf("expected mismatch status, got %+v", run)
	}
	if len(run.Mismatches) != 1 || run.Mismatches[0].Kind != MismatchBalance {
		t.Fatalf("expected balance mismatch, got %+v", run.Mismatches)
	}
	if len(store.saved) != 1 {
		t.Fatalf("expected saved reconciliation run, got %d", len(store.saved))
	}
}

func TestPaperSnapshotReaderBuildsBalancesAndOpenOrders(t *testing.T) {
	reader := NewPaperSnapshotReader(
		fakeAccountReader{account: execution.Account{
			BaseAsset:    "BTC",
			QuoteAsset:   "USDT",
			BaseBalance:  0.25,
			QuoteBalance: 1000,
		}},
		fakeOrderReader{orders: []execution.Order{
			{ClientOrderID: "client-open", Symbol: "BTCUSDT", Status: execution.OrderStatusSubmitted},
			{ClientOrderID: "client-filled", Symbol: "BTCUSDT", Status: execution.OrderStatusFilled},
		}},
		"BTCUSDT",
	)

	snapshot, err := reader.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}

	if len(snapshot.Balances) != 2 {
		t.Fatalf("expected base and quote balances, got %+v", snapshot.Balances)
	}
	if len(snapshot.Positions) != 1 || snapshot.Positions[0].Symbol != "BTCUSDT" {
		t.Fatalf("expected BTC position, got %+v", snapshot.Positions)
	}
	if len(snapshot.Orders) != 1 || snapshot.Orders[0].ClientOrderID != "client-open" {
		t.Fatalf("expected only open order in snapshot, got %+v", snapshot.Orders)
	}
}

func TestServiceDetectsMissingOpenOrderMismatch(t *testing.T) {
	service := NewService(Dependencies{
		Internal: fakeSnapshotReader{snapshot: Snapshot{
			Orders: []Order{{ClientOrderID: "client-open", Symbol: "BTCUSDT", Status: string(execution.OrderStatusSubmitted)}},
		}},
		External: fakeSnapshotReader{snapshot: Snapshot{}},
		Now:      func() time.Time { return time.Unix(100, 0).UTC() },
	})

	run, err := service.Run(context.Background())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if run.Status != StatusMismatch {
		t.Fatalf("expected mismatch status, got %+v", run)
	}
	if len(run.Mismatches) != 1 || run.Mismatches[0].Kind != MismatchOrder {
		t.Fatalf("expected order mismatch, got %+v", run.Mismatches)
	}
}

type fakeSnapshotReader struct {
	snapshot Snapshot
}

func (f fakeSnapshotReader) Snapshot(ctx context.Context) (Snapshot, error) {
	return f.snapshot, nil
}

type fakeStore struct {
	saved []Run
}

type fakeAccountReader struct {
	account execution.Account
}

func (f fakeAccountReader) Get(ctx context.Context) (execution.Account, error) {
	return f.account, nil
}

type fakeOrderReader struct {
	orders []execution.Order
}

func (f fakeOrderReader) ListOrders(ctx context.Context, query execution.Query) ([]execution.Order, error) {
	for index := range f.orders {
		if f.orders[index].Side == "" {
			f.orders[index].Side = strategy.SideBuy
		}
	}
	return f.orders, nil
}

func (f *fakeStore) Save(ctx context.Context, run Run) (Run, error) {
	run.ID = "reconciliation-1"
	f.saved = append(f.saved, run)
	return run, nil
}
