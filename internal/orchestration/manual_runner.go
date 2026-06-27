package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sentra/internal/execution"
	"sentra/internal/marketdata"
	"sentra/internal/risk"
	"sentra/internal/strategy"
)

type ManualRunRequest struct {
	Symbol   string `json:"symbol"`
	Interval string `json:"interval"`
}

func (r ManualRunRequest) Normalize() ManualRunRequest {
	r.Symbol = strings.ToUpper(strings.TrimSpace(r.Symbol))
	r.Interval = strings.TrimSpace(r.Interval)
	return r
}

type ManualRunResult struct {
	Status    string            `json:"status"`
	Signal    strategy.Signal   `json:"signal"`
	Decision  risk.Decision     `json:"risk_decision"`
	Execution *execution.Result `json:"execution,omitempty"`
}

type ManualRunnerDependencies struct {
	CandleReader     CandleReader
	StrategySettings StrategySettingsReader
	SignalStore      SignalStore
	RiskSettings     RiskSettingsReader
	DecisionStore    DecisionStore
	PriceReader      PriceReader
	AccountStore     AccountStore
	ExecutionStore   ExecutionStore
	ExecutionStats   ExecutionStatsReader
	Publisher        Publisher
	ExecutionEngine  *execution.PaperEngine
	Safety           SafetyChecker
}

type StrategySettingsReader interface {
	Get(ctx context.Context) (strategy.Settings, error)
}

type RiskSettingsReader interface {
	Get(ctx context.Context) (risk.Settings, error)
}

type ManualRunner struct {
	deps ManualRunnerDependencies
	cfg  Config
}

func NewManualRunner(deps ManualRunnerDependencies, cfg Config) *ManualRunner {
	return &ManualRunner{deps: deps, cfg: cfg}
}

func (r *ManualRunner) RunOnce(ctx context.Context, request ManualRunRequest) (ManualRunResult, error) {
	request = request.Normalize()
	settings, err := r.deps.StrategySettings.Get(ctx)
	if err != nil {
		return ManualRunResult{}, fmt.Errorf("read strategy settings: %w", err)
	}
	settings = settings.Normalized()
	if request.Symbol != "" {
		settings.Symbol = request.Symbol
	}
	if request.Interval != "" {
		settings.Interval = request.Interval
	}
	if err := settings.Validate(); err != nil {
		return ManualRunResult{}, err
	}
	candles, err := r.deps.CandleReader.List(ctx, marketdata.CandleQuery{
		Symbol:   settings.Symbol,
		Interval: settings.Interval,
		Limit:    settings.LookbackLimit,
		Desc:     true,
	})
	if err != nil {
		return ManualRunResult{}, fmt.Errorf("read strategy candles: %w", err)
	}
	reverseCandles(candles)
	evaluator, err := strategy.NewEvaluatorFromSettings(settings)
	if err != nil {
		return ManualRunResult{}, fmt.Errorf("create strategy evaluator: %w", err)
	}
	signal, err := evaluator.Evaluate(candles)
	if err != nil {
		return ManualRunResult{}, fmt.Errorf("evaluate strategy: %w", err)
	}
	signal.ID, err = r.deps.SignalStore.Save(ctx, signal)
	if err != nil {
		return ManualRunResult{}, fmt.Errorf("save signal: %w", err)
	}
	if r.deps.Publisher != nil {
		if err := r.deps.Publisher.PublishSignal(ctx, r.cfg.SignalStream, "", signal); err != nil {
			return ManualRunResult{}, fmt.Errorf("publish signal: %w", err)
		}
	}
	riskSettings, err := r.deps.RiskSettings.Get(ctx)
	if err != nil {
		return ManualRunResult{}, fmt.Errorf("read risk settings: %w", err)
	}
	price, account, err := r.executionContext(ctx, signal)
	if err != nil {
		return ManualRunResult{}, err
	}
	riskEvaluator := risk.NewEvaluator(riskSettings.Config())
	decision := riskEvaluator.EvaluateWithContext(signal, r.riskContext(ctx, signal.Symbol, price, account))
	decision.ID, err = r.deps.DecisionStore.Save(ctx, decision)
	if err != nil {
		return ManualRunResult{}, fmt.Errorf("save risk decision: %w", err)
	}
	if r.deps.Publisher != nil {
		if err := r.deps.Publisher.PublishDecision(ctx, r.cfg.RiskStream, "", decision); err != nil {
			return ManualRunResult{}, fmt.Errorf("publish risk decision: %w", err)
		}
	}
	result := ManualRunResult{Status: "risk_rejected", Signal: signal, Decision: decision}
	if decision.Decision != risk.DecisionApproved {
		return result, nil
	}
	if r.deps.Safety != nil {
		active, err := r.deps.Safety.IsKillSwitchActive(ctx)
		if err != nil {
			return ManualRunResult{}, fmt.Errorf("read safety status: %w", err)
		}
		if active {
			result.Status = "safety_blocked"
			return result, nil
		}
	}
	executionResult, updatedAccount, err := r.deps.ExecutionEngine.Execute(decision, price, account)
	if err != nil {
		return ManualRunResult{}, fmt.Errorf("execute paper order: %w", err)
	}
	orderID, tradeID, err := r.deps.ExecutionStore.Save(ctx, executionResult)
	if err != nil {
		return ManualRunResult{}, fmt.Errorf("save execution: %w", err)
	}
	executionResult.Order.ID = orderID
	executionResult.Trade.ID = tradeID
	executionResult.Trade.OrderID = orderID
	if err := r.deps.AccountStore.Save(ctx, updatedAccount); err != nil {
		return ManualRunResult{}, fmt.Errorf("save paper account: %w", err)
	}
	if r.deps.Publisher != nil {
		if err := r.deps.Publisher.PublishExecution(ctx, r.cfg.ExecutionStream, "", executionResult); err != nil {
			return ManualRunResult{}, fmt.Errorf("publish execution: %w", err)
		}
	}
	result.Status = "executed"
	result.Execution = &executionResult
	return result, nil
}

func (r *ManualRunner) executionContext(ctx context.Context, signal strategy.Signal) (float64, execution.Account, error) {
	if signal.Side != strategy.SideBuy && signal.Side != strategy.SideSell {
		return 0, execution.Account{}, nil
	}
	price, err := r.deps.PriceReader.LatestPrice(ctx, signal.Symbol, signal.Interval)
	if err != nil {
		return 0, execution.Account{}, fmt.Errorf("read latest price: %w", err)
	}
	account, err := r.deps.AccountStore.Get(ctx)
	if err != nil {
		return 0, execution.Account{}, fmt.Errorf("read paper account: %w", err)
	}
	return price, account, nil
}

func (r *ManualRunner) riskContext(ctx context.Context, symbol string, price float64, account execution.Account) risk.Context {
	riskContext := risk.Context{
		QuoteAmount:  r.cfg.QuoteOrderAmount,
		Price:        price,
		BaseBalance:  account.BaseBalance,
		QuoteBalance: account.QuoteBalance,
	}
	if r.deps.ExecutionStats == nil {
		return riskContext
	}
	if stats, err := r.deps.ExecutionStats.DailyStats(ctx, symbol, time.Now().UTC()); err == nil {
		riskContext.DailyTrades = stats.TradeCount
		riskContext.DailyLoss = stats.DailyLoss
		riskContext.LastTradeAt = stats.LastTradeAt
	}
	return riskContext
}
