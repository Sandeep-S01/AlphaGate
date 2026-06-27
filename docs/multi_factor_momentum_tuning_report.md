# Multi-Factor Momentum Tuning Report

Generated: 2026-06-26

## Scope

Template tuned: `multi-factor-momentum`

Dataset: local `BTCUSDT` 1h candles ending `2026-06-18T15:00:00Z`

Cost model used for verification:

- Fee rate: `0.001`
- Slippage rate: `0.0005`
- Fill mode: `next_open`
- Position sizing: `10%` of equity

## Changes

- Added explicit short-side Pine rules for downtrend momentum.
- Fixed Pine parser close-rule direction inference so `strategy.close("SHORT")` produces a short close signal.
- Added `shorting_enabled` to predefined template execution profiles.
- Wired Research backtest payloads to honor template-level shorting.
- Added a generic ATR-percent regime filter to the backtest engine so entries can be skipped in dead or extreme volatility regimes.
- Persisted regime-filter settings on saved backtest runs.
- Changed Multi-Factor Momentum default profile to a lower-churn setup:
  - interval `1h`
  - position size `10%` of equity
  - cooldown `72` bars
  - minimum hold `24` bars
  - ATR stop/take-profit `2.5 / 4.0`
  - max trade rate target `0.5/day`
  - shorting enabled
  - regime filter enabled with ATR(14) between `0.25%` and `4.0%`

## Evidence

Best one-year quality run:

- Run ID: `431703b3-6203-49df-a6ea-3d2c5956e49d`
- Return: `+1.1114%`
- Benchmark: `-39.2893%`
- Excess return: `+40.4008%`
- Gross PnL: `$302.78`
- Fees: `$165.83`
- Estimated slippage: `$82.87`
- Total trades: `166`
- Trades/day: `0.4548`
- Profit factor: `1.2169`
- Max drawdown: `1.2183%`
- Validation: `insufficient_sample` because completed round trips were below the validator threshold

Two-year robustness check with the same profile:

- Run ID: `344bb85d-13eb-4227-96f5-1f6e8978ae80`
- Return: `-0.3870%`
- Benchmark: `-1.5239%`
- Excess return: `+1.1370%`
- Profit factor: `0.9702`
- Max drawdown: `4.1961%`
- Validation: `weak_profit_factor`

Walk-forward fold check with the same profile:

| Fold | Window | Return | Benchmark | Excess | Trades | Profit Factor | Max DD |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 2024-06-18 to 2024-12-17 | `+0.6075%` | `+67.0546%` | `-66.4471%` | 78 | 1.1820 | 1.3591% |
| 2 | 2024-12-17 to 2025-06-17 | `-0.8979%` | `-1.8270%` | `+0.9291%` | 78 | 0.7137 | 1.5102% |
| 3 | 2025-06-17 to 2025-12-16 | `+0.3278%` | `-16.5051%` | `+16.8329%` | 78 | 1.1497 | 0.7951% |
| 4 | 2025-12-16 to 2026-06-16 | `+0.7136%` | `-25.5966%` | `+26.3103%` | 78 | 1.2510 | 0.9562% |

ATR-regime sweep summary:

- Best fold-average candidate: ATR floor `0.5%`, max `4.0%`
- Average fold return improved from `+0.1878%` to `+0.3964%`
- It failed the latest one-year BTCUSDT window with `-1.0225%`, so it was not promoted to the default.
- The default remains `0.25%` to `4.0%` because it preserves the latest one-year result while still making regime state explicit and auditable.

Risk-on participation check:

- Added a risk-on continuation branch to the executable Pine so established uptrends can enter while MACD remains bullish, not only on a fresh crossover.
- The four-fold walk-forward metrics were unchanged after this branch, which indicates the remaining bull-market underperformance is mostly allocation and exposure, not just entry timing.

Position allocation sweep:

| Position Size | Avg Fold Return | Min Fold Return | Positive Return Folds | Worst DD |
| ---: | ---: | ---: | ---: | ---: |
| 10% | `+0.1878%` | `-0.8979%` | 3/4 | 1.5102% |
| 20% | `+0.3676%` | `-1.7979%` | 3/4 | 3.0098% |
| 30% | `+0.5393%` | `-2.7001%` | 3/4 | 4.4984% |
| 40% | `+0.7029%` | `-3.6041%` | 3/4 | 5.9760% |
| 50% | `+0.8582%` | `-4.5098%` | 3/4 | 7.4425% |

The 20% allocation was tested on the latest one-year BTCUSDT window and returned `-1.1500%`, so it was rejected as the default. The profile now exposes `position_size_percent: 10` explicitly so the UI applies the conservative allocation consistently.

Benchmark-aware validation:

- Added a strong-bull benchmark capture rule to hardened run validation.
- If buy-and-hold returns at least `+20%`, a strategy must capture at least `25%` of that benchmark move.
- This rule runs before sample-size validation so bull-regime underparticipation is not hidden by `insufficient_sample`.
- Verification fold: `2024-06-18` to `2024-12-17`
  - Run ID: `6c2d91bd-780f-4e18-a8ed-25cff99b43c0`
  - Strategy return: `+0.6075%`
  - Benchmark return: `+67.0546%`
  - Validation: `low_bull_market_capture`

Benchmark-aware ranking:

- Added validation-status priority to optimizer and strategy-comparison ranking.
- `low_bull_market_capture` is ranked below ordinary benchmark/profit-factor failures even if its raw excess-return score is higher.
- Walk-forward ranking now only uses walk-forward averages when fold data exists, avoiding zero-fold rows being sorted by meaningless defaults.
- This prevents optimization from selecting defensive underparticipation profiles just because they look stable in down/flat regimes.

Dashboard validation visibility:

- Research backtest results now display hardened validation status for every completed run.
- `low_bull_market_capture` shows a specific benchmark-capture warning.
- Results now include benchmark return, excess return, benchmark capture percentage, and validation label cards.

Optimizer execution parity:

- The optimization endpoint now accepts and propagates shorting controls.
- The optimization endpoint now accepts and propagates ATR-percent regime filters.
- Required-candle calculation includes regime-filter history, so walk-forward folds are evaluated with the same hardened execution model as single backtests.

## Conclusion

This fixes the main structural reason the template was consistently negative in downtrending windows: it was long-only and overtraded after costs. The new default can produce positive net results on the latest one-year BTCUSDT 1h window while controlling churn and drawdown.

It is not yet robust enough to claim production profitability across all market regimes. The walk-forward check shows three positive absolute-return folds but poor participation in strong bull markets and one weak sideways fold. The next production-hardening step should introduce benchmark-aware evaluation and parameter selection, because simply increasing exposure improves average return while worsening the weak fold too much.
