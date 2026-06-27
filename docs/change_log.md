# Change Log

## 2026-06-20

- Started BTC Trend Pullback v2 slice 1.
- Updated strategy tests to require a fresh RSI midline cross for BTC trend pullback entries.
- Updated BTC trend pullback entry logic to require previous RSI <= 50 and current RSI > 50.
- Updated strategy settings and backtest required-candle calculations to include previous RSI history.
- Updated the BTC trend pullback backtest fixture to match the stricter entry confirmation.
- Updated comparison research-validation test data so walk-forward folds satisfy the stricter BTC trend pullback lookback.
- Updated backtesting, runbook, and README docs to describe BTC Trend Pullback v2 slice 1.
- Started BTC Trend Pullback v2 slice 2.
- Added optional ATR percent volatility-filter tests for acceptable, too-low, and too-high volatility.
- Added default BTC trend pullback ATR filter constants and enabled them through the strategy factory.
- Updated BTC trend pullback required-candle calculations to include default ATR filter history.
- Updated BTC trend pullback backtest fixture warmup candles so the entry setup occurs after ATR filter history is available.
- Updated strategy comparison and API test fixtures to provide enough candles for the default ATR filter lookback.
- Updated backtesting, runbook, and README docs to describe the default ATR volatility filter.
- Started research activation integrity hardening.
- Changed single backtest default execution timing to `next_open`.
- Added candidate validation rejection for `same_close` backtest evidence.
- Required train/test and walk-forward evidence before strategy activation.
- Updated backtest/API fixtures and docs to reflect stricter execution and activation gates.
- Started strategy-quality metrics slice.
- Added expectancy, trades/day, churn ratio, Sharpe ratio, and Sortino ratio to backtest runs.
- Propagated strategy-quality metrics into comparison and optimization results.
- Added migration `000023_strategy_quality_metrics` for saved backtests and comparison results.
- Updated dashboard Backtests result panel and BTC trend-pullback coverage calculation.
- Updated backtesting and README docs for the new metrics.
