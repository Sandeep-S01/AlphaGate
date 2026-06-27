package streams

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"sentra/internal/platform/events"
)

func TestConsumerRecoversAfterTransientRedisReadFailure(t *testing.T) {
	candle := map[string]any{"symbol": "BTCUSDT", "interval": "1m", "close": "100", "is_closed": true}
	envelope := events.NewEnvelope("market.candle.updated", "test", "", time.Unix(10, 0).UTC(), candle)
	payload, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	client := &fakeStreamClient{
		readResults: []readResult{
			{err: errors.New("redis down")},
			{streams: []redis.XStream{{Stream: "stream:market-data", Messages: []redis.XMessage{{ID: "1-0", Values: map[string]any{"payload": string(payload)}}}}}},
		},
	}
	consumer := NewConsumer(client, ConsumerConfig{
		Stream:         "stream:market-data",
		Group:          "group",
		Consumer:       "worker-1",
		Block:          time.Millisecond,
		ReadRetryDelay: 0,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handled := 0
	err = consumer.Run(ctx, func(ctx context.Context, envelope events.Envelope) error {
		handled++
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation after handled event, got %v", err)
	}
	if handled != 1 {
		t.Fatalf("expected one handled event after recovery, got %d", handled)
	}
	if client.reads != 2 {
		t.Fatalf("expected retry after read failure, got %d reads", client.reads)
	}
	if client.acks != 1 {
		t.Fatalf("expected ack after recovered message, got %d", client.acks)
	}
}

type readResult struct {
	streams []redis.XStream
	err     error
}

type fakeStreamClient struct {
	readResults []readResult
	reads       int
	acks        int
}

func (f *fakeStreamClient) XGroupCreateMkStream(ctx context.Context, stream string, group string, start string) *redis.StatusCmd {
	return redis.NewStatusResult("", nil)
}

func (f *fakeStreamClient) XReadGroup(ctx context.Context, args *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	result := f.readResults[f.reads]
	f.reads++
	return redis.NewXStreamSliceCmdResult(result.streams, result.err)
}

func (f *fakeStreamClient) XAck(ctx context.Context, stream string, group string, ids ...string) *redis.IntCmd {
	f.acks++
	return redis.NewIntResult(1, nil)
}
