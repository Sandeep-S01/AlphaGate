package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sentra/internal/execution"
	"sentra/internal/marketdata"
	"sentra/internal/platform/events"
	"sentra/internal/risk"
	"sentra/internal/strategy"
)

type Config struct {
	Symbol           string
	Interval         string
	LookbackLimit    int
	SignalStream     string
	RiskStream       string
	ExecutionStream  string
	QuoteOrderAmount float64
}

type Dependencies struct {
	Idempotency       IdempotencyStore
	CandleStore       CandleStore
	CandleReader      CandleReader
	StrategyEvaluator StrategyEvaluator
	SignalStore       SignalStore
	RiskEvaluator     RiskEvaluator
	DecisionStore     DecisionStore
	PriceReader       PriceReader
	AccountStore      AccountStore
	ExecutionStore    ExecutionStore
	ExecutionStats    ExecutionStatsReader
	Publisher         Publisher
	ExecutionEngine   *execution.PaperEngine
	Safety            SafetyChecker
	Metrics           Metrics
}

type Metrics interface {
	IncPipelineCompleted()
	IncPipelineFailed()
}

type IdempotencyStore interface {
	Begin(ctx context.Context, key string) (bool, error)
	Complete(ctx context.Context, key string) error
	Fail(ctx context.Context, key string, reason string) error
}

type CandleStore interface {
	Upsert(ctx context.Context, candle marketdata.Candle) error
}

type CandleReader interface {
	List(ctx context.Context, query marketdata.CandleQuery) ([]marketdata.Candle, error)
}

type StrategyEvaluator interface {
	Evaluate(candles []marketdata.Candle) (strategy.Signal, error)
}

type SignalStore interface {
	Save(ctx context.Context, signal strategy.Signal) (string, error)
}

type RiskEvaluator interface {
	Evaluate(signal strategy.Signal) risk.Decision
}

type ContextualRiskEvaluator interface {
	EvaluateWithContext(signal strategy.Signal, ctx risk.Context) risk.Decision
}

type DecisionStore interface {
	Save(ctx context.Context, decision risk.Decision) (string, error)
}

type PriceReader interface {
	LatestPrice(ctx context.Context, symbol string, interval string) (float64, error)
}

type AccountStore interface {
	Get(ctx context.Context) (execution.Account, error)
	Save(ctx context.Context, account execution.Account) error
}

type ExecutionStore interface {
	Save(ctx context.Context, result execution.Result) (string, string, error)
}

type ExecutionStatsReader interface {
	DailyStats(ctx context.Context, symbol string, day time.Time) (execution.DailyStats, error)
}

type SafetyChecker interface {
	IsKillSwitchActive(ctx context.Context) (bool, error)
}

type Publisher interface {
	PublishSignal(ctx context.Context, stream string, correlationID string, signal strategy.Signal) error
	PublishDecision(ctx context.Context, stream string, correlationID string, decision risk.Decision) error
	PublishExecution(ctx context.Context, stream string, correlationID string, result execution.Result) error
}

type Orchestrator struct {
	deps Dependencies
	cfg  Config
}

func NewOrchestrator(deps Dependencies, cfg Config) *Orchestrator {
	return &Orchestrator{deps: deps, cfg: cfg}
}

func (o *Orchestrator) Handle(ctx context.Context, envelope events.Envelope) error {
	if envelope.Type != "market.candle.updated" {
		return nil
	}

	candle, err := decodeCandle(envelope.Payload)
	if err != nil {
		return err
	}
	if !candle.IsClosed {
		return nil
	}

	key := idempotencyKey(candle)
	started, err := o.deps.Idempotency.Begin(ctx, key)
	if err != nil {
		return fmt.Errorf("begin orchestration: %w", err)
	}
	if !started {
		return nil
	}

	if err := o.run(ctx, candle, envelope.CorrelationID); err != nil {
		if o.deps.Metrics != nil {
			o.deps.Metrics.IncPipelineFailed()
		}
		_ = o.deps.Idempotency.Fail(ctx, key, err.Error())
		return err
	}

	if err := o.deps.Idempotency.Complete(ctx, key); err != nil {
		return fmt.Errorf("complete orchestration: %w", err)
	}
	if o.deps.Metrics != nil {
		o.deps.Metrics.IncPipelineCompleted()
	}
	return nil
}

func (o *Orchestrator) run(ctx context.Context, candle marketdata.Candle, correlationID string) error {
	if err := o.deps.CandleStore.Upsert(ctx, candle); err != nil {
		return fmt.Errorf("persist candle: %w", err)
	}

	limit := o.cfg.LookbackLimit
	if limit <= 0 {
		limit = 100
	}
	candles, err := o.deps.CandleReader.List(ctx, marketdata.CandleQuery{
		Symbol:   candle.Symbol,
		Interval: candle.Interval,
		Limit:    limit,
		Desc:     true,
	})
	if err != nil {
		return fmt.Errorf("read strategy candles: %w", err)
	}

	reverseCandles(candles)
	signal, err := o.deps.StrategyEvaluator.Evaluate(candles)
	if err != nil {
		return fmt.Errorf("evaluate strategy: %w", err)
	}
	signal.ID, err = o.deps.SignalStore.Save(ctx, signal)
	if err != nil {
		return fmt.Errorf("save signal: %w", err)
	}
	if err := o.deps.Publisher.PublishSignal(ctx, o.cfg.SignalStream, correlationID, signal); err != nil {
		return fmt.Errorf("publish signal: %w", err)
	}

	price, account, err := o.executionContext(ctx, signal)
	if err != nil {
		return err
	}

	decision := o.evaluateRisk(ctx, signal, price, account)
	decision.ID, err = o.deps.DecisionStore.Save(ctx, decision)
	if err != nil {
		return fmt.Errorf("save risk decision: %w", err)
	}
	if err := o.deps.Publisher.PublishDecision(ctx, o.cfg.RiskStream, correlationID, decision); err != nil {
		return fmt.Errorf("publish risk decision: %w", err)
	}
	if decision.Decision != risk.DecisionApproved {
		return nil
	}
	if o.deps.Safety != nil {
		active, err := o.deps.Safety.IsKillSwitchActive(ctx)
		if err != nil {
			return fmt.Errorf("read safety status: %w", err)
		}
		if active {
			return nil
		}
	}

	result, updatedAccount, err := o.deps.ExecutionEngine.Execute(decision, price, account)
	if err != nil {
		return o.recordFailedExecution(ctx, decision, price, err, correlationID)
	}
	orderID, tradeID, err := o.deps.ExecutionStore.Save(ctx, result)
	if err != nil {
		return fmt.Errorf("save execution: %w", err)
	}
	result.Order.ID = orderID
	result.Trade.ID = tradeID
	result.Trade.OrderID = orderID
	if err := o.deps.AccountStore.Save(ctx, updatedAccount); err != nil {
		return fmt.Errorf("save paper account: %w", err)
	}
	if err := o.deps.Publisher.PublishExecution(ctx, o.cfg.ExecutionStream, correlationID, result); err != nil {
		return fmt.Errorf("publish execution: %w", err)
	}
	return nil
}

func (o *Orchestrator) recordFailedExecution(ctx context.Context, decision risk.Decision, price float64, cause error, correlationID string) error {
	now := time.Now().UTC()
	requestedQuantity := 0.0
	if price > 0 {
		requestedQuantity = o.cfg.QuoteOrderAmount / price
	}
	result := execution.Result{
		Order: execution.Order{
			RiskDecisionID:    decision.ID,
			ClientOrderID:     fmt.Sprintf("paper-%s-%d", decision.ID, now.UnixNano()),
			Symbol:            decision.Symbol,
			Side:              decision.SignalSide,
			RequestedQuantity: requestedQuantity,
			Price:             price,
			QuoteAmount:       o.cfg.QuoteOrderAmount,
			Status:            execution.OrderStatusFailed,
			FailureReason:     cause.Error(),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		OrderEvents: []execution.OrderEvent{
			{Status: execution.OrderStatusCreated, CreatedAt: now},
			{Status: execution.OrderStatusFailed, Reason: cause.Error(), CreatedAt: now},
		},
	}
	orderID, _, err := o.deps.ExecutionStore.Save(ctx, result)
	if err != nil {
		return fmt.Errorf("save failed execution: %w", err)
	}
	result.Order.ID = orderID
	if err := o.deps.Publisher.PublishExecution(ctx, o.cfg.ExecutionStream, correlationID, result); err != nil {
		return fmt.Errorf("publish failed execution: %w", err)
	}
	return nil
}

func (o *Orchestrator) executionContext(ctx context.Context, signal strategy.Signal) (float64, execution.Account, error) {
	if signal.Side != strategy.SideBuy && signal.Side != strategy.SideSell {
		return 0, execution.Account{}, nil
	}
	price, err := o.deps.PriceReader.LatestPrice(ctx, signal.Symbol, signal.Interval)
	if err != nil {
		return 0, execution.Account{}, fmt.Errorf("read latest price: %w", err)
	}
	account, err := o.deps.AccountStore.Get(ctx)
	if err != nil {
		return 0, execution.Account{}, fmt.Errorf("read paper account: %w", err)
	}
	return price, account, nil
}

func (o *Orchestrator) evaluateRisk(ctx context.Context, signal strategy.Signal, price float64, account execution.Account) risk.Decision {
	contextual, ok := o.deps.RiskEvaluator.(ContextualRiskEvaluator)
	if !ok {
		return o.deps.RiskEvaluator.Evaluate(signal)
	}
	riskContext := risk.Context{
		QuoteAmount:  o.cfg.QuoteOrderAmount,
		Price:        price,
		BaseBalance:  account.BaseBalance,
		QuoteBalance: account.QuoteBalance,
	}
	if o.deps.ExecutionStats != nil {
		if stats, err := o.deps.ExecutionStats.DailyStats(ctx, signal.Symbol, time.Now().UTC()); err == nil {
			riskContext.DailyTrades = stats.TradeCount
			riskContext.DailyLoss = stats.DailyLoss
			riskContext.LastTradeAt = stats.LastTradeAt
		}
	}
	return contextual.EvaluateWithContext(signal, riskContext)
}

func decodeCandle(payload any) (marketdata.Candle, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return marketdata.Candle{}, fmt.Errorf("encode candle payload: %w", err)
	}
	var candle marketdata.Candle
	if err := json.Unmarshal(encoded, &candle); err != nil {
		return marketdata.Candle{}, fmt.Errorf("decode candle payload: %w", err)
	}
	return candle, nil
}

func idempotencyKey(candle marketdata.Candle) string {
	exchange := candle.Exchange
	if exchange == "" {
		exchange = "unknown"
	}
	return fmt.Sprintf("orchestration:%s:%s:%s:%s", exchange, candle.Symbol, candle.Interval, candle.OpenTime.UTC().Format("20060102150405"))
}

func reverseCandles(candles []marketdata.Candle) {
	for left, right := 0, len(candles)-1; left < right; left, right = left+1, right-1 {
		candles[left], candles[right] = candles[right], candles[left]
	}
}
