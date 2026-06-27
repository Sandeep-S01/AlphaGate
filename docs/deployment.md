# Deployment

Sentra currently deploys as separate Go commands from one container image:

- `api`: HTTP API, health/readiness checks, metrics, and read endpoints.
- `worker`: market-data collection, persistence, orchestration, risk, and paper execution workers controlled by environment flags.
- `migrate`: PostgreSQL migration runner.
- `backfill`, `strategy`, `risk`, `execution`: operational one-shot commands.

Live exchange order placement is not part of this deployment phase.

## Image

Build the image locally:

```powershell
docker build -t sentra:local .
```

The image contains compiled command binaries in `/app` and migration files in `/app/migrations`.

## Docker Compose

Start dependencies only:

```powershell
docker compose up -d postgres redis
```

Run migrations:

```powershell
docker compose --profile app run --rm migrate
```

Start the API and worker:

```powershell
docker compose --profile app up -d api worker
```

Check API health:

```powershell
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

Open the dashboard:

```text
http://localhost:8080/dashboard/
```

Stop application services:

```powershell
docker compose --profile app down
```

## Environment

Use `.env.production.example` as the production template. Required production decisions:

- Set `APP_ENV=production`.
- Set `AUTH_ENABLED=true`.
- Replace `ADMIN_API_KEY` with a strong random secret.
- Replace database credentials and do not commit the final `.env`.
- Keep `MARKET_DATA_ENABLED=false`, `ORCHESTRATION_ENABLED=false`, and `EXECUTION_ENABLED=false` unless the paper pipeline is intentionally enabled.
- Keep execution in paper mode; no live trading credentials are used in the current architecture.

## Migration Order

Apply migrations before starting long-running app services:

```powershell
docker compose --profile app run --rm migrate
docker compose --profile app up -d api worker
```

The migration runner records applied versions in `schema_migrations` and is safe to re-run.

## Health and Metrics

- `GET /health`: process health.
- `GET /ready`: PostgreSQL and Redis readiness.
- `GET /metrics`: in-process counters.
- `GET /api/v1/ops/pipeline-runs`: orchestration run history.
- `GET /api/v1/ops/streams`: Redis Streams consumer group state.
- `GET /api/v1/market/candles/coverage`: candle availability for a symbol, interval, and optional range.
- `GET /api/v1/strategy/settings`: current strategy settings.
- `PUT /api/v1/strategy/settings`: update persisted strategy settings for `sma-crossover`, `rsi-mean-reversion`, or `btc-trend-pullback`.
- `POST /api/v1/strategy/evaluate`: manually evaluate and save a signal.
- `POST /api/v1/backtests`: run an isolated historical backtest. Use `save_equity_curve: false` for long interactive runs when per-candle equity persistence is not needed.
- `GET /api/v1/backtests`: list backtest runs.
- `GET /api/v1/backtests/{id}`: read one backtest run and its simulated trades.
- `POST /api/v1/strategy/comparisons`: compare SMA and RSI over the same historical range.
- `GET /api/v1/strategy/comparisons`: list saved strategy comparisons.
- `GET /api/v1/strategy/comparisons/{id}`: read one comparison and its ranked results.
- `POST /api/v1/strategy/comparisons/{id}/activate`: activate a strategy from comparison evidence.
- `GET /api/v1/strategy/activations`: list strategy activation history.
- `GET /api/v1/risk/settings`: current persisted paper-risk settings.
- `PUT /api/v1/risk/settings`: update paper-risk limits and permissions.
- `GET /api/v1/safety/status`: current kill-switch status.
- `PUT /api/v1/safety/status`: enable or disable the kill switch.
- `GET /api/v1/audit/events`: list operator-visible audit events.
- `GET /api/v1/reports/paper/daily-pnl`: daily paper P&L summary.
- `GET /api/v1/reports/paper/trade-counts`: paper trade counts by day.
- `GET /api/v1/reports/risk/rejections`: rejected risk decisions by reason.
- `POST /api/v1/paper/account/reset`: reset the paper account assets and balances.
- `POST /api/v1/paper/cycles`: run one manual paper strategy/risk/execution cycle.

CSV exports are available by adding `format=csv` to:

- `GET /api/v1/audit/events`
- `GET /api/v1/paper/trades`
- `GET /api/v1/risk-decisions`

When authentication is enabled, send `X-API-Key` for `/api/v1/...` endpoints.
The dashboard shell is public, but its protected API calls still require the API key.
