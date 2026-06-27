package marketdata

import (
	"context"
	"errors"
	"testing"
)

func TestCollectorPublishesCandlesFromSubscriber(t *testing.T) {
	candles := make(chan Candle, 1)
	candles <- Candle{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", Close: "100.00"}
	close(candles)

	subscriber := &fakeSubscriber{candles: candles}
	publisher := &fakePublisher{}

	collector := NewCollector(subscriber, publisher, CollectorConfig{
		Symbol:      "BTCUSDT",
		Interval:    "1m",
		RedisStream: "stream:market-data",
	})

	if err := collector.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if subscriber.symbol != "BTCUSDT" || subscriber.interval != "1m" {
		t.Fatalf("subscriber received wrong request: %s %s", subscriber.symbol, subscriber.interval)
	}
	if len(publisher.published) != 1 {
		t.Fatalf("expected 1 published candle, got %d", len(publisher.published))
	}
	if publisher.stream != "stream:market-data" {
		t.Fatalf("expected stream:market-data, got %q", publisher.stream)
	}
}

func TestCollectorReconnectsWhenSubscriptionCloses(t *testing.T) {
	first := make(chan Candle)
	close(first)
	second := make(chan Candle, 1)
	second <- Candle{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", Close: "101.00"}
	close(second)

	subscriber := &fakeReconnectSubscriber{channels: []<-chan Candle{first, second}}
	publisher := &fakePublisher{}
	collector := NewCollector(subscriber, publisher, CollectorConfig{
		Symbol:         "BTCUSDT",
		Interval:       "1m",
		RedisStream:    "stream:market-data",
		MaxReconnects:  1,
		ReconnectDelay: 0,
	})

	if err := collector.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if subscriber.calls != 2 {
		t.Fatalf("expected 2 subscribe attempts, got %d", subscriber.calls)
	}
	if len(publisher.published) != 1 || publisher.published[0].Close != "101.00" {
		t.Fatalf("expected candle from second subscription, got %+v", publisher.published)
	}
}

func TestCollectorRecoversWhenBinanceSubscriptionIsUnavailable(t *testing.T) {
	second := make(chan Candle, 1)
	second <- Candle{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", Close: "101.00"}
	close(second)

	subscriber := &fakeFailingSubscriber{
		errors:   []error{errors.New("binance unavailable"), nil},
		channels: []<-chan Candle{nil, second},
	}
	publisher := &fakePublisher{}
	collector := NewCollector(subscriber, publisher, CollectorConfig{
		Symbol:         "BTCUSDT",
		Interval:       "1m",
		RedisStream:    "stream:market-data",
		MaxReconnects:  1,
		ReconnectDelay: 0,
	})

	if err := collector.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if subscriber.calls != 2 {
		t.Fatalf("expected retry after subscribe failure, got %d calls", subscriber.calls)
	}
	if len(publisher.published) != 1 || publisher.published[0].Close != "101.00" {
		t.Fatalf("expected candle after recovery, got %+v", publisher.published)
	}
}

func TestCollectorReturnsErrorWhenRedisPublishFails(t *testing.T) {
	candles := make(chan Candle, 1)
	candles <- Candle{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", Close: "100.00"}
	close(candles)

	collector := NewCollector(&fakeSubscriber{candles: candles}, &fakePublisher{err: errors.New("redis down")}, CollectorConfig{
		Symbol:      "BTCUSDT",
		Interval:    "1m",
		RedisStream: "stream:market-data",
	})

	if err := collector.Run(context.Background()); err == nil {
		t.Fatal("expected publish failure")
	}
}

func TestCollectorRetriesTemporaryRedisPublishFailure(t *testing.T) {
	candles := make(chan Candle, 1)
	candles <- Candle{Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", Close: "100.00"}
	close(candles)

	publisher := &fakePublisher{errors: []error{errors.New("redis down")}}
	collector := NewCollector(&fakeSubscriber{candles: candles}, publisher, CollectorConfig{
		Symbol:         "BTCUSDT",
		Interval:       "1m",
		RedisStream:    "stream:market-data",
		MaxReconnects:  2,
		ReconnectDelay: 0,
	})

	if err := collector.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error after transient Redis failure: %v", err)
	}
	if len(publisher.published) != 1 {
		t.Fatalf("expected candle to publish after retry, got %+v", publisher.published)
	}
	if publisher.attempts != 2 {
		t.Fatalf("expected one failed publish and one retry, got %d attempts", publisher.attempts)
	}
}

type fakeSubscriber struct {
	symbol   string
	interval string
	candles  <-chan Candle
}

func (f *fakeSubscriber) SubscribeKlines(ctx context.Context, symbol string, interval string) (<-chan Candle, error) {
	f.symbol = symbol
	f.interval = interval
	return f.candles, nil
}

type fakePublisher struct {
	stream    string
	published []Candle
	err       error
	errors    []error
	attempts  int
}

func (f *fakePublisher) PublishCandle(ctx context.Context, stream string, candle Candle) error {
	f.attempts++
	if f.err != nil {
		return f.err
	}
	if len(f.errors) > 0 {
		err := f.errors[0]
		f.errors = f.errors[1:]
		if err != nil {
			return err
		}
	}
	f.stream = stream
	f.published = append(f.published, candle)
	return nil
}

type fakeReconnectSubscriber struct {
	calls    int
	channels []<-chan Candle
}

func (f *fakeReconnectSubscriber) SubscribeKlines(ctx context.Context, symbol string, interval string) (<-chan Candle, error) {
	channel := f.channels[f.calls]
	f.calls++
	return channel, nil
}

type fakeFailingSubscriber struct {
	calls    int
	errors   []error
	channels []<-chan Candle
}

func (f *fakeFailingSubscriber) SubscribeKlines(ctx context.Context, symbol string, interval string) (<-chan Candle, error) {
	index := f.calls
	f.calls++
	if f.errors[index] != nil {
		return nil, f.errors[index]
	}
	return f.channels[index], nil
}
