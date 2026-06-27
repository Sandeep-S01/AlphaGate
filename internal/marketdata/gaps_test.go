package marketdata

import (
	"testing"
	"time"
)

func TestDetectGapsFindsMissingRanges(t *testing.T) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	existing := []time.Time{
		start,
		start.Add(time.Minute),
		start.Add(4 * time.Minute),
	}

	gaps, err := DetectGaps(start, start.Add(5*time.Minute), "1m", existing)
	if err != nil {
		t.Fatalf("DetectGaps returned error: %v", err)
	}

	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap, got %+v", gaps)
	}
	if !gaps[0].From.Equal(start.Add(2*time.Minute)) || !gaps[0].To.Equal(start.Add(4*time.Minute)) {
		t.Fatalf("unexpected gap: %+v", gaps[0])
	}
}
