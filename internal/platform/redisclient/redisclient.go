package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"sentra/internal/config"
)

func OptionsFromConfig(cfg config.RedisConfig) *redis.Options {
	return &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
}

func Connect(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(OptionsFromConfig(cfg))

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		if closeErr := client.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping redis: %w; close redis: %v", err, closeErr)
		}
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

type Pinger struct {
	client redis.Cmdable
}

func NewPinger(client redis.Cmdable) Pinger {
	return Pinger{client: client}
}

func (p Pinger) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}
