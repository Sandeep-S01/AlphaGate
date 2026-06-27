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
