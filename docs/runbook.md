# Runbook

## Service Startup

1. Start Postgres and Redis.
2. Run `migrate`.
3. Start `api`.
4. Start `worker` with only the intended feature flags enabled.

For Docker Compose:

```powershell
docker compose up -d postgres redis
docker compose --profile app run --rm migrate
docker compose --profile app up -d api worker
```

### Production Deployment

For production deployments, follow these additional steps:

1. Use the production environment file:
   ```powershell
   copy .env.production.example .env
   ```
   Then edit `.env` to set:
   - `APP_ENV=production`
   - `AUTH_ENABLED=true`
   - Set strong values for `ADMIN_API_KEY`, `POSTGRES_DSN`, and `REDIS_PASSWORD`
   - Configure appropriate resource limits and timeouts

2. Build and deploy the Docker image:
   ```powershell
   docker build -t sentra:production .
   docker compose -f docker-compose.prod.yml up -d
   ```

3. Verify all services are healthy:
   ```powershell
   docker compose ps
   curl -H "X-API-Key: $ADMIN_API_KEY" http://localhost:8080/health
   curl -H "X-API-Key: $ADMIN_API_KEY" http://localhost:8080/ready
   ```

## Basic Checks

```powershell
docker compose ps
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8080/metrics
```

Dashboard:

```text
http://localhost:8080/dashboard/
```

Authenticated operational endpoints:

```powershell
curl -H "X-API-Key: $env:ADMIN_API_KEY" http://localhost:8080/api/v1/ops/streams
curl -H "X-API-Key: $env:ADMIN_API_KEY" http://localhost:8080/api/v1/ops/pipeline-runs
```

## Controlled Trading Readiness

Sentra is currently a production-grade paper trading system plus a controlled live-execution foundation. It is not an autonomous Binance live-trading system. Binance live order placement remains disabled by default and must not be treated as available unless the execution status endpoint explicitly reports live execution enabled.

Before any live-capable deployment can be considered ready, all gates below must be true:

- Strategy lifecycle for the active strategy/symbol/interval is `LIVE_ENABLED`.
- Kill switch is off.
- Latest reconciliation status is `matched`.
- Risk engine is enabled.
- Active symbol is present in `allowed_symbols`, or the whitelist is intentionally empty.
- Latest risk decision is not rejected.
- Execution adapter reports `live_trading_enabled = true`.
- Execution adapter is not `binance_disabled`.

Inspect these gates in the dashboard:

- Trading workspace, `Trade Readiness`: lifecycle state, kill switch, reconciliation status, risk limits, latest risk rejection, execution adapter, retry policy, and live-blocked reasons.
- Risk workspace: allowed symbols, max order quote amount, max position quote amount, max total exposure quote amount, max open positions, side permissions, daily limits, and cooldown.
- Ops workspace, `Run Reconciliation`: latest reconciliation severity and mismatch detail.
- Trading workspace, Orders tab: order lifecycle status, client/exchange order IDs, requested vs filled quantity, average fill price, failure reason, submitted time, and updated time.

Useful readiness API checks:

```powershell
curl http://localhost:8080/api/v1/strategy/lifecycle?symbol=BTCUSDT
curl http://localhost:8080/api/v1/safety/status
curl http://localhost:8080/api/v1/reconciliation/runs?limit=5
curl http://localhost:8080/api/v1/risk/settings
curl http://localhost:8080/api/v1/risk-decisions?symbol=BTCUSDT&limit=5
curl http://localhost:8080/api/v1/execution/status
curl http://localhost:8080/api/v1/paper/orders?symbol=BTCUSDT&limit=20
```

Expected default execution status:

```json
{
  "mode": "paper",
  "exchange_adapter": "binance_disabled",
  "live_trading_enabled": false
}
```

If any readiness gate is blocked, do not advance to live. Fix the root cause, rerun reconciliation, and verify the audit trail before changing lifecycle state.

## Common Failures

`/ready` fails:

- Verify `POSTGRES_DSN` points to the running database host.
- Verify `REDIS_ADDR` points to the running Redis host.
- Check container logs with `docker compose logs api`.

API exits in production:

- `APP_ENV=production` requires `AUTH_ENABLED=true`.
- `ADMIN_API_KEY` must be configured for protected routes to be usable.

No candles are persisted:

- Confirm `MARKET_DATA_PERSISTENCE_ENABLED=true`.
- Confirm `MARKET_DATA_REDIS_STREAM` matches the producer stream.
- Check `GET /api/v1/ops/streams` for pending Redis messages.
- Binance websocket subscription failures are expected to retry up to the configured reconnect limit.
- Closed websocket candle streams are expected to reconnect up to the configured reconnect limit.
- Temporary Redis publish failures are retried for the same candle before the collector returns an error.
- Redis stream read interruptions are expected to retry in the worker consumer instead of stopping immediately.

No paper trades are created:

- Confirm `ORCHESTRATION_ENABLED=true`.
- Confirm `RISK_ENABLED=true`.
- Confirm `EXECUTION_ENABLED=true`.
- Check recent risk decisions and pipeline runs.
- Use `POST /api/v1/paper/cycles` from the dashboard Ops tab to run one paper cycle manually and inspect the returned signal/risk/execution result.
- Duplicate closed-candle events should be skipped by idempotency records. Idempotency keys include exchange, symbol, interval, and candle open time.
- Failed pipeline attempts should be marked failed and allowed to retry.

Strategy evaluate fails:

- Confirm migrations `000006_strategy_settings`, `000011_strategy_rsi`, and `000019_btc_trend_pullback_strategy` are applied.
- Confirm enough candles exist for the configured `lookback_limit`.
- For `sma-crossover`, confirm `slow_period` is greater than `fast_period`.
- For `rsi-mean-reversion`, confirm `rsi_oversold` is below `rsi_overbought`.
- For `btc-trend-pullback`, confirm `slow_period` is greater than `fast_period` and `lookback_limit` covers `max(slow_period + 1, rsi_period + 2, default_atr_period + 1)`.
- Confirm `lookback_limit` covers the selected strategy requirement.

Backtest fails:

- Confirm migration `000007_backtests` is applied.
- Confirm migration `000008_backtest_reporting` is applied.
- Confirm migration `000011_strategy_rsi` is applied.
- Confirm candles exist for the selected symbol, interval, and time range.
- For `sma-crossover`, confirm `slow_period` is greater than `fast_period`.
- For `rsi-mean-reversion`, confirm `rsi_oversold` is below `rsi_overbought`.
- For `btc-trend-pullback`, confirm `slow_period` is greater than `fast_period` and enough candles exist for `max(slow_period + 1, rsi_period + 2, default_atr_period + 1)`.
- Use `cmd/backfill` first if the selected historical range has no candles.
- Backtests are isolated and should not create paper orders, paper trades, or Redis events.

Strategy comparison fails:

- Confirm migration `000012_strategy_comparisons` is applied.
- Confirm candles exist for the selected symbol, interval, and time range.
- Confirm the range has enough candles for both SMA and RSI settings.
- Comparisons are isolated and should not create paper orders, paper trades, or Redis events.

Strategy activation fails:

- Confirm migration `000013_strategy_activations` is applied.
- Confirm the comparison exists and includes ranked results.
- Confirm the comparison is recent enough to be accepted as activation evidence.
- If the API returns `409 Conflict`, the activation gate blocked the winner. The selected result must have `validation_status = candidate`, positive excess return versus buy-and-hold, drawdown at or below 30%, and completed trades.
- Confirm the selected result includes train/test and walk-forward evidence, with `train_validation_status`, `test_validation_status`, and `walk_forward_validation_status` all set to `candidate`.
- Confirm the selected comparison result has `execution_fill_mode = next_open`. Old `same_close` evidence must be rerun before activation.
- Check `GET /api/v1/strategy/activations` and audit events after activation.

Risk rejects expected paper trades:

- Confirm migration `000009_risk_settings` is applied.
- Check `GET /api/v1/risk/settings` for side permissions, minimum signal strength, quote-size cap, per-order cap, position cap, total exposure cap, open-position cap, allowed symbols, daily trade cap, daily loss cap, and cooldown.
- Set a limit to `0` to disable that cap.
- Check recent `risk_decisions` snapshots to see the active rule values used for each decision.

Reset paper account:

```powershell
$body = @{
  base_asset = "BTC"
  quote_asset = "USDT"
  base_balance = 0
  quote_balance = 10000
} | ConvertTo-Json
curl -Method POST -ContentType "application/json" -Body $body http://localhost:8080/api/v1/paper/account/reset
```

Run one manual paper cycle:

```powershell
$body = @{ symbol = "BTCUSDT"; interval = "1m" } | ConvertTo-Json
curl -Method POST -ContentType "application/json" -Body $body http://localhost:8080/api/v1/paper/cycles
```

## Failure Simulation Coverage

Automated tests cover duplicate closed-candle recovery:

- completed duplicate candle events are ignored and do not create extra executions
- failed pipeline attempts can retry after the dependency recovers
- idempotency keys separate different exchanges for the same symbol, interval, and open time

Automated tests cover market-data dependency failures:

- Binance websocket subscription unavailable, then recovery
- websocket subscription closes, then reconnects
- temporary Redis publish failure, then same-candle retry
- persistent Redis publish failure returns an error
- historical Binance backfill fetch failure retries before marking a job failed

Automated tests cover PostgreSQL disconnect handling at shared persistence boundaries:

- candle writes, batch candle writes, candle deletes, candle reads, coverage reads, and open-time reads return wrapped operation errors
- paper order, paper trade, and paper account persistence return wrapped operation errors
- failed candle backfill writes mark the backfill job as failed with the last error
- failed orchestration writes mark the pipeline run as failed and allow retry after recovery

Manual cycles are paper-only. They require enough candles for the persisted strategy settings and still apply persisted risk settings before creating paper orders/trades.

## Audit And Reports

Filter audit events:

```powershell
curl "http://localhost:8080/api/v1/audit/events?event_type=risk.settings_changed&actor=operator&limit=50"
```

Export CSV:

```powershell
curl "http://localhost:8080/api/v1/audit/events?format=csv" -o audit-events.csv
curl "http://localhost:8080/api/v1/paper/trades?symbol=BTCUSDT&format=csv" -o paper-trades.csv
curl "http://localhost:8080/api/v1/risk-decisions?symbol=BTCUSDT&format=csv" -o risk-decisions.csv
```

Reports:

```powershell
curl "http://localhost:8080/api/v1/reports/paper/daily-pnl?symbol=BTCUSDT&limit=30"
curl "http://localhost:8080/api/v1/reports/paper/trade-counts?symbol=BTCUSDT&limit=30"
curl "http://localhost:8080/api/v1/reports/risk/rejections?symbol=BTCUSDT&limit=30"
```

## Emergency Stop

Enable the kill switch:

```powershell
$body = @{
  kill_switch_active = $true
  reason = "operator emergency stop"
  updated_by = "operator"
} | ConvertTo-Json
curl -Method PUT -ContentType "application/json" -Body $body http://localhost:8080/api/v1/safety/status
```

Verify status and audit:

```powershell
curl http://localhost:8080/api/v1/safety/status
curl http://localhost:8080/api/v1/audit/events?event_type=safety.status_changed
```

Recovery:

```powershell
$body = @{
  kill_switch_active = $false
  reason = "paper execution approved to resume"
  updated_by = "operator"
} | ConvertTo-Json
curl -Method PUT -ContentType "application/json" -Body $body http://localhost:8080/api/v1/safety/status
```

When the kill switch is active, strategy signals and risk decisions may still be recorded, but paper order/trade creation is blocked.

Critical reconciliation mismatches can arm the kill switch automatically. After any emergency stop:

1. Keep the kill switch armed.
2. Run reconciliation from the Ops workspace or `POST /api/v1/reconciliation/runs`.
3. Review mismatch severity and details in the Ops workspace.
4. Inspect failed or partial orders in the Trading workspace Orders tab.
5. Review audit events for `reconciliation.critical_mismatch` and `safety.status_changed`.
6. Only disarm the kill switch after the mismatch is explained and the latest reconciliation is `matched`.

Check candle coverage:

```powershell
curl "http://localhost:8080/api/v1/market/candles/coverage?symbol=BTCUSDT&interval=1m"
```

## Backfill

Run historical candle backfill as a one-shot container:

```powershell
docker compose --profile app run --rm backfill -symbol BTCUSDT -interval 1m -from 2026-06-16T00:00:00Z -to 2026-06-16T01:00:00Z
```

## Failure Simulation Coverage

Run the reliability test suite:

```powershell
go test ./internal/marketdata ./internal/platform/streams ./internal/orchestration
```

Covered simulations:

- Binance subscription unavailable then recovered.
- WebSocket subscription closes and reconnects.
- Redis publish unavailable.
- Redis stream read unavailable then recovered.
- PostgreSQL write failure in the paper pipeline.
- Duplicate candle event handling.
- Retry succeeds after a failed pipeline attempt.

Run full readiness verification before release:

```powershell
go test ./...
cd dashboard-src
npm run lint
npm run build
npx playwright test --project=chromium
```

## Logs

All services write structured JSON logs to stdout/stderr. Use:

```powershell
docker compose logs -f api
docker compose logs -f worker
```

## Monitoring and Observability

Sentra provides multiple ways to monitor system health and performance:

### Health Checks
- `GET /health` - Basic service health
- `GET /ready` - Dependency readiness (PostgreSQL, Redis)
- `GET /metrics` - Prometheus-compatible metrics

### Key Metrics to Monitor
- **HTTP Request Rates**: `http_requests_total` by endpoint and status code
- **Error Rates**: Increase in 5xx responses or pipeline failures
- **Latency**: API response times and database query durations
- **Pipeline Success Rate**: Ratio of successful vs failed pipeline runs
- **Database Connection Pool**: Active/idle connections and wait times
- **Redis Stream Lag**: Pending messages in consumer groups

### Alerting Guidelines
Set up alerts for:
- Service downtime (health check failures)
- High error rates (>5% error rate for 5 minutes)
- Elevated latency (95th percentile > 1s for API endpoints)
- Pipeline failure rate (>10% failure rate)
- Database connection exhaustion (>90% pool utilization)
- Redis stream backlog growth (>1000 pending messages)

### Log Aggregation
All services emit structured JSON logs to stdout/stderr. In production, configure log collection to:
- Aggregate logs from all containers
- Index logs for searching and analysis
- Set up alerts on error patterns
- Retain logs according to compliance requirements

## Backup and Disaster Recovery

### Backup Strategy
PostgreSQL stores durable business state and should be backed up regularly. Redis Streams contain transient operational data and do not require backup for business continuity.

Minimum production backup scope:

- **PostgreSQL database dumps**: Daily logical backups, hourly WAL archives
- **Compose/env configuration**: Version-controlled environment files
- **Application image tag or digest**: Record of deployed versions
- **Backup retention**: Maintain 30 days of daily backups, 12 months of monthly backups

### Backup Procedures
1. PostgreSQL logical backup:
   ```bash
   pg_dump -U sentra -Fc sentra > sentra-backup-$(date +%Y%m%d).dump
   ```

2. Verify backup integrity:
   ```bash
   pg_restore -l sentra-backup-$(date +%Y%m%d).dump > backup-contents.txt
   ```

### Disaster Recovery
To restore service after a catastrophic failure:

1. Provision new infrastructure (database, redis, compute)
2. Restore PostgreSQL from latest backup:
   ```bash
   pg_restore -U sentra -d sentra sentra-backup-latest.dump
   ```
3. Apply any pending migrations:
   ```bash
   go run ./cmd/migrate
   ```
4. Restore configuration and redeploy application
5. Verify service health:
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/ready
   ```

Note: Redis Streams will be rebuilt as new market data flows through the system.

## Performance Tuning

### Database Optimization
- Monitor connection pool utilization via `/metrics`
- Adjust `POSTGRES_MAX_CONNS` and `POSTGRES_MIN_CONNS` based on load
- Enable query performance monitoring in PostgreSQL
- Consider read replicas for high-volume query workloads

### Redis Optimization
- Monitor memory usage and eviction policies
- Adjust stream retention periods based on replay requirements
- Consider Redis clustering for high-throughput scenarios

### Application Tuning
- Increase `LOOKBACK_LIMIT` only when needed for strategy evaluation
- Tune worker consumer groups based on message volume
- Adjust batch sizes in backfill operations for optimal throughput
- Monitor garbage collection pauses in Go runtime

## Security Operations

### Certificate Management
If terminating SSL at the application level:
- Rotate certificates before expiration
- Use automated certificate management (Let's Encrypt) when possible
- Test certificate renewal procedures regularly

### Secret Rotation
- Rotate `ADMIN_API_KEY` periodically (every 90 days recommended)
- Rotate database credentials following organizational policies
- Update secrets in all deployment environments
- Audit service restarts after secret rotation

### Vulnerability Management
- Keep base images and dependencies updated
- Monitor for security advisories in used libraries (Go, PostgreSQL, Redis)
- Conduct periodic dependency scanning
- Apply security patches within recommended timeframes
