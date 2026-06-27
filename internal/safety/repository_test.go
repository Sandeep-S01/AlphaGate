package safety

import (
	"strings"
	"testing"
)

func TestBuildUpsertStatusSQLUsesSafetyFields(t *testing.T) {
	query, args := BuildUpsertStatusSQL(Status{
		KillSwitchActive: true,
		Reason:           "maintenance",
		UpdatedBy:        "operator",
	})

	if !strings.Contains(query, "INSERT INTO safety_status") {
		t.Fatalf("expected safety_status upsert, got %s", query)
	}
	if !strings.Contains(query, "kill_switch_active") || !strings.Contains(query, "RETURNING") {
		t.Fatalf("expected safety fields and returning clause, got %s", query)
	}
	if len(args) != 3 || args[0] != true || args[1] != "maintenance" || args[2] != "operator" {
		t.Fatalf("unexpected args: %#v", args)
	}
}
