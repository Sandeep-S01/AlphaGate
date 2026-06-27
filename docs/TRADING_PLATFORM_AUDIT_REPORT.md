# Sentra Trading Platform Audit Report

Date: 2026-06-20

## Executive Summary

Sentra has a solid application foundation for a research/paper-trading system: Go services, PostgreSQL persistence, Redis stream infrastructure, Binance market data ingestion, backfill/aggregation, paper execution, safety controls, API middleware, audit logs, and a dashboard.

The weak area is not project structure. The weak area is the quant research loop. Current strategies are simple prototypes, and the backtest engine still has realism gaps that can make results unreliable for strategy approval. The platform should not be used for live trading decisions yet.

Primary recommendation: fix the backtesting engine realism and data-quality gates first, then improve strategy research. Building more strategies before the engine is trustworthy will produce misleading results.

## Current Architecture Map

Core boundaries:

- `cmd/api`: HTTP API and dashboard server.
- `cmd/worker`: stream-driven market pipeline.
- `cmd/backfill`: historical candle backfill CLI.
- `cmd/migrate`: database migration runner.
- `internal/marketdata`: Binance candle storage, backfill, gap detection, timeframe aggregation.
- `internal/strategy`: signal evaluators for SMA crossover, RSI mean reversion, BTC trend pullback.
- `internal/backtest`: backtest simulation, metrics, comparison, SMA optimization.
- `internal/risk`: signal approval/rejection rules.
- `internal/execution`: paper order/trade execution and account updates.
- `internal/orchestration`: candle-to-signal-to-risk-to-paper-execution pipeline.
- `internal/safety`: operator kill switch.
- `internal/api`: REST API, dashboard endpoints, middleware, auth/rate limit/body limit.

## P0 Problems

### P0.1 Backtest Fill Timing Is Not Realistic

The engine evaluates a signal using candles through the current closed candle and then fills entry/exit on that same candle close. This creates same-bar execution bias. A real system only knows a candle after it closes, so the earliest realistic fill is normally the next candle open, next candle close, or an explicitly modeled delayed market order.

Impact: strategy results can be materially wrong, especially for fast intervals like `1m` and `15m`.

Fix: introduce execution timing modes, default to `next_open`, and keep `same_close` only as a research/debug mode.

### P0.2 Backtest Data Quality Is Not a Hard Engine Gate

The system has candle coverage and gap detection, but the backtest engine accepts whatever candle slice it receives. The API checks minimum candle count, but that is not enough to catch gaps, duplicates, non-monotonic timestamps, unclosed candles, or malformed OHLC values.

Impact: sparse or dirty ranges can generate false negative or false positive strategy results.

Fix: add a `ValidateCandleSeries` gate before simulation and return structured diagnostics.

### P0.3 Backtest Save Is Not Transactional

`backtest.Repository.SaveWithOptions` inserts the run, trades, round trips, and equity points sequentially without a database transaction.

Impact: if saving fails midway, the database can contain a partial backtest run.

Fix: add transaction-backed save support for backtest persistence.

### P0.4 Strategy Activation Is Too Close To Research Output

The app can save activations from strategy comparisons, but current strategy validation is still based on a single-run score. It does not enforce walk-forward success, out-of-sample pass rate, data-quality pass, execution-realism mode, or risk acceptance.

Impact: a weak or overfit strategy can be promoted too easily.

Fix: add activation gates: `candidate` status, clean data diagnostics, realistic fill mode, positive out-of-sample excess return, acceptable drawdown, and explicit operator approval.

## P1 Problems

### P1.1 Strategy Layer Is Too Basic

Current strategies:

- `sma-crossover`: basic trend-following crossover, prone to chop and overtrading.
- `rsi-mean-reversion`: basic threshold strategy without regime, volatility, or trend filters.
- `btc-trend-pullback`: better direction, but still early-stage and not optimized with walk-forward tooling.

Impact: the negative backtests are expected. The current strategies are baseline examples, not mature BTC trading systems.

Fix: keep these as benchmarks, then add a proper BTC strategy research module with regime filters, ATR exits, volatility sizing, cooldown, and walk-forward optimization.

### P1.2 Optimizer Is SMA-Only

The optimizer supports SMA grids, train/test, and walk-forward fields, but it does not optimize RSI or BTC trend-pullback parameters.

Impact: the strongest current strategy candidate cannot be researched properly from the UI/API.

Fix: add strategy-specific optimization requests and reusable parameter-grid support.

### P1.3 Metrics Are Still Incomplete

The engine calculates return, P&L, win rate, drawdown, profit factor, average trade, benchmark return, and excess return. Missing useful research metrics include Sharpe, Sortino, Calmar, expectancy, exposure time, recovery factor, consecutive losses, monthly returns, and trade distribution.

Impact: candidate selection is under-informed.

Fix: add a `performance` package or backtest metrics module and persist the most important metrics.

### P1.4 Risk Layer Is Rule-Based But Not Portfolio-Aware

Risk checks max quote amount, daily trades, daily loss, signal strength, buy/sell permissions, and cooldown. It does not model volatility-based sizing, max open exposure, consecutive losses, strategy-level drawdown, or exchange constraints.

Impact: paper trading safety is basic; live readiness is not there.

Fix: add portfolio/risk context and exchange constraints before any live execution work.

### P1.5 Paper Execution Is Simplified

Paper execution fills immediately at a price and fixed quote amount. It does not model spread, order types, liquidity, partial fills, exchange filters, or latency.

Impact: paper results can diverge from exchange behavior.

Fix: add execution model configuration and exchange symbol filters.

## P2 Problems

- Dashboard needs clearer parameter presets and warnings for low sample size, dirty data, and overtrading.
- Backtest and optimizer should become async jobs for large two-year runs.
- Candle storage will eventually need retention/partitioning/index review as symbols and intervals grow.
- API contracts should be documented with request/response examples.
- CI should run Go tests, JS syntax checks, and migration checks.
- Binaries under `bin/` should stay generated artifacts, not source-of-truth deliverables.

## Strategy Assessment

Do not judge the platform by the current strategy P&L alone. Current negative results mainly show that the strategy ideas are too naive and trading too frequently after fees.

Recommended classification:

- SMA crossover: keep as benchmark only.
- RSI mean reversion: keep as benchmark only.
- BTC trend pullback: keep as first serious research candidate, but do not activate until tested with realistic fills and walk-forward validation.

Next strategy direction:

1. BTC trend-pullback v2 on `15m` and `1h`.
2. Regime filter using higher-timeframe EMA or market structure.
3. ATR stop and target.
4. Percent-equity sizing, default 2-10%.
5. Minimum holding and cooldown.
6. Walk-forward validation across multiple BTC regimes.

## Backtesting Assessment

Current status: usable for rough research, not for production approval.

What is good:

- Fees and slippage exist.
- Percent-equity and fixed-quote sizing exist.
- ATR exits exist.
- Buy-and-hold benchmark exists.
- Equity curve can be skipped for performance.
- Backfill and aggregation exist.

What must improve:

- Signal bar and fill bar must be separated.
- Data quality must be validated inside the backtest path.
- Long runs should use async job execution.
- Persistence should be transactional.
- Strategy-specific optimization must be added.
- Performance metrics must be expanded.

## Database Assessment

The schema is appropriate for the current stage: candles, signals, risk decisions, paper execution, backtests, diagnostics, comparisons, activations, backfill jobs, safety, and audit.

Required improvements:

- Transactional save for backtest runs and child rows.
- Add backtest data-quality diagnostics table or JSON column.
- Add execution-model fields to `backtest_runs`: fill mode, spread model, latency model.
- Add strategy optimization result storage if optimizer results need history.
- Consider partitioning candles by interval/time later.

## API Assessment

Current API shape is reasonable and modular. Key endpoints exist for candles, backfills, aggregation, strategy settings, backtests, optimizations, comparisons, activations, risk settings, safety, reports, and paper execution.

Required API changes:

- Add `execution_fill_mode` to backtest requests.
- Add backtest data-quality diagnostics to response.
- Add async backtest job endpoints for large runs.
- Add strategy-specific optimization endpoint or generic parameter-grid schema.
- Add activation gate response explaining why activation is blocked.

## Testing Strategy

Immediate required tests:

- Backtest next-candle execution red/green tests.
- Candle series validation tests for gaps, duplicates, ordering, unclosed candles, and invalid OHLC.
- Transactional save failure test.
- Strategy activation gate tests.
- BTC trend-pullback optimizer tests.

Regression tests to keep:

- Existing strategy evaluator tests.
- Existing backtest sizing, ATR, benchmark, validation tests.
- Existing market aggregation and backfill tests.
- API middleware/security tests.

## Deployment And Operations

Current deployment is suitable for local/dev paper trading with Docker Compose.

Before production:

- Add CI pipeline.
- Add migration validation in CI.
- Add environment-based production safety defaults.
- Add structured operational runbooks for backfill, aggregation, paper reset, kill switch, and failed worker recovery.
- Add failure simulation tests for Binance unavailable, Redis down, DB down, websocket interruption, and duplicate events.
- Add exchange reconciliation only when live execution is introduced.

## 30-Day Roadmap

### Week 1: Backtest Correctness

1. Add candle-series validator.
2. Add next-candle execution mode.
3. Add execution mode fields to API/run output.
4. Add transactional backtest saves.

### Week 2: Research Quality

1. Add strategy-specific optimizer support.
2. Add BTC trend-pullback parameter sweep.
3. Add Sharpe, Sortino, Calmar, expectancy, and exposure metrics.
4. Add train/test and walk-forward reporting to the dashboard.

### Week 3: Strategy Improvement

1. Build BTC trend-pullback v2.
2. Add higher-timeframe regime filter.
3. Add volatility-based position sizing.
4. Add parameter presets for `15m` and `1h`.

### Week 4: Production Safety

1. Add strict activation gates.
2. Add failure simulation tests.
3. Add async backtest jobs.
4. Add operational runbooks and CI checks.

## Final Recommendation

Proceed in this order:

1. Backtest engine realism and data-quality gate.
2. Transactional persistence.
3. Strategy-specific optimizer for BTC trend-pullback.
4. Better performance metrics.
5. Strategy activation gates.
6. New strategy research.

Do not build live trading or exchange reconciliation yet. Those are required later, but only after the research and backtesting foundation is trustworthy.
