package marketdata

import (
	"testing"
	"time"
)

func TestIntervalDurationParsesSupportedIntervals(t *testing.T) {
	tests := map[string]time.Duration{
		"1m": time.Minute,
		"5m": 5 * time.Minute,
		"1h": time.Hour,
		"1d": 24 * time.Hour,
		"1w": 7 * 24 * time.Hour,
	}

	for interval, want := range tests {
		got, err := IntervalDuration(interval)
		if err != nil {
			t.Fatalf("IntervalDuration(%q) returned error: %v", interval, err)
		}
		if got != want {
			t.Fatalf("IntervalDuration(%q) = %s, want %s", interval, got, want)
		}
	}
}

func TestIntervalDurationRejectsUnsupportedInterval(t *testing.T) {
	_, err := IntervalDuration("2x")
	if err == nil {
		t.Fatal("expected unsupported interval to fail")
	}
}
