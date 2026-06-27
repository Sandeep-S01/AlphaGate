# Predefined Strategy Templates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add all 10 strategies from the provided Quantitative Trading Strategy Report as predefined Sentra strategy templates, with clear metadata, starter Pine/code specs, runtime support status, and dashboard access.

**Architecture:** Introduce a first-class strategy template catalog that is separate from saved user Pine strategies and separate from live Go evaluators. Templates can be copied into the Pine editor or used as presets for executable native strategies where Sentra already has the required OHLCV-only data support.

**Tech Stack:** Go API, React/Vite dashboard, existing Pine parser/evaluator, PostgreSQL migrations only if persisted template storage is chosen.

---

## Scope

Add predefined templates for:

1. Trend Following, multi-timeframe.
2. Momentum Breakout, volume-confirmed.
3. Adaptive Mean Reversion.
4. Statistical Arbitrage / Pairs Trading.
5. VWAP Reversion.
6. Crypto Market Making.
7. Funding Rate Arbitrage.
8. Grid Trading.
9. Multi-Factor Momentum.
10. Smart Money / Order Flow.

Do not implement live order execution for strategies requiring unsupported data in this step. Mark them as `template_only` or `blocked_by_data` with explicit requirements.

## File Structure

- Create: `internal/strategy/templates.go`
  - Owns predefined strategy template metadata and starter Pine/source text.
  - Exposes `PredefinedTemplates()` and `GetPredefinedTemplate(id string)`.

- Create: `internal/strategy/templates_test.go`
  - Verifies all 10 document strategies exist, IDs are stable, names are unique, support statuses are valid, and executable/native mappings are honest.

- Modify: `internal/strategy/types.go`
  - Add constants for the new strategy IDs that can be native or future-native.
  - Keep existing constants stable.

- Modify: `internal/api/router.go`
  - Add `GET /api/v1/strategies/templates`.
  - Add `GET /api/v1/strategies/templates/{id}`.
  - Do not require database access for these catalog endpoints.

- Modify: `internal/api/middleware_test.go`
  - Add route tests proving the template list/detail endpoints return the full catalog and safe errors.

- Modify: `dashboard-src/src/components/StrategyWorkspace.jsx`
  - Add a “Predefined Templates” panel/tab.
  - Let the user preview a template and load it into the Pine editor.
  - Preserve the existing Pine repository behavior.

- Optional Create: `dashboard-src/src/components/strategyTemplates.js`
  - Only if the UI needs a local fallback while API is unavailable.
  - Prefer API source of truth if possible.

- Modify: `docs/extending-strategies.md`
  - Document the difference between predefined templates, saved Pine strategies, and native evaluators.

## Template Data Model

Use a Go struct similar to:

```go
type TemplateSupportStatus string

const (
    TemplateExecutableNative TemplateSupportStatus = "executable_native"
    TemplateExecutablePine   TemplateSupportStatus = "executable_pine"
    TemplateTemplateOnly     TemplateSupportStatus = "template_only"
    TemplateBlockedByData    TemplateSupportStatus = "blocked_by_data"
)

type StrategyTemplate struct {
    ID              string                `json:"id"`
    Name            string                `json:"name"`
    Category        string                `json:"category"`
    Market          string                `json:"market"`
    TimeHorizon     string                `json:"time_horizon"`
    Summary         string                `json:"summary"`
    SupportStatus   TemplateSupportStatus `json:"support_status"`
    NativeStrategy  string                `json:"native_strategy,omitempty"`
    RequiredData    []string              `json:"required_data"`
    Indicators      []string              `json:"indicators"`
    EntryRules      []string              `json:"entry_rules"`
    ExitRules       []string              `json:"exit_rules"`
    RiskRules       []string              `json:"risk_rules"`
    DefaultSettings Settings              `json:"default_settings"`
    PineCode        string                `json:"pine_code,omitempty"`
    Blockers        []string              `json:"blockers,omitempty"`
}
```

## Support Mapping

- `trend-following-mtf`
  - Status: `executable_pine` first, later native.
  - Data: OHLCV.
  - Indicators: EMA20, SMA50, ADX14, ATR14, volume SMA20, 20-day high.
  - Note: Pine parser may not fully support ADX/highest today, so include blocker if validation fails.

- `momentum-breakout-volume`
  - Status: `executable_pine` if parser supports Bollinger Bands and comparisons; otherwise `template_only`.
  - Data: OHLCV.
  - Indicators: RSI14, Bollinger Bands, volume SMA, ATR.

- `adaptive-mean-reversion`
  - Status: `executable_pine` or native candidate.
  - Data: OHLCV.
  - Indicators: Bollinger Bands, RSI2, SMA200, ATR14, volume SMA20.

- `stat-arb-pairs`
  - Status: `blocked_by_data`.
  - Data: two-symbol synchronized OHLCV, spread/z-score, cointegration test.
  - Blocker: current strategy evaluator receives one candle series only.

- `vwap-reversion`
  - Status: `template_only` initially.
  - Data: intraday OHLCV with reliable session boundaries.
  - Blocker: current model lacks session VWAP helper and market-session calendar.

- `crypto-market-making`
  - Status: `blocked_by_data`.
  - Data: L1/L2 order book, trade flow, funding rate, inventory state, low-latency order gateway.

- `funding-rate-arbitrage`
  - Status: `blocked_by_data`.
  - Data: spot/perp prices, funding rates, borrow rates, multi-leg execution.

- `grid-trading`
  - Status: `template_only`.
  - Data: OHLCV plus persistent open grid order state.
  - Blocker: evaluator is candle-signal based and does not manage standing order ladders.

- `multi-factor-momentum`
  - Status: `executable_pine` first, strongest native candidate.
  - Data: OHLCV.
  - Indicators: SMA50, SMA200, RSI14, MACD, ATR14, volume SMA20.

- `smart-money-order-flow`
  - Status: `blocked_by_data`.
  - Data: CVD, bid/ask imbalance, liquidation feed, open interest, funding.

## Tasks

### Task 1: Add Catalog Types And Data

**Files:**
- Create: `internal/strategy/templates.go`
- Test: `internal/strategy/templates_test.go`

- [x] **Step 1: Write tests for catalog completeness**

Test that `PredefinedTemplates()` returns exactly 10 entries and includes the 10 stable IDs listed in this plan.

- [x] **Step 2: Write tests for catalog integrity**

Assert every template has non-empty name, category, summary, required data, risk rules, and a valid support status.

- [x] **Step 3: Implement `StrategyTemplate` and constants**

Add support-status constants and catalog structs in `internal/strategy/templates.go`.

- [x] **Step 4: Add the 10 templates**

Populate all document strategies with entry rules, exit rules, risk rules, required data, and blockers where needed.

- [x] **Step 5: Run strategy tests**

Run: `go test ./internal/strategy`

Expected: pass.

### Task 2: Add API Endpoints

**Files:**
- Modify: `internal/api/router.go`
- Test: `internal/api/middleware_test.go`

- [x] **Step 1: Write route tests**

Add tests for:

- `GET /api/v1/strategies/templates` returns 10 templates.
- `GET /api/v1/strategies/templates/multi-factor-momentum` returns the detail payload.
- Unknown template returns `404`.

- [x] **Step 2: Implement handlers**

Add two handlers near the existing Pine routes:

- `strategyTemplatesHandler`
- `strategyTemplateDetailHandler`

- [x] **Step 3: Register routes**

Register before `/api/v1/strategies/pine/{id}` to avoid path ambiguity.

- [x] **Step 4: Run API tests**

Run: `go test ./internal/api`

Expected: pass.

### Task 3: Add Pine Starter Code Where Feasible

**Files:**
- Modify: `internal/strategy/templates.go`
- Test: `internal/strategy/templates_test.go`
- Optional Test: `internal/pine/parser_test.go`

- [x] **Step 1: Add parser validation test for Pine-backed templates**

For templates marked `executable_pine`, parse `PineCode` with `pine.NewParser()` and assert no fatal errors.

- [x] **Step 2: Downgrade unsupported Pine templates honestly**

If the current parser cannot handle a strategy, keep its Pine-like source as reference text but mark it `template_only`, not `executable_pine`.

- [x] **Step 3: Keep `multi-factor-momentum` as the first executable template target**

It best matches current Sentra needs and the document recommendation.

- [x] **Step 4: Run parser and strategy tests**

Run: `go test ./internal/pine ./internal/strategy`

Expected: pass.

### Task 4: Expose Templates In Strategy Workspace

**Files:**
- Modify: `dashboard-src/src/components/StrategyWorkspace.jsx`

- [x] **Step 1: Add template state and fetch**

Fetch `/api/v1/strategies/templates` on mount alongside saved Pine strategies.

- [x] **Step 2: Add template/source tabs**

Use two compact tabs in the left panel:

- `Templates`
- `Saved`

- [x] **Step 3: Add template list cards**

Each template row should show name, market/category, and support badge:

- Native
- Pine
- Template
- Data Blocked

- [x] **Step 4: Add template preview/load behavior**

Clicking a template should:

- Set active template.
- Fill strategy name.
- Load `pine_code` if present.
- Show template rules and blockers in diagnostics.
- Disable save or show warning if no Pine code exists.

- [x] **Step 5: Preserve existing saved strategy behavior**

Do not break list, validate, save, or select behavior for `/api/v1/strategies/pine`.

### Task 5: Add Documentation

**Files:**
- Modify: `docs/extending-strategies.md`

- [x] **Step 1: Document template statuses**

Explain `executable_native`, `executable_pine`, `template_only`, and `blocked_by_data`.

- [x] **Step 2: Document the 10 imported templates**

Add a small table with ID, strategy, support status, and required data.

- [x] **Step 3: Document promotion path**

Explain how to promote a template into a native evaluator after required data and tests exist.

### Task 6: Verification

**Files:**
- No code file ownership; verification only.

- [x] **Step 1: Run Go tests**

Run: `go test ./...`

Expected: pass.

- [x] **Step 2: Run frontend lint/build**

Run:

```powershell
cd dashboard-src
npm run lint
npm run build
```

Expected: pass.

- [ ] **Step 3: Manual dashboard check**

Blocked in this run: the already-running API on `127.0.0.1:8080` is serving an older router and returns 404 for the new template endpoint. Code-level route tests, lint, and production build pass.

Open Strategy workspace and verify:

- Templates list loads.
- All 10 templates are visible.
- Multi-Factor Momentum can be loaded into the editor.
- Blocked strategies clearly show missing data requirements.
- Existing saved Pine strategy list still works.

## Non-Goals For This Step

- Do not add exchange funding-rate collectors.
- Do not add L2 order book persistence.
- Do not implement multi-leg execution.
- Do not make market-making or funding-arbitrage live.
- Do not alter trading/risk execution behavior until templates are visible and verified.

## Recommended Next Implementation Order

1. Catalog and tests.
2. API list/detail endpoints.
3. Dashboard template browser.
4. Pine validation for feasible templates.
5. Documentation and full verification.
