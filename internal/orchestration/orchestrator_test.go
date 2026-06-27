package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"sentra/internal/execution"
	"sentra/internal/marketdata"
	"sentra/internal/platform/events"
	"sentra/internal/risk"
	"sentra/internal/strategy"
)

func TestOrchestratorRunsFullPaperPipelineForClosedCandle(t *testing.T) {
	fixture := newFixture()
	orchestrator := fixture.orchestrator()

	err := orchestrator.Handle(context.Background(), candleEnvelope(true))
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if len(fixture.signals.saved) != 1 {
		t.Fatalf("expected 1 saved signal, got %d", len(fixture.signals.saved))
	}
	if len(fixture.decisions.saved) != 1 || fixture.decisions.saved[0].Decision != risk.DecisionApproved {
		t.Fatalf("expected approved decision, got %+v", fixture.decisions.saved)
	}
	if len(fixture.executions.saved) != 1 {
		t.Fatalf("expected 1 saved execution, got %d", len(fixture.executions.saved))
	}
	if fixture.accounts.saved.QuoteBalance != 900 {
		t.Fatalf("expected quote balance 900, got %.8f", fixture.accounts.saved.QuoteBalance)
	}
	if fixture.publisher.executions != 1 {
		t.Fatalf("expected execution event, got %d", fixture.publisher.executions)
	}
	if !fixture.idempotency.completed {
		t.Fatal("expected idempotency key completed")
	}
}

func TestOrchestratorDoesNotExecuteRejectedRiskDecision(t *testing.T) {
	fixture := newFixture()
	fixture.riskDecision = risk.Decision{
		ID:         "risk-1",
		SignalID:   "signal-1",
		Symbol:     "BTCUSDT",
		SignalSide: strategy.SideBuy,
		Decision:   risk.DecisionRejected,
		Reason:     "buy disabled",
	}
	orchestrator := fixture.orchestrator()

	err := orchestrator.Handle(context.Background(), candleEnvelope(true))
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if len(fixture.executions.saved) != 0 {
		t.Fatalf("expected no execution, got %d", len(fixture.executions.saved))
	}
	if fixture.publisher.riskDecisions != 1 {
		t.Fatalf("expected risk decision event, got %d", fixture.publisher.riskDecisions)
	}
}

func TestOrchestratorRejectsSellBeforeExecutionWhenBaseBalanceIsInsufficient(t *testing.T) {
	fixture := newFixture()
	fixture.evaluator.signal = strategy.Signal{
		StrategyName: "btc-trend-pullback",
		Version:      "v1",
		Symbol:       "BTCUSDT",
		Interval:     "1m",
		Side:         strategy.SideSell,
		Strength:     10,
		Reason:       "test sell",
		GeneratedAt:  time.Unix(70, 0).UTC(),
	}
	fixture.accounts.account = execution.Account{BaseBalance: 0.001, QuoteBalance: 1000}
	orchestrator := fixture.orchestrator()
	orchestrator.deps.RiskEvaluator = risk.NewEvaluator(risk.Config{Enabled: true, AllowBuy: true, AllowSell: true})

	err := orchestrator.Handle(context.Background(), candleEnvelope(true))
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if len(fixture.decisions.saved) != 1 || fixture.decisions.saved[0].Decision != risk.DecisionRejected {
		t.Fatalf("expected rejected risk decision, got %+v", fixture.decisions.saved)
	}
	if fixture.decisions.saved[0].Reason != "base balance 0.001000 is below required 0.002000" {
		t.Fatalf("expected insufficient base balance reason, got %+v", fixture.decisions.saved[0])
	}
	if len(fixture.executions.saved) != 0 {
		t.Fatalf("expected risk rejection before execution, got %d executions", len(fixture.executions.saved))
	}
	if !fixture.idempotency.completed {
		t.Fatal("expected pipeline to complete after risk rejection")
	}
}

func TestOrchestratorRejectsBuyBeforeExecutionWhenQuoteBalanceIsInsufficient(t *testing.T) {
	fixture := newFixture()
	fixture.accounts.account = execution.Account{BaseBalance: 0, QuoteBalance: 50}
	orchestrator := fixture.orchestrator()
	orchestrator.deps.RiskEvaluator = risk.NewEvaluator(risk.Config{Enabled: true, AllowBuy: true, AllowSell: true})

	err := orchestrator.Handle(context.Background(), candleEnvelope(true))
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if len(fixture.decisions.saved) != 1 || fixture.decisions.saved[0].Decision != risk.DecisionRejected {
		t.Fatalf("expected rejected risk decision, got %+v", fixture.decisions.saved)
	}
	if fixture.decisions.saved[0].Reason != "quote balance 50.000000 is below required 100.000000" {
		t.Fatalf("expected insufficient quote balance reason, got %+v", fixture.decisions.saved[0])
	}
	if len(fixture.executions.saved) != 0 {
		t.Fatalf("expected risk rejection before execution, got %d executions", len(fixture.executions.saved))
	}
	if !fixture.idempotency.completed {
		t.Fatal("expected pipeline to complete after risk rejection")
	}
}

func TestOrchestratorBlocksExecutionWhenKillSwitchActive(t *testing.T) {
	fixture := newFixture()
	fixture.safety = &fakeSafety{active: true}
	orchestrator := fixture.orchestrator()

	err := orchestrator.Handle(context.Background(), candleEnvelope(true))
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if len(fixture.executions.saved) != 0 {
		t.Fatalf("expected no execution while kill switch active, got %d", len(fixture.executions.saved))
	}
	if !fixture.idempotency.completed {
		t.Fatal("expected pipeline to complete after safety block")
	}
}

func TestOrchestratorRecordsFailedOrderWhenPaperExecutionRejects(t *testing.T) {
	fixture := newFixture()
	fixture.riskDecision.SignalSide = strategy.SideSell
	fixture.accounts.account = execution.Account{BaseBalance: 0, QuoteBalance: 1000}
	orchestrator := fixture.orchestrator()

	err := orchestrator.Handle(context.Background(), candleEnvelope(true))
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if len(fixture.executions.saved) != 1 {
		t.Fatalf("expected failed execution record, got %d", len(fixture.executions.saved))
	}
	order := fixture.executions.saved[0].Order
	if order.Status != execution.OrderStatusFailed {
		t.Fatalf("expected failed order status, got %+v", order)
	}
	if order.FailureReason != "insufficient base balance" {
		t.Fatalf("expected failure reason to be persisted, got %+v", order)
	}
	if fixture.accounts.saved.UpdatedAt != (time.Time{}) {
		t.Fatalf("expected account to remain unchanged, got %+v", fixture.accounts.saved)
	}
	if !fixture.idempotency.completed {
		t.Fatal("expected pipeline to complete after recording failed execution")
	}
}

func TestOrchestratorSkipsDuplicateCandle(t *testing.T) {
	fixture := newFixture()
	fixture.idempotency.begin = false
	orchestrator := fixture.orchestrator()

	err := orchestrator.Handle(context.Background(), candleEnvelope(true))
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if len(fixture.signals.saved) != 0 {
		t.Fatalf("expected no signal on duplicate, got %d", len(fixture.signals.saved))
	}
}

func TestOrchestratorStressSkipsCompletedDuplicatesAndRetriesFailedEvent(t *testing.T) {
	fixture := newFixture()
	stateful := newStatefulIdempotency()
	fixture.idempotency = nil
	orchestrator := fixture.orchestrator()
	orchestrator.deps.Idempotency = stateful

	if err := orchestrator.Handle(context.Background(), candleEnvelope(true)); err != nil {
		t.Fatalf("first handle returned error: %v", err)
	}
	for index := 0; index < 25; index++ {
		if err := orchestrator.Handle(context.Background(), candleEnvelope(true)); err != nil {
			t.Fatalf("duplicate handle %d returned error: %v", index, err)
		}
	}

	if len(fixture.executions.saved) != 1 {
		t.Fatalf("expected completed duplicates to skip execution, got %d executions", len(fixture.executions.saved))
	}

	failedFixture := newFixture()
	failedStateful := newStatefulIdempotency()
	failedFixture.idempotency = nil
	failedFixture.candles.err = errors.New("database disconnect")
	retryOrchestrator := failedFixture.orchestrator()
	retryOrchestrator.deps.Idempotency = failedStateful

	if err := retryOrchestrator.Handle(context.Background(), candleEnvelope(true)); err == nil {
		t.Fatal("expected first failed event")
	}
	failedFixture.candles.err = nil
	if err := retryOrchestrator.Handle(context.Background(), candleEnvelope(true)); err != nil {
		t.Fatalf("expected failed event to retry successfully, got %v", err)
	}
	if len(failedFixture.executions.saved) != 1 {
		t.Fatalf("expected retry to execute once, got %d", len(failedFixture.executions.saved))
	}
}

func TestIdempotencyKeySeparatesExchanges(t *testing.T) {
	candle := marketdata.Candle{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		OpenTime: time.Unix(10, 0).UTC(),
	}
	other := candle
	other.Exchange = "coinbase"

	if idempotencyKey(candle) == idempotencyKey(other) {
		t.Fatalf("expected different exchanges to produce different idempotency keys, got %q", idempotencyKey(candle))
	}
}

func TestOrchestratorMarksPipelineFailedWhenDatabaseWriteFails(t *testing.T) {
	fixture := newFixture()
	fixture.candles.err = errors.New("database disconnect")
	orchestrator := fixture.orchestrator()

	err := orchestrator.Handle(context.Background(), candleEnvelope(true))
	if err == nil {
		t.Fatal("expected database failure")
	}
	if !fixture.idempotency.failed {
		t.Fatal("expected idempotency failure record")
	}
	if fixture.idempotency.completed {
		t.Fatal("did not expect pipeline completion")
	}
	if len(fixture.signals.saved) != 0 {
		t.Fatalf("expected no downstream signal after DB failure, got %d", len(fixture.signals.saved))
	}
}

func TestOrchestratorMarksPipelineFailedWhenRedisPublishFails(t *testing.T) {
	fixture := newFixture()
	fixture.publisher.err = errors.New("redis down")
	orchestrator := fixture.orchestrator()

	err := orchestrator.Handle(context.Background(), candleEnvelope(true))
	if err == nil {
		t.Fatal("expected Redis publish failure")
	}
	if !fixture.idempotency.failed {
		t.Fatal("expected idempotency failure record")
	}
	if len(fixture.decisions.saved) != 0 {
		t.Fatalf("expected no risk decision after signal publish failure, got %d", len(fixture.decisions.saved))
	}
}

func TestOrchestratorCanRecoverWhenRetrySucceedsAfterFailure(t *testing.T) {
	fixture := newFixture()
	fixture.candles.err = errors.New("database disconnect")
	orchestrator := fixture.orchestrator()

	if err := orchestrator.Handle(context.Background(), candleEnvelope(true)); err == nil {
		t.Fatal("expected first attempt to fail")
	}
	fixture.candles.err = nil
	fixture.idempotency.begin = true

	if err := orchestrator.Handle(context.Background(), candleEnvelope(true)); err != nil {
		t.Fatalf("expected retry to recover, got %v", err)
	}
	if !fixture.idempotency.completed {
		t.Fatal("expected retry to complete")
	}
	if len(fixture.executions.saved) != 1 {
		t.Fatalf("expected one execution after recovery, got %d", len(fixture.executions.saved))
	}
}

func TestOrchestratorRecordsFailureMetricsOnPipelineFailure(t *testing.T) {
	fixture := newFixture()
	fixture.candles.err = errors.New("database disconnect")
	metrics := &fakeMetrics{}
	orchestrator := fixture.orchestrator()
	orchestrator.deps.Metrics = metrics

	err := orchestrator.Handle(context.Background(), candleEnvelope(true))
	if err == nil {
		t.Fatal("expected pipeline failure")
	}
	if metrics.failed != 1 {
		t.Fatalf("expected failed metric, got %d", metrics.failed)
	}
}

func TestOrchestratorSkipsOpenCandle(t *testing.T) {
	fixture := newFixture()
	orchestrator := fixture.orchestrator()

	err := orchestrator.Handle(context.Background(), candleEnvelope(false))
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if fixture.idempotency.beginCalled {
		t.Fatal("expected open candle to skip before idempotency")
	}
}

func candleEnvelope(closed bool) events.Envelope {
	candle := marketdata.Candle{
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		OpenTime:  time.Unix(10, 0).UTC(),
		CloseTime: time.Unix(70, 0).UTC(),
		Close:     "50000",
		IsClosed:  closed,
	}
	return events.NewEnvelope("market.candle.updated", "market-data", "", candle.EventTime, candle)
}

type fixture struct {
	idempotency  *fakeIdempotency
	candles      *fakeCandleStore
	candleReader *fakeCandleReader
	evaluator    *fakeStrategyEvaluator
	signals      *fakeSignalStore
	risk         *fakeRiskEvaluator
	decisions    *fakeDecisionStore
	prices       *fakePriceReader
	accounts     *fakeAccountStore
	executions   *fakeExecutionStore
	publisher    *fakePublisher
	riskDecision risk.Decision
	safety       *fakeSafety
}

func newFixture() *fixture {
	riskDecision := risk.Decision{
		ID:         "risk-1",
		SignalID:   "signal-1",
		Symbol:     "BTCUSDT",
		SignalSide: strategy.SideBuy,
		Decision:   risk.DecisionApproved,
		Reason:     "approved",
	}
	return &fixture{
		idempotency:  &fakeIdempotency{begin: true},
		candles:      &fakeCandleStore{},
		candleReader: &fakeCandleReader{candles: []marketdata.Candle{{Symbol: "BTCUSDT", Interval: "1m", Close: "50000"}}},
		evaluator:    &fakeStrategyEvaluator{},
		signals:      &fakeSignalStore{},
		risk:         &fakeRiskEvaluator{},
		decisions:    &fakeDecisionStore{},
		prices:       &fakePriceReader{price: 50000},
		accounts:     &fakeAccountStore{account: execution.Account{BaseBalance: 0, QuoteBalance: 1000}},
		executions:   &fakeExecutionStore{},
		publisher:    &fakePublisher{},
		riskDecision: riskDecision,
		safety:       &fakeSafety{},
	}
}

func (f *fixture) orchestrator() *Orchestrator {
	f.risk.decision = f.riskDecision
	return NewOrchestrator(Dependencies{
		Idempotency:       f.idempotency,
		CandleStore:       f.candles,
		CandleReader:      f.candleReader,
		StrategyEvaluator: f.evaluator,
		SignalStore:       f.signals,
		RiskEvaluator:     f.risk,
		DecisionStore:     f.decisions,
		PriceReader:       f.prices,
		AccountStore:      f.accounts,
		ExecutionStore:    f.executions,
		Publisher:         f.publisher,
		Safety:            f.safety,
		ExecutionEngine: execution.NewPaperEngine(execution.Config{
			Enabled:          true,
			Symbol:           "BTCUSDT",
			BaseAsset:        "BTC",
			QuoteAsset:       "USDT",
			QuoteOrderAmount: 100,
			FeeRate:          0.001,
		}),
	}, Config{
		Symbol:           "BTCUSDT",
		Interval:         "1m",
		LookbackLimit:    10,
		SignalStream:     "stream:strategy-signals",
		RiskStream:       "stream:risk-decisions",
		ExecutionStream:  "stream:execution-results",
		QuoteOrderAmount: 100,
	})
}

type fakeSafety struct {
	active bool
}

func (f *fakeSafety) IsKillSwitchActive(ctx context.Context) (bool, error) {
	return f.active, nil
}

type fakeIdempotency struct {
	begin       bool
	beginCalled bool
	completed   bool
	failed      bool
}

func (f *fakeIdempotency) Begin(ctx context.Context, key string) (bool, error) {
	f.beginCalled = true
	return f.begin, nil
}

func (f *fakeIdempotency) Complete(ctx context.Context, key string) error {
	f.completed = true
	return nil
}

func (f *fakeIdempotency) Fail(ctx context.Context, key string, reason string) error {
	f.failed = true
	return nil
}

type statefulIdempotency struct {
	status map[string]string
}

func newStatefulIdempotency() *statefulIdempotency {
	return &statefulIdempotency{status: map[string]string{}}
}

func (s *statefulIdempotency) Begin(ctx context.Context, key string) (bool, error) {
	switch s.status[key] {
	case "", "failed":
		s.status[key] = "processing"
		return true, nil
	default:
		return false, nil
	}
}

func (s *statefulIdempotency) Complete(ctx context.Context, key string) error {
	s.status[key] = "completed"
	return nil
}

func (s *statefulIdempotency) Fail(ctx context.Context, key string, reason string) error {
	s.status[key] = "failed"
	return nil
}

type fakeCandleStore struct {
	upserted []marketdata.Candle
	err      error
}

func (f *fakeCandleStore) Upsert(ctx context.Context, candle marketdata.Candle) error {
	if f.err != nil {
		return f.err
	}
	f.upserted = append(f.upserted, candle)
	return nil
}

type fakeCandleReader struct {
	candles []marketdata.Candle
	query   marketdata.CandleQuery
}

func (f *fakeCandleReader) List(ctx context.Context, query marketdata.CandleQuery) ([]marketdata.Candle, error) {
	f.query = query
	candles := append([]marketdata.Candle(nil), f.candles...)
	if query.Desc {
		reverseCandles(candles)
	}
	return candles, nil
}

type fakeStrategyEvaluator struct {
	signal strategy.Signal
}

func (f *fakeStrategyEvaluator) Evaluate(candles []marketdata.Candle) (strategy.Signal, error) {
	if f.signal.Side != "" {
		return f.signal, nil
	}
	return strategy.Signal{
		StrategyName: "sma-crossover",
		Version:      "v1",
		Symbol:       "BTCUSDT",
		Interval:     "1m",
		Side:         strategy.SideBuy,
		Strength:     1,
		Reason:       "test",
		GeneratedAt:  time.Unix(70, 0).UTC(),
	}, nil
}

type fakeSignalStore struct {
	saved []strategy.Signal
}

func (f *fakeSignalStore) Save(ctx context.Context, signal strategy.Signal) (string, error) {
	f.saved = append(f.saved, signal)
	return "signal-1", nil
}

type fakeRiskEvaluator struct {
	decision risk.Decision
}

func (f *fakeRiskEvaluator) Evaluate(signal strategy.Signal) risk.Decision {
	decision := f.decision
	decision.SignalID = signal.ID
	return decision
}

type fakeDecisionStore struct {
	saved []risk.Decision
}

func (f *fakeDecisionStore) Save(ctx context.Context, decision risk.Decision) (string, error) {
	f.saved = append(f.saved, decision)
	return "risk-1", nil
}

type fakePriceReader struct {
	price float64
}

func (f *fakePriceReader) LatestPrice(ctx context.Context, symbol string, interval string) (float64, error) {
	return f.price, nil
}

type fakeAccountStore struct {
	account execution.Account
	saved   execution.Account
}

func (f *fakeAccountStore) Get(ctx context.Context) (execution.Account, error) {
	return f.account, nil
}

func (f *fakeAccountStore) Save(ctx context.Context, account execution.Account) error {
	f.saved = account
	return nil
}

type fakeExecutionStore struct {
	saved []execution.Result
}

func (f *fakeExecutionStore) Save(ctx context.Context, result execution.Result) (string, string, error) {
	f.saved = append(f.saved, result)
	return "order-1", "trade-1", nil
}

type fakePublisher struct {
	signals       int
	riskDecisions int
	executions    int
	err           error
}

func (f *fakePublisher) PublishSignal(ctx context.Context, stream string, correlationID string, signal strategy.Signal) error {
	if f.err != nil {
		return f.err
	}
	f.signals++
	return nil
}

func (f *fakePublisher) PublishDecision(ctx context.Context, stream string, correlationID string, decision risk.Decision) error {
	if f.err != nil {
		return f.err
	}
	f.riskDecisions++
	return nil
}

func (f *fakePublisher) PublishExecution(ctx context.Context, stream string, correlationID string, result execution.Result) error {
	if f.err != nil {
		return f.err
	}
	f.executions++
	return nil
}

type fakeMetrics struct {
	completed int
	failed    int
}

func (f *fakeMetrics) IncPipelineCompleted() {
	f.completed++
}

func (f *fakeMetrics) IncPipelineFailed() {
	f.failed++
}
