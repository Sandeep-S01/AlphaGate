package backtest

import (
	"strconv"
	"testing"
	"time"

	"sentra/internal/marketdata"
	"sentra/internal/strategy"
)

func TestEngineRunsSMABacktestWithoutExternalSideEffects(t *testing.T) {
	request := Request{
		StrategyName:    "sma-crossover",
		Version:         "v1",
		Symbol:          "BTCUSDT",
		Interval:        "1m",
		From:            time.Unix(1000, 0).UTC(),
		To:              time.Unix(2000, 0).UTC(),
		FastPeriod:      2,
		SlowPeriod:      3,
		StartingBalance: 1000,
		FeeRate:         0.001,
	}

	run, trades, err := NewEngine().Run(request, candlesForCrosses())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if run.Symbol != "BTCUSDT" || run.Interval != "1m" {
		t.Fatalf("unexpected run market: %+v", run)
	}
	if run.EndingBalance <= 0 {
		t.Fatalf("expected ending balance, got %+v", run)
	}
	if run.TotalTrades != len(trades) {
		t.Fatalf("expected trade count %d, got %d", len(trades), run.TotalTrades)
	}
	if len(trades) == 0 {
		t.Fatal("expected at least one simulated trade")
	}
	if trades[0].RunID != "" {
		t.Fatalf("engine should not assign persistence IDs, got run id %q", trades[0].RunID)
	}
}

func TestEngineReportsTradeQualityMetrics(t *testing.T) {
	request := Request{
		StrategyName:    "sma-crossover",
		Version:         "v1",
		Symbol:          "BTCUSDT",
		Interval:        "1m",
		From:            time.Unix(1000, 0).UTC(),
		To:              time.Unix(2000, 0).UTC(),
		FastPeriod:      2,
		SlowPeriod:      3,
		StartingBalance: 1000,
		FeeRate:         0.001,
	}

	run, trades, err := NewEngine().Run(request, candlesForCrosses())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if run.BuyCount == 0 {
		t.Fatalf("expected buy count, got %+v with trades %+v", run, trades)
	}
	if run.SellCount == 0 {
		t.Fatalf("expected sell count, got %+v with trades %+v", run, trades)
	}
	if run.BestTrade == 0 && run.WorstTrade == 0 {
		t.Fatalf("expected trade quality metrics, got %+v", run)
	}
	if run.OpenPosition && trades[len(trades)-1].Side == SideSell {
		t.Fatalf("did not expect open position after final sell: %+v", run)
	}
}

func TestEngineBuildsDiagnosticsForCompletedRoundTrips(t *testing.T) {
	request := Request{
		StrategyName:    "sma-crossover",
		Version:         "v1",
		Symbol:          "BTCUSDT",
		Interval:        "1m",
		From:            time.Unix(1000, 0).UTC(),
		To:              time.Unix(2000, 0).UTC(),
		FastPeriod:      2,
		SlowPeriod:      3,
		StartingBalance: 1000,
		FeeRate:         0.001,
		SlippageRate:    0.0005,
	}

	run, _, err := NewEngine().Run(request, candlesForCrosses())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(run.RoundTrips) == 0 {
		t.Fatalf("expected completed round trips, got %+v", run)
	}
	first := run.RoundTrips[0]
	if !first.ExitTime.After(first.EntryTime) {
		t.Fatalf("expected exit after entry, got %+v", first)
	}
	if first.EntryPrice <= 0 || first.ExitPrice <= 0 || first.Fees <= 0 {
		t.Fatalf("expected realistic fill prices and fees, got %+v", first)
	}
	if first.HoldingSeconds <= 0 {
		t.Fatalf("expected holding duration, got %+v", first)
	}
	if len(run.EquityCurve) == 0 {
		t.Fatal("expected equity curve points")
	}
	if run.ProfitFactor < 0 {
		t.Fatalf("expected non-negative profit factor, got %f", run.ProfitFactor)
	}
	if run.AverageHoldingSeconds <= 0 {
		t.Fatalf("expected average holding seconds, got %f", run.AverageHoldingSeconds)
	}
	if run.Expectancy == 0 {
		t.Fatalf("expected expectancy metric, got %+v", run)
	}
	if run.TradesPerDay <= 0 {
		t.Fatalf("expected trades per day metric, got %+v", run)
	}
	if run.ChurnRatio <= 0 {
		t.Fatalf("expected churn ratio metric, got %+v", run)
	}
	if run.SharpeRatio == 0 {
		t.Fatalf("expected Sharpe ratio metric, got %+v", run)
	}
	if run.SortinoRatio == 0 {
		t.Fatalf("expected Sortino ratio metric, got %+v", run)
	}
	if run.ValidationStatus == "" || run.ValidationReason == "" {
		t.Fatalf("expected validation result, got status=%q reason=%q", run.ValidationStatus, run.ValidationReason)
	}
}

func TestEngineReportsCostDiagnostics(t *testing.T) {
	request := Request{
		StrategyName:       "sma-crossover",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriod:         2,
		SlowPeriod:         3,
		StartingBalance:    1000,
		FeeRate:            0.001,
		SlippageRate:       0.0005,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
		ExecutionFillMode:  ExecutionFillModeNextOpen,
	}

	run, trades, err := NewEngine().Run(request, candlesForCrosses())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(trades) == 0 {
		t.Fatal("expected trades")
	}
	if run.TotalFees <= 0 {
		t.Fatalf("expected total fees to be reported, got %+v", run)
	}
	if run.EstimatedSlippageCost <= 0 {
		t.Fatalf("expected slippage cost to be reported, got %+v", run)
	}
	if run.GrossProfitLoss-run.TotalFees-run.EstimatedSlippageCost > run.ProfitLoss+0.000001 {
		t.Fatalf("expected net PnL to include costs, got gross=%f fees=%f slippage=%f net=%f", run.GrossProfitLoss, run.TotalFees, run.EstimatedSlippageCost, run.ProfitLoss)
	}
	if run.RoundTripCostPercent <= 0 {
		t.Fatalf("expected round-trip cost percent, got %+v", run)
	}
	if run.BreakEvenMovePercent <= 0 {
		t.Fatalf("expected break-even move percent, got %+v", run)
	}
}

func TestEngineAppliesSlippageToFillPrices(t *testing.T) {
	baseRequest := Request{
		StrategyName:    "sma-crossover",
		Version:         "v1",
		Symbol:          "BTCUSDT",
		Interval:        "1m",
		From:            time.Unix(1000, 0).UTC(),
		To:              time.Unix(2000, 0).UTC(),
		FastPeriod:      2,
		SlowPeriod:      3,
		StartingBalance: 1000,
		FeeRate:         0.001,
	}
	noSlippage, _, err := NewEngine().Run(baseRequest, candlesForCrosses())
	if err != nil {
		t.Fatalf("Run without slippage returned error: %v", err)
	}
	slippedRequest := baseRequest
	slippedRequest.SlippageRate = 0.01
	withSlippage, _, err := NewEngine().Run(slippedRequest, candlesForCrosses())
	if err != nil {
		t.Fatalf("Run with slippage returned error: %v", err)
	}
	if len(noSlippage.RoundTrips) == 0 || len(withSlippage.RoundTrips) == 0 {
		t.Fatal("expected round trips in both runs")
	}
	if withSlippage.RoundTrips[0].EntryPrice <= noSlippage.RoundTrips[0].EntryPrice {
		t.Fatalf("expected buy entry price to increase with slippage: no=%f slipped=%f", noSlippage.RoundTrips[0].EntryPrice, withSlippage.RoundTrips[0].EntryPrice)
	}
	if withSlippage.RoundTrips[0].ExitPrice >= noSlippage.RoundTrips[0].ExitPrice {
		t.Fatalf("expected sell exit price to decrease with slippage: no=%f slipped=%f", noSlippage.RoundTrips[0].ExitPrice, withSlippage.RoundTrips[0].ExitPrice)
	}
}

func TestEngineForceClosesOpenPositionAtEndWithCosts(t *testing.T) {
	request := Request{
		StrategyName:       "sma-crossover",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriod:         2,
		SlowPeriod:         3,
		StartingBalance:    1000,
		FeeRate:            0.001,
		SlippageRate:       0.01,
		PositionSizingMode: PositionSizingFixedQuote,
		PositionSizeValue:  100,
	}

	run, trades, err := NewEngine().Run(request, candlesForEndOfBacktestClose())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if run.OpenPosition {
		t.Fatalf("expected final position to be force-closed, got %+v", run)
	}
	if len(trades) != 2 || trades[1].Side != SideSell {
		t.Fatalf("expected buy plus forced sell, got %+v", trades)
	}
	if len(run.RoundTrips) != 1 || run.RoundTrips[0].ExitReason != "end_of_backtest" {
		t.Fatalf("expected end_of_backtest round trip, got %+v", run.RoundTrips)
	}
	if trades[1].Price >= 18 {
		t.Fatalf("expected final sell to include sell slippage below last close, got %f", trades[1].Price)
	}
	if trades[1].Fee <= 0 {
		t.Fatalf("expected forced close sell fee, got %+v", trades[1])
	}
	markToMarketWithoutExitCosts := 900 + trades[0].Quantity*18
	if run.EndingBalance >= markToMarketWithoutExitCosts {
		t.Fatalf("expected ending balance to include final exit costs, got ending=%f mark=%f", run.EndingBalance, markToMarketWithoutExitCosts)
	}
	if run.SellCount != 1 || run.TotalTrades != 2 {
		t.Fatalf("expected forced close counts in run, got %+v", run)
	}
}

func TestEngineNextOpenFillUsesNextCandleOpenAfterSignal(t *testing.T) {
	request := Request{
		StrategyName:       "sma-crossover",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriod:         2,
		SlowPeriod:         3,
		StartingBalance:    1000,
		FeeRate:            0,
		PositionSizingMode: PositionSizingFixedQuote,
		PositionSizeValue:  100,
		ExecutionFillMode:  ExecutionFillModeNextOpen,
	}

	run, trades, err := NewEngine().Run(request, candlesForNextOpenFill())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if run.ExecutionFillMode != ExecutionFillModeNextOpen {
		t.Fatalf("expected fill mode copied to run, got %+v", run)
	}
	if len(trades) == 0 || trades[0].Side != SideBuy {
		t.Fatalf("expected first trade to be a buy, got %+v", trades)
	}
	if trades[0].Price != 50 {
		t.Fatalf("expected buy fill at next candle open 50, got %f", trades[0].Price)
	}
	if !trades[0].ExecutedAt.Equal(candlesForNextOpenFill()[4].OpenTime) {
		t.Fatalf("expected buy execution at next candle open time, got %s", trades[0].ExecutedAt)
	}
}

func TestEngineUsesFixedQuotePositionSizing(t *testing.T) {
	request := Request{
		StrategyName:       "sma-crossover",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriod:         2,
		SlowPeriod:         3,
		StartingBalance:    1000,
		FeeRate:            0.001,
		PositionSizingMode: PositionSizingFixedQuote,
		PositionSizeValue:  100,
	}

	run, trades, err := NewEngine().Run(request, candlesForCrosses())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(trades) == 0 || trades[0].Side != SideBuy {
		t.Fatalf("expected first trade to be a buy, got %+v", trades)
	}
	if trades[0].QuoteAmount != 100 {
		t.Fatalf("expected fixed quote buy size 100, got %f", trades[0].QuoteAmount)
	}
	if run.PositionSizingMode != PositionSizingFixedQuote || run.PositionSizeValue != 100 {
		t.Fatalf("expected position sizing to be copied to run, got %+v", run)
	}
	if run.EndingBalance < 800 {
		t.Fatalf("expected unused quote balance to remain in equity, got %f", run.EndingBalance)
	}
}

func TestEngineUsesPercentEquityPositionSizing(t *testing.T) {
	request := Request{
		StrategyName:       "sma-crossover",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriod:         2,
		SlowPeriod:         3,
		StartingBalance:    1000,
		FeeRate:            0.001,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
		ExecutionFillMode:  ExecutionFillModeSameClose,
	}

	run, trades, err := NewEngine().Run(request, candlesForCrosses())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(trades) == 0 || trades[0].Side != SideBuy {
		t.Fatalf("expected first trade to be a buy, got %+v", trades)
	}
	if trades[0].QuoteAmount < 99.99 || trades[0].QuoteAmount > 100.01 {
		t.Fatalf("expected first buy to use 10%% of equity, got %f", trades[0].QuoteAmount)
	}
	if run.PositionSizingMode != PositionSizingPercentEquity || run.PositionSizeValue != 10 {
		t.Fatalf("expected position sizing to be copied to run, got %+v", run)
	}
}

func TestEngineTrendFilterBlocksSMABuysBelowTrend(t *testing.T) {
	baseRequest := Request{
		StrategyName:       "sma-crossover",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriod:         2,
		SlowPeriod:         3,
		StartingBalance:    1000,
		FeeRate:            0.001,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
		ExecutionFillMode:  ExecutionFillModeSameClose,
	}
	unfiltered, _, err := NewEngine().Run(baseRequest, candlesForCrosses())
	if err != nil {
		t.Fatalf("unfiltered run returned error: %v", err)
	}
	filteredRequest := baseRequest
	filteredRequest.TrendFilterEnabled = true
	filteredRequest.TrendPeriod = 5
	filtered, _, err := NewEngine().Run(filteredRequest, candlesForTrendFilter())
	if err != nil {
		t.Fatalf("filtered run returned error: %v", err)
	}
	if unfiltered.BuyCount == 0 {
		t.Fatal("expected baseline to buy")
	}
	if filtered.BuyCount != 0 {
		t.Fatalf("expected trend filter to block buys below trend, got %d", filtered.BuyCount)
	}
	if !filtered.TrendFilterEnabled || filtered.TrendPeriod != 5 {
		t.Fatalf("expected trend filter settings copied to run, got %+v", filtered)
	}
}

func TestEngineRegimeFilterBlocksEntriesOutsideATRPercentRange(t *testing.T) {
	baseRequest := Request{
		StrategyName:       "sma-crossover",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriod:         2,
		SlowPeriod:         3,
		StartingBalance:    1000,
		FeeRate:            0.001,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
		ExecutionFillMode:  ExecutionFillModeSameClose,
	}
	unfiltered, _, err := NewEngine().Run(baseRequest, candlesForCrosses())
	if err != nil {
		t.Fatalf("unfiltered run returned error: %v", err)
	}
	filteredRequest := baseRequest
	filteredRequest.RegimeFilterEnabled = true
	filteredRequest.RegimeFilterPeriod = 3
	filteredRequest.RegimeMinATRPercent = 20
	filtered, _, err := NewEngine().Run(filteredRequest, candlesForCrosses())
	if err != nil {
		t.Fatalf("filtered run returned error: %v", err)
	}
	if unfiltered.BuyCount == 0 {
		t.Fatal("expected baseline to buy")
	}
	if filtered.TotalTrades != 0 {
		t.Fatalf("expected regime filter to block all entries, got trades=%d run=%+v", filtered.TotalTrades, filtered)
	}
	if !filtered.RegimeFilterEnabled || filtered.RegimeFilterPeriod != 3 || filtered.RegimeMinATRPercent != 20 {
		t.Fatalf("expected regime filter settings copied to run, got %+v", filtered)
	}
}

func TestEngineCooldownBarsBlocksRapidReentry(t *testing.T) {
	baseRequest := Request{
		StrategyName:       "sma-crossover",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriod:         2,
		SlowPeriod:         3,
		StartingBalance:    1000,
		FeeRate:            0.001,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
		ExecutionFillMode:  ExecutionFillModeSameClose,
	}
	unfiltered, _, err := NewEngine().Run(baseRequest, candlesForCrosses())
	if err != nil {
		t.Fatalf("unfiltered run returned error: %v", err)
	}
	cooldownRequest := baseRequest
	cooldownRequest.CooldownBars = 10
	filtered, _, err := NewEngine().Run(cooldownRequest, candlesForCrosses())
	if err != nil {
		t.Fatalf("cooldown run returned error: %v", err)
	}
	if unfiltered.BuyCount < 2 {
		t.Fatalf("expected baseline to have repeated entries, got %d", unfiltered.BuyCount)
	}
	if filtered.BuyCount >= unfiltered.BuyCount {
		t.Fatalf("expected cooldown to reduce buy count, baseline=%d filtered=%d", unfiltered.BuyCount, filtered.BuyCount)
	}
	if filtered.CooldownBars != 10 {
		t.Fatalf("expected cooldown setting copied to run, got %+v", filtered)
	}
}

func TestEngineMinimumHoldingBarsDelaysExit(t *testing.T) {
	baseRequest := Request{
		StrategyName:       "sma-crossover",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriod:         2,
		SlowPeriod:         3,
		StartingBalance:    1000,
		FeeRate:            0.001,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
		ExecutionFillMode:  ExecutionFillModeSameClose,
	}
	unfiltered, _, err := NewEngine().Run(baseRequest, candlesForCrosses())
	if err != nil {
		t.Fatalf("unfiltered run returned error: %v", err)
	}
	holdingRequest := baseRequest
	holdingRequest.MinHoldingBars = 5
	filtered, _, err := NewEngine().Run(holdingRequest, candlesForCrosses())
	if err != nil {
		t.Fatalf("holding run returned error: %v", err)
	}
	if unfiltered.SellCount == 0 {
		t.Fatal("expected baseline to sell")
	}
	if filtered.SellCount >= unfiltered.SellCount {
		t.Fatalf("expected minimum holding period to reduce sells, baseline=%d filtered=%d", unfiltered.SellCount, filtered.SellCount)
	}
	if filtered.MinHoldingBars != 5 {
		t.Fatalf("expected minimum holding setting copied to run, got %+v", filtered)
	}
}

func TestEngineATRStopLossExitsOpenPositionBeforeStrategySell(t *testing.T) {
	request := Request{
		StrategyName:            "sma-crossover",
		Version:                 "v1",
		Symbol:                  "BTCUSDT",
		Interval:                "1m",
		From:                    time.Unix(1000, 0).UTC(),
		To:                      time.Unix(2000, 0).UTC(),
		FastPeriod:              2,
		SlowPeriod:              3,
		StartingBalance:         1000,
		FeeRate:                 0,
		PositionSizingMode:      PositionSizingPercentEquity,
		PositionSizeValue:       10,
		ATRExitEnabled:          true,
		ATRPeriod:               3,
		ATRStopMultiplier:       1,
		ATRTakeProfitMultiplier: 0,
		ExecutionFillMode:       ExecutionFillModeSameClose,
	}

	run, trades, err := NewEngine().Run(request, candlesForATRStop())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if run.BuyCount != 1 || run.SellCount != 1 {
		t.Fatalf("expected one buy and one ATR stop sell, got buys=%d sells=%d trades=%+v", run.BuyCount, run.SellCount, trades)
	}
	if len(run.RoundTrips) != 1 {
		t.Fatalf("expected one round trip, got %+v", run.RoundTrips)
	}
	if run.RoundTrips[0].ExitReason != "ATR stop-loss" {
		t.Fatalf("expected ATR stop-loss exit reason, got %+v", run.RoundTrips[0])
	}
	if !run.ATRExitEnabled || run.ATRPeriod != 3 || run.ATRStopMultiplier != 1 {
		t.Fatalf("expected ATR settings copied to run, got %+v", run)
	}
}

func TestEngineCalculatesBuyAndHoldBenchmark(t *testing.T) {
	request := Request{
		StrategyName:    "sma-crossover",
		Version:         "v1",
		Symbol:          "BTCUSDT",
		Interval:        "1m",
		From:            time.Unix(1000, 0).UTC(),
		To:              time.Unix(2000, 0).UTC(),
		FastPeriod:      2,
		SlowPeriod:      3,
		StartingBalance: 1000,
		FeeRate:         0.001,
	}

	run, _, err := NewEngine().Run(request, candlesForCrosses())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if run.BenchmarkEndingBalance <= 0 {
		t.Fatalf("expected benchmark ending balance, got %+v", run)
	}
	if run.BenchmarkReturnPercent <= 0 {
		t.Fatalf("expected buy-and-hold benchmark to gain on rising test data, got %+v", run)
	}
	expectedExcess := run.ReturnPercent - run.BenchmarkReturnPercent
	if mathAbs(run.ExcessReturnPercent-expectedExcess) > 0.000001 {
		t.Fatalf("expected excess return %f, got %f", expectedExcess, run.ExcessReturnPercent)
	}
}

func TestBenchmarkBuyAndHoldAppliesExitCosts(t *testing.T) {
	request := Request{
		StrategyName:    "sma-crossover",
		Version:         "v1",
		Symbol:          "BTCUSDT",
		Interval:        "1m",
		StartingBalance: 1000,
		FeeRate:         0.001,
		SlippageRate:    0.0005,
	}
	candles := []marketdata.Candle{
		{Close: "100", OpenTime: time.Unix(1000, 0).UTC(), CloseTime: time.Unix(1060, 0).UTC()},
		{Close: "110", OpenTime: time.Unix(1060, 0).UTC(), CloseTime: time.Unix(1120, 0).UTC()},
	}

	ending, _, _, err := benchmarkBuyAndHold(request, candles)
	if err != nil {
		t.Fatalf("benchmark returned error: %v", err)
	}

	entryFill := 100.0 * (1 + request.SlippageRate)
	entryFee := request.StartingBalance * request.FeeRate
	quantity := (request.StartingBalance - entryFee) / entryFill
	exitFill := 110.0 * (1 - request.SlippageRate)
	exitGross := quantity * exitFill
	expected := exitGross - exitGross*request.FeeRate

	if mathAbs(ending-expected) > 0.000001 {
		t.Fatalf("expected benchmark ending %f with exit costs, got %f", expected, ending)
	}
}

func TestValidateRunCandidateRejectsNegativeExcessReturn(t *testing.T) {
	status, reason := validateRunCandidate(runValidationInput{
		Interval:            "15m",
		From:                time.Unix(0, 0).UTC(),
		To:                  time.Unix(86400*30, 0).UTC(),
		CompletedTrades:     120,
		TotalTrades:         240,
		ProfitFactor:        1.5,
		MaxDrawdown:         12,
		AverageTrade:        2,
		ExcessReturnPercent: -1,
	})

	if status != "underperforms_benchmark" || reason != "excess return must be positive" {
		t.Fatalf("expected benchmark rejection, got status=%q reason=%q", status, reason)
	}
}

func TestValidateRunCandidateRejectsLowBullMarketCapture(t *testing.T) {
	status, reason := validateRunCandidate(runValidationInput{
		Interval:               "15m",
		From:                   time.Unix(0, 0).UTC(),
		To:                     time.Unix(86400*30, 0).UTC(),
		CompletedTrades:        120,
		TotalTrades:            60,
		ProfitFactor:           1.5,
		MaxDrawdown:            12,
		AverageTrade:           2,
		ReturnPercent:          2,
		BenchmarkReturnPercent: 40,
		ExcessReturnPercent:    -38,
		ExecutionFillMode:      ExecutionFillModeNextOpen,
	})

	if status != "low_bull_market_capture" || reason != "strategy captures too little of a strong positive benchmark move" {
		t.Fatalf("expected bull capture rejection, got status=%q reason=%q", status, reason)
	}
}

func TestValidateRunCandidatePrioritizesLowBullMarketCaptureOverSampleSize(t *testing.T) {
	status, reason := validateRunCandidate(runValidationInput{
		Interval:               "15m",
		From:                   time.Unix(0, 0).UTC(),
		To:                     time.Unix(86400*30, 0).UTC(),
		CompletedTrades:        20,
		TotalTrades:            40,
		ProfitFactor:           1.5,
		MaxDrawdown:            12,
		AverageTrade:           2,
		ReturnPercent:          1,
		BenchmarkReturnPercent: 60,
		ExcessReturnPercent:    -59,
		ExecutionFillMode:      ExecutionFillModeNextOpen,
	})

	if status != "low_bull_market_capture" || reason != "strategy captures too little of a strong positive benchmark move" {
		t.Fatalf("expected bull capture rejection before sample-size rejection, got status=%q reason=%q", status, reason)
	}
}

func TestValidateRunCandidateAllowsDefensiveExcessInDownMarket(t *testing.T) {
	status, reason := validateRunCandidate(runValidationInput{
		Interval:               "15m",
		From:                   time.Unix(0, 0).UTC(),
		To:                     time.Unix(86400*30, 0).UTC(),
		CompletedTrades:        120,
		TotalTrades:            60,
		ProfitFactor:           1.5,
		MaxDrawdown:            12,
		AverageTrade:           2,
		ReturnPercent:          2,
		BenchmarkReturnPercent: -40,
		ExcessReturnPercent:    42,
		ExecutionFillMode:      ExecutionFillModeNextOpen,
	})

	if status != "candidate" || reason != "strategy meets hardened validation rules" {
		t.Fatalf("expected defensive candidate, got status=%q reason=%q", status, reason)
	}
}

func TestValidateRunCandidateRejectsWeakProfitFactor(t *testing.T) {
	status, reason := validateRunCandidate(runValidationInput{
		Interval:            "15m",
		From:                time.Unix(0, 0).UTC(),
		To:                  time.Unix(86400*30, 0).UTC(),
		CompletedTrades:     120,
		TotalTrades:         240,
		ProfitFactor:        1.2,
		MaxDrawdown:         12,
		AverageTrade:        2,
		ExcessReturnPercent: 5,
	})

	if status != "weak_profit_factor" || reason != "profit factor must be greater than 1.2" {
		t.Fatalf("expected profit factor rejection, got status=%q reason=%q", status, reason)
	}
}

func TestValidateRunCandidateRejectsNegativeAverageTrade(t *testing.T) {
	status, reason := validateRunCandidate(runValidationInput{
		Interval:            "15m",
		From:                time.Unix(0, 0).UTC(),
		To:                  time.Unix(86400*30, 0).UTC(),
		CompletedTrades:     120,
		TotalTrades:         240,
		ProfitFactor:        1.5,
		MaxDrawdown:         12,
		AverageTrade:        -0.01,
		ExcessReturnPercent: 5,
	})

	if status != "negative_average_trade" || reason != "average completed trade must be positive after fees" {
		t.Fatalf("expected average trade rejection, got status=%q reason=%q", status, reason)
	}
}

func TestValidateRunCandidateRejectsAverageTradeBelowCost(t *testing.T) {
	status, reason := validateRunCandidate(runValidationInput{
		Interval:             "1m",
		CompletedTrades:      200,
		TotalTrades:          400,
		ProfitFactor:         1.3,
		MaxDrawdown:          5,
		AverageTrade:         0.10,
		BreakEvenMovePercent: 0.30,
		ExcessReturnPercent:  5,
		ExecutionFillMode:    ExecutionFillModeNextOpen,
	})

	if status != "cost_drag" || reason != "average trade does not exceed estimated round-trip cost" {
		t.Fatalf("expected cost drag rejection, got status=%q reason=%q", status, reason)
	}
}

func TestValidateRunCandidateRejectsExplicitExcessiveChurn(t *testing.T) {
	status, reason := validateRunCandidate(runValidationInput{
		Interval:             "1m",
		CompletedTrades:      200,
		TotalTrades:          2000,
		ProfitFactor:         1.3,
		MaxDrawdown:          5,
		AverageTrade:         2,
		BreakEvenMovePercent: 0.30,
		ExcessReturnPercent:  5,
		ExecutionFillMode:    ExecutionFillModeNextOpen,
		TradesPerDay:         80,
	})

	if status != "overtrading" || reason != "trade frequency is too high for the selected interval" {
		t.Fatalf("expected overtrading rejection, got status=%q reason=%q", status, reason)
	}
}

func TestValidateRunCandidateRejectsExcessiveTradeFrequency(t *testing.T) {
	status, reason := validateRunCandidate(runValidationInput{
		Interval:            "15m",
		From:                time.Unix(0, 0).UTC(),
		To:                  time.Unix(86400*30, 0).UTC(),
		CompletedTrades:     120,
		TotalTrades:         180,
		ProfitFactor:        1.5,
		MaxDrawdown:         12,
		AverageTrade:        2,
		ExcessReturnPercent: 5,
	})

	if status != "overtrading" || reason != "trade frequency is too high for the selected interval" {
		t.Fatalf("expected overtrading rejection, got status=%q reason=%q", status, reason)
	}
}

func TestValidateRunCandidateRejectsSameCloseExecutionEvidence(t *testing.T) {
	status, reason := validateRunCandidate(runValidationInput{
		Interval:            "15m",
		From:                time.Unix(0, 0).UTC(),
		To:                  time.Unix(86400*30, 0).UTC(),
		CompletedTrades:     120,
		TotalTrades:         90,
		ProfitFactor:        1.5,
		MaxDrawdown:         12,
		AverageTrade:        2,
		ExcessReturnPercent: 5,
		ExecutionFillMode:   ExecutionFillModeSameClose,
	})

	if status != "unsafe_execution_timing" || reason != "candidate backtests must use next_open execution" {
		t.Fatalf("expected execution timing rejection, got status=%q reason=%q", status, reason)
	}
}

func TestValidateRunCandidateAcceptsStrongCandidate(t *testing.T) {
	status, reason := validateRunCandidate(runValidationInput{
		Interval:            "15m",
		From:                time.Unix(0, 0).UTC(),
		To:                  time.Unix(86400*30, 0).UTC(),
		CompletedTrades:     120,
		TotalTrades:         90,
		ProfitFactor:        1.5,
		MaxDrawdown:         12,
		AverageTrade:        2,
		ExcessReturnPercent: 5,
		ExecutionFillMode:   ExecutionFillModeNextOpen,
	})

	if status != "candidate" || reason != "strategy meets hardened validation rules" {
		t.Fatalf("expected candidate, got status=%q reason=%q", status, reason)
	}
}

func TestValidationRankTreatsLowBullMarketCaptureAsSevereBenchmarkFailure(t *testing.T) {
	if !validationRankLess("underperforms_benchmark", "low_bull_market_capture") {
		t.Fatal("expected ordinary benchmark underperformance to rank ahead of low bull-market capture")
	}
	if !validationRankLess("low_bull_market_capture", "cost_drag") {
		t.Fatal("expected low bull-market capture to rank ahead of cost-drag execution failures")
	}
}

func TestEngineRunsRSIBacktest(t *testing.T) {
	request := Request{
		StrategyName:    "rsi-mean-reversion",
		Version:         "v1",
		Symbol:          "BTCUSDT",
		Interval:        "1m",
		From:            time.Unix(1000, 0).UTC(),
		To:              time.Unix(2000, 0).UTC(),
		RSIPeriod:       3,
		RSIOversold:     30,
		RSIOverbought:   70,
		StartingBalance: 1000,
		FeeRate:         0.001,
	}

	run, trades, err := NewEngine().Run(request, candlesForRSI())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if run.StrategyName != "rsi-mean-reversion" {
		t.Fatalf("expected RSI run, got %+v", run)
	}
	if len(trades) == 0 {
		t.Fatal("expected RSI backtest trades")
	}
}

func TestEngineRunsBTCTrendPullbackBacktest(t *testing.T) {
	request := Request{
		StrategyName:       "btc-trend-pullback",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "15m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(2000, 0).UTC(),
		FastPeriod:         3,
		SlowPeriod:         5,
		RSIPeriod:          3,
		StartingBalance:    1000,
		FeeRate:            0.001,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
	}

	run, trades, err := NewEngine().Run(request, candlesForTrendPullbackBacktest())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if run.StrategyName != "btc-trend-pullback" {
		t.Fatalf("expected BTC trend pullback run, got %+v", run)
	}
	if len(trades) == 0 {
		t.Fatal("expected trend pullback backtest trades")
	}
}

func TestRequestValidateAllowsExecutableStrategyTemplate(t *testing.T) {
	request := DefaultRequest()
	request.StrategyName = strategy.StrategyMultiFactorMomentum
	request.From = time.Unix(1000, 0).UTC()
	request.To = time.Unix(2000, 0).UTC()
	request.FastPeriod = 50
	request.SlowPeriod = 200

	if err := request.Normalize().Validate(); err != nil {
		t.Fatalf("expected executable template request to validate, got %v", err)
	}
}

func TestRequestValidateRejectsDataBlockedStrategyTemplate(t *testing.T) {
	request := DefaultRequest()
	request.StrategyName = strategy.StrategyFundingRateArbitrage
	request.From = time.Unix(1000, 0).UTC()
	request.To = time.Unix(2000, 0).UTC()

	if err := request.Normalize().Validate(); err == nil {
		t.Fatalf("expected data-blocked template request to be rejected")
	}
}

func TestEngineStressRunsTwoYearsOfFifteenMinuteCandles(t *testing.T) {
	const candleCount = 70080
	request := Request{
		StrategyName:       "sma-crossover",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "15m",
		From:               time.Unix(1000, 0).UTC(),
		To:                 time.Unix(1000, 0).UTC().Add(candleCount * 15 * time.Minute),
		FastPeriod:         9,
		SlowPeriod:         21,
		StartingBalance:    1000,
		FeeRate:            0.001,
		SlippageRate:       0.0005,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
		ExecutionFillMode:  ExecutionFillModeNextOpen,
	}

	run, trades, err := NewEngine().Run(request, candlesForLargeStress(candleCount, "15m", 15*time.Minute))
	if err != nil {
		t.Fatalf("Run returned error for large stress set: %v", err)
	}

	if run.Interval != "15m" || run.ExecutionFillMode != ExecutionFillModeNextOpen {
		t.Fatalf("expected stress run metadata to be preserved, got %+v", run)
	}
	if len(run.EquityCurve) == 0 {
		t.Fatal("expected equity curve to be calculated during stress run")
	}
	if run.TotalTrades != len(trades) {
		t.Fatalf("expected trade count %d, got %d", len(trades), run.TotalTrades)
	}
}

func TestEngineRunsExecutablePineTemplateWithoutQuadraticRecalculation(t *testing.T) {
	const candleCount = 1200
	request := Request{
		StrategyName:            strategy.StrategyMultiFactorMomentum,
		Version:                 "v1",
		Symbol:                  "BTCUSDT",
		Interval:                "1m",
		From:                    time.Unix(1000, 0).UTC(),
		To:                      time.Unix(1000, 0).UTC().Add(candleCount * time.Minute),
		FastPeriod:              50,
		SlowPeriod:              200,
		RSIPeriod:               14,
		RSIOversold:             50,
		RSIOverbought:           70,
		StartingBalance:         10000,
		FeeRate:                 0.001,
		SlippageRate:            0.0005,
		PositionSizingMode:      PositionSizingPercentEquity,
		PositionSizeValue:       10,
		ATRExitEnabled:          true,
		ATRPeriod:               14,
		ATRStopMultiplier:       2,
		ATRTakeProfitMultiplier: 3,
		ExecutionFillMode:       ExecutionFillModeNextOpen,
	}

	startedAt := time.Now()
	run, trades, err := NewEngine().Run(request, candlesForLargeStress(candleCount, "1m", time.Minute))
	elapsed := time.Since(startedAt)
	if err != nil {
		t.Fatalf("Run returned error for executable Pine template stress set: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("expected executable Pine template run to avoid full recalculation per candle, took %s", elapsed)
	}
	if run.StrategyName != strategy.StrategyMultiFactorMomentum {
		t.Fatalf("expected strategy metadata to be preserved, got %+v", run)
	}
	if run.TotalTrades != len(trades) {
		t.Fatalf("expected trade count %d, got %d", len(trades), run.TotalTrades)
	}
}

func TestEngineReturnsCandleSeriesDiagnosticsError(t *testing.T) {
	request := Request{
		StrategyName:    "sma-crossover",
		Version:         "v1",
		Symbol:          "BTCUSDT",
		Interval:        "1m",
		From:            time.Unix(1000, 0).UTC(),
		To:              time.Unix(2000, 0).UTC(),
		FastPeriod:      2,
		SlowPeriod:      3,
		StartingBalance: 1000,
		FeeRate:         0.001,
	}
	candles := candlesForCrosses()
	candles[3].OpenTime = candles[2].OpenTime

	_, _, err := NewEngine().Run(request, candles)
	if err == nil {
		t.Fatal("expected invalid candle series error")
	}
	seriesErr, ok := err.(CandleSeriesError)
	if !ok {
		t.Fatalf("expected CandleSeriesError, got %T %v", err, err)
	}
	if seriesErr.Diagnostics.DuplicateCount == 0 {
		t.Fatalf("expected duplicate diagnostics, got %+v", seriesErr.Diagnostics)
	}
}

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func TestRequestValidateRejectsInvalidDateRange(t *testing.T) {
	request := DefaultRequest()
	request.From = time.Unix(100, 0).UTC()
	request.To = time.Unix(10, 0).UTC()

	if err := request.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRequestValidateRejectsATRExitWithoutMultiplier(t *testing.T) {
	request := DefaultRequest()
	request.From = time.Unix(10, 0).UTC()
	request.To = time.Unix(100, 0).UTC()
	request.ATRExitEnabled = true
	request.ATRPeriod = 14

	if err := request.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRequestValidateRejectsInvalidRegimeFilterRange(t *testing.T) {
	request := DefaultRequest()
	request.From = time.Unix(10, 0).UTC()
	request.To = time.Unix(100, 0).UTC()
	request.RegimeFilterEnabled = true
	request.RegimeFilterPeriod = 14
	request.RegimeMinATRPercent = 5
	request.RegimeMaxATRPercent = 2

	if err := request.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRequestRequiredCandlesIncludesRegimeFilterPeriod(t *testing.T) {
	request := DefaultRequest()
	request.RegimeFilterEnabled = true
	request.RegimeFilterPeriod = 50

	if required := request.RequiredCandles(); required != 51 {
		t.Fatalf("expected regime filter history to require 51 candles, got %d", required)
	}
}

func TestRequestNormalizeDefaultsRSIFieldsForPersistableSMARuns(t *testing.T) {
	request := Request{
		StrategyName:    "sma-crossover",
		Version:         "v1",
		Symbol:          "BTCUSDT",
		Interval:        "15m",
		From:            time.Unix(10, 0).UTC(),
		To:              time.Unix(100, 0).UTC(),
		FastPeriod:      5,
		SlowPeriod:      20,
		StartingBalance: 1000,
	}

	normalized := request.Normalize()

	if normalized.RSIPeriod != 14 || normalized.RSIOversold != 30 || normalized.RSIOverbought != 70 {
		t.Fatalf("expected default RSI fields for persistence, got %+v", normalized)
	}
}

func TestRequestNormalizeDefaultsToSavingEquityCurve(t *testing.T) {
	request := DefaultRequest()

	normalized := request.Normalize()

	if normalized.SaveEquityCurve == nil || !*normalized.SaveEquityCurve {
		t.Fatalf("expected default save_equity_curve true, got %+v", normalized.SaveEquityCurve)
	}
}

func TestRequestNormalizeDefaultsToNextOpenExecution(t *testing.T) {
	request := DefaultRequest()
	request.ExecutionFillMode = ""

	normalized := request.Normalize()

	if normalized.ExecutionFillMode != ExecutionFillModeNextOpen {
		t.Fatalf("expected default execution fill mode next_open, got %q", normalized.ExecutionFillMode)
	}
}

func TestRequestNormalizePreservesDisabledEquityCurve(t *testing.T) {
	disabled := false
	request := DefaultRequest()
	request.SaveEquityCurve = &disabled

	normalized := request.Normalize()

	if normalized.SaveEquityCurve == nil || *normalized.SaveEquityCurve {
		t.Fatalf("expected save_equity_curve false, got %+v", normalized.SaveEquityCurve)
	}
}

func TestRequestRequiredCandlesForTrendPullbackIncludesPreviousRSI(t *testing.T) {
	request := DefaultRequest()
	request.StrategyName = "btc-trend-pullback"
	request.FastPeriod = 2
	request.SlowPeriod = 3
	request.RSIPeriod = 14

	if required := request.RequiredCandles(); required != 16 {
		t.Fatalf("expected trend pullback to require previous and current RSI windows, got %d", required)
	}
}

func TestRequestRequiredCandlesForTrendPullbackIncludesDefaultATRFilter(t *testing.T) {
	request := DefaultRequest()
	request.StrategyName = "btc-trend-pullback"
	request.FastPeriod = 2
	request.SlowPeriod = 3
	request.RSIPeriod = 3

	if required := request.RequiredCandles(); required != 15 {
		t.Fatalf("expected trend pullback to require default ATR filter history, got %d", required)
	}
}

func TestValidateCandleSeriesRejectsGapsAndUnclosedCandles(t *testing.T) {
	candles := candlesForCrosses()
	candles[4].OpenTime = candles[4].OpenTime.Add(time.Minute)

	diagnostics := ValidateCandleSeries("BTCUSDT", "1m", candles)
	if diagnostics.Valid {
		t.Fatalf("expected invalid diagnostics for gap, got %+v", diagnostics)
	}
	if diagnostics.GapCount == 0 {
		t.Fatalf("expected gap count, got %+v", diagnostics)
	}

	candles = candlesForCrosses()
	candles[2].IsClosed = false
	diagnostics = ValidateCandleSeries("BTCUSDT", "1m", candles)
	if diagnostics.Valid || diagnostics.UnclosedCount != 1 {
		t.Fatalf("expected one unclosed candle, got %+v", diagnostics)
	}
}

func TestValidateCandleSeriesRejectsDuplicateAndInvalidOHLC(t *testing.T) {
	candles := candlesForCrosses()
	candles[3].OpenTime = candles[2].OpenTime

	diagnostics := ValidateCandleSeries("BTCUSDT", "1m", candles)
	if diagnostics.Valid || diagnostics.DuplicateCount == 0 {
		t.Fatalf("expected duplicate candle diagnostics, got %+v", diagnostics)
	}

	candles = candlesForCrosses()
	candles[1].High = "9"
	candles[1].Low = "11"
	candles[1].Open = "10"
	candles[1].Close = "10"
	diagnostics = ValidateCandleSeries("BTCUSDT", "1m", candles)
	if diagnostics.Valid || diagnostics.InvalidOHLCCount != 1 {
		t.Fatalf("expected invalid OHLC diagnostics, got %+v", diagnostics)
	}
}

func candlesForCrosses() []marketdata.Candle {
	base := time.Unix(1000, 0).UTC()
	closes := []string{"11", "10", "10", "12", "15", "18", "16", "14", "11", "9", "12", "16"}
	candles := make([]marketdata.Candle, 0, len(closes))
	for index, closeValue := range closes {
		openTime := base.Add(time.Duration(index) * time.Minute)
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Open:      closeValue,
			High:      closeValue,
			Low:       closeValue,
			Close:     closeValue,
			IsClosed:  true,
		})
	}
	return candles
}

func candlesForNextOpenFill() []marketdata.Candle {
	base := time.Unix(1000, 0).UTC()
	rows := []struct {
		open  string
		close string
	}{
		{"11", "11"},
		{"10", "10"},
		{"10", "10"},
		{"12", "12"},
		{"50", "15"},
		{"18", "18"},
	}
	candles := make([]marketdata.Candle, 0, len(rows))
	for index, row := range rows {
		openTime := base.Add(time.Duration(index) * time.Minute)
		high := row.open
		low := row.open
		if row.close > high {
			high = row.close
		}
		if row.close < low {
			low = row.close
		}
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Open:      row.open,
			High:      high,
			Low:       low,
			Close:     row.close,
			IsClosed:  true,
		})
	}
	return candles
}

func candlesForEndOfBacktestClose() []marketdata.Candle {
	base := time.Unix(1000, 0).UTC()
	closes := []string{"11", "10", "10", "12", "14", "16", "18"}
	candles := make([]marketdata.Candle, 0, len(closes))
	for index, closeValue := range closes {
		openTime := base.Add(time.Duration(index) * time.Minute)
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Open:      closeValue,
			High:      closeValue,
			Low:       closeValue,
			Close:     closeValue,
			IsClosed:  true,
		})
	}
	return candles
}

func candlesForRSI() []marketdata.Candle {
	base := time.Unix(1000, 0).UTC()
	closes := []string{"100", "95", "90", "85", "80", "85", "90", "95", "100", "105", "110", "100"}
	candles := make([]marketdata.Candle, 0, len(closes))
	for index, closeValue := range closes {
		openTime := base.Add(time.Duration(index) * time.Minute)
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Open:      closeValue,
			High:      closeValue,
			Low:       closeValue,
			Close:     closeValue,
			IsClosed:  true,
		})
	}
	return candles
}

func candlesForTrendFilter() []marketdata.Candle {
	base := time.Unix(1000, 0).UTC()
	closes := []string{"40", "40", "40", "40", "10", "11", "20"}
	candles := make([]marketdata.Candle, 0, len(closes))
	for index, closeValue := range closes {
		openTime := base.Add(time.Duration(index) * time.Minute)
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Open:      closeValue,
			High:      closeValue,
			Low:       closeValue,
			Close:     closeValue,
			IsClosed:  true,
		})
	}
	return candles
}

func candlesForATRStop() []marketdata.Candle {
	base := time.Unix(1000, 0).UTC()
	rows := []struct {
		high  string
		low   string
		close string
	}{
		{"11", "11", "11"},
		{"11", "9", "10"},
		{"11", "9", "10"},
		{"13", "11", "12"},
		{"12", "9", "12"},
		{"12", "11", "12"},
	}
	candles := make([]marketdata.Candle, 0, len(rows))
	for index, row := range rows {
		openTime := base.Add(time.Duration(index) * time.Minute)
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Open:      row.close,
			High:      row.high,
			Low:       row.low,
			Close:     row.close,
			IsClosed:  true,
		})
	}
	return candles
}

func candlesForTrendPullbackBacktest() []marketdata.Candle {
	base := time.Unix(1000, 0).UTC()
	closes := []string{"100", "100", "100", "100", "100", "100", "100", "100", "100", "104", "108", "112", "116", "110", "108", "117", "111", "100"}
	candles := make([]marketdata.Candle, 0, len(closes))
	for index, closeValue := range closes {
		openTime := base.Add(time.Duration(index) * 15 * time.Minute)
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "15m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(15 * time.Minute),
			Open:      closeValue,
			High:      closeValue,
			Low:       closeValue,
			Close:     closeValue,
			IsClosed:  true,
		})
	}
	return candles
}

func candlesForLargeStress(count int, interval string, step time.Duration) []marketdata.Candle {
	base := time.Unix(1000, 0).UTC()
	candles := make([]marketdata.Candle, 0, count)
	for index := 0; index < count; index++ {
		openTime := base.Add(time.Duration(index) * step)
		price := 10000 + float64(index%200) - 100
		if (index/200)%2 == 1 {
			price = 10000 - float64(index%200) + 100
		}
		open := formatTestPrice(price)
		closeValue := formatTestPrice(price + float64((index%5)-2))
		high := formatTestPrice(price + 5)
		low := formatTestPrice(price - 5)
		candles = append(candles, marketdata.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  interval,
			OpenTime:  openTime,
			CloseTime: openTime.Add(step),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closeValue,
			Volume:    "100.00",
			IsClosed:  true,
		})
	}
	return candles
}

func formatTestPrice(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func TestEngineRunsShortingSMABacktest(t *testing.T) {
	request := Request{
		StrategyName:      "sma-crossover",
		Version:           "v1",
		Symbol:            "BTCUSDT",
		Interval:          "1m",
		From:              time.Unix(1000, 0).UTC(),
		To:                time.Unix(2000, 0).UTC(),
		FastPeriod:        2,
		SlowPeriod:        3,
		StartingBalance:   1000,
		FeeRate:           0.001,
		ShortingEnabled:   true,
		ExecutionFillMode: "same_close",
	}

	run, trades, err := NewEngine().Run(request, candlesForCrosses())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(trades) < 4 {
		t.Fatalf("expected at least 4 trades with shorting enabled, got %d", len(trades))
	}

	if len(run.RoundTrips) < 2 {
		t.Fatalf("expected at least 2 completed round trips, got %d", len(run.RoundTrips))
	}

	hasShort := false
	for _, rt := range run.RoundTrips {
		if rt.EntryReason == "strategy sell signal" {
			hasShort = true
			if rt.ExitReason != "strategy buy signal" {
				t.Fatalf("expected short exit reason 'strategy buy signal', got %q", rt.ExitReason)
			}
			t.Logf("Short roundtrip: entry %f, exit %f, net pnl %f", rt.EntryPrice, rt.ExitPrice, rt.NetProfitLoss)
		}
	}

	if !hasShort {
		t.Fatal("expected at least one short position round trip")
	}
}
