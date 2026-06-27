package streams

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"sentra/internal/platform/events"
)

func TestProducerPublishesEnvelopeAsSinglePayload(t *testing.T) {
	writer := &fakeStreamWriter{}
	producer := NewProducer(writer)
	envelope := events.NewEnvelope("market.candle.updated", "market-data", "corr-1", time.Unix(10, 0).UTC(), map[string]string{
		"symbol": "BTCUSDT",
	})

	if err := producer.Publish(context.Background(), "stream:market-data", envelope); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if writer.args.Stream != "stream:market-data" {
		t.Fatalf("expected stream:market-data, got %q", writer.args.Stream)
	}
	values, ok := writer.args.Values.(map[string]any)
	if !ok {
		t.Fatalf("expected values map, got %#v", writer.args.Values)
	}
	rawPayload, ok := values["payload"].(string)
	if !ok {
		t.Fatalf("expected payload string, got %#v", values["payload"])
	}

	var decoded events.Envelope
	if err := json.Unmarshal([]byte(rawPayload), &decoded); err != nil {
		t.Fatalf("payload is not an envelope: %v", err)
	}
	if decoded.ID != envelope.ID || decoded.Type != envelope.Type {
		t.Fatalf("unexpected decoded payload: %+v", decoded)
	}
}

type fakeStreamWriter struct {
	args *redis.XAddArgs
}

func (f *fakeStreamWriter) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	f.args = args
	return redis.NewStringResult("1-0", nil)
}
