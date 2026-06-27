package observability

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type StreamStats struct {
	Name    string `json:"name"`
	Groups  int    `json:"groups"`
	Pending int64  `json:"pending"`
}

type RedisStreamStatsReader struct {
	client  redis.Cmdable
	streams []string
}

func NewRedisStreamStatsReader(client redis.Cmdable, streams []string) *RedisStreamStatsReader {
	return &RedisStreamStatsReader{client: client, streams: streams}
}

func (r *RedisStreamStatsReader) Stats(ctx context.Context) ([]StreamStats, error) {
	stats := make([]StreamStats, 0, len(r.streams))
	for _, stream := range r.streams {
		groups, err := r.client.XInfoGroups(ctx, stream).Result()
		if err != nil {
			if strings.Contains(err.Error(), "no such key") {
				stats = append(stats, StreamStats{Name: stream})
				continue
			}
			return nil, fmt.Errorf("get stream groups for %s: %w", stream, err)
		}
		item := StreamStats{Name: stream, Groups: len(groups)}
		for _, group := range groups {
			item.Pending += group.Pending
		}
		stats = append(stats, item)
	}
	return stats, nil
}
