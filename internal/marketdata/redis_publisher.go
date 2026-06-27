package marketdata

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"sentra/internal/platform/events"
	"sentra/internal/platform/streams"
)

type RedisPublisher struct {
	producer *streams.Producer
}

func NewRedisPublisher(client redis.Cmdable) *RedisPublisher {
	return &RedisPublisher{producer: streams.NewProducer(client)}
}

func (p *RedisPublisher) PublishCandle(ctx context.Context, stream string, candle Candle) error {
	occurredAt := candle.EventTime
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	envelope := events.NewEnvelope("market.candle.updated", "market-data", "", occurredAt, candle)
	return p.producer.Publish(ctx, stream, envelope)
}
