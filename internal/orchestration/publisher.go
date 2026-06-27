package orchestration

import (
	"context"

	"github.com/redis/go-redis/v9"

	"sentra/internal/execution"
	"sentra/internal/platform/events"
	"sentra/internal/platform/streams"
	"sentra/internal/risk"
	"sentra/internal/strategy"
)

type RedisPublisher struct {
	producer *streams.Producer
}

func NewRedisPublisher(client redis.Cmdable) *RedisPublisher {
	return &RedisPublisher{producer: streams.NewProducer(client)}
}

func (p *RedisPublisher) PublishSignal(ctx context.Context, stream string, correlationID string, signal strategy.Signal) error {
	envelope := events.NewEnvelope("strategy.signal.generated", "orchestration", correlationID, signal.GeneratedAt, signal)
	return p.producer.Publish(ctx, stream, envelope)
}

func (p *RedisPublisher) PublishDecision(ctx context.Context, stream string, correlationID string, decision risk.Decision) error {
	envelope := events.NewEnvelope("risk.decision.created", "orchestration", correlationID, decision.EvaluatedAt, decision)
	return p.producer.Publish(ctx, stream, envelope)
}

func (p *RedisPublisher) PublishExecution(ctx context.Context, stream string, correlationID string, result execution.Result) error {
	envelope := events.NewEnvelope("execution.paper_order.filled", "orchestration", correlationID, result.Order.CreatedAt, result)
	return p.producer.Publish(ctx, stream, envelope)
}
