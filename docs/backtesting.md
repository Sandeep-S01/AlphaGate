# Backtesting

Sentra backtests run against persisted PostgreSQL candles. They do not publish Redis events and do not mutate the paper trading account.

## Execution Model

The backtest engine evaluates the selected strategy over the chosen candle range.

Fill model:

- `execution_fill_mode: "next_open"` fills strategy buy/sell signals on the next candle open
- `execution_fill_mode: "same_close"` keeps the older research/debug behavior and fills on the signal candle close
- backtest requests default to `next_open` when `execution_fill_mode` is omitted
- dashboard backtests default to `next_open`
- strategy comparisons and SMA optimizations default to `next_open`
- optional ATR exits can close an open long position using candle high/low
- any long position still open at the end of the selected range is force-closed at the final candle close with normal sell slippage and fees
- fees are applied using `fee_rate`
- one long position at a time
- entries can use all available quote balance, a fixed quote amount, or a percent of current equity

Use `next_open` for serious research. A strategy only knows a candle after it closes, so filling on the same close is optimistic and is rejected from candidate approval.

## Position Sizing

Backtest requests support:

- `all_in`: use the available quote balance on each entry
- `fixed_quote`: use `position_size_value` quote currency per entry, capped by available quote balance
- `percent_equity`: use `position_size_value` percent of current equity per entry, capped by available quote balance

Use `percent_equity` for realistic long historical tests. It prevents one losing strategy from compounding every signal with the full account balance.

## Supported Strategies

- `sma-crossover`: fast SMA crosses slow SMA.
- `rsi-mean-reversion`: RSI oversold/overbought signals.
- `btc-trend-pullback`: long-only trend pullback using pullback EMA, trend EMA, and RSI midline-cross momentum confirmation.

For `btc-trend-pullback`, `fast_period` is the pullback EMA period, `slow_period` is the trend EMA period, and `rsi_period` is the momentum confirmation period. It buys only after price recovers above the pullback EMA in a bullish EMA trend while RSI crosses from `<= 50` to `> 50`, the default ATR percent volatility filter is in range, and it exits when the trend is invalidated. Required candles include both the previous/current RSI windows and the default ATR filter window.

## SMA Research Filters

SMA backtests can optionally apply research filters before simulated execution:

- `trend_filter_enabled`: only allow SMA buy entries when the current close is above the configured trend SMA
- `trend_period`: the long SMA period used by the trend filter, commonly `200`
- `cooldown_bars`: minimum number of candles after any trade before another entry is allowed
- `min_holding_bars`: minimum number of candles after entry before a sell signal can close the position

These filters are backtest/research controls. They are intended to reduce signal churn before broader parameter sweeps.

## Strategy Quality Metrics

Backtest runs and comparison results include additional research-quality metrics:

- `expectancy`: average net P&L per completed round trip
- `trades_per_day`: total executions divided by backtest duration in days
- `churn_ratio`: percent of evaluated candles that generated an execution
- `sharpe_ratio`: annualized return-to-volatility ratio from equity-curve step returns
- `sortino_ratio`: annualized return-to-downside-volatility ratio from equity-curve step returns

These metrics are diagnostic. Read them together with profit factor, drawdown, excess return, train/test results, and walk-forward results.

## ATR Risk Exits

Backtest requests can enable ATR-based exits:

- `atr_exit_enabled`: enables ATR stop/target checks for open long positions
- `atr_period`: candle period used for average true range, default `14` when enabled
- `atr_stop_multiplier`: stop-loss distance below entry price, expressed as ATR multiple
- `atr_take_profit_multiplier`: take-profit distance above entry price, expressed as ATR multiple

At least one multiplier must be positive when ATR exits are enabled. ATR levels are fixed when the position opens. On later candles, the engine checks the candle low against the ATR stop first, then the candle high against the ATR target. This gives the backtest a basic risk-first exit model without adding a new strategy family.

## SMA Parameter Sweep

The optimizer endpoint runs a bounded SMA grid over the selected candle range and ranks the results. The dashboard uses this initial grid:

- fast periods: `5`, `10`, `15`, `20`
- slow periods: `50`, `100`, `200`

Invalid combinations where `slow <= fast` are skipped. Results are ranked by candidate status, excess return, profit factor, lower drawdown, and lower trade count.

Dashboard SMA sweeps enable train/test validation by default:

- train segment: first 70% of candles
- test segment: final 30% of candles

Each parameter set is evaluated on the full range, train segment, and test segment. This helps reject settings that only look good on one historical window.

Dashboard SMA sweeps also enable walk-forward validation by default:

- the selected candle range is split into 4 chronological folds
- each parameter set is tested independently on every fold
- results include average walk-forward return, average walk-forward excess return, worst fold drawdown, pass count, and walk-forward validation status

Walk-forward validation is stricter than a single train/test split because a parameter set must behave across multiple market windows.

Strategy comparisons can now use the same evidence fields:

```json
{
  "execution_fill_mode": "next_open",
  "train_test_enabled": true,
  "train_ratio": 0.7,
  "walk_forward_enabled": true,
  "walk_forward_folds": 4
}
```

When enabled, each comparison result includes execution fill mode, train/test validation status, and aggregate walk-forward status. Comparison ranking prefers walk-forward candidates first, then train/test candidates, then baseline candidate scoring.

## Benchmark

Each run includes a buy-and-hold benchmark using the same starting balance, fee rate, slippage rate, first candle close, and last candle close. The run also records excess return, which is:

`strategy_return_percent - benchmark_return_percent`

## Diagnostics

Each backtest run records:

- total return
- win rate
- maximum drawdown
- best and worst trade
- average win and average loss
- winning and losing completed trades
- profit factor
- average net P&L per completed trade
- average holding time
- buy-and-hold benchmark return
- excess return versus benchmark
- validation status and reason

Completed round trips are stored separately from raw buy/sell events. A round trip contains one entry and one exit, with net P&L, holding duration, and exit reason. Forced final liquidations use exit reason `end_of_backtest`.

Equity curve points are stored for each evaluated candle so drawdown and account growth can be inspected after the run.

For long interactive dashboard runs, the request sends `save_equity_curve: false`. The engine still calculates drawdown and summary metrics, but the API skips persisting per-candle equity points. This keeps multi-year backtests from spending most of their time inserting tens of thousands of rows.

API clients can omit `save_equity_curve` to preserve the original behavior, or set it explicitly:

- `save_equity_curve: true`: persist every evaluated equity point
- `save_equity_curve: false`: persist summary, trades, and round trips only

Backtest saves are transaction-backed when the repository is connected to a PostgreSQL pool. If a child insert fails, the run insert is rolled back with it.

## Stress Coverage

The automated stress suite includes a synthetic two-year `15m` run with 70,080 candles using `execution_fill_mode: "next_open"`. This protects the core engine from regressions on long historical ranges while keeping normal test runtime bounded.

Dirty candle data is rejected before simulation. API callers receive `candle_diagnostics` with counts for gaps, duplicates, out-of-order candles, unclosed candles, symbol/interval mismatches, and invalid OHLC rows.

## Validation

Backtest validation is intentionally conservative:

- fewer than 100 completed trades: `insufficient_sample`
- excess return <= 0: `underperforms_benchmark`
- profit factor <= 1.2: `weak_profit_factor`
- drawdown above 30%: `high_drawdown`
- average completed trade <= 0 after fees: `negative_average_trade`
- trade frequency above the interval limit: `overtrading`
- otherwise: `candidate`

A strategy marked `candidate` is not approved for live trading. It only means the backtest passed baseline diagnostics and deserves further review.

## Activation Gate

Activating a comparison winner into `strategy_settings` is blocked unless the selected result is a `candidate`, has positive excess return versus buy-and-hold, has maximum drawdown at or below 30%, and has completed trades.

Activation requires train/test and walk-forward evidence on the selected comparison result, and all of that evidence must have `candidate` validation status.

Activation also requires `execution_fill_mode = next_open`. Older comparison evidence that used `same_close` is blocked because it can be look-ahead biased.

`POST /api/v1/strategy/comparisons/{id}/activate` returns `409 Conflict` when this gate blocks activation. Blocked activations do not update strategy settings, activation history, or audit logs.

Default trade frequency limits are:

- `1m`: 20 trade events per day
- `5m`: 12 trade events per day
- `15m`: 4 trade events per day
- `1h`: 2 trade events per day

## Recommended Testing Order

For BTCUSDT historical testing:

1. Test `15m` first.
2. Test SMA with percent-of-equity sizing.
3. Add the SMA trend filter, cooldown, minimum holding bars, and ATR exits.
4. Run the SMA sweep.
5. Inspect profit factor, drawdown, excess return, trade count, and holding time.
6. Compare against RSI only after filtered SMA behavior is understood.
7. Use `1m` only after higher timeframe behavior looks reasonable.
