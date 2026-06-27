package marketdata

import (
	"context"
	"encoding/json"
	"fmt"

	"sentra/internal/platform/events"
)

type CandleStore interface {
	Upsert(ctx context.Context, candle Candle) error
}

type Persister struct {
	store CandleStore
}

func NewPersister(store CandleStore) *Persister {
	return &Persister{store: store}
}

func (p *Persister) Handle(ctx context.Context, envelope events.Envelope) error {
	if envelope.Type != "market.candle.updated" {
		return nil
	}

	payload, err := json.Marshal(envelope.Payload)
	if err != nil {
		return fmt.Errorf("encode candle payload: %w", err)
	}

	var candle Candle
	if err := json.Unmarshal(payload, &candle); err != nil {
		return fmt.Errorf("decode candle payload: %w", err)
	}

	return p.store.Upsert(ctx, candle)
}
