package backtest

import (
	"strings"
	"testing"
	"time"
)

func TestBuildInsertComparisonSQLUsesComparisonFields(t *testing.T) {
	comparison := Comparison{
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(10, 0).UTC(),
		To:                 time.Unix(100, 0).UTC(),
		StartingBalance:    1000,
		FeeRate:            0.001,
		SlippageRate:       0.0005,
		ExecutionFillMode:  ExecutionFillModeNextOpen,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
		TrendFilterEnabled: true,
		TrendPeriod:        200,
		CooldownBars:       20,
		MinHoldingBars:     10,
		TrainTestEnabled:   true,
		TrainRatio:         0.7,
		TrainFrom:          time.Unix(10, 0).UTC(),
		TrainTo:            time.Unix(70, 0).UTC(),
		TestFrom:           time.Unix(71, 0).UTC(),
		TestTo:             time.Unix(100, 0).UTC(),
		WalkForwardEnabled: true,
		WalkForwardFolds:   4,
		WinnerStrategy:     "sma-crossover",
	}

	query, args := BuildInsertComparisonSQL(comparison)

	if !strings.Contains(query, "INSERT INTO strategy_comparisons") {
		t.Fatalf("expected strategy_comparisons insert, got %s", query)
	}
	if len(args) != 23 {
		t.Fatalf("expected 23 args, got %d", len(args))
	}
	if args[0] != "BTCUSDT" || args[5] != 0.001 || args[6] != 0.0005 || args[7] != ExecutionFillModeNextOpen || args[8] != PositionSizingPercentEquity || args[9] != 10.0 || args[10] != true || args[11] != 200 || args[12] != 20 || args[13] != 10 || args[14] != true || args[15] != 0.7 || args[21] != 4 || args[22] != "sma-crossover" {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestBuildInsertComparisonResultSQLUsesResultFields(t *testing.T) {
	result := ComparisonResult{
		ComparisonID:                "comparison-1",
		Rank:                        1,
		StrategyName:                "rsi-mean-reversion",
		ReturnPercent:               4.5,
		MaxDrawdown:                 1.2,
		WinRate:                     60,
		TotalTrades:                 4,
		ProfitFactor:                1.5,
		Expectancy:                  2.5,
		TradesPerDay:                4,
		ChurnRatio:                  8,
		SharpeRatio:                 1.2,
		SortinoRatio:                1.6,
		ExecutionFillMode:           ExecutionFillModeNextOpen,
		PositionSizingMode:          PositionSizingFixedQuote,
		PositionSizeValue:           250,
		TrendFilterEnabled:          true,
		TrendPeriod:                 200,
		CooldownBars:                20,
		MinHoldingBars:              10,
		BenchmarkEndingBalance:      900,
		BenchmarkProfitLoss:         -100,
		BenchmarkReturnPercent:      -10,
		ExcessReturnPercent:         14.5,
		ValidationStatus:            "candidate",
		ValidationReason:            "strategy meets baseline validation rules",
		TrainExcessReturn:           6,
		TrainValidationStatus:       "candidate",
		TestExcessReturn:            3,
		TestValidationStatus:        "candidate",
		WalkForwardFolds:            4,
		WalkForwardPasses:           3,
		WalkForwardAverageExcess:    2.5,
		WalkForwardValidationStatus: "unstable_walk_forward",
	}

	query, args := BuildInsertComparisonResultSQL(result)

	if !strings.Contains(query, "INSERT INTO strategy_comparison_results") {
		t.Fatalf("expected strategy_comparison_results insert, got %s", query)
	}
	if !strings.Contains(query, "expectancy") || !strings.Contains(query, "sortino_ratio") {
		t.Fatalf("expected strategy quality metrics in insert, got %s", query)
	}
	if len(args) != 64 {
		t.Fatalf("expected 64 args, got %d", len(args))
	}
	if args[0] != "comparison-1" || args[1] != 1 || args[2] != "rsi-mean-reversion" {
		t.Fatalf("unexpected args: %+v", args)
	}
	if args[24] != 2.5 || args[25] != 4.0 || args[26] != 8.0 || args[27] != 1.2 || args[28] != 1.6 {
		t.Fatalf("unexpected strategy quality metric args: %+v", args)
	}
	if args[29] != ExecutionFillModeNextOpen || args[30] != PositionSizingFixedQuote || args[31] != 250.0 || args[32] != true || args[33] != 200 || args[34] != 20 || args[35] != 10 || args[39] != 14.5 {
		t.Fatalf("unexpected sizing/benchmark args: %+v", args)
	}
	if args[43] != 6.0 || args[47] != "candidate" || args[50] != 3.0 || args[54] != "candidate" || args[56] != 4 || args[57] != 3 || args[59] != 2.5 || args[61] != "unstable_walk_forward" {
		t.Fatalf("unexpected research validation args: %+v", args)
	}
}
