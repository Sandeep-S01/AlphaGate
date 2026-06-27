package audit

import (
	"strings"
	"testing"
	"time"
)

func TestBuildInsertEventSQLUsesAuditFields(t *testing.T) {
	query, args := BuildInsertEventSQL(Event{
		EventType:   "safety.status_changed",
		Actor:       "operator",
		Summary:     "kill switch enabled",
		DetailsJSON: `{"kill_switch_active":true}`,
	})

	if !strings.Contains(query, "INSERT INTO audit_events") {
		t.Fatalf("expected audit_events insert, got %s", query)
	}
	if !strings.Contains(query, "RETURNING id") {
		t.Fatalf("expected returning id, got %s", query)
	}
	if len(args) != 4 || args[0] != "safety.status_changed" || args[1] != "operator" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildListEventsSQLSupportsFilters(t *testing.T) {
	query, args := BuildListEventsSQL(Query{
		EventType: "safety.status_changed",
		Actor:     "operator",
		From:      time.Unix(10, 0).UTC(),
		To:        time.Unix(20, 0).UTC(),
		Limit:     25,
	})

	for _, expected := range []string{"event_type = $1", "actor = $2", "created_at >= $3", "created_at < $4", "LIMIT $5"} {
		if !strings.Contains(query, expected) {
			t.Fatalf("expected %q in query: %s", expected, query)
		}
	}
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %#v", args)
	}
}
