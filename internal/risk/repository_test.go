package risk

import (
	"strings"
	"testing"
	"time"

	"sentra/internal/strategy"
)

func TestBuildInsertDecisionSQLUsesDecisionFields(t *testing.T) {
	decision := Decision{
		SignalID:     "signal-1",
		Symbol:       "BTCUSDT",
		SignalSide:   strategy.SideBuy,
		Decision:     DecisionApproved,
		Reason:       "approved by risk rules",
		EvaluatedAt:  time.Unix(20, 0).UTC(),
		RiskSnapshot: `{"max_signal_strength":10}`,
	}

	query, args := BuildInsertDecisionSQL(decision)

	if !strings.Contains(query, "INSERT INTO risk_decisions") {
		t.Fatalf("expected risk_decisions insert, got %s", query)
	}
	if !strings.Contains(query, "RETURNING id") {
		t.Fatalf("expected RETURNING id, got %s", query)
	}
	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d", len(args))
	}
	if args[0] != "signal-1" || args[2] != string(strategy.SideBuy) || args[3] != string(DecisionApproved) {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestBuildRejectedReasonsSQLGroupsRejectedDecisions(t *testing.T) {
	query, args := BuildRejectedReasonsSQL(DecisionQuery{Symbol: "BTCUSDT", Limit: 10})

	if !strings.Contains(query, "decision = 'rejected'") {
		t.Fatalf("expected rejected filter, got %s", query)
	}
	if !strings.Contains(query, "GROUP BY reason") {
		t.Fatalf("expected reason grouping, got %s", query)
	}
	if len(args) != 2 || args[0] != "BTCUSDT" || args[1] != 10 {
		t.Fatalf("unexpected args: %#v", args)
	}
}
