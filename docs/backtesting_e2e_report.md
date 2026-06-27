# Backtesting E2E Micro-Agent Report

Generated: `2026-06-27T20:06:11Z`

Base URL: `http://127.0.0.1:8080`

Summary: **18 pass**, **0 warn**, **0 fail**, **0 blocked** out of **18** scenarios.

## Scenario Matrix

| Level | Scenario | Status | HTTP | Latency | Evidence |
| --- | --- | --- | --- | ---: | --- |
| L0-Min | Health endpoint returns OK | PASS | 200 | 3ms | {"service":"sentra","status":"ok","timestamp":"2026-06-27T20:06:11Z"} |
| L0-Min | Readiness endpoint returns OK | PASS | 200 | 1270ms | {"checks":{"candles":{"latency_ms":1262,"status":"ok"},"postgres":{"latency_ms":2,"status":"ok"},"redis":{"latency_ms":0,"status":"ok"},"signals":{"latency_ms":4,"status":"ok"}},"s... |
| L1-Easy | Template catalog returns all predefined strategies | PASS | 200 | 1ms | catalog contains 10 templates |
| L1-Easy | Market candle seed data is available | PASS | 200 | 17ms | 600 candles available |
| L1-Easy | Malformed JSON is rejected | PASS | 400 | 0ms | invalid JSON body |
| L1-Easy | Unknown strategy is rejected | PASS | 400 | 0ms | unsupported strategy_name "unknown-strategy" |
| L2-Medium | Data-blocked template is rejected | PASS | 400 | 0ms | strategy template "crypto-market-making" is not executable: support status is "blocked_by_data" |
| L2-Medium | Future empty range returns insufficient-candles error | PASS | 400 | 6ms | not enough candles for selected backtest range |
| L3-Hard | Native SMA backtest creates a run | PASS | 201 | 125ms | strategy=sma-crossover return=-0.20% trades=14 fees=1.40 roundTripCost=0.30% validation=insufficient_sample |
| L3-Hard | Native RSI backtest creates a run | PASS | 201 | 85ms | strategy=rsi-mean-reversion return=-0.02% trades=10 fees=1.00 roundTripCost=0.30% validation=insufficient_sample |
| L3-Hard | Native trend-pullback backtest creates a run | PASS | 201 | 31ms | strategy=btc-trend-pullback return=0.00% trades=0 fees=0.00 roundTripCost=0.30% validation=insufficient_sample |
| L4-Template | Executable template backtest creates a run: adaptive-mean-reversion | PASS | 201 | 37ms | strategy=adaptive-mean-reversion return=-0.07% trades=6 fees=0.60 roundTripCost=0.30% validation=insufficient_sample |
| L4-Template | Executable template backtest creates a run: momentum-breakout-volume | PASS | 201 | 118ms | strategy=momentum-breakout-volume return=-0.47% trades=36 fees=3.59 roundTripCost=0.30% validation=insufficient_sample |
| L4-Template | Executable template backtest creates a run: multi-factor-momentum | PASS | 201 | 61ms | strategy=multi-factor-momentum return=-0.17% trades=16 fees=1.60 roundTripCost=0.30% validation=insufficient_sample |
| L4-Template | Executable template backtest creates a run: trend-following-mtf | PASS | 201 | 106ms | strategy=trend-following-mtf return=-0.42% trades=32 fees=3.19 roundTripCost=0.30% validation=insufficient_sample |
| L5-Extreme | Oversized historical range is handled without server error | PASS | 400 | 0ms | backtest range is too large |
| L5-Extreme | Invalid economics are rejected | PASS | 400 | 0ms | starting_balance must be positive |
| L5-Extreme | Protected API rate limit eventually returns 429 | PASS | 429 | 48ms | received 429 after burst requests |

## Findings

- **MEDIUM / Strategy Templates:** 4 predefined templates are intentionally blocked by missing market-data/execution infrastructure. Recommendation: Keep them visible but disabled in backtesting until L2/order-flow/funding/pairs/grid state support exists.
