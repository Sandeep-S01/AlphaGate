package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"sentra/internal/marketdata"
	"sentra/internal/strategy"
)

const maxOptimizationCombinations = 100

type OptimizationRequest struct {
	StrategyName            string    `json:"strategy_name"`
	Symbol                  string    `json:"symbol"`
	Interval                string    `json:"interval"`
	From                    time.Time `json:"from"`
	To                      time.Time `json:"to"`
	FastPeriods             []int     `json:"fast_periods"`
	SlowPeriods             []int     `json:"slow_periods"`
	RSIPeriods              []int     `json:"rsi_periods"`
	RSIOversoldValues       []float64 `json:"rsi_oversold_values"`
	RSIOverboughtValues     []float64 `json:"rsi_overbought_values"`
	StartingBalance         float64   `json:"starting_balance"`
	FeeRate                 float64   `json:"fee_rate"`
	SlippageRate            float64   `json:"slippage_rate"`
	ExecutionFillMode       string    `json:"execution_fill_mode"`
	PositionSizingMode      string    `json:"position_sizing_mode"`
	PositionSizeValue       float64   `json:"position_size_value"`
	TrendFilterEnabled      bool      `json:"trend_filter_enabled"`
	TrendPeriod             int       `json:"trend_period"`
	CooldownBars            int       `json:"cooldown_bars"`
	MinHoldingBars          int       `json:"min_holding_bars"`
	ATRExitEnabled          bool      `json:"atr_exit_enabled"`
	ATRPeriod               int       `json:"atr_period"`
	ATRStopMultiplier       float64   `json:"atr_stop_multiplier"`
	ATRTakeProfitMultiplier float64   `json:"atr_take_profit_multiplier"`
	RegimeFilterEnabled     bool      `json:"regime_filter_enabled"`
	RegimeFilterPeriod      int       `json:"regime_filter_period"`
	RegimeMinATRPercent     float64   `json:"regime_min_atr_percent"`
	RegimeMaxATRPercent     float64   `json:"regime_max_atr_percent"`
	ShortingEnabled         bool      `json:"shorting_enabled"`
	TrainTestEnabled        bool      `json:"train_test_enabled"`
	TrainRatio              float64   `json:"train_ratio"`
	WalkForwardEnabled      bool      `json:"walk_forward_enabled"`
	WalkForwardFolds        int       `json:"walk_forward_folds"`
	MaxCombinations         int       `json:"max_combinations"`
}

type Optimization struct {
	StrategyName            string               `json:"strategy_name"`
	Symbol                  string               `json:"symbol"`
	Interval                string               `json:"interval"`
	From                    time.Time            `json:"from"`
	To                      time.Time            `json:"to"`
	TotalCombinations       int                  `json:"total_combinations"`
	PositionSizingMode      string               `json:"position_sizing_mode"`
	PositionSizeValue       float64              `json:"position_size_value"`
	TrendFilterEnabled      bool                 `json:"trend_filter_enabled"`
	TrendPeriod             int                  `json:"trend_period"`
	CooldownBars            int                  `json:"cooldown_bars"`
	MinHoldingBars          int                  `json:"min_holding_bars"`
	ATRExitEnabled          bool                 `json:"atr_exit_enabled"`
	ATRPeriod               int                  `json:"atr_period"`
	ATRStopMultiplier       float64              `json:"atr_stop_multiplier"`
	ATRTakeProfitMultiplier float64              `json:"atr_take_profit_multiplier"`
	RegimeFilterEnabled     bool                 `json:"regime_filter_enabled"`
	RegimeFilterPeriod      int                  `json:"regime_filter_period"`
	RegimeMinATRPercent     float64              `json:"regime_min_atr_percent"`
	RegimeMaxATRPercent     float64              `json:"regime_max_atr_percent"`
	ShortingEnabled         bool                 `json:"shorting_enabled"`
	ExecutionFillMode       string               `json:"execution_fill_mode"`
	TrainTestEnabled        bool                 `json:"train_test_enabled"`
	TrainRatio              float64              `json:"train_ratio"`
	TrainFrom               time.Time            `json:"train_from,omitempty"`
	TrainTo                 time.Time            `json:"train_to,omitempty"`
	TestFrom                time.Time            `json:"test_from,omitempty"`
	TestTo                  time.Time            `json:"test_to,omitempty"`
	WalkForwardEnabled      bool                 `json:"walk_forward_enabled"`
	WalkForwardFolds        int                  `json:"walk_forward_folds"`
	Results                 []OptimizationResult `json:"results"`
}

type OptimizationResult struct {
	Rank                        int                 `json:"rank"`
	StrategyName                string              `json:"strategy_name"`
	FastPeriod                  int                 `json:"fast_period"`
	SlowPeriod                  int                 `json:"slow_period"`
	RSIPeriod                   int                 `json:"rsi_period"`
	RSIOversold                 float64             `json:"rsi_oversold"`
	RSIOverbought               float64             `json:"rsi_overbought"`
	EndingBalance               float64             `json:"ending_balance"`
	ProfitLoss                  float64             `json:"profit_loss"`
	ReturnPercent               float64             `json:"return_percent"`
	ExcessReturnPercent         float64             `json:"excess_return_percent"`
	BenchmarkReturnPercent      float64             `json:"benchmark_return_percent"`
	ProfitFactor                float64             `json:"profit_factor"`
	MaxDrawdown                 float64             `json:"max_drawdown"`
	WinRate                     float64             `json:"win_rate"`
	TotalTrades                 int                 `json:"total_trades"`
	AverageTrade                float64             `json:"average_trade"`
	AverageHoldingSeconds       float64             `json:"average_holding_seconds"`
	Expectancy                  float64             `json:"expectancy"`
	TradesPerDay                float64             `json:"trades_per_day"`
	ChurnRatio                  float64             `json:"churn_ratio"`
	SharpeRatio                 float64             `json:"sharpe_ratio"`
	SortinoRatio                float64             `json:"sortino_ratio"`
	ExecutionFillMode           string              `json:"execution_fill_mode"`
	ValidationStatus            string              `json:"validation_status"`
	ValidationReason            string              `json:"validation_reason"`
	TrainReturnPercent          float64             `json:"train_return_percent"`
	TrainExcessReturn           float64             `json:"train_excess_return_percent"`
	TrainProfitFactor           float64             `json:"train_profit_factor"`
	TrainMaxDrawdown            float64             `json:"train_max_drawdown"`
	TrainTotalTrades            int                 `json:"train_total_trades"`
	TrainValidationStatus       string              `json:"train_validation_status"`
	TrainValidationReason       string              `json:"train_validation_reason"`
	TestReturnPercent           float64             `json:"test_return_percent"`
	TestExcessReturn            float64             `json:"test_excess_return_percent"`
	TestProfitFactor            float64             `json:"test_profit_factor"`
	TestMaxDrawdown             float64             `json:"test_max_drawdown"`
	TestTotalTrades             int                 `json:"test_total_trades"`
	TestValidationStatus        string              `json:"test_validation_status"`
	TestValidationReason        string              `json:"test_validation_reason"`
	WalkForwardFolds            int                 `json:"walk_forward_folds"`
	WalkForwardPasses           int                 `json:"walk_forward_passes"`
	WalkForwardAverageReturn    float64             `json:"walk_forward_average_return"`
	WalkForwardAverageExcess    float64             `json:"walk_forward_average_excess"`
	WalkForwardWorstDrawdown    float64             `json:"walk_forward_worst_drawdown"`
	WalkForwardValidationStatus string              `json:"walk_forward_validation_status"`
	WalkForwardValidationReason string              `json:"walk_forward_validation_reason"`
	WalkForwardResults          []WalkForwardResult `json:"walk_forward_results,omitempty"`
}

type WalkForwardResult struct {
	Fold             int       `json:"fold"`
	From             time.Time `json:"from"`
	To               time.Time `json:"to"`
	ReturnPercent    float64   `json:"return_percent"`
	ExcessReturn     float64   `json:"excess_return_percent"`
	ProfitFactor     float64   `json:"profit_factor"`
	MaxDrawdown      float64   `json:"max_drawdown"`
	TotalTrades      int       `json:"total_trades"`
	ValidationStatus string    `json:"validation_status"`
	ValidationReason string    `json:"validation_reason"`
}

type Optimizer struct {
	engine *Engine
}

type optimizationCandidate struct {
	fast          int
	slow          int
	rsiPeriod     int
	rsiOversold   float64
	rsiOverbought float64
}

func NewOptimizer(engine *Engine) *Optimizer {
	if engine == nil {
		engine = NewEngine()
	}
	return &Optimizer{engine: engine}
}

func DefaultOptimizationRequest() OptimizationRequest {
	request := DefaultRequest()
	return OptimizationRequest{
		StrategyName:        request.StrategyName,
		Symbol:              request.Symbol,
		Interval:            request.Interval,
		FastPeriods:         []int{5, 10, 15, 20},
		SlowPeriods:         []int{50, 100, 200},
		RSIPeriods:          []int{14},
		RSIOversoldValues:   []float64{30},
		RSIOverboughtValues: []float64{70},
		StartingBalance:     request.StartingBalance,
		FeeRate:             request.FeeRate,
		SlippageRate:        request.SlippageRate,
		ExecutionFillMode:   ExecutionFillModeNextOpen,
		PositionSizingMode:  PositionSizingPercentEquity,
		PositionSizeValue:   10,
		TrendFilterEnabled:  false,
		TrendPeriod:         200,
		ATRPeriod:           14,
		RegimeFilterPeriod:  14,
		TrainTestEnabled:    false,
		TrainRatio:          0.7,
		WalkForwardEnabled:  false,
		WalkForwardFolds:    4,
		MaxCombinations:     maxOptimizationCombinations,
	}
}

func (r OptimizationRequest) Normalize() OptimizationRequest {
	r.StrategyName = strings.TrimSpace(r.StrategyName)
	if r.StrategyName == "" {
		r.StrategyName = strategy.StrategySMACrossover
	}
	r.Symbol = strings.ToUpper(strings.TrimSpace(r.Symbol))
	if r.Symbol == "" {
		r.Symbol = "BTCUSDT"
	}
	r.Interval = strings.TrimSpace(r.Interval)
	if r.Interval == "" {
		r.Interval = "15m"
	}
	if len(r.FastPeriods) == 0 {
		r.FastPeriods = []int{5, 10, 15, 20}
	}
	if len(r.SlowPeriods) == 0 {
		r.SlowPeriods = []int{50, 100, 200}
	}
	if len(r.RSIPeriods) == 0 {
		r.RSIPeriods = []int{14}
	}
	if len(r.RSIOversoldValues) == 0 {
		r.RSIOversoldValues = []float64{30}
	}
	if len(r.RSIOverboughtValues) == 0 {
		r.RSIOverboughtValues = []float64{70}
	}
	if r.StartingBalance <= 0 {
		r.StartingBalance = 1000
	}
	r.PositionSizingMode = strings.TrimSpace(r.PositionSizingMode)
	if r.PositionSizingMode == "" {
		r.PositionSizingMode = PositionSizingPercentEquity
	}
	r.ExecutionFillMode = strings.TrimSpace(r.ExecutionFillMode)
	if r.ExecutionFillMode == "" {
		r.ExecutionFillMode = ExecutionFillModeNextOpen
	}
	if r.PositionSizeValue <= 0 {
		r.PositionSizeValue = 10
	}
	if r.TrendFilterEnabled && r.TrendPeriod <= 0 {
		r.TrendPeriod = 200
	}
	if r.ATRExitEnabled && r.ATRPeriod <= 0 {
		r.ATRPeriod = 14
	}
	if r.RegimeFilterEnabled && r.RegimeFilterPeriod <= 0 {
		r.RegimeFilterPeriod = 14
	}
	if r.TrainTestEnabled && r.TrainRatio == 0 {
		r.TrainRatio = 0.7
	}
	if r.WalkForwardEnabled && r.WalkForwardFolds == 0 {
		r.WalkForwardFolds = 4
	}
	if r.MaxCombinations <= 0 {
		r.MaxCombinations = maxOptimizationCombinations
	}
	return r
}

func (r OptimizationRequest) Validate() error {
	if strings.TrimSpace(r.Symbol) == "" {
		return fmt.Errorf("symbol is required")
	}
	if strings.TrimSpace(r.Interval) == "" {
		return fmt.Errorf("interval is required")
	}
	if r.From.IsZero() || r.To.IsZero() {
		return fmt.Errorf("from and to are required")
	}
	if !r.To.After(r.From) {
		return fmt.Errorf("to must be after from")
	}
	if r.StartingBalance <= 0 {
		return fmt.Errorf("starting_balance must be positive")
	}
	if r.FeeRate < 0 {
		return fmt.Errorf("fee_rate cannot be negative")
	}
	if r.SlippageRate < 0 {
		return fmt.Errorf("slippage_rate cannot be negative")
	}
	switch r.ExecutionFillMode {
	case ExecutionFillModeSameClose, ExecutionFillModeNextOpen:
	default:
		return fmt.Errorf("unsupported execution_fill_mode %q", r.ExecutionFillMode)
	}
	candidates, err := r.candidates()
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return fmt.Errorf("optimization grid produced no valid combinations")
	}
	if len(candidates) > r.MaxCombinations {
		return fmt.Errorf("optimization combinations exceed limit %d", r.MaxCombinations)
	}
	if r.TrainTestEnabled && (r.TrainRatio <= 0 || r.TrainRatio >= 1) {
		return fmt.Errorf("train_ratio must be greater than 0 and less than 1")
	}
	if r.WalkForwardEnabled && r.WalkForwardFolds < 2 {
		return fmt.Errorf("walk_forward_folds must be at least 2")
	}
	for _, period := range append([]int{}, r.FastPeriods...) {
		if period <= 0 {
			return fmt.Errorf("fast_periods must be positive")
		}
	}
	for _, period := range append([]int{}, r.SlowPeriods...) {
		if period <= 0 {
			return fmt.Errorf("slow_periods must be positive")
		}
	}
	for _, period := range append([]int{}, r.RSIPeriods...) {
		if period <= 0 {
			return fmt.Errorf("rsi_periods must be positive")
		}
	}
	for _, value := range append([]float64{}, r.RSIOversoldValues...) {
		if value <= 0 {
			return fmt.Errorf("rsi_oversold_values must be positive")
		}
	}
	for _, value := range append([]float64{}, r.RSIOverboughtValues...) {
		if value >= 100 {
			return fmt.Errorf("rsi_overbought_values must be less than 100")
		}
	}
	if err := (Request{
		StrategyName:            r.StrategyName,
		Version:                 "v1",
		Symbol:                  r.Symbol,
		Interval:                r.Interval,
		From:                    r.From,
		To:                      r.To,
		FastPeriod:              1,
		SlowPeriod:              2,
		StartingBalance:         r.StartingBalance,
		FeeRate:                 r.FeeRate,
		SlippageRate:            r.SlippageRate,
		ExecutionFillMode:       r.ExecutionFillMode,
		PositionSizingMode:      r.PositionSizingMode,
		PositionSizeValue:       r.PositionSizeValue,
		TrendFilterEnabled:      r.TrendFilterEnabled,
		TrendPeriod:             r.TrendPeriod,
		CooldownBars:            r.CooldownBars,
		MinHoldingBars:          r.MinHoldingBars,
		ATRExitEnabled:          r.ATRExitEnabled,
		ATRPeriod:               r.ATRPeriod,
		ATRStopMultiplier:       r.ATRStopMultiplier,
		ATRTakeProfitMultiplier: r.ATRTakeProfitMultiplier,
		RegimeFilterEnabled:     r.RegimeFilterEnabled,
		RegimeFilterPeriod:      r.RegimeFilterPeriod,
		RegimeMinATRPercent:     r.RegimeMinATRPercent,
		RegimeMaxATRPercent:     r.RegimeMaxATRPercent,
		ShortingEnabled:         r.ShortingEnabled,
	}).Normalize().Validate(); err != nil {
		return err
	}
	return nil
}

func (r OptimizationRequest) RequiredCandles() int {
	required := 0
	for _, candidate := range r.candidatesUnchecked() {
		if candidate.slow+1 > required {
			required = candidate.slow + 1
		}
		if (r.StrategyName == strategy.StrategyRSIMeanReversion || r.StrategyName == strategy.StrategyBTCTrendPullback) && candidate.rsiPeriod+1 > required {
			required = candidate.rsiPeriod + 1
		}
	}
	if r.TrendFilterEnabled && r.TrendPeriod > required {
		required = r.TrendPeriod
	}
	if r.ATRExitEnabled && r.ATRPeriod+1 > required {
		required = r.ATRPeriod + 1
	}
	if r.RegimeFilterEnabled && r.RegimeFilterPeriod+1 > required {
		required = r.RegimeFilterPeriod + 1
	}
	return required
}

func (r OptimizationRequest) candidates() ([]optimizationCandidate, error) {
	candidates := make([]optimizationCandidate, 0)
	for _, candidate := range r.candidatesUnchecked() {
		if candidate.fast <= 0 || candidate.slow <= 0 || candidate.rsiPeriod <= 0 {
			return nil, fmt.Errorf("optimization periods must be positive")
		}
		if r.StrategyName != strategy.StrategyRSIMeanReversion && candidate.slow <= candidate.fast {
			continue
		}
		if candidate.rsiOversold <= 0 || candidate.rsiOverbought <= candidate.rsiOversold || candidate.rsiOverbought >= 100 {
			return nil, fmt.Errorf("RSI thresholds are invalid")
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func (r OptimizationRequest) candidatesUnchecked() []optimizationCandidate {
	switch r.StrategyName {
	case strategy.StrategyRSIMeanReversion:
		candidates := make([]optimizationCandidate, 0, len(r.RSIPeriods)*len(r.RSIOversoldValues)*len(r.RSIOverboughtValues))
		for _, rsiPeriod := range r.RSIPeriods {
			for _, oversold := range r.RSIOversoldValues {
				for _, overbought := range r.RSIOverboughtValues {
					candidates = append(candidates, optimizationCandidate{
						fast:          9,
						slow:          21,
						rsiPeriod:     rsiPeriod,
						rsiOversold:   oversold,
						rsiOverbought: overbought,
					})
				}
			}
		}
		return candidates
	case strategy.StrategyBTCTrendPullback:
		candidates := make([]optimizationCandidate, 0, len(r.FastPeriods)*len(r.SlowPeriods)*len(r.RSIPeriods))
		for _, fast := range r.FastPeriods {
			for _, slow := range r.SlowPeriods {
				for _, rsiPeriod := range r.RSIPeriods {
					candidates = append(candidates, optimizationCandidate{
						fast:          fast,
						slow:          slow,
						rsiPeriod:     rsiPeriod,
						rsiOversold:   30,
						rsiOverbought: 70,
					})
				}
			}
		}
		return candidates
	default:
		candidates := make([]optimizationCandidate, 0, len(r.FastPeriods)*len(r.SlowPeriods))
		for _, fast := range r.FastPeriods {
			for _, slow := range r.SlowPeriods {
				candidates = append(candidates, optimizationCandidate{
					fast:          fast,
					slow:          slow,
					rsiPeriod:     14,
					rsiOversold:   30,
					rsiOverbought: 70,
				})
			}
		}
		return candidates
	}
}

func (o *Optimizer) Optimize(request OptimizationRequest, candles []marketdata.Candle) (Optimization, error) {
	request = request.Normalize()
	if err := request.Validate(); err != nil {
		return Optimization{}, err
	}
	required := request.RequiredCandles()
	if len(candles) < required {
		return Optimization{}, fmt.Errorf("insufficient candles: need %d, got %d", required, len(candles))
	}
	var split optimizationSplit
	if request.TrainTestEnabled {
		var err error
		split, err = splitOptimizationCandles(candles, request.TrainRatio, required)
		if err != nil {
			return Optimization{}, err
		}
	}
	var walkForwardFolds []walkForwardFold
	if request.WalkForwardEnabled {
		var err error
		walkForwardFolds, err = splitWalkForwardCandles(candles, request.WalkForwardFolds, required)
		if err != nil {
			return Optimization{}, err
		}
	}
	results := []OptimizationResult{}
	candidates, err := request.candidates()
	if err != nil {
		return Optimization{}, err
	}
	for _, candidate := range candidates {
		run, _, err := o.engine.Run(request.backtestRequestForCandidate(candidate), candles)
		if err != nil {
			return Optimization{}, err
		}
		row := optimizationResultFromRun(run)
		if request.TrainTestEnabled {
			trainRun, _, err := o.engine.Run(request.backtestRequestForCandidateAndCandles(candidate, split.trainCandles), split.trainCandles)
			if err != nil {
				return Optimization{}, err
			}
			testRun, _, err := o.engine.Run(request.backtestRequestForCandidateAndCandles(candidate, split.testCandles), split.testCandles)
			if err != nil {
				return Optimization{}, err
			}
			applyTrainTestMetrics(&row, trainRun, testRun)
		}
		if request.WalkForwardEnabled {
			if err := o.applyWalkForwardMetrics(&row, request, candidate, walkForwardFolds); err != nil {
				return Optimization{}, err
			}
		}
		results = append(results, row)
	}
	rankOptimizationResults(results)
	for index := range results {
		results[index].Rank = index + 1
	}
	return Optimization{
		StrategyName:            request.StrategyName,
		Symbol:                  request.Symbol,
		Interval:                request.Interval,
		From:                    request.From,
		To:                      request.To,
		TotalCombinations:       len(results),
		PositionSizingMode:      request.PositionSizingMode,
		PositionSizeValue:       request.PositionSizeValue,
		TrendFilterEnabled:      request.TrendFilterEnabled,
		TrendPeriod:             request.TrendPeriod,
		CooldownBars:            request.CooldownBars,
		MinHoldingBars:          request.MinHoldingBars,
		ATRExitEnabled:          request.ATRExitEnabled,
		ATRPeriod:               request.ATRPeriod,
		ATRStopMultiplier:       request.ATRStopMultiplier,
		ATRTakeProfitMultiplier: request.ATRTakeProfitMultiplier,
		RegimeFilterEnabled:     request.RegimeFilterEnabled,
		RegimeFilterPeriod:      request.RegimeFilterPeriod,
		RegimeMinATRPercent:     request.RegimeMinATRPercent,
		RegimeMaxATRPercent:     request.RegimeMaxATRPercent,
		ShortingEnabled:         request.ShortingEnabled,
		ExecutionFillMode:       request.ExecutionFillMode,
		TrainTestEnabled:        request.TrainTestEnabled,
		TrainRatio:              request.TrainRatio,
		TrainFrom:               split.trainFrom,
		TrainTo:                 split.trainTo,
		TestFrom:                split.testFrom,
		TestTo:                  split.testTo,
		WalkForwardEnabled:      request.WalkForwardEnabled,
		WalkForwardFolds:        request.WalkForwardFolds,
		Results:                 results,
	}, nil
}

func (r OptimizationRequest) backtestRequest(fast int, slow int) Request {
	return r.backtestRequestForCandidate(optimizationCandidate{
		fast:          fast,
		slow:          slow,
		rsiPeriod:     firstInt(r.RSIPeriods, 14),
		rsiOversold:   firstFloat(r.RSIOversoldValues, 30),
		rsiOverbought: firstFloat(r.RSIOverboughtValues, 70),
	})
}

func (r OptimizationRequest) backtestRequestForCandidate(candidate optimizationCandidate) Request {
	return Request{
		StrategyName:            r.StrategyName,
		Version:                 "v1",
		Symbol:                  r.Symbol,
		Interval:                r.Interval,
		From:                    r.From,
		To:                      r.To,
		FastPeriod:              candidate.fast,
		SlowPeriod:              candidate.slow,
		RSIPeriod:               candidate.rsiPeriod,
		RSIOversold:             candidate.rsiOversold,
		RSIOverbought:           candidate.rsiOverbought,
		StartingBalance:         r.StartingBalance,
		FeeRate:                 r.FeeRate,
		SlippageRate:            r.SlippageRate,
		ExecutionFillMode:       r.ExecutionFillMode,
		PositionSizingMode:      r.PositionSizingMode,
		PositionSizeValue:       r.PositionSizeValue,
		TrendFilterEnabled:      r.TrendFilterEnabled,
		TrendPeriod:             r.TrendPeriod,
		CooldownBars:            r.CooldownBars,
		MinHoldingBars:          r.MinHoldingBars,
		ATRExitEnabled:          r.ATRExitEnabled,
		ATRPeriod:               r.ATRPeriod,
		ATRStopMultiplier:       r.ATRStopMultiplier,
		ATRTakeProfitMultiplier: r.ATRTakeProfitMultiplier,
		RegimeFilterEnabled:     r.RegimeFilterEnabled,
		RegimeFilterPeriod:      r.RegimeFilterPeriod,
		RegimeMinATRPercent:     r.RegimeMinATRPercent,
		RegimeMaxATRPercent:     r.RegimeMaxATRPercent,
		ShortingEnabled:         r.ShortingEnabled,
	}
}

func (r OptimizationRequest) backtestRequestForCandles(fast int, slow int, candles []marketdata.Candle) Request {
	request := r.backtestRequest(fast, slow)
	if len(candles) > 0 {
		request.From = candles[0].OpenTime
		request.To = candles[len(candles)-1].CloseTime
	}
	return request
}

func (r OptimizationRequest) backtestRequestForCandidateAndCandles(candidate optimizationCandidate, candles []marketdata.Candle) Request {
	request := r.backtestRequestForCandidate(candidate)
	if len(candles) > 0 {
		request.From = candles[0].OpenTime
		request.To = candles[len(candles)-1].CloseTime
	}
	return request
}

func firstInt(values []int, fallback int) int {
	if len(values) == 0 {
		return fallback
	}
	return values[0]
}

func firstFloat(values []float64, fallback float64) float64 {
	if len(values) == 0 {
		return fallback
	}
	return values[0]
}

func optimizationResultFromRun(run Run) OptimizationResult {
	return OptimizationResult{
		StrategyName:           run.StrategyName,
		FastPeriod:             run.FastPeriod,
		SlowPeriod:             run.SlowPeriod,
		RSIPeriod:              run.RSIPeriod,
		RSIOversold:            run.RSIOversold,
		RSIOverbought:          run.RSIOverbought,
		EndingBalance:          run.EndingBalance,
		ProfitLoss:             run.ProfitLoss,
		ReturnPercent:          run.ReturnPercent,
		ExcessReturnPercent:    run.ExcessReturnPercent,
		BenchmarkReturnPercent: run.BenchmarkReturnPercent,
		ProfitFactor:           run.ProfitFactor,
		MaxDrawdown:            run.MaxDrawdown,
		WinRate:                run.WinRate,
		TotalTrades:            run.TotalTrades,
		AverageTrade:           run.AverageTrade,
		AverageHoldingSeconds:  run.AverageHoldingSeconds,
		Expectancy:             run.Expectancy,
		TradesPerDay:           run.TradesPerDay,
		ChurnRatio:             run.ChurnRatio,
		SharpeRatio:            run.SharpeRatio,
		SortinoRatio:           run.SortinoRatio,
		ExecutionFillMode:      run.ExecutionFillMode,
		ValidationStatus:       run.ValidationStatus,
		ValidationReason:       run.ValidationReason,
	}
}

func applyTrainTestMetrics(row *OptimizationResult, train Run, test Run) {
	row.TrainReturnPercent = train.ReturnPercent
	row.TrainExcessReturn = train.ExcessReturnPercent
	row.TrainProfitFactor = train.ProfitFactor
	row.TrainMaxDrawdown = train.MaxDrawdown
	row.TrainTotalTrades = train.TotalTrades
	row.TrainValidationStatus = train.ValidationStatus
	row.TrainValidationReason = train.ValidationReason
	row.TestReturnPercent = test.ReturnPercent
	row.TestExcessReturn = test.ExcessReturnPercent
	row.TestProfitFactor = test.ProfitFactor
	row.TestMaxDrawdown = test.MaxDrawdown
	row.TestTotalTrades = test.TotalTrades
	row.TestValidationStatus = test.ValidationStatus
	row.TestValidationReason = test.ValidationReason
}

func (o *Optimizer) applyWalkForwardMetrics(row *OptimizationResult, request OptimizationRequest, candidate optimizationCandidate, folds []walkForwardFold) error {
	results := make([]WalkForwardResult, 0, len(folds))
	passes := 0
	totalReturn := 0.0
	totalExcess := 0.0
	worstDrawdown := 0.0
	for _, fold := range folds {
		run, _, err := o.engine.Run(request.backtestRequestForCandidateAndCandles(candidate, fold.candles), fold.candles)
		if err != nil {
			return err
		}
		if run.ValidationStatus == "candidate" {
			passes++
		}
		totalReturn += run.ReturnPercent
		totalExcess += run.ExcessReturnPercent
		if run.MaxDrawdown > worstDrawdown {
			worstDrawdown = run.MaxDrawdown
		}
		results = append(results, WalkForwardResult{
			Fold:             fold.index,
			From:             fold.from,
			To:               fold.to,
			ReturnPercent:    run.ReturnPercent,
			ExcessReturn:     run.ExcessReturnPercent,
			ProfitFactor:     run.ProfitFactor,
			MaxDrawdown:      run.MaxDrawdown,
			TotalTrades:      run.TotalTrades,
			ValidationStatus: run.ValidationStatus,
			ValidationReason: run.ValidationReason,
		})
	}
	row.WalkForwardFolds = len(results)
	row.WalkForwardPasses = passes
	row.WalkForwardResults = results
	if len(results) > 0 {
		row.WalkForwardAverageReturn = totalReturn / float64(len(results))
		row.WalkForwardAverageExcess = totalExcess / float64(len(results))
	}
	row.WalkForwardWorstDrawdown = worstDrawdown
	row.WalkForwardValidationStatus, row.WalkForwardValidationReason = validateWalkForward(row)
	return nil
}

func validateWalkForward(row *OptimizationResult) (string, string) {
	if row.WalkForwardFolds == 0 {
		return "", ""
	}
	if row.WalkForwardPasses == row.WalkForwardFolds {
		return "candidate", "all walk-forward folds passed"
	}
	if row.WalkForwardAverageExcess <= 0 {
		return "underperforms_walk_forward", "average walk-forward excess return must be positive"
	}
	return "unstable_walk_forward", "not all walk-forward folds passed validation"
}

func rankOptimizationResults(results []OptimizationResult) {
	sort.SliceStable(results, func(i, j int) bool {
		leftHasWalkForward := results[i].WalkForwardFolds > 0
		rightHasWalkForward := results[j].WalkForwardFolds > 0
		if leftHasWalkForward || rightHasWalkForward {
			leftWalkForward := survivesWalkForward(results[i])
			rightWalkForward := survivesWalkForward(results[j])
			if leftWalkForward != rightWalkForward {
				return leftWalkForward
			}
			if !leftWalkForward && !rightWalkForward && results[i].WalkForwardValidationStatus != results[j].WalkForwardValidationStatus {
				return validationRankLess(results[i].WalkForwardValidationStatus, results[j].WalkForwardValidationStatus)
			}
			if results[i].WalkForwardAverageExcess != results[j].WalkForwardAverageExcess {
				return results[i].WalkForwardAverageExcess > results[j].WalkForwardAverageExcess
			}
		}
		leftSurvivesSplit := survivesTrainTest(results[i])
		rightSurvivesSplit := survivesTrainTest(results[j])
		if leftSurvivesSplit != rightSurvivesSplit {
			return leftSurvivesSplit
		}
		if !leftSurvivesSplit && !rightSurvivesSplit {
			leftSplitRank := maxValidationRank(results[i].TrainValidationStatus, results[i].TestValidationStatus)
			rightSplitRank := maxValidationRank(results[j].TrainValidationStatus, results[j].TestValidationStatus)
			if leftSplitRank != rightSplitRank {
				return leftSplitRank < rightSplitRank
			}
		}
		if results[i].TestExcessReturn != results[j].TestExcessReturn {
			return results[i].TestExcessReturn > results[j].TestExcessReturn
		}
		leftCandidate := results[i].ValidationStatus == "candidate"
		rightCandidate := results[j].ValidationStatus == "candidate"
		if leftCandidate != rightCandidate {
			return leftCandidate
		}
		if !leftCandidate && !rightCandidate && results[i].ValidationStatus != results[j].ValidationStatus {
			return validationRankLess(results[i].ValidationStatus, results[j].ValidationStatus)
		}
		leftScore := optimizationScore(results[i])
		rightScore := optimizationScore(results[j])
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		if results[i].ProfitFactor != results[j].ProfitFactor {
			return results[i].ProfitFactor > results[j].ProfitFactor
		}
		if results[i].MaxDrawdown != results[j].MaxDrawdown {
			return results[i].MaxDrawdown < results[j].MaxDrawdown
		}
		return results[i].TotalTrades < results[j].TotalTrades
	})
}

func optimizationScore(result OptimizationResult) float64 {
	score := result.ExcessReturnPercent
	score += result.ProfitFactor * 2
	score += result.AverageTrade
	if result.TradesPerDay > 20 {
		score -= (result.TradesPerDay - 20) * 0.25
	}
	return score
}

func survivesWalkForward(result OptimizationResult) bool {
	if result.WalkForwardValidationStatus == "" {
		return false
	}
	return result.WalkForwardValidationStatus == "candidate"
}

func survivesTrainTest(result OptimizationResult) bool {
	if result.TrainValidationStatus == "" && result.TestValidationStatus == "" {
		return false
	}
	return result.TrainValidationStatus == "candidate" && result.TestValidationStatus == "candidate"
}

type optimizationSplit struct {
	trainCandles []marketdata.Candle
	testCandles  []marketdata.Candle
	trainFrom    time.Time
	trainTo      time.Time
	testFrom     time.Time
	testTo       time.Time
}

func splitOptimizationCandles(candles []marketdata.Candle, trainRatio float64, requiredCandles int) (optimizationSplit, error) {
	trainLength := int(float64(len(candles)) * trainRatio)
	if trainLength < requiredCandles {
		return optimizationSplit{}, fmt.Errorf("insufficient train candles: need %d, got %d", requiredCandles, trainLength)
	}
	testLength := len(candles) - trainLength
	if testLength < requiredCandles {
		return optimizationSplit{}, fmt.Errorf("insufficient test candles: need %d, got %d", requiredCandles, testLength)
	}
	trainCandles := candles[:trainLength]
	testCandles := candles[trainLength:]
	return optimizationSplit{
		trainCandles: trainCandles,
		testCandles:  testCandles,
		trainFrom:    trainCandles[0].OpenTime,
		trainTo:      trainCandles[len(trainCandles)-1].CloseTime,
		testFrom:     testCandles[0].OpenTime,
		testTo:       testCandles[len(testCandles)-1].CloseTime,
	}, nil
}

type walkForwardFold struct {
	index   int
	candles []marketdata.Candle
	from    time.Time
	to      time.Time
}

func splitWalkForwardCandles(candles []marketdata.Candle, folds int, requiredCandles int) ([]walkForwardFold, error) {
	if folds < 2 {
		return nil, fmt.Errorf("walk_forward_folds must be at least 2")
	}
	foldSize := len(candles) / folds
	if foldSize < requiredCandles {
		return nil, fmt.Errorf("insufficient walk-forward candles per fold: need %d, got %d", requiredCandles, foldSize)
	}
	results := make([]walkForwardFold, 0, folds)
	for index := 0; index < folds; index++ {
		start := index * foldSize
		end := start + foldSize
		if index == folds-1 {
			end = len(candles)
		}
		foldCandles := candles[start:end]
		if len(foldCandles) < requiredCandles {
			return nil, fmt.Errorf("insufficient walk-forward candles in fold %d: need %d, got %d", index+1, requiredCandles, len(foldCandles))
		}
		results = append(results, walkForwardFold{
			index:   index + 1,
			candles: foldCandles,
			from:    foldCandles[0].OpenTime,
			to:      foldCandles[len(foldCandles)-1].CloseTime,
		})
	}
	return results, nil
}
