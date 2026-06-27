package risk

import (
	"testing"
	"time"

	"sentra/internal/strategy"
)

func TestEvaluatorApprovesBuyWithinLimits(t *testing.T) {
	evaluator := NewEvaluator(Config{
		Enabled:           true,
		MaxSignalStrength: 50,
		AllowBuy:          true,
		AllowSell:         true,
	})

	decision := evaluator.Evaluate(signal(strategy.SideBuy, 12.5))

	if decision.Decision != DecisionApproved {
		t.Fatalf("expected approved, got %+v", decision)
	}
	if decision.SignalSide != strategy.SideBuy {
		t.Fatalf("expected buy side, got %q", decision.SignalSide)
	}
}

func TestEvaluatorRejectsHoldSignal(t *testing.T) {
	evaluator := NewEvaluator(Config{Enabled: true, AllowBuy: true, AllowSell: true})

	decision := evaluator.Evaluate(signal(strategy.SideHold, 0))

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
	if decision.Reason == "" {
		t.Fatal("expected rejection reason")
	}
}

func TestEvaluatorRejectsDisabledRiskEngine(t *testing.T) {
	evaluator := NewEvaluator(Config{Enabled: false, AllowBuy: true, AllowSell: true})

	decision := evaluator.Evaluate(signal(strategy.SideBuy, 1))

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
}

func TestEvaluatorRejectsStrengthAboveLimit(t *testing.T) {
	evaluator := NewEvaluator(Config{
		Enabled:           true,
		MaxSignalStrength: 10,
		AllowBuy:          true,
		AllowSell:         true,
	})

	decision := evaluator.Evaluate(signal(strategy.SideBuy, 11))

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
}

func TestEvaluatorRejectsDisallowedSell(t *testing.T) {
	evaluator := NewEvaluator(Config{
		Enabled:   true,
		AllowBuy:  true,
		AllowSell: false,
	})

	decision := evaluator.Evaluate(signal(strategy.SideSell, 1))

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
}

func TestEvaluatorRejectsBelowMinimumStrength(t *testing.T) {
	evaluator := NewEvaluator(Config{
		Enabled:           true,
		MinSignalStrength: 5,
		AllowBuy:          true,
		AllowSell:         true,
	})

	decision := evaluator.Evaluate(signal(strategy.SideBuy, 2))

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
}

func TestEvaluatorRejectsQuoteAmountAboveLimit(t *testing.T) {
	evaluator := NewEvaluator(Config{
		Enabled:        true,
		AllowBuy:       true,
		AllowSell:      true,
		MaxQuoteAmount: 100,
	})

	decision := evaluator.EvaluateWithContext(signal(strategy.SideBuy, 10), Context{QuoteAmount: 150})

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
}

func TestEvaluatorRejectsSellWhenBaseBalanceIsInsufficient(t *testing.T) {
	evaluator := NewEvaluator(Config{Enabled: true, AllowBuy: true, AllowSell: true})

	decision := evaluator.EvaluateWithContext(signal(strategy.SideSell, 10), Context{
		QuoteAmount: 100,
		Price:       50000,
		BaseBalance: 0.001,
	})

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
	if decision.Reason != "base balance 0.001000 is below required 0.002000" {
		t.Fatalf("expected insufficient base balance reason, got %q", decision.Reason)
	}
}

func TestEvaluatorRejectsBuyWhenQuoteBalanceIsInsufficient(t *testing.T) {
	evaluator := NewEvaluator(Config{Enabled: true, AllowBuy: true, AllowSell: true})

	decision := evaluator.EvaluateWithContext(signal(strategy.SideBuy, 10), Context{
		QuoteAmount:  100,
		QuoteBalance: 50,
	})

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
	if decision.Reason != "quote balance 50.000000 is below required 100.000000" {
		t.Fatalf("expected insufficient quote balance reason, got %q", decision.Reason)
	}
}

func TestEvaluatorRejectsDailyTradeLimit(t *testing.T) {
	evaluator := NewEvaluator(Config{
		Enabled:        true,
		AllowBuy:       true,
		AllowSell:      true,
		MaxDailyTrades: 2,
	})

	decision := evaluator.EvaluateWithContext(signal(strategy.SideBuy, 10), Context{DailyTrades: 2})

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
}

func TestEvaluatorRejectsDailyLossLimit(t *testing.T) {
	evaluator := NewEvaluator(Config{
		Enabled:      true,
		AllowBuy:     true,
		AllowSell:    true,
		MaxDailyLoss: 50,
	})

	decision := evaluator.EvaluateWithContext(signal(strategy.SideBuy, 10), Context{DailyLoss: 75})

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
}

func TestEvaluatorRejectsCooldownWindow(t *testing.T) {
	now := time.Unix(1000, 0).UTC()
	evaluator := NewEvaluator(Config{
		Enabled:   true,
		AllowBuy:  true,
		AllowSell: true,
		Cooldown:  10 * time.Minute,
		Now:       func() time.Time { return now },
	})

	decision := evaluator.EvaluateWithContext(signal(strategy.SideBuy, 10), Context{LastTradeAt: now.Add(-5 * time.Minute)})

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
}

func TestEvaluatorRejectsDisallowedSymbol(t *testing.T) {
	evaluator := NewEvaluator(Config{
		Enabled:        true,
		AllowBuy:       true,
		AllowSell:      true,
		AllowedSymbols: []string{"ETHUSDT"},
	})

	decision := evaluator.EvaluateWithContext(signal(strategy.SideBuy, 10), Context{QuoteAmount: 50})

	if decision.Decision != DecisionRejected {
		t.Fatalf("expected rejected, got %+v", decision)
	}
	if decision.Reason == "" {
		t.Fatal("expected concrete rejection reason")
	}
}

func TestEvaluatorRejectsExposureAndOpenPositionLimits(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		context Context
	}{
		{
			name:    "max order quote amount",
			config:  Config{Enabled: true, AllowBuy: true, AllowSell: true, MaxOrderQuoteAmount: 100},
			context: Context{QuoteAmount: 150},
		},
		{
			name:    "max open positions",
			config:  Config{Enabled: true, AllowBuy: true, AllowSell: true, MaxOpenPositions: 2},
			context: Context{OpenPositions: 2, QuoteAmount: 50},
		},
		{
			name:    "max position quote",
			config:  Config{Enabled: true, AllowBuy: true, AllowSell: true, MaxPositionQuoteAmount: 500},
			context: Context{PositionQuoteAmount: 450, QuoteAmount: 75},
		},
		{
			name:    "max total exposure",
			config:  Config{Enabled: true, AllowBuy: true, AllowSell: true, MaxTotalExposureQuoteAmount: 1000},
			context: Context{TotalExposureQuoteAmount: 950, QuoteAmount: 75},
		},
	}

	for _, tt := range tests {
		decision := NewEvaluator(tt.config).EvaluateWithContext(signal(strategy.SideBuy, 10), tt.context)
		if decision.Decision != DecisionRejected {
			t.Fatalf("%s: expected rejected, got %+v", tt.name, decision)
		}
		if decision.Reason == "" {
			t.Fatalf("%s: expected rejection reason", tt.name)
		}
	}
}

func signal(side strategy.Side, strength float64) strategy.Signal {
	return strategy.Signal{
		ID:           "signal-1",
		StrategyName: "sma-crossover",
		Version:      "v1",
		Symbol:       "BTCUSDT",
		Interval:     "1m",
		Side:         side,
		Strength:     strength,
		Reason:       "test",
		GeneratedAt:  time.Unix(10, 0).UTC(),
	}
}
