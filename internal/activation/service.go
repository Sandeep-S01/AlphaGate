package activation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sentra/internal/backtest"
	"sentra/internal/strategy"
)

const maxEvidenceAge = 7 * 24 * time.Hour
const minActivationTrades = 100
const minActivationProfitFactor = 1.3
const maxActivationDrawdown = 20.0

var ErrActivationGate = errors.New("activation gate blocked")

type LifecycleState string

const (
	StateDraft        LifecycleState = "DRAFT"
	StateBacktesting  LifecycleState = "BACKTESTING"
	StateValidated    LifecycleState = "VALIDATED"
	StatePaperTrading LifecycleState = "PAPER_TRADING"
	StateApproved     LifecycleState = "APPROVED"
	StateLiveEnabled  LifecycleState = "LIVE_ENABLED"
)

type Request struct {
	ComparisonID string `json:"comparison_id"`
	StrategyName string `json:"strategy_name"`
	Actor        string `json:"actor"`
}

type LifecycleRecord struct {
	ID           string         `json:"id"`
	StrategyName string         `json:"strategy_name"`
	Symbol       string         `json:"symbol"`
	Interval     string         `json:"interval"`
	State        LifecycleState `json:"state"`
	Reason       string         `json:"reason"`
	UpdatedBy    string         `json:"updated_by"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type Record struct {
	ID                    string            `json:"id"`
	ComparisonID          string            `json:"comparison_id"`
	ComparisonResultID    string            `json:"comparison_result_id"`
	StrategyName          string            `json:"strategy_name"`
	Actor                 string            `json:"actor"`
	ActivatedSettings     strategy.Settings `json:"activated_settings"`
	ComparisonReturn      float64           `json:"comparison_return"`
	ComparisonDrawdown    float64           `json:"comparison_drawdown"`
	ComparisonWinRate     float64           `json:"comparison_win_rate"`
	ComparisonTotalTrades int               `json:"comparison_total_trades"`
	CreatedAt             time.Time         `json:"created_at"`
}

type ComparisonReader interface {
	Get(ctx context.Context, id string) (backtest.Comparison, error)
}

type SettingsStore interface {
	Save(ctx context.Context, settings strategy.Settings) (strategy.Settings, error)
}

type Store interface {
	Save(ctx context.Context, record Record) (Record, error)
}

type LifecycleStore interface {
	SaveLifecycle(ctx context.Context, record LifecycleRecord) (LifecycleRecord, error)
}

type Dependencies struct {
	Comparisons ComparisonReader
	Settings    SettingsStore
	Activations Store
	Lifecycles  LifecycleStore
	Now         func() time.Time
}

type Service struct {
	deps Dependencies
}

func NewService(deps Dependencies) *Service {
	if deps.Now == nil {
		deps.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{deps: deps}
}

func (s *Service) Activate(ctx context.Context, request Request) (Record, error) {
	if request.ComparisonID == "" {
		return Record{}, fmt.Errorf("comparison_id is required")
	}
	if request.Actor == "" {
		request.Actor = "operator"
	}
	comparison, err := s.deps.Comparisons.Get(ctx, request.ComparisonID)
	if err != nil {
		return Record{}, fmt.Errorf("read comparison: %w", err)
	}
	if comparison.ID == "" {
		return Record{}, fmt.Errorf("comparison not found")
	}
	now := s.deps.Now()
	if comparison.CreatedAt.IsZero() || now.Sub(comparison.CreatedAt) > maxEvidenceAge {
		return Record{}, fmt.Errorf("comparison evidence is stale")
	}
	result, err := selectResult(comparison, request.StrategyName)
	if err != nil {
		return Record{}, err
	}
	if err := validateActivationCandidate(result); err != nil {
		return Record{}, err
	}
	settings := settingsFromResult(comparison, result)
	savedSettings, err := s.deps.Settings.Save(ctx, settings)
	if err != nil {
		return Record{}, fmt.Errorf("save strategy settings: %w", err)
	}
	record := Record{
		ComparisonID:          comparison.ID,
		ComparisonResultID:    result.ID,
		StrategyName:          result.StrategyName,
		Actor:                 request.Actor,
		ActivatedSettings:     savedSettings,
		ComparisonReturn:      result.ReturnPercent,
		ComparisonDrawdown:    result.MaxDrawdown,
		ComparisonWinRate:     result.WinRate,
		ComparisonTotalTrades: result.TotalTrades,
	}
	if s.deps.Activations == nil {
		if err := s.saveValidatedLifecycle(ctx, comparison, result, request.Actor, now); err != nil {
			return Record{}, err
		}
		return record, nil
	}
	saved, err := s.deps.Activations.Save(ctx, record)
	if err != nil {
		return Record{}, fmt.Errorf("save strategy activation: %w", err)
	}
	if err := s.saveValidatedLifecycle(ctx, comparison, result, request.Actor, now); err != nil {
		return Record{}, err
	}
	return saved, nil
}

func (s *Service) saveValidatedLifecycle(ctx context.Context, comparison backtest.Comparison, result backtest.ComparisonResult, actor string, now time.Time) error {
	if s.deps.Lifecycles == nil {
		return nil
	}
	_, err := s.deps.Lifecycles.SaveLifecycle(ctx, LifecycleRecord{
		StrategyName: result.StrategyName,
		Symbol:       comparison.Symbol,
		Interval:     comparison.Interval,
		State:        StateValidated,
		Reason:       "strategy validated from comparison evidence",
		UpdatedBy:    actor,
		UpdatedAt:    now,
	})
	if err != nil {
		return fmt.Errorf("save strategy lifecycle: %w", err)
	}
	return nil
}

func selectResult(comparison backtest.Comparison, requestedStrategy string) (backtest.ComparisonResult, error) {
	strategyName := requestedStrategy
	if strategyName == "" {
		strategyName = comparison.WinnerStrategy
	}
	for _, result := range comparison.Results {
		if result.StrategyName == strategyName {
			return result, nil
		}
	}
	return backtest.ComparisonResult{}, fmt.Errorf("comparison result for strategy %q not found", strategyName)
}

func validateActivationCandidate(result backtest.ComparisonResult) error {
	if result.ValidationStatus != "candidate" {
		reason := result.ValidationReason
		if reason == "" {
			reason = fmt.Sprintf("winner validation status is %q", result.ValidationStatus)
		}
		return fmt.Errorf("%w: %s", ErrActivationGate, reason)
	}
	if result.ExcessReturnPercent <= 0 {
		return fmt.Errorf("%w: winner must outperform buy-and-hold benchmark", ErrActivationGate)
	}
	if result.MaxDrawdown > maxActivationDrawdown {
		return fmt.Errorf("%w: winner maximum drawdown %.2f%% exceeds %.2f%%", ErrActivationGate, result.MaxDrawdown, maxActivationDrawdown)
	}
	if result.TotalTrades < minActivationTrades {
		return fmt.Errorf("%w: winner completed trades %d below minimum %d", ErrActivationGate, result.TotalTrades, minActivationTrades)
	}
	if result.ProfitFactor < minActivationProfitFactor {
		return fmt.Errorf("%w: winner profit factor %.2f below %.2f", ErrActivationGate, result.ProfitFactor, minActivationProfitFactor)
	}
	if result.Expectancy <= 0 {
		return fmt.Errorf("%w: winner expectancy must be positive", ErrActivationGate)
	}
	if result.ExecutionFillMode != backtest.ExecutionFillModeNextOpen {
		return fmt.Errorf("%w: winner must use next_open execution evidence", ErrActivationGate)
	}
	if result.TrainValidationStatus == "" || result.TestValidationStatus == "" {
		return fmt.Errorf("%w: winner must include train/test validation evidence", ErrActivationGate)
	}
	if result.TrainValidationStatus != "candidate" || result.TestValidationStatus != "candidate" {
		reason := result.TestValidationReason
		if reason == "" {
			reason = result.TrainValidationReason
		}
		if reason == "" {
			reason = "winner must pass train/test validation"
		}
		return fmt.Errorf("%w: %s", ErrActivationGate, reason)
	}
	if result.WalkForwardFolds <= 0 || result.WalkForwardValidationStatus == "" {
		return fmt.Errorf("%w: winner must include walk-forward validation evidence", ErrActivationGate)
	}
	if result.WalkForwardValidationStatus != "candidate" {
		reason := result.WalkForwardValidationReason
		if reason == "" {
			reason = "winner must pass walk-forward validation"
		}
		return fmt.Errorf("%w: %s", ErrActivationGate, reason)
	}
	return nil
}

func settingsFromResult(comparison backtest.Comparison, result backtest.ComparisonResult) strategy.Settings {
	settings := strategy.Settings{
		StrategyName:  result.StrategyName,
		Version:       result.Version,
		Symbol:        comparison.Symbol,
		Interval:      comparison.Interval,
		FastPeriod:    result.FastPeriod,
		SlowPeriod:    result.SlowPeriod,
		RSIPeriod:     result.RSIPeriod,
		RSIOversold:   result.RSIOversold,
		RSIOverbought: result.RSIOverbought,
	}
	settings.LookbackLimit = requiredLookback(settings)
	return settings
}

func requiredLookback(settings strategy.Settings) int {
	required := settings.RequiredCandles()
	if required < 100 {
		return 100
	}
	return required
}
