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

## Checkpoint 3 - 2026-06-27 22:02 IST

Status: PASS with runtime configuration correction

Runtime:
- Docker Desktop restarted.
- Project Postgres container running: `sentra-postgres-1`
- Project Redis container recreated and reachable on `localhost:6379`: `sentra-redis-1`
- API process running: yes
- Worker process running: yes
- Dashboard/API URL: `http://localhost:8080`
- Execution mode: paper
- Paper execution enabled: true
- Exchange adapter: `binance_disabled`
- Live trading enabled: false

Configuration correction:
- Initial restart used `MARKET_DATA_INTERVAL=1m` while active strategy settings were `btc-trend-pullback` on `15m`.
- This produced fresh strategy signals labeled `15m` while consuming `1m` market-data events.
- Worker was restarted with `MARKET_DATA_INTERVAL=15m` and `EXECUTION_INTERVAL=15m` to match active strategy settings.

Market data:
- Redis `stream:market-data` length: `97`
- Latest persisted BTCUSDT `1m` candle: `2026-06-27 16:31:00 UTC`
- Latest persisted BTCUSDT `15m` candle: `2026-06-27 16:30:00 UTC`

Strategy/risk/execution:
- Active strategy settings: `btc-trend-pullback`, `BTCUSDT`, `15m`
- Latest real strategy signal: `hold`
- Latest real risk decision: `rejected`
- Latest real risk reason: `hold signal is not executable`
- No new natural paper order was created after restarting the real strategy path.
- Existing paper test orders remain filled from the forced RSI lifecycle test.

Safety and reconciliation:
- Reconciliation run: `7d533d48-e16b-4069-81ab-8b4dca86188a`
- Reconciliation status: `matched`
- Reconciliation mismatches: `0`

Next observation target:
- Continue paper proving on the corrected `15m` runtime path.
- Wait for a natural `buy` or `sell` signal from `btc-trend-pullback`, then verify risk, paper order lifecycle, trade persistence, account update, and reconciliation again.

## Checkpoint 4 - 2026-06-28 00:21 IST

Status: PASS with strategy strength-scaling fix

Runtime:
- API process rebuilt and restarted.
- Worker process rebuilt and restarted.
- Worker market-data interval: `15m`
- Dashboard/API URL: `http://localhost:8080`
- Execution mode: paper
- Paper execution enabled: true
- Exchange adapter: `binance_disabled`
- Live trading enabled: false

Defect found and fixed:
- `btc-trend-pullback` signal strength mixed a raw BTC price gap with RSI values.
- Evidence before fix: the same active strategy signal produced strength `1144.202386`, causing risk rejection against max strength `100`.
- Root cause: `abs(currentClose - currentPullback)` was measured in price units, so BTC-scale prices produced strength values in the thousands.
- Fix: normalize the price-gap component as a percentage of current close before adding RSI components.
- Regression: `TestBTCTrendPullbackStrengthIsPriceScaleInvariant` proves the same candle pattern has the same strength at low and BTC-like price scales.

Controlled active-strategy paper lifecycle test:
- Strategy: `btc-trend-pullback`, `BTCUSDT`, `15m`
- Trigger: `POST /api/v1/paper/cycles`
- Signal: `sell`
- Strength after fix: `35.289679`
- Risk decision: `approved`
- Risk reason: `approved by risk rules`
- Result: `executed`
- Order ID: `6bc0e09c-0647-4cc8-9716-f73a503299df`
- Order status: `filled`
- Order lifecycle events persisted: `created -> submitted -> filled`
- Trade ID: `acbe0ddc-25fb-4807-8eb7-78358b897694`
- Paper account after sell: `0.0016571242918733613 BTC`, `9899.9 USDT`
- Reconciliation run: `8429eb69-2f7e-4686-a6a6-4d4829a5f49d`
- Reconciliation status: `matched`
- Reconciliation mismatches: `0`

Notes:
- No risk thresholds were loosened.
- The fix makes strategy strength compatible with the existing risk threshold model instead of bypassing risk.
- Continue paper proving to observe the worker-driven path over multiple natural 15m candles.

## Checkpoint 5 - 2026-06-28 00:32 IST

Status: PASS

Runtime:
- API process running: yes
- Worker process running: yes
- Worker market-data interval: `15m`
- Dashboard/API URL: `http://localhost:8080`
- Execution mode: paper
- Paper execution enabled: true
- Exchange adapter: `binance_disabled`
- Live trading enabled: false

Natural worker-driven paper lifecycle:
- Observation window crossed the `00:30 IST` 15m boundary.
- Worker-generated strategy signal: `d810fe4d-6fc2-468b-af90-5d74b80399ad`
- Strategy: `btc-trend-pullback`, `BTCUSDT`, `15m`
- Signal: `sell`
- Strength: `35.011919`
- Risk decision: `approved`
- Risk reason: `approved by risk rules`
- Paper order ID: `ecc2f466-e1d6-4c6f-b5b3-44d376aec9ba`
- Paper order status: `filled`
- Order lifecycle events persisted: `created -> submitted -> filled`
- Trade ID: `f2303b2a-0844-4663-bb4a-d92357915749`
- Paper account after natural worker sell: `0.000006165398438465414 BTC`, `9999.8 USDT`
- Reconciliation run: `38dadda9-9585-4a39-9243-059879bd08d0`
- Reconciliation status: `matched`
- Reconciliation mismatches: `0`

Notes:
- This checkpoint proves the background worker path, not only the manual cycle path.
- Older pre-fix high-strength signals remain in history, but the new worker-generated signal used normalized strength below the max risk threshold.
- Continue observing for duplicate-order behavior and longer-run account/reconciliation stability across more 15m candles.
