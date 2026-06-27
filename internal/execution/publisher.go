package execution

import (
	"context"

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

func (p *RedisPublisher) PublishExecution(ctx context.Context, stream string, result Result) error {
	envelope := events.NewEnvelope("execution.paper_order.filled", "execution", "", result.Order.CreatedAt, result)
	return p.producer.Publish(ctx, stream, envelope)
}
