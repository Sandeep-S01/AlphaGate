package marketdata

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type HistoricalKlineRequest struct {
	Symbol    string
	Interval  string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

type HistoricalKlineFetcher interface {
	FetchKlines(ctx context.Context, request HistoricalKlineRequest) ([]Candle, error)
}

type BackfillRequest struct {
	Symbol   string
	Interval string
	From     time.Time
	To       time.Time
	Limit    int
}

type BackfillStatus string

const (
	BackfillStatusPending   BackfillStatus = "pending"
	BackfillStatusRunning   BackfillStatus = "running"
	BackfillStatusFailed    BackfillStatus = "failed"
	BackfillStatusCompleted BackfillStatus = "completed"
)

type BackfillJob struct {
	ID              string         `json:"id"`
	Symbol          string         `json:"symbol"`
	BaseInterval    string         `json:"base_interval"`
	From            time.Time      `json:"from_time"`
	To              time.Time      `json:"to_time"`
	NextOpenTime    time.Time      `json:"next_open_time"`
	Status          BackfillStatus `json:"status"`
	CandlesInserted int            `json:"candles_inserted"`
	LastError       string         `json:"last_error,omitempty"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type BackfillJobQuery struct {
	Symbol string
	Limit  int
}

type BackfillResult struct {
	Job             BackfillJob `json:"job"`
	CandlesInserted int         `json:"candles_inserted"`
}

type BackfillJobStore interface {
	Create(ctx context.Context, job BackfillJob) (BackfillJob, error)
	Get(ctx context.Context, id string) (BackfillJob, error)
	List(ctx context.Context, query BackfillJobQuery) ([]BackfillJob, error)
	Save(ctx context.Context, job BackfillJob) (BackfillJob, error)
}

type batchCandleStore interface {
	UpsertBatch(ctx context.Context, candles []Candle) error
}

type BackfillOption func(*BackfillService)

type BackfillService struct {
	fetcher      HistoricalKlineFetcher
	store        CandleStore
	jobs         BackfillJobStore
	maxAttempts  int
	retryDelay   time.Duration
	requestDelay time.Duration
}

func NewBackfillService(fetcher HistoricalKlineFetcher, store CandleStore, opts ...BackfillOption) *BackfillService {
	service := &BackfillService{
		fetcher:      fetcher,
		store:        store,
		maxAttempts:  3,
		retryDelay:   time.Second,
		requestDelay: 200 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func WithBackfillJobs(jobs BackfillJobStore) BackfillOption {
	return func(service *BackfillService) {
		service.jobs = jobs
	}
}

func WithBackfillRetryDelay(delay time.Duration) BackfillOption {
	return func(service *BackfillService) {
		service.retryDelay = delay
		service.requestDelay = delay
	}
}

func (s *BackfillService) Backfill(ctx context.Context, request BackfillRequest) (int, error) {
	result, err := s.Start(ctx, request)
	if err != nil {
		return 0, err
	}
	return result.CandlesInserted, nil
}

func (s *BackfillService) Start(ctx context.Context, request BackfillRequest) (BackfillResult, error) {
	request, err := normalizeBackfillRequest(request)
	if err != nil {
		return BackfillResult{}, err
	}
	job := BackfillJob{
		Symbol:       strings.ToUpper(request.Symbol),
		BaseInterval: request.Interval,
		From:         request.From.UTC(),
		To:           request.To.UTC(),
		NextOpenTime: request.From.UTC(),
		Status:       BackfillStatusPending,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if s.jobs != nil {
		job, err = s.jobs.Create(ctx, job)
		if err != nil {
			return BackfillResult{}, fmt.Errorf("create backfill job: %w", err)
		}
	}
	return s.runJob(ctx, job, request.Limit)
}

func (s *BackfillService) Resume(ctx context.Context, jobID string) (BackfillResult, error) {
	if s.jobs == nil {
		return BackfillResult{}, fmt.Errorf("backfill job store is required")
	}
	if strings.TrimSpace(jobID) == "" {
		return BackfillResult{}, fmt.Errorf("job id is required")
	}
	job, err := s.jobs.Get(ctx, jobID)
	if err != nil {
		return BackfillResult{}, fmt.Errorf("get backfill job: %w", err)
	}
	return s.runJob(ctx, job, 1000)
}

func (s *BackfillService) runJob(ctx context.Context, job BackfillJob, limit int) (BackfillResult, error) {
	step, err := IntervalDuration(job.BaseInterval)
	if err != nil {
		return BackfillResult{}, err
	}
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}

	startedAt := time.Now().UTC()
	job.Status = BackfillStatusRunning
	job.LastError = ""
	job.StartedAt = &startedAt
	job.UpdatedAt = startedAt
	if s.jobs != nil {
		if job, err = s.jobs.Save(ctx, job); err != nil {
			return BackfillResult{}, fmt.Errorf("mark backfill running: %w", err)
		}
	}

	current := job.NextOpenTime.UTC()
	if current.IsZero() || current.Before(job.From) {
		current = job.From.UTC()
	}
	for current.Before(job.To) {
		candles, err := s.fetchWithRetry(ctx, HistoricalKlineRequest{
			Symbol:    job.Symbol,
			Interval:  job.BaseInterval,
			StartTime: current,
			EndTime:   job.To,
			Limit:     limit,
		})
		if err != nil {
			return BackfillResult{}, s.failJob(ctx, job, fmt.Errorf("fetch klines: %w", err))
		}
		if len(candles) == 0 {
			break
		}
		if err := s.storeCandles(ctx, candles); err != nil {
			return BackfillResult{}, s.failJob(ctx, job, fmt.Errorf("store candle batch: %w", err))
		}

		job.CandlesInserted += len(candles)
		current = candles[len(candles)-1].OpenTime.UTC().Add(step)
		job.NextOpenTime = current
		job.UpdatedAt = time.Now().UTC()
		if s.jobs != nil {
			if job, err = s.jobs.Save(ctx, job); err != nil {
				return BackfillResult{}, fmt.Errorf("save backfill progress: %w", err)
			}
		}
		if len(candles) < limit {
			break
		}
		if s.requestDelay > 0 {
			timer := time.NewTimer(s.requestDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return BackfillResult{}, ctx.Err()
			case <-timer.C:
			}
		}
	}

	completedAt := time.Now().UTC()
	job.Status = BackfillStatusCompleted
	job.CompletedAt = &completedAt
	job.UpdatedAt = completedAt
	if job.NextOpenTime.IsZero() {
		job.NextOpenTime = current
	}
	if s.jobs != nil {
		if job, err = s.jobs.Save(ctx, job); err != nil {
			return BackfillResult{}, fmt.Errorf("mark backfill completed: %w", err)
		}
	}
	return BackfillResult{Job: job, CandlesInserted: job.CandlesInserted}, nil
}

func (s *BackfillService) fetchWithRetry(ctx context.Context, request HistoricalKlineRequest) ([]Candle, error) {
	var lastErr error
	for attempt := 1; attempt <= s.maxAttempts; attempt++ {
		candles, err := s.fetcher.FetchKlines(ctx, request)
		if err == nil {
			return candles, nil
		}
		lastErr = err
		if attempt == s.maxAttempts {
			break
		}
		if s.retryDelay > 0 {
			timer := time.NewTimer(s.retryDelay * time.Duration(attempt))
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return nil, lastErr
}

func (s *BackfillService) storeCandles(ctx context.Context, candles []Candle) error {
	if batchStore, ok := s.store.(batchCandleStore); ok {
		return batchStore.UpsertBatch(ctx, candles)
	}
	for _, candle := range candles {
		if err := s.store.Upsert(ctx, candle); err != nil {
			return err
		}
	}
	return nil
}

func (s *BackfillService) failJob(ctx context.Context, job BackfillJob, err error) error {
	job.Status = BackfillStatusFailed
	job.LastError = err.Error()
	job.UpdatedAt = time.Now().UTC()
	if s.jobs != nil {
		_, _ = s.jobs.Save(ctx, job)
	}
	return err
}

func normalizeBackfillRequest(request BackfillRequest) (BackfillRequest, error) {
	if request.Symbol == "" {
		return BackfillRequest{}, fmt.Errorf("symbol is required")
	}
	if request.Interval == "" {
		return BackfillRequest{}, fmt.Errorf("interval is required")
	}
	if !request.From.Before(request.To) {
		return BackfillRequest{}, fmt.Errorf("from must be before to")
	}
	if _, err := IntervalDuration(request.Interval); err != nil {
		return BackfillRequest{}, err
	}
	if request.Limit <= 0 || request.Limit > 1000 {
		request.Limit = 1000
	}
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	request.Interval = strings.TrimSpace(request.Interval)
	request.From = request.From.UTC()
	request.To = request.To.UTC()
	return request, nil
}
