package risk

import (
	"context"

	"github.com/redis/go-redis/v9"

	"sentra/internal/platform/events"
	"sentra/internal/platform/streams"
)

type RedisDecisionPublisher struct {
	producer *streams.Producer
}

func NewRedisDecisionPublisher(client redis.Cmdable) *RedisDecisionPublisher {
	return &RedisDecisionPublisher{producer: streams.NewProducer(client)}
}

func (p *RedisDecisionPublisher) PublishDecision(ctx context.Context, stream string, decision Decision) error {
	envelope := events.NewEnvelope("risk.decision.created", "risk", "", decision.EvaluatedAt, decision)
	return p.producer.Publish(ctx, stream, envelope)
}
