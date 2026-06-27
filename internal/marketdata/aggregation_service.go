package marketdata

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type CandleReadWriter interface {
	CandleStore
	UpsertBatch(ctx context.Context, candles []Candle) error
	DeleteRange(ctx context.Context, query CandleQuery) error
	List(ctx context.Context, query CandleQuery) ([]Candle, error)
}

type AggregationRequest struct {
	Symbol          string    `json:"symbol"`
	SourceInterval  string    `json:"source_interval"`
	TargetIntervals []string  `json:"target_intervals"`
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
}

type AggregationResult struct {
	Symbol          string                      `json:"symbol"`
	SourceInterval  string                      `json:"source_interval"`
	From            time.Time                   `json:"from"`
	To              time.Time                   `json:"to"`
	TargetIntervals []AggregationIntervalResult `json:"target_intervals"`
}

type AggregationIntervalResult struct {
	Interval string `json:"interval"`
	Count    int    `json:"count"`
}

type AggregationService struct {
	candles CandleReadWriter
}

func NewAggregationService(candles CandleReadWriter) *AggregationService {
	return &AggregationService{candles: candles}
}

func (s *AggregationService) Aggregate(ctx context.Context, request AggregationRequest) (AggregationResult, error) {
	request, err := normalizeAggregationRequest(request)
	if err != nil {
		return AggregationResult{}, err
	}
	limit, err := ExpectedCandleCount(request.From, request.To, request.SourceInterval)
	if err != nil {
		return AggregationResult{}, err
	}
	source, err := s.candles.List(ctx, CandleQuery{
		Symbol:   request.Symbol,
		Interval: request.SourceInterval,
		From:     request.From,
		To:       request.To,
		Limit:    limit,
	})
	if err != nil {
		return AggregationResult{}, fmt.Errorf("list source candles: %w", err)
	}

	result := AggregationResult{
		Symbol:          request.Symbol,
		SourceInterval:  request.SourceInterval,
		From:            request.From,
		To:              request.To,
		TargetIntervals: make([]AggregationIntervalResult, 0, len(request.TargetIntervals)),
	}
	for _, interval := range request.TargetIntervals {
		aggregated, err := AggregateCandles(source, interval)
		if err != nil {
			return AggregationResult{}, err
		}
		if err := s.candles.DeleteRange(ctx, CandleQuery{
			Symbol:   request.Symbol,
			Interval: interval,
			From:     request.From,
			To:       request.To,
		}); err != nil {
			return AggregationResult{}, fmt.Errorf("delete %s candle range: %w", interval, err)
		}
		if err := s.candles.UpsertBatch(ctx, aggregated); err != nil {
			return AggregationResult{}, fmt.Errorf("store %s candles: %w", interval, err)
		}
		result.TargetIntervals = append(result.TargetIntervals, AggregationIntervalResult{Interval: interval, Count: len(aggregated)})
	}
	return result, nil
}

func normalizeAggregationRequest(request AggregationRequest) (AggregationRequest, error) {
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	request.SourceInterval = strings.TrimSpace(request.SourceInterval)
	if request.Symbol == "" {
		return AggregationRequest{}, fmt.Errorf("symbol is required")
	}
	if request.SourceInterval == "" {
		request.SourceInterval = "1m"
	}
	if len(request.TargetIntervals) == 0 {
		request.TargetIntervals = []string{"5m", "15m", "1h"}
	}
	if !request.From.Before(request.To) {
		return AggregationRequest{}, fmt.Errorf("from must be before to")
	}
	request.From = request.From.UTC()
	request.To = request.To.UTC()
	return request, nil
}

func ExpectedCandleCount(from time.Time, to time.Time, interval string) (int, error) {
	if !from.Before(to) {
		return 0, fmt.Errorf("from must be before to")
	}
	step, err := IntervalDuration(interval)
	if err != nil {
		return 0, err
	}
	return int(to.Sub(from) / step), nil
}
