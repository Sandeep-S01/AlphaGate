package backtest

import (
	"testing"
	"time"

	"sentra/internal/marketdata"
)

func TestCompareStrategiesRanksRunsWithDiagnostics(t *testing.T) {
	request := ComparisonRequest{
		Symbol:          "BTCUSDT",
		Interval:        "1m",
		From:            time.Unix(1000, 0).UTC(),
		To:              time.Unix(2000, 0).UTC(),
		FastPeriod:      2,
		SlowPeriod:      3,
		RSIPeriod:       3,
		RSIOversold:     30,
		RSIOverbought:   70,
		StartingBalance: 1000,
		FeeRate:         0.001,
	}

	comparison, err := NewComparator(NewEngine()).Compare(request, candlesForLargeStress(18, "1m", time.Minute))
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}

	if comparison.Symbol != "BTCUSDT" || comparison.Interval != "1m" {
		t.Fatalf("unexpected comparison market: %+v", comparison)
	}
	if len(comparison.Results) != 3 {
		t.Fatalf("expected three strategy results, got %d", len(comparison.Results))
	}
	if comparison.Results[0].Rank != 1 || comparison.Results[1].Rank != 2 || comparison.Results[2].Rank != 3 {
		t.Fatalf("expected ranked results, got %+v", comparison.Results)
	}
	if comparison.WinnerStrategy == "" {
		t.Fatalf("expected winner strategy, got %+v", comparison)
	}
	if comparison.Results[0].ValidationStatus == "" || comparison.Results[0].ProfitFactor < 0 {
		t.Fatalf("expected diagnostic fields in comparison result, got %+v", comparison.Results[0])
	}
}

func candlesWithOpen(candles []marketdata.Candle) []marketdata.Candle {
	normalized := make([]marketdata.Candle, len(candles))
	copy(normalized, candles)
	for index := range normalized {
		if normalized[index].Open == "" {
			normalized[index].Open = normalized[index].Close
		}
		if normalized[index].High == "" {
			normalized[index].High = normalized[index].Close
		}
		if normalized[index].Low == "" {
			normalized[index].Low = normalized[index].Close
		}
	}
	return normalized
}

func TestCompareStrategiesAddsResearchValidationMetricsWhenEnabled(t *testing.T) {
	request := ComparisonRequest{
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(5000, 0).UTC(),
		FastPeriod:         2,
		SlowPeriod:         3,
		RSIPeriod:          3,
		RSIOversold:        30,
		RSIOverbought:      70,
		StartingBalance:    1000,
		FeeRate:            0.001,
		TrainTestEnabled:   true,
		TrainRatio:         0.6,
		WalkForwardEnabled: true,
		WalkForwardFolds:   3,
	}

	comparison, err := NewComparator(NewEngine()).Compare(request, candlesForLargeStress(45, "1m", time.Minute))
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}

	if !comparison.TrainTestEnabled || comparison.TrainRatio != 0.6 {
		t.Fatalf("expected train/test settings on comparison, got %+v", comparison)
	}
	if !comparison.WalkForwardEnabled || comparison.WalkForwardFolds != 3 {
		t.Fatalf("expected walk-forward settings on comparison, got %+v", comparison)
	}
	for _, row := range comparison.Results {
		if row.TrainValidationStatus == "" || row.TestValidationStatus == "" {
			t.Fatalf("expected train/test validation on result, got %+v", row)
		}
		if row.WalkForwardFolds != 3 || row.WalkForwardValidationStatus == "" {
			t.Fatalf("expected walk-forward validation on result, got %+v", row)
		}
	}
}

func TestCompareStrategiesPropagatesNextOpenFillMode(t *testing.T) {
	request := ComparisonRequest{
		Symbol:            "BTCUSDT",
		Interval:          "1m",
		From:              time.Unix(1000, 0).UTC(),
		To:                time.Unix(2000, 0).UTC(),
		FastPeriod:        2,
		SlowPeriod:        3,
		RSIPeriod:         3,
		RSIOversold:       30,
		RSIOverbought:     70,
		StartingBalance:   1000,
		FeeRate:           0.001,
		ExecutionFillMode: ExecutionFillModeNextOpen,
	}

	comparison, err := NewComparator(NewEngine()).Compare(request, candlesForLargeStress(18, "1m", time.Minute))
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}

	if comparison.ExecutionFillMode != ExecutionFillModeNextOpen {
		t.Fatalf("expected comparison fill mode next_open, got %q", comparison.ExecutionFillMode)
	}
	for _, result := range comparison.Results {
		if result.ExecutionFillMode != ExecutionFillModeNextOpen {
			t.Fatalf("expected result fill mode next_open, got %+v", result)
		}
	}
}

func TestComparisonRequestRejectsTooFewCandlesForBothStrategies(t *testing.T) {
	request := DefaultComparisonRequest()
	request.From = time.Unix(1000, 0).UTC()
	request.To = time.Unix(2000, 0).UTC()
	request.SlowPeriod = 5
	request.RSIPeriod = 9

	_, err := NewComparator(NewEngine()).Compare(request, candlesForCrosses()[:6])
	if err == nil {
		t.Fatal("expected insufficient candles error")
	}
}

func TestComparisonRequestDefaultsToNextOpenFillMode(t *testing.T) {
	request := DefaultComparisonRequest()
	request.ExecutionFillMode = ""

	normalized := request.Normalize()

	if normalized.ExecutionFillMode != ExecutionFillModeNextOpen {
		t.Fatalf("expected comparison default fill mode next_open, got %q", normalized.ExecutionFillMode)
	}
}

func TestComparisonRequestRejectsInvalidResearchValidationSettings(t *testing.T) {
	request := DefaultComparisonRequest()
	request.From = time.Unix(1000, 0).UTC()
	request.To = time.Unix(2000, 0).UTC()
	request.TrainTestEnabled = true
	request.TrainRatio = 1

	if err := request.Normalize().Validate(); err == nil {
		t.Fatal("expected invalid train ratio validation error")
	}

	request = DefaultComparisonRequest()
	request.From = time.Unix(1000, 0).UTC()
	request.To = time.Unix(2000, 0).UTC()
	request.WalkForwardEnabled = true
	request.WalkForwardFolds = 1

	if err := request.Normalize().Validate(); err == nil {
		t.Fatal("expected invalid walk-forward folds validation error")
	}
}

func TestComparatorRanksCandidateProfitFactorBeforeRawReturn(t *testing.T) {
	runs := []Run{
		{
			StrategyName:     "high-return-weak",
			ReturnPercent:    20,
			ProfitFactor:     0.9,
			MaxDrawdown:      5,
			WinRate:          70,
			TotalTrades:      200,
			ValidationStatus: "not_profitable",
			ValidationReason: "profit factor must be greater than 1",
		},
		{
			StrategyName:     "lower-return-candidate",
			ReturnPercent:    5,
			ProfitFactor:     1.4,
			MaxDrawdown:      4,
			WinRate:          55,
			TotalTrades:      120,
			ValidationStatus: "candidate",
			ValidationReason: "strategy meets baseline validation rules",
		},
	}

	rankRuns(runs)

	if runs[0].StrategyName != "lower-return-candidate" {
		t.Fatalf("expected validated candidate first, got %+v", runs)
	}
}

func TestComparatorRanksLowBullMarketCaptureBelowOtherValidationFailures(t *testing.T) {
	results := []ComparisonResult{
		{
			StrategyName:             "defensive-underparticipant",
			ReturnPercent:            30,
			BenchmarkReturnPercent:   65,
			ExcessReturnPercent:      40,
			ProfitFactor:             1.8,
			ValidationStatus:         "low_bull_market_capture",
			WalkForwardAverageExcess: 100,
		},
		{
			StrategyName:             "weak-profit-factor",
			ReturnPercent:            -1,
			BenchmarkReturnPercent:   -10,
			ExcessReturnPercent:      -5,
			ProfitFactor:             0.9,
			ValidationStatus:         "weak_profit_factor",
			WalkForwardAverageExcess: 0,
		},
	}

	rankComparisonResults(results)

	if results[0].ValidationStatus == "low_bull_market_capture" {
		t.Fatalf("expected benchmark-capture failure to rank below other validation failures, got %+v", results)
	}
}
