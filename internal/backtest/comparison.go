package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"sentra/internal/marketdata"
	"sentra/internal/strategy"
)

type ComparisonRequest struct {
	Symbol             string    `json:"symbol"`
	Interval           string    `json:"interval"`
	From               time.Time `json:"from"`
	To                 time.Time `json:"to"`
	FastPeriod         int       `json:"fast_period"`
	SlowPeriod         int       `json:"slow_period"`
	RSIPeriod          int       `json:"rsi_period"`
	RSIOversold        float64   `json:"rsi_oversold"`
	RSIOverbought      float64   `json:"rsi_overbought"`
	StartingBalance    float64   `json:"starting_balance"`
	FeeRate            float64   `json:"fee_rate"`
	SlippageRate       float64   `json:"slippage_rate"`
	ExecutionFillMode  string    `json:"execution_fill_mode"`
	PositionSizingMode string    `json:"position_sizing_mode"`
	PositionSizeValue  float64   `json:"position_size_value"`
	TrendFilterEnabled bool      `json:"trend_filter_enabled"`
	TrendPeriod        int       `json:"trend_period"`
	CooldownBars       int       `json:"cooldown_bars"`
	MinHoldingBars     int       `json:"min_holding_bars"`
	TrainTestEnabled   bool      `json:"train_test_enabled"`
	TrainRatio         float64   `json:"train_ratio"`
	WalkForwardEnabled bool      `json:"walk_forward_enabled"`
	WalkForwardFolds   int       `json:"walk_forward_folds"`
}

type Comparison struct {
	ID                 string             `json:"id"`
	Symbol             string             `json:"symbol"`
	Interval           string             `json:"interval"`
	From               time.Time          `json:"from"`
	To                 time.Time          `json:"to"`
	StartingBalance    float64            `json:"starting_balance"`
	FeeRate            float64            `json:"fee_rate"`
	SlippageRate       float64            `json:"slippage_rate"`
	ExecutionFillMode  string             `json:"execution_fill_mode"`
	PositionSizingMode string             `json:"position_sizing_mode"`
	PositionSizeValue  float64            `json:"position_size_value"`
	TrendFilterEnabled bool               `json:"trend_filter_enabled"`
	TrendPeriod        int                `json:"trend_period"`
	CooldownBars       int                `json:"cooldown_bars"`
	MinHoldingBars     int                `json:"min_holding_bars"`
	TrainTestEnabled   bool               `json:"train_test_enabled"`
	TrainRatio         float64            `json:"train_ratio"`
	TrainFrom          time.Time          `json:"train_from,omitempty"`
	TrainTo            time.Time          `json:"train_to,omitempty"`
	TestFrom           time.Time          `json:"test_from,omitempty"`
	TestTo             time.Time          `json:"test_to,omitempty"`
	WalkForwardEnabled bool               `json:"walk_forward_enabled"`
	WalkForwardFolds   int                `json:"walk_forward_folds"`
	WinnerStrategy     string             `json:"winner_strategy"`
	CreatedAt          time.Time          `json:"created_at"`
	Results            []ComparisonResult `json:"results"`
}

type ComparisonResult struct {
	ID                          string  `json:"id"`
	ComparisonID                string  `json:"comparison_id"`
	Rank                        int     `json:"rank"`
	StrategyName                string  `json:"strategy_name"`
	Version                     string  `json:"version"`
	FastPeriod                  int     `json:"fast_period"`
	SlowPeriod                  int     `json:"slow_period"`
	RSIPeriod                   int     `json:"rsi_period"`
	RSIOversold                 float64 `json:"rsi_oversold"`
	RSIOverbought               float64 `json:"rsi_overbought"`
	EndingBalance               float64 `json:"ending_balance"`
	ProfitLoss                  float64 `json:"profit_loss"`
	ReturnPercent               float64 `json:"return_percent"`
	WinRate                     float64 `json:"win_rate"`
	MaxDrawdown                 float64 `json:"max_drawdown"`
	TotalTrades                 int     `json:"total_trades"`
	BuyCount                    int     `json:"buy_count"`
	SellCount                   int     `json:"sell_count"`
	BestTrade                   float64 `json:"best_trade"`
	WorstTrade                  float64 `json:"worst_trade"`
	AverageWin                  float64 `json:"average_win"`
	AverageLoss                 float64 `json:"average_loss"`
	ProfitFactor                float64 `json:"profit_factor"`
	AverageTrade                float64 `json:"average_trade"`
	AverageHoldingSeconds       float64 `json:"average_holding_seconds"`
	Expectancy                  float64 `json:"expectancy"`
	TradesPerDay                float64 `json:"trades_per_day"`
	ChurnRatio                  float64 `json:"churn_ratio"`
	SharpeRatio                 float64 `json:"sharpe_ratio"`
	SortinoRatio                float64 `json:"sortino_ratio"`
	ExecutionFillMode           string  `json:"execution_fill_mode"`
	PositionSizingMode          string  `json:"position_sizing_mode"`
	PositionSizeValue           float64 `json:"position_size_value"`
	TrendFilterEnabled          bool    `json:"trend_filter_enabled"`
	TrendPeriod                 int     `json:"trend_period"`
	CooldownBars                int     `json:"cooldown_bars"`
	MinHoldingBars              int     `json:"min_holding_bars"`
	BenchmarkEndingBalance      float64 `json:"benchmark_ending_balance"`
	BenchmarkProfitLoss         float64 `json:"benchmark_profit_loss"`
	BenchmarkReturnPercent      float64 `json:"benchmark_return_percent"`
	ExcessReturnPercent         float64 `json:"excess_return_percent"`
	ValidationStatus            string  `json:"validation_status"`
	ValidationReason            string  `json:"validation_reason"`
	TrainReturnPercent          float64 `json:"train_return_percent"`
	TrainExcessReturn           float64 `json:"train_excess_return_percent"`
	TrainProfitFactor           float64 `json:"train_profit_factor"`
	TrainMaxDrawdown            float64 `json:"train_max_drawdown"`
	TrainTotalTrades            int     `json:"train_total_trades"`
	TrainValidationStatus       string  `json:"train_validation_status"`
	TrainValidationReason       string  `json:"train_validation_reason"`
	TestReturnPercent           float64 `json:"test_return_percent"`
	TestExcessReturn            float64 `json:"test_excess_return_percent"`
	TestProfitFactor            float64 `json:"test_profit_factor"`
	TestMaxDrawdown             float64 `json:"test_max_drawdown"`
	TestTotalTrades             int     `json:"test_total_trades"`
	TestValidationStatus        string  `json:"test_validation_status"`
	TestValidationReason        string  `json:"test_validation_reason"`
	WalkForwardFolds            int     `json:"walk_forward_folds"`
	WalkForwardPasses           int     `json:"walk_forward_passes"`
	WalkForwardAverageReturn    float64 `json:"walk_forward_average_return"`
	WalkForwardAverageExcess    float64 `json:"walk_forward_average_excess"`
	WalkForwardWorstDrawdown    float64 `json:"walk_forward_worst_drawdown"`
	WalkForwardValidationStatus string  `json:"walk_forward_validation_status"`
	WalkForwardValidationReason string  `json:"walk_forward_validation_reason"`
	OpenPosition                bool    `json:"open_position"`
}

type Comparator struct {
	engine *Engine
}

func NewComparator(engine *Engine) *Comparator {
	if engine == nil {
		engine = NewEngine()
	}
	return &Comparator{engine: engine}
}

func DefaultComparisonRequest() ComparisonRequest {
	request := DefaultRequest()
	return ComparisonRequest{
		Symbol:             request.Symbol,
		Interval:           request.Interval,
		FastPeriod:         request.FastPeriod,
		SlowPeriod:         request.SlowPeriod,
		RSIPeriod:          request.RSIPeriod,
		RSIOversold:        request.RSIOversold,
		RSIOverbought:      request.RSIOverbought,
		StartingBalance:    request.StartingBalance,
		FeeRate:            request.FeeRate,
		SlippageRate:       request.SlippageRate,
		ExecutionFillMode:  ExecutionFillModeNextOpen,
		PositionSizingMode: request.PositionSizingMode,
		PositionSizeValue:  request.PositionSizeValue,
		TrendFilterEnabled: request.TrendFilterEnabled,
		TrendPeriod:        request.TrendPeriod,
		CooldownBars:       request.CooldownBars,
		MinHoldingBars:     request.MinHoldingBars,
		TrainTestEnabled:   false,
		TrainRatio:         0.7,
		WalkForwardEnabled: false,
		WalkForwardFolds:   4,
	}
}

func (r ComparisonRequest) Normalize() ComparisonRequest {
	r.Symbol = strings.ToUpper(strings.TrimSpace(r.Symbol))
	r.Interval = strings.TrimSpace(r.Interval)
	r.PositionSizingMode = strings.TrimSpace(r.PositionSizingMode)
	if r.PositionSizingMode == "" {
		r.PositionSizingMode = PositionSizingAllIn
	}
	r.ExecutionFillMode = strings.TrimSpace(r.ExecutionFillMode)
	if r.ExecutionFillMode == "" {
		r.ExecutionFillMode = ExecutionFillModeNextOpen
	}
	if r.PositionSizingMode == PositionSizingAllIn && r.PositionSizeValue <= 0 {
		r.PositionSizeValue = 100
	}
	if r.TrendFilterEnabled && r.TrendPeriod <= 0 {
		r.TrendPeriod = 200
	}
	if r.TrainTestEnabled && r.TrainRatio == 0 {
		r.TrainRatio = 0.7
	}
	if r.WalkForwardEnabled && r.WalkForwardFolds == 0 {
		r.WalkForwardFolds = 4
	}
	return r
}

func (r ComparisonRequest) Validate() error {
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
	if r.FastPeriod <= 0 {
		return fmt.Errorf("fast_period must be positive")
	}
	if r.SlowPeriod <= r.FastPeriod {
		return fmt.Errorf("slow_period must be greater than fast_period")
	}
	if r.RSIPeriod <= 0 {
		return fmt.Errorf("rsi_period must be positive")
	}
	if r.RSIOversold <= 0 || r.RSIOverbought <= r.RSIOversold || r.RSIOverbought >= 100 {
		return fmt.Errorf("RSI thresholds are invalid")
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
	switch r.PositionSizingMode {
	case PositionSizingAllIn:
	case PositionSizingFixedQuote:
		if r.PositionSizeValue <= 0 {
			return fmt.Errorf("position_size_value must be positive for fixed_quote")
		}
	case PositionSizingPercentEquity:
		if r.PositionSizeValue <= 0 || r.PositionSizeValue > 100 {
			return fmt.Errorf("position_size_value must be between 0 and 100 for percent_equity")
		}
	default:
		return fmt.Errorf("unsupported position_sizing_mode %q", r.PositionSizingMode)
	}
	if r.TrendFilterEnabled && r.TrendPeriod <= 0 {
		return fmt.Errorf("trend_period must be positive when trend filter is enabled")
	}
	if r.CooldownBars < 0 {
		return fmt.Errorf("cooldown_bars cannot be negative")
	}
	if r.MinHoldingBars < 0 {
		return fmt.Errorf("min_holding_bars cannot be negative")
	}
	if r.TrainTestEnabled && (r.TrainRatio <= 0 || r.TrainRatio >= 1) {
		return fmt.Errorf("train_ratio must be greater than 0 and less than 1")
	}
	if r.WalkForwardEnabled && r.WalkForwardFolds < 2 {
		return fmt.Errorf("walk_forward_folds must be at least 2")
	}
	return nil
}

func (r ComparisonRequest) RequiredCandles() int {
	smaRequired := r.SlowPeriod + 1
	if r.TrendFilterEnabled && r.TrendPeriod > smaRequired {
		smaRequired = r.TrendPeriod
	}
	rsiRequired := r.RSIPeriod + 1
	if rsiRequired > smaRequired {
		return rsiRequired
	}
	return smaRequired
}

func (c *Comparator) Compare(request ComparisonRequest, candles []marketdata.Candle) (Comparison, error) {
	request = request.Normalize()
	if err := request.Validate(); err != nil {
		return Comparison{}, err
	}
	required := request.RequiredCandles()
	if len(candles) < required {
		return Comparison{}, fmt.Errorf("insufficient candles: need %d, got %d", required, len(candles))
	}
	var split optimizationSplit
	if request.TrainTestEnabled {
		var err error
		split, err = splitOptimizationCandles(candles, request.TrainRatio, required)
		if err != nil {
			return Comparison{}, err
		}
	}
	var walkForwardFolds []walkForwardFold
	if request.WalkForwardEnabled {
		var err error
		walkForwardFolds, err = splitWalkForwardCandles(candles, request.WalkForwardFolds, required)
		if err != nil {
			return Comparison{}, err
		}
	}

	strategyNames := []string{strategy.StrategySMACrossover, strategy.StrategyRSIMeanReversion, strategy.StrategyBTCTrendPullback}
	results := make([]ComparisonResult, 0, len(strategyNames))
	for _, strategyName := range strategyNames {
		run, _, err := c.engine.Run(request.BacktestRequest(strategyName), candles)
		if err != nil {
			return Comparison{}, err
		}
		row := comparisonResultFromRun(0, run)
		if request.TrainTestEnabled {
			trainRun, _, err := c.engine.Run(request.BacktestRequestForCandles(strategyName, split.trainCandles), split.trainCandles)
			if err != nil {
				return Comparison{}, err
			}
			testRun, _, err := c.engine.Run(request.BacktestRequestForCandles(strategyName, split.testCandles), split.testCandles)
			if err != nil {
				return Comparison{}, err
			}
			applyComparisonTrainTestMetrics(&row, trainRun, testRun)
		}
		if request.WalkForwardEnabled {
			if err := c.applyComparisonWalkForwardMetrics(&row, request, strategyName, walkForwardFolds); err != nil {
				return Comparison{}, err
			}
		}
		results = append(results, row)
	}

	rankComparisonResults(results)
	for index := range results {
		results[index].Rank = index + 1
	}
	return Comparison{
		Symbol:             request.Symbol,
		Interval:           request.Interval,
		From:               request.From,
		To:                 request.To,
		StartingBalance:    request.StartingBalance,
		FeeRate:            request.FeeRate,
		SlippageRate:       request.SlippageRate,
		ExecutionFillMode:  request.ExecutionFillMode,
		PositionSizingMode: request.PositionSizingMode,
		PositionSizeValue:  request.PositionSizeValue,
		TrendFilterEnabled: request.TrendFilterEnabled,
		TrendPeriod:        request.TrendPeriod,
		CooldownBars:       request.CooldownBars,
		MinHoldingBars:     request.MinHoldingBars,
		TrainTestEnabled:   request.TrainTestEnabled,
		TrainRatio:         request.TrainRatio,
		TrainFrom:          split.trainFrom,
		TrainTo:            split.trainTo,
		TestFrom:           split.testFrom,
		TestTo:             split.testTo,
		WalkForwardEnabled: request.WalkForwardEnabled,
		WalkForwardFolds:   request.WalkForwardFolds,
		WinnerStrategy:     results[0].StrategyName,
		Results:            results,
	}, nil
}

func (r ComparisonRequest) BacktestRequest(strategyName string) Request {
	return Request{
		StrategyName:       strategyName,
		Version:            "v1",
		Symbol:             r.Symbol,
		Interval:           r.Interval,
		From:               r.From,
		To:                 r.To,
		FastPeriod:         r.FastPeriod,
		SlowPeriod:         r.SlowPeriod,
		RSIPeriod:          r.RSIPeriod,
		RSIOversold:        r.RSIOversold,
		RSIOverbought:      r.RSIOverbought,
		StartingBalance:    r.StartingBalance,
		FeeRate:            r.FeeRate,
		SlippageRate:       r.SlippageRate,
		ExecutionFillMode:  r.ExecutionFillMode,
		PositionSizingMode: r.PositionSizingMode,
		PositionSizeValue:  r.PositionSizeValue,
		TrendFilterEnabled: r.TrendFilterEnabled,
		TrendPeriod:        r.TrendPeriod,
		CooldownBars:       r.CooldownBars,
		MinHoldingBars:     r.MinHoldingBars,
	}
}

func (r ComparisonRequest) BacktestRequestForCandles(strategyName string, candles []marketdata.Candle) Request {
	request := r.BacktestRequest(strategyName)
	if len(candles) > 0 {
		request.From = candles[0].OpenTime
		request.To = candles[len(candles)-1].CloseTime
	}
	return request
}

func rankRuns(runs []Run) {
	sort.SliceStable(runs, func(i, j int) bool {
		leftCandidate := runs[i].ValidationStatus == "candidate"
		rightCandidate := runs[j].ValidationStatus == "candidate"
		if leftCandidate != rightCandidate {
			return leftCandidate
		}
		if runs[i].ProfitFactor != runs[j].ProfitFactor {
			return runs[i].ProfitFactor > runs[j].ProfitFactor
		}
		if runs[i].MaxDrawdown != runs[j].MaxDrawdown {
			return runs[i].MaxDrawdown < runs[j].MaxDrawdown
		}
		if runs[i].ReturnPercent != runs[j].ReturnPercent {
			return runs[i].ReturnPercent > runs[j].ReturnPercent
		}
		return runs[i].TotalTrades > runs[j].TotalTrades
	})
}

func rankComparisonResults(results []ComparisonResult) {
	sort.SliceStable(results, func(i, j int) bool {
		leftHasWalkForward := results[i].WalkForwardFolds > 0
		rightHasWalkForward := results[j].WalkForwardFolds > 0
		if leftHasWalkForward || rightHasWalkForward {
			leftWalkForward := comparisonSurvivesWalkForward(results[i])
			rightWalkForward := comparisonSurvivesWalkForward(results[j])
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
		leftSurvivesSplit := comparisonSurvivesTrainTest(results[i])
		rightSurvivesSplit := comparisonSurvivesTrainTest(results[j])
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
		if results[i].ProfitFactor != results[j].ProfitFactor {
			return results[i].ProfitFactor > results[j].ProfitFactor
		}
		if results[i].MaxDrawdown != results[j].MaxDrawdown {
			return results[i].MaxDrawdown < results[j].MaxDrawdown
		}
		if results[i].ReturnPercent != results[j].ReturnPercent {
			return results[i].ReturnPercent > results[j].ReturnPercent
		}
		return results[i].TotalTrades > results[j].TotalTrades
	})
}

func comparisonSurvivesWalkForward(result ComparisonResult) bool {
	if result.WalkForwardValidationStatus == "" {
		return false
	}
	return result.WalkForwardValidationStatus == "candidate"
}

func comparisonSurvivesTrainTest(result ComparisonResult) bool {
	if result.TrainValidationStatus == "" && result.TestValidationStatus == "" {
		return false
	}
	return result.TrainValidationStatus == "candidate" && result.TestValidationStatus == "candidate"
}

func applyComparisonTrainTestMetrics(row *ComparisonResult, train Run, test Run) {
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

func (c *Comparator) applyComparisonWalkForwardMetrics(row *ComparisonResult, request ComparisonRequest, strategyName string, folds []walkForwardFold) error {
	passes := 0
	totalReturn := 0.0
	totalExcess := 0.0
	worstDrawdown := 0.0
	for _, fold := range folds {
		run, _, err := c.engine.Run(request.BacktestRequestForCandles(strategyName, fold.candles), fold.candles)
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
	}
	row.WalkForwardFolds = len(folds)
	row.WalkForwardPasses = passes
	if len(folds) > 0 {
		row.WalkForwardAverageReturn = totalReturn / float64(len(folds))
		row.WalkForwardAverageExcess = totalExcess / float64(len(folds))
	}
	row.WalkForwardWorstDrawdown = worstDrawdown
	row.WalkForwardValidationStatus, row.WalkForwardValidationReason = validateComparisonWalkForward(row)
	return nil
}

func validateComparisonWalkForward(row *ComparisonResult) (string, string) {
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

func comparisonResultFromRun(rank int, run Run) ComparisonResult {
	return ComparisonResult{
		Rank:                   rank,
		StrategyName:           run.StrategyName,
		Version:                run.Version,
		FastPeriod:             run.FastPeriod,
		SlowPeriod:             run.SlowPeriod,
		RSIPeriod:              run.RSIPeriod,
		RSIOversold:            run.RSIOversold,
		RSIOverbought:          run.RSIOverbought,
		EndingBalance:          run.EndingBalance,
		ProfitLoss:             run.ProfitLoss,
		ReturnPercent:          run.ReturnPercent,
		WinRate:                run.WinRate,
		MaxDrawdown:            run.MaxDrawdown,
		TotalTrades:            run.TotalTrades,
		BuyCount:               run.BuyCount,
		SellCount:              run.SellCount,
		BestTrade:              run.BestTrade,
		WorstTrade:             run.WorstTrade,
		AverageWin:             run.AverageWin,
		AverageLoss:            run.AverageLoss,
		ProfitFactor:           run.ProfitFactor,
		AverageTrade:           run.AverageTrade,
		AverageHoldingSeconds:  run.AverageHoldingSeconds,
		Expectancy:             run.Expectancy,
		TradesPerDay:           run.TradesPerDay,
		ChurnRatio:             run.ChurnRatio,
		SharpeRatio:            run.SharpeRatio,
		SortinoRatio:           run.SortinoRatio,
		ExecutionFillMode:      run.ExecutionFillMode,
		PositionSizingMode:     run.PositionSizingMode,
		PositionSizeValue:      run.PositionSizeValue,
		TrendFilterEnabled:     run.TrendFilterEnabled,
		TrendPeriod:            run.TrendPeriod,
		CooldownBars:           run.CooldownBars,
		MinHoldingBars:         run.MinHoldingBars,
		BenchmarkEndingBalance: run.BenchmarkEndingBalance,
		BenchmarkProfitLoss:    run.BenchmarkProfitLoss,
		BenchmarkReturnPercent: run.BenchmarkReturnPercent,
		ExcessReturnPercent:    run.ExcessReturnPercent,
		ValidationStatus:       run.ValidationStatus,
		ValidationReason:       run.ValidationReason,
		OpenPosition:           run.OpenPosition,
	}
}
