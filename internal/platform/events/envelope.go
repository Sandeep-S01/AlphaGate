package events

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Envelope struct {
	ID            string    `json:"event_id"`
	Type          string    `json:"event_type"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
	Source        string    `json:"source"`
	Version       int       `json:"version"`
	Payload       any       `json:"payload"`
}

func NewEnvelope(eventType string, source string, correlationID string, occurredAt time.Time, payload any) Envelope {
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return Envelope{
		ID:            newID(),
		Type:          eventType,
		CorrelationID: correlationID,
		OccurredAt:    occurredAt.UTC(),
		Source:        source,
		Version:       1,
		Payload:       payload,
	}
}

func newID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(bytes[:])
}
