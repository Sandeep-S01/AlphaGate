# Sentra Project Rules

## Backtest Module — Critical Safety Rules

These rules encode hard-won lessons from a catastrophic bug where the backtest engine
produced -99% to -100% PnL due to fee-compounding from overtrading. The root causes
were: (1) all_in default position sizing, (2) zero cooldown/holding bars, (3) SMA signal
thrashing from loose crossover comparisons. These rules exist to prevent recurrence.

### Position Sizing
- **NEVER** default `PositionSizingMode` to `all_in`. The safe default is `percent_equity`.
- When `all_in` is selected and `FeeRate > 0`, emit a `SanityWarning` to the caller.
- Always run `SanityCheckRequest()` before executing a backtest.

### Cooldown and Minimum Holding
- `CooldownBars` and `MinHoldingBars` MUST default to at least 1 in `Normalize()`.
- Never allow zero values to pass through without normalization — they enable same-candle
  round trips that destroy capital through fee compounding.

### Strategy Signal Quality
- SMA crossover signals MUST use strict `<` / `>` (not `<=` / `>=`) for the previous-bar
  comparison to prevent whipsaw signals when SMAs are equal.
- SMA crossover MUST include a minimum gap threshold (currently 0.01% of slow SMA).
- Trend Pullback sell condition MUST require BOTH `currentClose < trend` AND `RSI < 35`
  (AND logic). Never use OR — `RSI < 45` alone fires during normal consolidation.

### RSI Implementation
- RSI MUST use Wilder's exponential smoothing, not simple-average. The `rsi()` function
  takes a `period` parameter and applies proper smoothing when sufficient candles exist.

### Same-Candle Re-entry
- After an ATR stop-loss or take-profit exit, the engine MUST NOT open a new position
  on the same candle. The `atrExitedThisCandle` flag enforces this.

### Testing
- Backtest engine tests MUST validate that fee-paying round trips do NOT produce
  returns below -50% with starting balance $1000 and realistic BTC prices.
- Any change to position sizing, cooldown, or signal logic requires re-running the
  full test suite: `go test ./internal/backtest/... ./internal/strategy/... -v`
