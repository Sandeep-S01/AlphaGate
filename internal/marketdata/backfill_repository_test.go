package marketdata

import (
	"strings"
	"testing"
	"time"
)

func TestBuildInsertBackfillJobSQLCreatesPendingProgress(t *testing.T) {
	start := time.Date(2024, 6, 18, 0, 0, 0, 0, time.UTC)
	job := BackfillJob{
		Symbol:       "BTCUSDT",
		BaseInterval: "1m",
		From:         start,
		To:           start.Add(24 * time.Hour),
		NextOpenTime: start,
		Status:       BackfillStatusPending,
	}

	query, args := BuildInsertBackfillJobSQL(job)

	if !strings.Contains(query, "INSERT INTO market_data_backfill_jobs") {
		t.Fatalf("expected insert query, got %s", query)
	}
	if !strings.Contains(query, "RETURNING") {
		t.Fatalf("expected returning clause, got %s", query)
	}
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d", len(args))
	}
}

func TestBuildSaveBackfillJobSQLPersistsProgress(t *testing.T) {
	query, args := BuildSaveBackfillJobSQL(BackfillJob{
		ID:              "job-1",
		NextOpenTime:    time.Unix(10, 0).UTC(),
		Status:          BackfillStatusRunning,
		CandlesInserted: 1000,
		LastError:       "",
	})

	if !strings.Contains(query, "UPDATE market_data_backfill_jobs") {
		t.Fatalf("expected update query, got %s", query)
	}
	if !strings.Contains(query, "candles_inserted = $4") {
		t.Fatalf("expected inserted-count update, got %s", query)
	}
	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d", len(args))
	}
}

func TestBuildListBackfillJobsSQLFiltersBySymbol(t *testing.T) {
	query, args := BuildListBackfillJobsSQL(BackfillJobQuery{Symbol: "BTCUSDT", Limit: 20})

	if !strings.Contains(query, "WHERE 1 = 1") || !strings.Contains(query, "symbol = $1") {
		t.Fatalf("expected symbol filter, got %s", query)
	}
	if !strings.Contains(query, "ORDER BY created_at DESC LIMIT $2") {
		t.Fatalf("expected ordered limit, got %s", query)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
}
