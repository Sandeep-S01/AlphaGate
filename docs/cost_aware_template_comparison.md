# Cost-Aware Template Comparison

Generated: 2026-06-26

Symbol: `BTCUSDT`

Assumptions:
- Starting balance: `$10,000`
- Fee rate: `0.10%` per side
- Slippage rate: `0.05%` per side
- Round-trip cost estimate: `0.30%`
- Execution fill mode: `next_open`
- Position sizing: `10%` equity
- Equity curve persistence disabled for the comparison runs
- Each template used its execution profile interval, cooldown, minimum hold, and ATR settings.

## Ranked Results

| Rank | Strategy | Interval | Return | Benchmark | Excess | Trades | Fees | Slippage | Gross PnL | Net PnL | Profit Factor | Validation |
| ---: | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| 1 | Multi-Factor Momentum | `1h` | `-1.21%` | `-39.29%` | `+38.08%` | 210 | `$208.10` | `$104.00` | `$190.79` | `-$121.31` | `0.8254` | `weak_profit_factor` |
| 2 | Adaptive Mean Reversion | `15m` | `-13.21%` | `-40.31%` | `+27.10%` | 1,032 | `$960.89` | `$480.21` | `$120.55` | `-$1,320.55` | `0.1515` | `weak_profit_factor` |
| 3 | Momentum Breakout (Volume-Confirmed) | `15m` | `-28.05%` | `-40.31%` | `+12.26%` | 2,124 | `$1,818.41` | `$908.75` | `-$77.33` | `-$2,804.50` | `0.2728` | `weak_profit_factor` |
| 4 | Trend Following (Multi-Timeframe) | `15m` | `-28.32%` | `-40.31%` | `+11.99%` | 2,182 | `$1,858.90` | `$928.99` | `-$43.81` | `-$2,831.69` | `0.2905` | `weak_profit_factor` |

## Run IDs

| Strategy | Run ID |
| --- | --- |
| `multi-factor-momentum` | `20b26781-c493-4491-87f4-321a21678133` |
| `adaptive-mean-reversion` | `64fcecf8-8fa6-4c90-bcaf-a50a63135c38` |
| `momentum-breakout-volume` | `c00bba03-524e-4832-acb6-af1e453ac4f6` |
| `trend-following-mtf` | `81e02dbb-c340-47e3-8be2-7f558a3bf1ad` |

## Findings

1. `multi-factor-momentum` is the best candidate to tune next. It nearly breaks even net while strongly outperforming buy-and-hold on the same period.
2. The `15m` templates still overtrade relative to their edge. Fees plus slippage are larger than their gross PnL.
3. `adaptive-mean-reversion` has positive gross PnL before costs, but costs fully erase the edge.
4. `trend-following-mtf` and `momentum-breakout-volume` both have negative gross PnL before costs on this period, so tuning execution costs alone will not fix them.
5. All four fail production validation due to weak profit factor, not insufficient sample.

## Recommended Next Optimization Target

Tune `multi-factor-momentum` first:

- Keep interval at `1h`.
- Reduce trading further with stricter entry confirmation.
- Test `cooldown_bars` in `[6, 12, 24]`.
- Test `min_holding_bars` in `[3, 6, 12]`.
- Test ATR stop/take-profit pairs:
  - `2.5 / 4.0`
  - `3.0 / 5.0`
  - `4.0 / 6.0`
- Add MACD acceleration or slope confirmation because the current starter Pine omits it.
