package orchestration

import (
	"strings"
	"testing"
)

func TestBuildBeginSQLRetriesFailedRunsOnly(t *testing.T) {
	query := BuildBeginSQL()

	if !strings.Contains(query, "INSERT INTO pipeline_runs") {
		t.Fatalf("expected pipeline_runs insert, got %s", query)
	}
	if !strings.Contains(query, "ON CONFLICT") {
		t.Fatalf("expected conflict handling, got %s", query)
	}
	if !strings.Contains(query, "status = 'failed'") {
		t.Fatalf("expected failed runs to be retryable, got %s", query)
	}
}

func TestBuildListRunsSQLUsesStatusAndLimit(t *testing.T) {
	query, args := BuildListRunsSQL(RunQuery{Status: "failed", Limit: 25})

	if !strings.Contains(query, "FROM pipeline_runs") {
		t.Fatalf("expected pipeline_runs query, got %s", query)
	}
	if !strings.Contains(query, "status = $1") {
		t.Fatalf("expected status filter, got %s", query)
	}
	if !strings.Contains(query, "LIMIT $2") {
		t.Fatalf("expected limit placeholder, got %s", query)
	}
	if len(args) != 2 || args[0] != "failed" || args[1] != 25 {
		t.Fatalf("unexpected args: %+v", args)
	}
}
