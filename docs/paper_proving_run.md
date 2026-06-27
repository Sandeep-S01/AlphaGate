# AlphaGate Paper Proving Run

## Checkpoint 1 - 2026-06-27 14:33 IST

Status: PASS

Runtime:
- API process running: yes
- Worker process running: yes
- Dashboard URL: `http://localhost:8080`
- Postgres container: `sentra-postgres-1`
- Redis source: existing `smartsearch_redis` on `localhost:6379`

Safety:
- Execution mode: paper
- Paper execution enabled: true
- Exchange adapter: `binance_disabled`
- Live trading enabled: false
- Dashboard live state: `LIVE BLOCKED`
- Reconciliation status: matched
- Reconciliation mismatches: 0

Market data:
- `stream:market-data` length: 373
- BTCUSDT 1m candles: 1,061,289
- Latest persisted candle: `2026-06-27 09:03:00 UTC`

Strategy/risk/execution:
- Signals: 17
- Risk decisions: 15
- Latest risk decision: rejected
- Latest risk reason: `hold signal is not executable`
- Paper orders: 0

Notes:
- Real-time Binance websocket publishing is now working after fixing numeric kline decimal parsing.
- Current strategy output is producing hold signals, so no paper orders have been created.
- Continue observation for at least 24 hours before evaluating whether paper execution behavior is stable enough for longer proving.

## Checkpoint 2 - 2026-06-27 14:54 IST

Status: PASS with one fixed defect

Runtime:
- API process restarted with patched manual paper-cycle code.
- Dashboard/API URL: `http://localhost:8080`
- Execution mode: paper
- Paper execution enabled: true
- Exchange adapter: `binance_disabled`
- Live trading enabled: false

Controlled paper lifecycle test:
- Temporary strategy profile: `rsi-mean-reversion`, `BTCUSDT`, `1m`, RSI period `14`, oversold `98`, overbought `99`
- Trigger: `POST /api/v1/paper/cycles`
- Result: `executed`
- Signal: `buy`, strength `17.761124`, reason `RSI is oversold`
- Signal timestamp after fix: `2026-06-27 14:41:59.999 IST`
- Risk decision: `approved`, reason `approved by risk rules`
- Order ID: `34504750-1db0-4313-97be-f2f0d1dc3c53`
- Order status: `filled`
- Order lifecycle events persisted: `created -> submitted -> filled`
- Trade ID: `d110a652-4130-4168-bf57-792585c8b170`
- Paper account after test: `0.0033091517921839423 BTC`, `9800 USDT`
- Reconciliation run: `b0e5f31b-7d0e-40c5-9ffa-af1bb384277c`, status `matched`, mismatches `0`

Defect found and fixed:
- Manual paper-cycle evaluation previously requested candle rows in ascending order with a `LIMIT`, which selected the oldest candles in the table.
- Evidence: first controlled cycle generated a signal from `2024-06-18` even though `BTCUSDT` 1m data existed through `2026-06-27`.
- Fix: manual runner now requests newest candles first (`Desc: true`) and reverses them into chronological order before strategy evaluation, matching the streaming orchestrator path.
- Regression: `TestManualRunnerRunsOnePaperCycle` now asserts newest-first candle reads and validates signal generation uses the newest candle.

Restoration:
- Active strategy settings restored to `btc-trend-pullback`, `BTCUSDT`, `15m`, fast `21`, slow `200`, lookback `201`, RSI `14/30/70`.
