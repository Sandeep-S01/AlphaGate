package streams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"sentra/internal/platform/events"
)

type Handler func(ctx context.Context, envelope events.Envelope) error

type ConsumerConfig struct {
	Stream         string
	Group          string
	Consumer       string
	Block          time.Duration
	Count          int64
	ReadRetryDelay time.Duration
}

type redisStreamClient interface {
	XGroupCreateMkStream(ctx context.Context, stream string, group string, start string) *redis.StatusCmd
	XReadGroup(ctx context.Context, a *redis.XReadGroupArgs) *redis.XStreamSliceCmd
	XAck(ctx context.Context, stream string, group string, ids ...string) *redis.IntCmd
}

type Consumer struct {
	client redisStreamClient
	cfg    ConsumerConfig
}

func NewConsumer(client redisStreamClient, cfg ConsumerConfig) *Consumer {
	if cfg.Block == 0 {
		cfg.Block = 5 * time.Second
	}
	if cfg.Count == 0 {
		cfg.Count = 10
	}
	if cfg.ReadRetryDelay == 0 {
		cfg.ReadRetryDelay = time.Second
	}
	return &Consumer{client: client, cfg: cfg}
}

func (c *Consumer) EnsureGroup(ctx context.Context) error {
	err := c.client.XGroupCreateMkStream(ctx, c.cfg.Stream, c.cfg.Group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create consumer group: %w", err)
	}
	return nil
}

func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	if err := c.EnsureGroup(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.cfg.Group,
			Consumer: c.cfg.Consumer,
			Streams:  []string{c.cfg.Stream, ">"},
			Count:    c.cfg.Count,
			Block:    c.cfg.Block,
		}).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			if waitErr := c.waitBeforeReadRetry(ctx); waitErr != nil {
				return waitErr
			}
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				envelope, err := decodeEnvelope(message)
				if err != nil {
					return err
				}
				if err := handler(ctx, envelope); err != nil {
					return err
				}
				if err := c.client.XAck(ctx, c.cfg.Stream, c.cfg.Group, message.ID).Err(); err != nil {
					return fmt.Errorf("ack stream message: %w", err)
				}
			}
		}
	}
}

func (c *Consumer) waitBeforeReadRetry(ctx context.Context) error {
	if c.cfg.ReadRetryDelay <= 0 {
		return nil
	}
	timer := time.NewTimer(c.cfg.ReadRetryDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func decodeEnvelope(message redis.XMessage) (events.Envelope, error) {
	raw, ok := message.Values["payload"].(string)
	if !ok {
		return events.Envelope{}, fmt.Errorf("stream message %s missing payload", message.ID)
	}

	var envelope events.Envelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return events.Envelope{}, fmt.Errorf("decode event envelope: %w", err)
	}
	return envelope, nil
}
