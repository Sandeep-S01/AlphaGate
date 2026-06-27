package streams

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"

	"sentra/internal/platform/events"
)

type StreamWriter interface {
	XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd
}

type Producer struct {
	writer StreamWriter
}

func NewProducer(writer StreamWriter) *Producer {
	return &Producer{writer: writer}
}

func (p *Producer) Publish(ctx context.Context, stream string, envelope events.Envelope) error {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("encode event envelope: %w", err)
	}

	return p.writer.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{
			"event_id":   envelope.ID,
			"event_type": envelope.Type,
			"payload":    string(payload),
		},
	}).Err()
}
