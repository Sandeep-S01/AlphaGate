package backtest

import (
	"testing"
	"time"

	"sentra/internal/marketdata"
	"sentra/internal/strategy"
)

func TestOptimizerRunsSMAGridAndRanksResults(t *testing.T) {
	request := OptimizationRequest{
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriods:        []int{2, 3},
		SlowPeriods:        []int{4, 5},
		StartingBalance:    1000,
		FeeRate:            0.001,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
	}

	result, err := NewOptimizer(NewEngine()).Optimize(request, candlesForCrosses())
	if err != nil {
		t.Fatalf("Optimize returned error: %v", err)
	}

	if result.Symbol != "BTCUSDT" || result.Interval != "1m" {
		t.Fatalf("unexpected result market: %+v", result)
	}
	if len(result.Results) != 4 {
		t.Fatalf("expected 4 parameter results, got %d", len(result.Results))
	}
	for index, row := range result.Results {
		if row.Rank != index+1 {
			t.Fatalf("expected rank %d, got %+v", index+1, row)
		}
		if row.FastPeriod <= 0 || row.SlowPeriod <= row.FastPeriod {
			t.Fatalf("unexpected SMA params: %+v", row)
		}
	}
}

func TestOptimizerRanksCostAwareResultsAboveHighChurnResults(t *testing.T) {
	results := []OptimizationResult{
		{ReturnPercent: 5, ExcessReturnPercent: 6, ProfitFactor: 1.3, TradesPerDay: 80, AverageTrade: 0.1},
		{ReturnPercent: 4, ExcessReturnPercent: 5, ProfitFactor: 1.5, TradesPerDay: 8, AverageTrade: 2.0},
	}

	rankOptimizationResults(results)

	if results[0].TradesPerDay != 8 {
		t.Fatalf("expected lower churn cost-aware result first, got %+v", results)
	}
}

func TestOptimizerRanksLowBullMarketCaptureBelowOtherValidationFailures(t *testing.T) {
	results := []OptimizationResult{
		{
			FastPeriod:             9,
			SlowPeriod:             21,
			ReturnPercent:          30,
			ExcessReturnPercent:    40,
			BenchmarkReturnPercent: 65,
			ProfitFactor:           1.8,
			AverageTrade:           3,
			ValidationStatus:       "low_bull_market_capture",
		},
		{
			FastPeriod:          12,
			SlowPeriod:          30,
			ReturnPercent:       -1,
			ExcessReturnPercent: -5,
			ProfitFactor:        0.9,
			AverageTrade:        1,
			ValidationStatus:    "weak_profit_factor",
		},
	}

	rankOptimizationResults(results)

	if results[0].ValidationStatus == "low_bull_market_capture" {
		t.Fatalf("expected benchmark-capture failure to rank below other validation failures, got %+v", results)
	}
}

func TestOptimizerRanksWalkForwardBullCaptureFailureBelowUnstableWalkForward(t *testing.T) {
	results := []OptimizationResult{
		{
			FastPeriod:                  9,
			SlowPeriod:                  21,
			WalkForwardFolds:            4,
			WalkForwardAverageExcess:    100,
			WalkForwardValidationStatus: "low_bull_market_capture",
			WalkForwardValidationReason: "strategy captures too little of a strong positive benchmark move",
			ValidationStatus:            "weak_profit_factor",
			ProfitFactor:                1.8,
			AverageTrade:                3,
		},
		{
			FastPeriod:                  12,
			SlowPeriod:                  30,
			WalkForwardFolds:            4,
			WalkForwardAverageExcess:    -5,
			WalkForwardValidationStatus: "unstable_walk_forward",
			ValidationStatus:            "weak_profit_factor",
			ProfitFactor:                0.9,
			AverageTrade:                1,
		},
	}

	rankOptimizationResults(results)

	if results[0].WalkForwardValidationStatus == "low_bull_market_capture" {
		t.Fatalf("expected walk-forward bull-capture failure to rank below unstable walk-forward, got %+v", results)
	}
}

func TestOptimizerSkipsInvalidSMAPeriodCombinations(t *testing.T) {
	request := OptimizationRequest{
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriods:        []int{5},
		SlowPeriods:        []int{3, 6},
		StartingBalance:    1000,
		FeeRate:            0.001,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
	}

	result, err := NewOptimizer(NewEngine()).Optimize(request, candlesForCrosses())
	if err != nil {
		t.Fatalf("Optimize returned error: %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected only valid fast/slow combo, got %+v", result.Results)
	}
	if result.Results[0].FastPeriod != 5 || result.Results[0].SlowPeriod != 6 {
		t.Fatalf("unexpected remaining combo: %+v", result.Results[0])
	}
}

func TestOptimizerAddsTrainTestMetricsWhenEnabled(t *testing.T) {
	request := OptimizationRequest{
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(4000, 0).UTC(),
		FastPeriods:        []int{2},
		SlowPeriods:        []int{4},
		StartingBalance:    1000,
		FeeRate:            0.001,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
		TrainTestEnabled:   true,
		TrainRatio:         0.5,
	}

	result, err := NewOptimizer(NewEngine()).Optimize(request, candlesWithOpen(candlesForOptimizerSplit()))
	if err != nil {
		t.Fatalf("Optimize returned error: %v", err)
	}

	if !result.TrainTestEnabled || result.TrainRatio != 0.5 {
		t.Fatalf("expected train/test settings copied to result, got %+v", result)
	}
	if result.TrainFrom.IsZero() || result.TrainTo.IsZero() || result.TestFrom.IsZero() || result.TestTo.IsZero() {
		t.Fatalf("expected train/test date ranges, got %+v", result)
	}
	row := result.Results[0]
	if row.TrainValidationStatus == "" || row.TestValidationStatus == "" {
		t.Fatalf("expected train/test validation statuses, got %+v", row)
	}
	if row.TrainTotalTrades == 0 && row.TestTotalTrades == 0 {
		t.Fatalf("expected train/test trade metrics, got %+v", row)
	}
}

func TestOptimizerAddsWalkForwardMetricsWhenEnabled(t *testing.T) {
	request := OptimizationRequest{
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(5000, 0).UTC(),
		FastPeriods:        []int{2},
		SlowPeriods:        []int{4},
		StartingBalance:    1000,
		FeeRate:            0.001,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
		WalkForwardEnabled: true,
		WalkForwardFolds:   3,
	}

	result, err := NewOptimizer(NewEngine()).Optimize(request, candlesWithOpen(candlesForOptimizerWalkForward()))
	if err != nil {
		t.Fatalf("Optimize returned error: %v", err)
	}

	if !result.WalkForwardEnabled || result.WalkForwardFolds != 3 {
		t.Fatalf("expected walk-forward settings copied to result, got %+v", result)
	}
	row := result.Results[0]
	if row.WalkForwardFolds != 3 {
		t.Fatalf("expected 3 walk-forward folds, got %+v", row)
	}
	if row.WalkForwardValidationStatus == "" {
		t.Fatalf("expected walk-forward validation status, got %+v", row)
	}
	if len(row.WalkForwardResults) != 3 {
		t.Fatalf("expected fold details, got %+v", row.WalkForwardResults)
	}
	if row.WalkForwardAverageExcess == 0 && row.WalkForwardAverageReturn == 0 {
		t.Fatalf("expected aggregate walk-forward metrics, got %+v", row)
	}
}

func TestOptimizerPropagatesNextOpenFillMode(t *testing.T) {
	request := OptimizationRequest{
		Symbol:            "BTCUSDT",
		Interval:          "1m",
		From:              time.Unix(1000, 0).UTC(),
		To:                time.Unix(5000, 0).UTC(),
		FastPeriods:       []int{2},
		SlowPeriods:       []int{4},
		StartingBalance:   1000,
		FeeRate:           0.001,
		ExecutionFillMode: ExecutionFillModeNextOpen,
		MaxCombinations:   10,
	}

	result, err := NewOptimizer(NewEngine()).Optimize(request, candlesWithOpen(candlesForOptimizerWalkForward()))
	if err != nil {
		t.Fatalf("Optimize returned error: %v", err)
	}

	if result.ExecutionFillMode != ExecutionFillModeNextOpen {
		t.Fatalf("expected optimization fill mode next_open, got %q", result.ExecutionFillMode)
	}
	if len(result.Results) == 0 || result.Results[0].ExecutionFillMode != ExecutionFillModeNextOpen {
		t.Fatalf("expected result fill mode next_open, got %+v", result.Results)
	}
}

func TestOptimizationRequestPropagatesHardenedExecutionControls(t *testing.T) {
	request := OptimizationRequest{
		Symbol:                  "BTCUSDT",
		Interval:                "1m",
		From:                    time.Unix(1000, 0).UTC(),
		To:                      time.Unix(5000, 0).UTC(),
		FastPeriods:             []int{2},
		SlowPeriods:             []int{4},
		StartingBalance:         1000,
		FeeRate:                 0.001,
		ExecutionFillMode:       ExecutionFillModeNextOpen,
		PositionSizingMode:      PositionSizingPercentEquity,
		PositionSizeValue:       10,
		ATRExitEnabled:          true,
		ATRPeriod:               14,
		ATRStopMultiplier:       2,
		ATRTakeProfitMultiplier: 3,
		RegimeFilterEnabled:     true,
		RegimeFilterPeriod:      20,
		RegimeMinATRPercent:     0.25,
		RegimeMaxATRPercent:     4,
		ShortingEnabled:         true,
		MaxCombinations:         10,
	}

	if err := request.Normalize().Validate(); err != nil {
		t.Fatalf("expected hardened optimization request to validate: %v", err)
	}
	backtestRequest := request.Normalize().backtestRequest(2, 4)
	if !backtestRequest.RegimeFilterEnabled || backtestRequest.RegimeFilterPeriod != 20 {
		t.Fatalf("expected regime filter propagated, got %+v", backtestRequest)
	}
	if backtestRequest.RegimeMinATRPercent != 0.25 || backtestRequest.RegimeMaxATRPercent != 4 {
		t.Fatalf("expected regime bounds propagated, got %+v", backtestRequest)
	}
	if !backtestRequest.ShortingEnabled {
		t.Fatalf("expected shorting flag propagated, got %+v", backtestRequest)
	}
	if required := request.Normalize().RequiredCandles(); required != 21 {
		t.Fatalf("expected regime filter required candles, got %d", required)
	}
}

func TestOptimizationRequestPropagatesStrategyName(t *testing.T) {
	request := OptimizationRequest{
		StrategyName:      strategy.StrategyRSIMeanReversion,
		Symbol:            "BTCUSDT",
		Interval:          "1m",
		From:              time.Unix(1000, 0).UTC(),
		To:                time.Unix(5000, 0).UTC(),
		FastPeriods:       []int{2},
		SlowPeriods:       []int{4},
		StartingBalance:   1000,
		FeeRate:           0.001,
		ExecutionFillMode: ExecutionFillModeNextOpen,
		MaxCombinations:   10,
	}

	backtestRequest := request.Normalize().backtestRequest(2, 4)

	if backtestRequest.StrategyName != strategy.StrategyRSIMeanReversion {
		t.Fatalf("expected strategy name propagated, got %q", backtestRequest.StrategyName)
	}
	if err := request.Normalize().Validate(); err != nil {
		t.Fatalf("expected RSI optimization request to validate: %v", err)
	}
}

func TestOptimizerRunsRSIStrategySpecificGrid(t *testing.T) {
	request := OptimizationRequest{
		StrategyName:        strategy.StrategyRSIMeanReversion,
		Symbol:              "BTCUSDT",
		Interval:            "1m",
		From:                time.Unix(1000, 0).UTC(),
		To:                  time.Unix(5000, 0).UTC(),
		RSIPeriods:          []int{7, 14},
		RSIOversoldValues:   []float64{25, 30},
		RSIOverboughtValues: []float64{70},
		StartingBalance:     1000,
		FeeRate:             0.001,
		ExecutionFillMode:   ExecutionFillModeNextOpen,
		MaxCombinations:     10,
	}

	result, err := NewOptimizer(NewEngine()).Optimize(request, candlesWithOpen(candlesForOptimizerWalkForward()))
	if err != nil {
		t.Fatalf("Optimize returned error: %v", err)
	}

	if result.TotalCombinations != 4 {
		t.Fatalf("expected 4 RSI combinations, got %d", result.TotalCombinations)
	}
	for _, row := range result.Results {
		if row.StrategyName != strategy.StrategyRSIMeanReversion {
			t.Fatalf("expected RSI strategy row, got %+v", row)
		}
		if row.RSIPeriod != 7 && row.RSIPeriod != 14 {
			t.Fatalf("expected optimized RSI period in row, got %+v", row)
		}
		if row.RSIOversold != 25 && row.RSIOversold != 30 {
			t.Fatalf("expected optimized oversold threshold in row, got %+v", row)
		}
		if row.RSIOverbought != 70 {
			t.Fatalf("expected optimized overbought threshold in row, got %+v", row)
		}
	}
}

func TestOptimizerRunsTrendPullbackFastSlowAndRSIGrid(t *testing.T) {
	request := OptimizationRequest{
		StrategyName:      strategy.StrategyBTCTrendPullback,
		Symbol:            "BTCUSDT",
		Interval:          "1m",
		From:              time.Unix(1000, 0).UTC(),
		To:                time.Unix(5000, 0).UTC(),
		FastPeriods:       []int{2},
		SlowPeriods:       []int{4, 5},
		RSIPeriods:        []int{7, 14},
		StartingBalance:   1000,
		FeeRate:           0.001,
		ExecutionFillMode: ExecutionFillModeNextOpen,
		MaxCombinations:   10,
	}

	result, err := NewOptimizer(NewEngine()).Optimize(request, candlesWithOpen(candlesForOptimizerWalkForward()))
	if err != nil {
		t.Fatalf("Optimize returned error: %v", err)
	}

	if result.TotalCombinations != 4 {
		t.Fatalf("expected 4 trend-pullback combinations, got %d", result.TotalCombinations)
	}
	for _, row := range result.Results {
		if row.RSIPeriod != 7 && row.RSIPeriod != 14 {
			t.Fatalf("expected trend-pullback RSI period in row, got %+v", row)
		}
	}
}

func TestOptimizerReturnsSelectedStrategyName(t *testing.T) {
	request := OptimizationRequest{
		StrategyName:      strategy.StrategyBTCTrendPullback,
		Symbol:            "BTCUSDT",
		Interval:          "1m",
		From:              time.Unix(1000, 0).UTC(),
		To:                time.Unix(5000, 0).UTC(),
		FastPeriods:       []int{2},
		SlowPeriods:       []int{4},
		StartingBalance:   1000,
		FeeRate:           0.001,
		ExecutionFillMode: ExecutionFillModeNextOpen,
		MaxCombinations:   10,
	}

	result, err := NewOptimizer(NewEngine()).Optimize(request, candlesWithOpen(candlesForOptimizerWalkForward()))
	if err != nil {
		t.Fatalf("Optimize returned error: %v", err)
	}

	if result.StrategyName != strategy.StrategyBTCTrendPullback {
		t.Fatalf("expected optimization strategy %q, got %q", strategy.StrategyBTCTrendPullback, result.StrategyName)
	}
	if len(result.Results) == 0 || result.Results[0].StrategyName != strategy.StrategyBTCTrendPullback {
		t.Fatalf("expected result rows to carry selected strategy, got %+v", result.Results)
	}
}

func TestOptimizationRequestRejectsUnsupportedStrategyName(t *testing.T) {
	request := DefaultOptimizationRequest()
	request.From = time.Unix(1000, 0).UTC()
	request.To = time.Unix(2000, 0).UTC()
	request.StrategyName = "not-a-strategy"

	if err := request.Normalize().Validate(); err == nil {
		t.Fatal("expected unsupported strategy validation error")
	}
}

func TestOptimizationRequestDefaultsToNextOpenFillMode(t *testing.T) {
	request := DefaultOptimizationRequest()
	request.ExecutionFillMode = ""

	normalized := request.Normalize()

	if normalized.ExecutionFillMode != ExecutionFillModeNextOpen {
		t.Fatalf("expected optimization default fill mode next_open, got %q", normalized.ExecutionFillMode)
	}
}

func TestOptimizationRequestRejectsInvalidWalkForwardFolds(t *testing.T) {
	request := DefaultOptimizationRequest()
	request.From = time.Unix(1000, 0).UTC()
	request.To = time.Unix(2000, 0).UTC()
	request.WalkForwardEnabled = true
	request.WalkForwardFolds = 1

	if err := request.Normalize().Validate(); err == nil {
		t.Fatal("expected invalid walk-forward folds validation error")
	}
}

func TestOptimizationRequestRejectsInvalidTrainRatio(t *testing.T) {
	request := DefaultOptimizationRequest()
	request.From = time.Unix(1000, 0).UTC()
	request.To = time.Unix(2000, 0).UTC()
	request.TrainTestEnabled = true
	request.TrainRatio = 1

	if err := request.Normalize().Validate(); err == nil {
		t.Fatal("expected invalid train ratio validation error")
	}
}

func TestOptimizationRequestRejectsTooManyCombinations(t *testing.T) {
	request := DefaultOptimizationRequest()
	request.From = time.Unix(1000, 0).UTC()
	request.To = time.Unix(2000, 0).UTC()
	request.FastPeriods = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	request.SlowPeriods = []int{20, 21, 22, 23, 24, 25, 26, 27, 28, 29}

	if err := request.Normalize().Validate(); err == nil {
		t.Fatal("expected too many combinations validation error")
	}
}

func candlesForOptimizerSplit() []marketdata.Candle {
	base := time.Unix(1000, 0).UTC()
	closes := []string{
		"10", "10", "10", "12", "15", "18", "16", "14", "11", "9",
		"12", "16", "18", "15", "12", "10", "13", "17", "19", "16",
		"12", "9", "11", "15", "18", "20", "17", "13", "10", "8",
		"11", "14", "18", "21", "19", "16", "12", "9", "13", "17",
	}
	candles := make([]marketdata.Candle, 0, len(closes))
	for index, closeValue := range closes {
		openTime := base.Add(time.Duration(index) * time.Minute)
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Close:     closeValue,
			IsClosed:  true,
		})
	}
	return candles
}

func candlesForOptimizerWalkForward() []marketdata.Candle {
	base := time.Unix(1000, 0).UTC()
	closes := []string{
		"10", "10", "10", "12", "15", "18", "16", "14", "11", "9",
		"12", "16", "18", "15", "12", "10", "13", "17", "19", "16",
		"12", "9", "11", "15", "18", "20", "17", "13", "10", "8",
		"11", "14", "18", "21", "19", "16", "12", "9", "13", "17",
		"20", "18", "14", "11", "9", "12", "16", "19", "21", "18",
		"15", "11", "8", "10", "14", "17", "20", "22", "19", "15",
	}
	candles := make([]marketdata.Candle, 0, len(closes))
	for index, closeValue := range closes {
		openTime := base.Add(time.Duration(index) * time.Minute)
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Close:     closeValue,
			IsClosed:  true,
		})
	}
	return candles
}
