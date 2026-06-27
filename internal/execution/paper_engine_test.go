package execution

import (
	"strings"
	"testing"
	"time"

	"sentra/internal/risk"
	"sentra/internal/strategy"
)

func TestPaperEngineExecutesApprovedBuy(t *testing.T) {
	engine := NewPaperEngine(Config{
		Enabled:          true,
		Symbol:           "BTCUSDT",
		BaseAsset:        "BTC",
		QuoteAsset:       "USDT",
		QuoteOrderAmount: 100,
		FeeRate:          0.001,
	})
	account := Account{BaseBalance: 0, QuoteBalance: 1000}

	result, updated, err := engine.Execute(approvedDecision(strategy.SideBuy), 50000, account)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Order.Status != OrderStatusFilled {
		t.Fatalf("expected filled order, got %q", result.Order.Status)
	}
	if got := orderEventStatuses(result.OrderEvents); got != "created,submitted,filled" {
		t.Fatalf("expected lifecycle events created,submitted,filled, got %s", got)
	}
	if result.Order.Side != strategy.SideBuy {
		t.Fatalf("expected buy side, got %q", result.Order.Side)
	}
	if result.Trade.Quantity != 0.001998 {
		t.Fatalf("expected quantity 0.001998, got %.8f", result.Trade.Quantity)
	}
	if updated.QuoteBalance != 900 {
		t.Fatalf("expected quote balance 900, got %.8f", updated.QuoteBalance)
	}
	if updated.BaseBalance != 0.001998 {
		t.Fatalf("expected base balance 0.001998, got %.8f", updated.BaseBalance)
	}
}

func TestPaperEngineCanSimulatePartialFill(t *testing.T) {
	engine := NewPaperEngine(Config{
		Enabled:          true,
		Symbol:           "BTCUSDT",
		BaseAsset:        "BTC",
		QuoteAsset:       "USDT",
		QuoteOrderAmount: 100,
		FeeRate:          0.001,
		FillRatio:        0.5,
	})
	account := Account{BaseBalance: 0, QuoteBalance: 1000}

	result, updated, err := engine.Execute(approvedDecision(strategy.SideBuy), 50000, account)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Order.Status != OrderStatusPartiallyFilled {
		t.Fatalf("expected partial fill status, got %q", result.Order.Status)
	}
	if result.Order.FilledQuantity != result.Trade.Quantity {
		t.Fatalf("expected order fill quantity to match trade, got %+v", result)
	}
	if updated.QuoteBalance != 950 {
		t.Fatalf("expected only filled quote amount deducted, got %.8f", updated.QuoteBalance)
	}
	if got := orderEventStatuses(result.OrderEvents); got != "created,submitted,partially_filled" {
		t.Fatalf("expected lifecycle events created,submitted,partially_filled, got %s", got)
	}
}

func TestPaperEngineExecutesApprovedSell(t *testing.T) {
	engine := NewPaperEngine(Config{
		Enabled:          true,
		Symbol:           "BTCUSDT",
		BaseAsset:        "BTC",
		QuoteAsset:       "USDT",
		QuoteOrderAmount: 100,
		FeeRate:          0.001,
	})
	account := Account{BaseBalance: 0.01, QuoteBalance: 1000}

	result, updated, err := engine.Execute(approvedDecision(strategy.SideSell), 50000, account)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Order.Side != strategy.SideSell {
		t.Fatalf("expected sell side, got %q", result.Order.Side)
	}
	if result.Trade.Quantity != 0.002 {
		t.Fatalf("expected quantity 0.002, got %.8f", result.Trade.Quantity)
	}
	if updated.BaseBalance != 0.008 {
		t.Fatalf("expected base balance 0.008, got %.8f", updated.BaseBalance)
	}
	if updated.QuoteBalance != 1099.9 {
		t.Fatalf("expected quote balance 1099.9, got %.8f", updated.QuoteBalance)
	}
}

func TestPaperEngineRejectsNonApprovedDecision(t *testing.T) {
	engine := NewPaperEngine(Config{Enabled: true, QuoteOrderAmount: 100})

	_, _, err := engine.Execute(risk.Decision{Decision: risk.DecisionRejected}, 50000, Account{QuoteBalance: 1000})
	if err == nil {
		t.Fatal("expected rejected risk decision to fail")
	}
}

func TestPaperEngineRejectsInsufficientQuoteBalance(t *testing.T) {
	engine := NewPaperEngine(Config{
		Enabled:          true,
		QuoteOrderAmount: 100,
		FeeRate:          0.001,
	})

	_, _, err := engine.Execute(approvedDecision(strategy.SideBuy), 50000, Account{QuoteBalance: 50})
	if err == nil {
		t.Fatal("expected insufficient quote balance to fail")
	}
}

func approvedDecision(side strategy.Side) risk.Decision {
	return risk.Decision{
		ID:          "risk-1",
		SignalID:    "signal-1",
		Symbol:      "BTCUSDT",
		SignalSide:  side,
		Decision:    risk.DecisionApproved,
		Reason:      "approved",
		EvaluatedAt: time.Unix(10, 0).UTC(),
	}
}

func orderEventStatuses(events []OrderEvent) string {
	statuses := make([]string, 0, len(events))
	for _, event := range events {
		statuses = append(statuses, string(event.Status))
	}
	return strings.Join(statuses, ",")
}
