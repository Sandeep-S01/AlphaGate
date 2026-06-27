package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEnvelopeSetsRequiredMetadata(t *testing.T) {
	payload := map[string]string{"symbol": "BTCUSDT"}
	occurredAt := time.Date(2026, 6, 16, 10, 30, 0, 0, time.UTC)

	envelope := NewEnvelope("market.candle.updated", "market-data", "corr-1", occurredAt, payload)

	if envelope.ID == "" {
		t.Fatal("expected event ID")
	}
	if envelope.Type != "market.candle.updated" {
		t.Fatalf("expected event type, got %q", envelope.Type)
	}
	if envelope.Source != "market-data" {
		t.Fatalf("expected source market-data, got %q", envelope.Source)
	}
	if envelope.CorrelationID != "corr-1" {
		t.Fatalf("expected correlation ID corr-1, got %q", envelope.CorrelationID)
	}
	if envelope.Version != 1 {
		t.Fatalf("expected version 1, got %d", envelope.Version)
	}
	if !envelope.OccurredAt.Equal(occurredAt) {
		t.Fatalf("expected occurred_at %s, got %s", occurredAt, envelope.OccurredAt)
	}
}

func TestEnvelopeRoundTripJSON(t *testing.T) {
	envelope := NewEnvelope("market.candle.updated", "market-data", "", time.Unix(10, 0).UTC(), map[string]string{
		"symbol": "BTCUSDT",
	})

	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded Envelope
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded.Type != envelope.Type || decoded.Source != envelope.Source || decoded.Version != envelope.Version {
		t.Fatalf("decoded envelope mismatch: %+v", decoded)
	}
}
