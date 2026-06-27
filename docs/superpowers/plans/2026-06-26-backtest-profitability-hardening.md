# Backtest Profitability Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Sentra backtests explain why strategies lose, model trading costs fairly, reduce destructive overtrading defaults, and expose enough diagnostics to tune strategies toward positive net expectancy.

**Architecture:** Keep the backtest engine as the source of truth for execution accounting and diagnostics. Add explicit cost/gross/net metrics to `backtest.Run`, persist them, show them in the dashboard, and add strategy-template execution profiles so each predefined strategy uses sensible timeframe/cooldown/holding defaults instead of a global 1-minute churn profile.

**Tech Stack:** Go backtest engine/API/PostgreSQL migrations, React/Vite dashboard, existing Go unit tests and API tests.

---

## Current Root Cause Summary

The current negative PnL pattern is mostly caused by weak starter strategy rules being run on high-frequency `1m` candles with realistic costs:

- `fee_rate=0.001` and `slippage_rate=0.0005` creates roughly `0.30%` round-trip drag.
- Current templates often enter and exit quickly, producing hundreds or thousands of small trades.
- Zero-cost sensitivity testing showed `multi-factor-momentum` can move from `-13.96%` to `+0.06%` on the same sample, proving costs and churn dominate.
- Benchmark validation is slightly unfair because buy-and-hold applies entry fee/slippage but does not model final liquidation costs.
- UI does not show gross PnL, fees, slippage drag, average trade cost, or break-even move, so users only see negative final PnL without the cause.

## File Map

- Modify: `internal/backtest/types.go`
  Add cost breakdown fields to `Run`; add default normalization hooks for strategy execution profiles.
- Modify: `internal/backtest/engine.go`
  Track gross PnL, total fees, estimated slippage impact, average cost per round trip, break-even move, fair benchmark exit cost, and richer validation reason.
- Modify: `internal/backtest/engine_test.go`
  Add regression tests for cost accounting, zero-cost vs cost behavior, fair benchmark costs, and overtrading diagnostics.
- Modify: `internal/backtest/repository.go`
  Persist and scan new cost fields.
- Modify: `internal/backtest/repository_test.go`
  Update insert/select SQL tests for new persisted fields.
- Create: `internal/platform/migrations/0000XX_backtest_cost_diagnostics.sql`
  Add nullable numeric columns for cost diagnostics to existing backtest tables.
- Modify: `internal/backtest/optimizer.go`
  Rank optimization results with net expectancy and cost-aware diagnostics.
- Modify: `internal/backtest/optimizer_test.go`
  Verify optimization prefers better net expectancy and lower churn when returns are close.
- Modify: `internal/strategy/templates.go`
  Add per-template execution defaults: recommended interval, cooldown, min hold, ATR default, cost warning threshold.
- Modify: `internal/strategy/templates_test.go`
  Verify every executable template has a complete execution profile.
- Modify: `dashboard-src/src/components/ResearchWorkspace.jsx`
  Show cost diagnostics in backtest result panels and apply template defaults when selected.
- Modify: `dashboard-src/src/components/StrategyWorkspace.jsx`
  Show template execution profile and cost warnings in the strategy catalog.
- Modify: `docs/backtesting_e2e_report.md`
  Add a manual verification note after implementation.
- Modify: `docs/extending-strategies.md`
  Document how strategy templates should define cost-aware execution profiles.

---

### Task 1: Add Cost Diagnostic Fields To Backtest Run

**Files:**
- Modify: `internal/backtest/types.go`
- Modify: `internal/backtest/engine_test.go`
- Modify: `internal/backtest/engine.go`

- [ ] **Step 1: Write the failing cost-field test**

Add this test to `internal/backtest/engine_test.go` near existing fee/slippage tests:

```go
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
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```powershell
go test ./internal/backtest -run TestEngineReportsCostDiagnostics -count=1
```

Expected: build failure because `Run.TotalFees`, `Run.EstimatedSlippageCost`, `Run.GrossProfitLoss`, `Run.RoundTripCostPercent`, and `Run.BreakEvenMovePercent` do not exist.

- [ ] **Step 3: Add fields to `Run`**

In `internal/backtest/types.go`, add these fields after `ProfitLoss`:

```go
GrossProfitLoss       float64 `json:"gross_profit_loss"`
TotalFees             float64 `json:"total_fees"`
EstimatedSlippageCost float64 `json:"estimated_slippage_cost"`
RoundTripCostPercent  float64 `json:"round_trip_cost_percent"`
BreakEvenMovePercent  float64 `json:"break_even_move_percent"`
```

- [ ] **Step 4: Track costs in engine**

In `internal/backtest/engine.go`, initialize these variables near the existing balance variables:

```go
grossProfitLoss := 0.0
totalFees := 0.0
estimatedSlippageCost := 0.0
```

In entry branches, after computing `fee`, add:

```go
totalFees += fee
estimatedSlippageCost += math.Abs(fillPriceBase-fillPrice) * quantity
```

In `closePosition`, after computing close fee and quantity, add:

```go
totalFees += fee
estimatedSlippageCost += math.Abs(price-fillPrice) * quantity
if openTrade != nil {
	if openTrade.Side == SideBuy {
		grossProfitLoss += (fillPrice - openTrade.Price) * openTrade.Quantity
	} else {
		grossProfitLoss += (openTrade.Price - fillPrice) * openTrade.Quantity
	}
}
```

Before constructing `Run`, add:

```go
roundTripCostPercent := (request.FeeRate*2 + request.SlippageRate*2) * 100
breakEvenMovePercent := roundTripCostPercent
```

Populate the new `Run` fields:

```go
GrossProfitLoss:       grossProfitLoss,
TotalFees:             totalFees,
EstimatedSlippageCost: estimatedSlippageCost,
RoundTripCostPercent:  roundTripCostPercent,
BreakEvenMovePercent:  breakEvenMovePercent,
```

- [ ] **Step 5: Run the test and verify it passes**

Run:

```powershell
go test ./internal/backtest -run TestEngineReportsCostDiagnostics -count=1
```

Expected: PASS.

- [ ] **Step 6: Run focused package tests**

Run:

```powershell
go test ./internal/backtest
```

Expected: PASS.

---

### Task 2: Make Benchmark Cost Modeling Fair

**Files:**
- Modify: `internal/backtest/engine.go`
- Modify: `internal/backtest/engine_test.go`

- [ ] **Step 1: Write the failing benchmark test**

Add this test near `TestEngineCalculatesBenchmarkMetrics`:

```go
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
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```powershell
go test ./internal/backtest -run TestBenchmarkBuyAndHoldAppliesExitCosts -count=1
```

Expected: FAIL because current benchmark does not apply exit slippage/fee.

- [ ] **Step 3: Update benchmark liquidation logic**

In `benchmarkBuyAndHold`, replace:

```go
endingBalance := quantity * lastPrice
```

with:

```go
exitPrice := applySellSlippage(lastPrice, request.SlippageRate)
exitGross := quantity * exitPrice
exitFee := exitGross * request.FeeRate
endingBalance := exitGross - exitFee
```

- [ ] **Step 4: Run the benchmark test**

Run:

```powershell
go test ./internal/backtest -run TestBenchmarkBuyAndHoldAppliesExitCosts -count=1
```

Expected: PASS.

---

### Task 3: Persist Cost Diagnostics

**Files:**
- Create: `internal/platform/migrations/0000XX_backtest_cost_diagnostics.sql`
- Modify: `internal/backtest/repository.go`
- Modify: `internal/backtest/repository_test.go`

- [ ] **Step 1: Create migration**

Create the next migration number after the current highest file in `internal/platform/migrations`. Use the actual next number in the filename.

Migration content:

```sql
ALTER TABLE backtest_runs
    ADD COLUMN IF NOT EXISTS gross_profit_loss DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_fees DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS estimated_slippage_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS round_trip_cost_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS break_even_move_percent DOUBLE PRECISION NOT NULL DEFAULT 0;

ALTER TABLE strategy_comparison_results
    ADD COLUMN IF NOT EXISTS gross_profit_loss DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_fees DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS estimated_slippage_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS round_trip_cost_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS break_even_move_percent DOUBLE PRECISION NOT NULL DEFAULT 0;
```

- [ ] **Step 2: Update repository SQL tests first**

In `TestBuildInsertRunSQLUsesRunFields`, set:

```go
GrossProfitLoss:       25,
TotalFees:             2,
EstimatedSlippageCost: 1.5,
RoundTripCostPercent:  0.3,
BreakEvenMovePercent:  0.3,
```

Update expected argument count by adding 5 to the current expected value.

Add assertions:

```go
if !strings.Contains(query, "gross_profit_loss") || !strings.Contains(query, "total_fees") || !strings.Contains(query, "estimated_slippage_cost") {
	t.Fatalf("expected cost diagnostics in insert, got %s", query)
}
```

- [ ] **Step 3: Run repository test and verify it fails**

Run:

```powershell
go test ./internal/backtest -run TestBuildInsertRunSQLUsesRunFields -count=1
```

Expected: FAIL because insert SQL does not include new fields.

- [ ] **Step 4: Update insert/select/scan SQL**

In `BuildInsertRunSQL`, add the five columns and five args in the same order as the `Run` fields.

In `scanRun`, scan the five new fields into:

```go
&run.GrossProfitLoss,
&run.TotalFees,
&run.EstimatedSlippageCost,
&run.RoundTripCostPercent,
&run.BreakEvenMovePercent,
```

In `runSelectSQL`, add:

```sql
r.gross_profit_loss, r.total_fees, r.estimated_slippage_cost,
r.round_trip_cost_percent, r.break_even_move_percent,
```

- [ ] **Step 5: Run repository package tests**

Run:

```powershell
go test ./internal/backtest
```

Expected: PASS.

---

### Task 4: Add Overtrading And Cost Validation Diagnostics

**Files:**
- Modify: `internal/backtest/engine.go`
- Modify: `internal/backtest/engine_test.go`
- Modify: `internal/backtest/validation.go`

- [ ] **Step 1: Write failing validation tests**

Add these tests near existing `validateRunCandidate` tests:

```go
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

func TestValidateRunCandidateRejectsExcessiveChurn(t *testing.T) {
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
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
go test ./internal/backtest -run "TestValidateRunCandidateRejectsAverageTradeBelowCost|TestValidateRunCandidateRejectsExcessiveChurn" -count=1
```

Expected: build failure or wrong status because new validation inputs are missing.

- [ ] **Step 3: Extend validation input**

In `runValidationInput`, add:

```go
BreakEvenMovePercent float64
TradesPerDay         float64
```

When calling `validateRunCandidate`, pass:

```go
BreakEvenMovePercent: breakEvenMovePercent,
TradesPerDay:         tradesPerDay,
```

- [ ] **Step 4: Add validation logic**

In `validateRunCandidate`, after insufficient sample checks and before benchmark checks, add:

```go
if input.TradesPerDay > 50 && input.Interval == "1m" {
	return "overtrading", "trade frequency is too high for the selected interval"
}
if input.BreakEvenMovePercent > 0 && input.AverageTrade > 0 && input.AverageTrade < input.BreakEvenMovePercent {
	return "cost_drag", "average trade does not exceed estimated round-trip cost"
}
```

- [ ] **Step 5: Run validation tests**

Run:

```powershell
go test ./internal/backtest -run "TestValidateRunCandidateRejectsAverageTradeBelowCost|TestValidateRunCandidateRejectsExcessiveChurn" -count=1
```

Expected: PASS.

---

### Task 5: Add Strategy Template Execution Profiles

**Files:**
- Modify: `internal/strategy/templates.go`
- Modify: `internal/strategy/templates_test.go`
- Modify: `internal/backtest/types.go`

- [ ] **Step 1: Add failing template-profile test**

In `internal/strategy/templates_test.go`, add:

```go
func TestExecutableTemplatesHaveExecutionProfiles(t *testing.T) {
	for _, tmpl := range PredefinedTemplates() {
		if tmpl.SupportStatus != TemplateExecutablePine {
			continue
		}
		if tmpl.ExecutionProfile.RecommendedInterval == "" {
			t.Fatalf("template %s missing recommended interval", tmpl.ID)
		}
		if tmpl.ExecutionProfile.CooldownBars <= 0 {
			t.Fatalf("template %s missing cooldown bars", tmpl.ID)
		}
		if tmpl.ExecutionProfile.MinHoldingBars <= 0 {
			t.Fatalf("template %s missing min holding bars", tmpl.ID)
		}
		if tmpl.ExecutionProfile.MaxTradesPerDay <= 0 {
			t.Fatalf("template %s missing max trades per day", tmpl.ID)
		}
	}
}
```

- [ ] **Step 2: Run test and verify failure**

Run:

```powershell
go test ./internal/strategy -run TestExecutableTemplatesHaveExecutionProfiles -count=1
```

Expected: build failure because `ExecutionProfile` does not exist.

- [ ] **Step 3: Add profile type**

In `internal/strategy/templates.go`, add:

```go
type TemplateExecutionProfile struct {
	RecommendedInterval string  `json:"recommended_interval"`
	CooldownBars        int     `json:"cooldown_bars"`
	MinHoldingBars      int     `json:"min_holding_bars"`
	ATRExitEnabled      bool    `json:"atr_exit_enabled"`
	ATRPeriod           int     `json:"atr_period"`
	ATRStopMultiplier   float64 `json:"atr_stop_multiplier"`
	ATRTakeProfitMultiplier float64 `json:"atr_take_profit_multiplier"`
	MaxTradesPerDay     float64 `json:"max_trades_per_day"`
}
```

Add to `StrategyTemplate`:

```go
ExecutionProfile TemplateExecutionProfile `json:"execution_profile"`
```

- [ ] **Step 4: Populate executable template profiles**

Use these initial profiles:

```go
ExecutionProfile: TemplateExecutionProfile{
	RecommendedInterval: "15m",
	CooldownBars:        8,
	MinHoldingBars:      4,
	ATRExitEnabled:      true,
	ATRPeriod:           14,
	ATRStopMultiplier:   2,
	ATRTakeProfitMultiplier: 3,
	MaxTradesPerDay:     12,
},
```

For `multi-factor-momentum`, use a slower profile:

```go
ExecutionProfile: TemplateExecutionProfile{
	RecommendedInterval: "1h",
	CooldownBars:        6,
	MinHoldingBars:      3,
	ATRExitEnabled:      true,
	ATRPeriod:           14,
	ATRStopMultiplier:   2.5,
	ATRTakeProfitMultiplier: 4,
	MaxTradesPerDay:     6,
},
```

- [ ] **Step 5: Run strategy tests**

Run:

```powershell
go test ./internal/strategy
```

Expected: PASS.

---

### Task 6: Apply Template Profiles In Dashboard Backtest Form

**Files:**
- Modify: `dashboard-src/src/components/ResearchWorkspace.jsx`
- Modify: `dashboard-src/src/components/StrategyWorkspace.jsx`

- [ ] **Step 1: Locate strategy selection handler**

Run:

```powershell
rg -n "strategyModel|setStrategy|templates|cooldown|atr_exit|backtest" dashboard-src/src/components/ResearchWorkspace.jsx
```

Expected: find the strategy dropdown state and request body creation.

- [ ] **Step 2: Apply profile values when a predefined template is selected**

In the template selection handler, add logic equivalent to:

```jsx
const applyTemplateProfile = (template) => {
  if (!template?.execution_profile) return
  const profile = template.execution_profile
  setBacktestParams((current) => ({
    ...current,
    interval: profile.recommended_interval || current.interval,
    cooldown_bars: profile.cooldown_bars || current.cooldown_bars,
    min_holding_bars: profile.min_holding_bars || current.min_holding_bars,
    atr_exit_enabled: Boolean(profile.atr_exit_enabled),
    atr_period: profile.atr_period || current.atr_period,
    atr_stop_multiplier: profile.atr_stop_multiplier || current.atr_stop_multiplier,
    atr_take_profit_multiplier: profile.atr_take_profit_multiplier || current.atr_take_profit_multiplier,
  }))
}
```

- [ ] **Step 3: Add visible profile note**

Render a compact note near the strategy dropdown:

```jsx
{selectedTemplate?.execution_profile && (
  <div className="mt-2 text-[12px] text-slate-300">
    Profile: {selectedTemplate.execution_profile.recommended_interval} · cooldown {selectedTemplate.execution_profile.cooldown_bars} bars · max {selectedTemplate.execution_profile.max_trades_per_day}/day
  </div>
)}
```

- [ ] **Step 4: Run frontend checks**

Run:

```powershell
cd dashboard-src
npm run lint
npm run build
```

Expected: both PASS.

---

### Task 7: Show Gross/Net/Cost Diagnostics In UI

**Files:**
- Modify: `dashboard-src/src/components/ResearchWorkspace.jsx`

- [ ] **Step 1: Add cost metrics to result panel**

In the backtest result summary area, add rows for:

```jsx
<Metric label="Gross PnL" value={formatCurrency(result.gross_profit_loss)} />
<Metric label="Fees" value={formatCurrency(result.total_fees)} />
<Metric label="Slippage Cost" value={formatCurrency(result.estimated_slippage_cost)} />
<Metric label="Round Trip Cost" value={`${formatNumber(result.round_trip_cost_percent)}%`} />
<Metric label="Break Even Move" value={`${formatNumber(result.break_even_move_percent)}%`} />
```

Use the existing local metric/card component if one already exists. Do not create a new visual style.

- [ ] **Step 2: Add cost warning**

Near validation status, render:

```jsx
{result.validation_status === 'cost_drag' && (
  <div className="mt-2 border border-amber-400/40 bg-amber-400/10 px-3 py-2 text-[12px] text-amber-100">
    Average trade is below estimated round-trip cost. Reduce churn, use a higher timeframe, or improve entry quality.
  </div>
)}
```

- [ ] **Step 3: Run frontend checks**

Run:

```powershell
cd dashboard-src
npm run lint
npm run build
```

Expected: both PASS.

---

### Task 8: Add Cost-Aware Optimization Ranking

**Files:**
- Modify: `internal/backtest/optimizer.go`
- Modify: `internal/backtest/optimizer_test.go`

- [ ] **Step 1: Write failing optimizer ranking test**

Add:

```go
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
```

- [ ] **Step 2: Run test and verify failure**

Run:

```powershell
go test ./internal/backtest -run TestOptimizerRanksCostAwareResultsAboveHighChurnResults -count=1
```

Expected: FAIL because current ranking prioritizes excess return directly.

- [ ] **Step 3: Add score helper**

In `optimizer.go`, add:

```go
func optimizationScore(result OptimizationResult) float64 {
	score := result.ExcessReturnPercent
	score += result.ProfitFactor * 2
	score += result.AverageTrade
	if result.TradesPerDay > 20 {
		score -= (result.TradesPerDay - 20) * 0.25
	}
	return score
}
```

Update ranking to sort by `optimizationScore`.

- [ ] **Step 4: Run optimizer tests**

Run:

```powershell
go test ./internal/backtest -run Optimizer -count=1
```

Expected: PASS.

---

### Task 9: Add API/E2E Cost Diagnostics Coverage

**Files:**
- Modify: `internal/api/middleware_test.go`
- Modify: `tools/backtest-agent/main.go`
- Modify: `docs/backtesting_e2e_report.md`

- [ ] **Step 1: Add API test assertions**

In the existing successful backtest API test, assert:

```go
if data["total_fees"] == nil {
	t.Fatalf("expected total_fees in response body: %+v", data)
}
if data["round_trip_cost_percent"] == nil {
	t.Fatalf("expected round_trip_cost_percent in response body: %+v", data)
}
```

- [ ] **Step 2: Run API test**

Run:

```powershell
go test ./internal/api -run Backtest -count=1
```

Expected: PASS after Task 3 persistence/API JSON changes.

- [ ] **Step 3: Update E2E report output**

In `tools/backtest-agent/main.go`, include cost fields in successful run evidence:

```go
Evidence: fmt.Sprintf("return=%.2f%% trades=%d fees=%.2f roundTripCost=%.2f%% validation=%s", run.ReturnPercent, run.TotalTrades, run.TotalFees, run.RoundTripCostPercent, run.ValidationStatus),
```

- [ ] **Step 4: Run E2E agent**

Run:

```powershell
go run ./tools/backtest-agent -base-url http://127.0.0.1:8080 -output-dir docs
```

Expected: report generated with cost diagnostics.

---

### Task 10: Documentation And Final Verification

**Files:**
- Modify: `docs/extending-strategies.md`
- Modify: `docs/backtesting_e2e_report.md`

- [ ] **Step 1: Document cost-aware strategy requirements**

Add this section to `docs/extending-strategies.md`:

```markdown
## Cost-Aware Strategy Requirements

Every executable strategy must be evaluated net of fees and slippage. A strategy should not be considered production-ready unless its average trade is greater than the estimated round-trip cost and it remains positive against a fair buy-and-hold benchmark.

For high-frequency intervals such as `1m`, use stricter cooldown and minimum holding bars. A useful first threshold is:

- average trade greater than round-trip cost
- profit factor greater than 1.2
- positive excess return
- trades per day below the template profile limit
- max drawdown inside the strategy's declared risk limit
```

- [ ] **Step 2: Run all backend tests**

Run:

```powershell
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Run frontend checks**

Run:

```powershell
cd dashboard-src
npm run lint
npm run build
```

Expected: PASS.

- [ ] **Step 4: Manual verification**

Run the API locally, open the dashboard, and verify:

- Selecting `multi-factor-momentum` applies the profile interval/cooldown/min-hold.
- Backtest results show gross PnL, net PnL, fees, slippage, round-trip cost, and break-even move.
- A losing high-churn result explains whether the failure is `cost_drag`, `overtrading`, `weak_profit_factor`, or `underperforms_benchmark`.
- A zero-cost sensitivity run shows visibly different results from realistic-cost run.

---

## Self-Review

**Spec coverage:** This plan covers cost accounting, benchmark fairness, diagnostics, template defaults, UI visibility, optimizer ranking, API/E2E coverage, and docs.

**Placeholders:** The migration number uses `0000XX` only because the next migration number must be selected from the local folder at execution time. All implementation logic is otherwise concrete.

**Type consistency:** New cost fields are named consistently across `Run`, JSON, SQL, UI, and tests:

- `gross_profit_loss`
- `total_fees`
- `estimated_slippage_cost`
- `round_trip_cost_percent`
- `break_even_move_percent`

**Execution order:** Tasks are sequenced so tests fail before production edits, then backend persistence/API, then UI and docs.

---

## Implementation Note: Optimizer UI Flow

Added a Research workspace optimizer flow for the backend optimization contract. The UI now:

- Runs the selected executable strategy through the fast/slow parameter optimizer from the backtest panel.
- Uses strategy-specific grids for native strategies: SMA fast/slow, RSI period plus threshold bands, and BTC trend-pullback fast/slow plus RSI period.
- Reuses the selected fee, sizing, ATR exit, regime filter, shorting, and execution timing controls.
- Enables train/test and 3-fold walk-forward validation in the optimizer request.
- Displays ranked candidates with return, excess return, benchmark capture, profit factor, drawdown, and validation status.
- Allows applying a ranked candidate back into the Fast Period and Slow Period fields for a follow-up single backtest.

Current scope note: the optimizer now accepts `strategy_name` and uses the same strategy validation path as single backtests. Native strategies have strategy-aware grids. Predefined Pine/template strategies can be executed and ranked through the optimizer, but deeper tuning of Pine internals still needs template-specific parameter metadata.
