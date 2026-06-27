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

## Checkpoint 6 - 2026-06-28 01:02 IST

Status: PASS with failed-execution resilience fix

Defect found:
- At the `00:45 IST` 15m boundary, the worker generated a normalized `sell` signal and risk approved it.
- The paper account no longer had enough BTC for another `$100` sell.
- Paper execution returned `insufficient base balance`.
- Before the fix, this error stopped the paper orchestration worker and did not persist a failed order.

Fix:
- Orchestrator now records a failed paper order when paper execution rejects an approved risk decision.
- Failed order events are persisted as `created -> failed`.
- Execution repository skips trade insertion for failed orders.
- The pipeline completes after recording the failed execution, so the worker remains alive and the event is auditable.

Verification:
- Tests added:
  - `TestOrchestratorRecordsFailedOrderWhenPaperExecutionRejects`
  - `TestRepositorySaveSkipsTradeForFailedOrder`
- Full verification: `go test ./...`
- Worker rebuilt and restarted on the corrected `15m` paper path.

Natural worker retry evidence:
- Observation crossed the `01:00 IST` 15m boundary.
- Worker process remained running after the execution failure.
- Worker-generated signal: `e33e7936-3a54-4305-a17e-879200104409`
- Signal: `sell`
- Strength: `35.891964`
- Risk decision: `approved`
- Failed order ID: `953e3785-02fe-4c12-b50c-932e8a316eab`
- Failed order reason: `insufficient base balance`
- Failed order lifecycle events persisted: `created -> failed`
- No trade row was inserted for the failed order.
- Paper account remained unchanged: `0.000006165398438465414 BTC`, `9999.8 USDT`
- Reconciliation run: `9da9d686-645e-4788-bf32-40cfd1ca1e66`
- Reconciliation status: `matched`
- Reconciliation mismatches: `0`

Safety:
- Execution mode: paper
- Exchange adapter: `binance_disabled`
- Live trading enabled: false

## Checkpoint 8 - 2026-06-28 01:27 IST

Status: PASS with symmetric buy-side balance risk guard

Defect addressed:
- Risk had a pre-execution balance guard for `sell` orders, but not for `buy` orders.
- A `buy` signal with insufficient USDT could still reach paper execution and fail there.

Fix:
- Risk context now includes paper quote balance.
- Risk evaluator rejects `buy` signals when `quote_balance < quote_order_amount`.
- Orchestrator and manual paper runner pass quote balance into risk before execution.

Verification:
- Tests added:
  - `TestEvaluatorRejectsBuyWhenQuoteBalanceIsInsufficient`
  - `TestOrchestratorRejectsBuyBeforeExecutionWhenQuoteBalanceIsInsufficient`
  - `TestManualRunnerRejectsBuyBeforeExecutionWhenQuoteBalanceIsInsufficient`
- Full verification: `go test ./...`

Runtime evidence:
- API and worker were rebuilt and restarted with the updated risk guard.
- Execution mode: paper
- Exchange adapter: `binance_disabled`
- Live trading enabled: false
- Paper account: `0.000006165398438465414 BTC`, `9999.8 USDT`
- Manual paper cycle on `BTCUSDT` `15m` returned `risk_rejected`.
- Runtime signal side: `sell`
- Runtime risk reason: `base balance 0.000006 is below required 0.001662`
- No new order was created after the previous failed order `953e3785-02fe-4c12-b50c-932e8a316eab`.
- Reconciliation run: `d16b6dce-b575-44c6-9ace-233759f827b1`
- Reconciliation status: `matched`
- Reconciliation mismatches: `0`

Notes:
- The active paper account currently has enough USDT, so the insufficient-quote runtime case was not forced by mutating account state.
- The buy-side guard is covered by focused risk, orchestrator, and manual-runner tests.

## Checkpoint 7 - 2026-06-28 01:13 IST

Status: PASS with pre-execution balance risk guard

Defect addressed:
- Risk approved a `sell` when the paper account did not have enough BTC to satisfy the configured `$100` quote order.
- Execution could record the failure, but the stronger control is to reject the order before execution.

Fix:
- Risk context now includes latest price and paper base balance.
- Risk evaluator rejects `sell` signals when `base_balance < quote_order_amount / latest_price`.
- Orchestrator and manual paper runner now read latest price/account before risk evaluation for executable buy/sell signals.
- Approved decisions reuse the same price/account for paper execution.

Verification:
- Tests added:
  - `TestEvaluatorRejectsSellWhenBaseBalanceIsInsufficient`
  - `TestOrchestratorRejectsSellBeforeExecutionWhenBaseBalanceIsInsufficient`
  - `TestManualRunnerRejectsSellBeforeExecutionWhenBaseBalanceIsInsufficient`
- Full verification: `go test ./...`

Runtime evidence:
- API process running: yes
- Worker process running: yes
- Worker market-data interval: `15m`
- Latest risk decision: `ed71974f-ed5c-4d45-b8d0-98c5add5f00f`
- Signal side: `sell`
- Risk decision: `rejected`
- Risk reason: `base balance 0.000006 is below required 0.001662`
- No new order was created after the previous failed order `953e3785-02fe-4c12-b50c-932e8a316eab`.
- Paper account remained unchanged: `0.000006165398438465414 BTC`, `9999.8 USDT`
- Reconciliation run: `456d4df2-dabc-4b0b-9136-2204d222a8ff`
- Reconciliation status: `matched`
- Reconciliation mismatches: `0`

Safety:
- Execution mode: paper
- Exchange adapter: `binance_disabled`
- Live trading enabled: false
