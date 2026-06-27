package marketdata

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type KlineSubscriber interface {
	SubscribeKlines(ctx context.Context, symbol string, interval string) (<-chan Candle, error)
}

type CandlePublisher interface {
	PublishCandle(ctx context.Context, stream string, candle Candle) error
}

type CollectorConfig struct {
	Symbol         string
	Interval       string
	RedisStream    string
	MaxReconnects  int
	ReconnectDelay time.Duration
}

type Collector struct {
	subscriber KlineSubscriber
	publisher  CandlePublisher
	cfg        CollectorConfig
}

func NewCollector(subscriber KlineSubscriber, publisher CandlePublisher, cfg CollectorConfig) *Collector {
	return &Collector{
		subscriber: subscriber,
		publisher:  publisher,
		cfg:        cfg,
	}
}

func (c *Collector) Run(ctx context.Context) error {
	reconnects := 0
	for {
		candles, err := c.subscriber.SubscribeKlines(ctx, c.cfg.Symbol, c.cfg.Interval)
		if err != nil {
			if reconnects >= c.cfg.MaxReconnects {
				return fmt.Errorf("subscribe klines: %w", err)
			}
			reconnects++
			if err := c.waitBeforeReconnect(ctx); err != nil {
				return err
			}
			continue
		}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case candle, ok := <-candles:
				if !ok {
					if reconnects >= c.cfg.MaxReconnects {
						return nil
					}
					reconnects++
					if err := c.waitBeforeReconnect(ctx); err != nil {
						return err
					}
					goto reconnect
				}
				if err := c.publishWithRetry(ctx, candle); err != nil {
					return err
				}
			}
		}
	reconnect:
	}
}

func (c *Collector) publishWithRetry(ctx context.Context, candle Candle) error {
	attempts := c.cfg.MaxReconnects + 1
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := c.publisher.PublishCandle(ctx, c.cfg.RedisStream, candle); err != nil {
			lastErr = err
			slog.Warn("market data candle publish failed", "stream", c.cfg.RedisStream, "symbol", candle.Symbol, "interval", candle.Interval, "open_time", candle.OpenTime, "attempt", attempt, "error", err)
			if attempt == attempts {
				break
			}
			if err := c.waitBeforeReconnect(ctx); err != nil {
				return err
			}
			continue
		}
		slog.Debug("market data candle published", "stream", c.cfg.RedisStream, "symbol", candle.Symbol, "interval", candle.Interval, "open_time", candle.OpenTime, "closed", candle.IsClosed)
		return nil
	}
	return fmt.Errorf("publish candle: %w", lastErr)
}

func (c *Collector) waitBeforeReconnect(ctx context.Context) error {
	if c.cfg.ReconnectDelay <= 0 {
		return nil
	}

	timer := time.NewTimer(c.cfg.ReconnectDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
