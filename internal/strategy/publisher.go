package strategy

import (
	"context"

	"github.com/redis/go-redis/v9"

	"sentra/internal/platform/events"
	"sentra/internal/platform/streams"
)

type RedisSignalPublisher struct {
	producer *streams.Producer
}

func NewRedisSignalPublisher(client redis.Cmdable) *RedisSignalPublisher {
	return &RedisSignalPublisher{producer: streams.NewProducer(client)}
}

func (p *RedisSignalPublisher) PublishSignal(ctx context.Context, stream string, signal Signal) error {
	envelope := events.NewEnvelope("strategy.signal.generated", "strategy", "", signal.GeneratedAt, signal)
	return p.producer.Publish(ctx, stream, envelope)
}
