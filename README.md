# Sentra

Automated crypto trading platform foundation. The current foundation contains project bootstrap, configuration, PostgreSQL and Redis connectivity, logging, Docker Compose dependencies, database migrations, public Binance market data collection, Redis Streams event envelopes, candle persistence infrastructure, historical candle backfill, candle query APIs, SMA crossover and RSI mean-reversion strategy runners, a configurable risk decision engine, paper trading execution, and event-driven paper-pipeline orchestration.

Live trading is intentionally not implemented yet.

## Requirements

- Go 1.23+
- Docker and Docker Compose

## API Documentation

Sentra provides a comprehensive RESTful API for interacting with the trading platform. The API is versioned under `/api/v1/` and includes endpoints for market data, strategy management, risk controls, paper trading, backtesting, and system operations.

### Authentication

API endpoints can be protected with authentication by setting `AUTH_ENABLED=true` and providing an `ADMIN_API_KEY`. When authentication is enabled, clients must send the API key in the `X-API-Key` header:

```bash
curl -H "X-API-Key: your-admin-api-key" http://localhost:8080/api/v1/strategy/settings
```

The following endpoints remain public even when authentication is enabled:
- `GET /health`
- `GET /ready`
- `GET /metrics`
- `GET /dashboard/`

### Market Data Endpoints

#### Get Historical Candles
Retrieve historical OHLCV candles for a symbol and time range.

```bash
GET /api/v1/market/candles?symbol=BTCUSDT&interval=1m&from=2026-06-16T00:00:00Z&to=2026-06-16T01:00:00Z&limit=500
```

Response:
```json
[
  {
    "symbol": "BTCUSDT",
    "interval": "1m",
    "open_time": "2026-06-16T00:00:00Z",
    "open": "25000.00",
    "high": "25100.00",
    "low": "24950.00",
    "close": "25050.00",
    "volume": "10.5",
    "quote_volume": "262875.00",
    "trade_count": 150,
    "taker_buy_base_volume": "5.2",
    "taker_buy_quote_volume": "130250.00"
  }
]
```

#### Get Candle Coverage
Check what historical data is available for a symbol and time range.

```bash
GET /api/v1/market/candles/coverage?symbol=BTCUSDT&interval=1m&from=2026-06-16T00:00:00Z&to=2026-06-16T01:00:00Z
```

Response:
```json
{
  "symbol": "BTCUSDT",
  "interval": "1m",
  "has_data": true,
  "earliest": "2026-06-16T00:00:00Z",
  "latest": "2026-06-16T00:59:00Z",
  "count": 60
}
```

### Strategy Endpoints

#### Get Current Strategy Settings
Retrieve the currently active strategy configuration.

```bash
GET /api/v1/strategy/settings
```

Response:
```json
{
  "strategy_name": "sma-crossover",
  "version": "v1",
  "symbol": "BTCUSDT",
  "interval": "1m",
  "fast_period": 9,
  "slow_period": 21,
  "lookback_limit": 100,
  "rsi_period": 14,
  "rsi_oversold": 30,
  "rsi_overbought": 70,
  "updated_at": "2026-06-22T10:30:00Z"
}
```

#### Update Strategy Settings
Modify the strategy configuration (requires authentication when enabled).

```bash
PUT /api/v1/strategy/settings
Content-Type: application/json

{
  "strategy_name": "sma-crossover",
  "version": "v1",
  "symbol": "BTCUSDT",
  "interval": "1m",
  "fast_period": 12,
  "slow_period": 26,
  "lookback_limit": 100
}
```

Response:
```json
{
  "strategy_name": "sma-crossover",
  "version": "v1",
  "symbol": "BTCUSDT",
  "interval": "1m",
  "fast_period": 12,
  "slow_period": 26,
  "lookback_limit": 100,
  "rsi_period": 14,
  "rsi_oversold": 30,
  "rsi_overbought": 70,
  "updated_at": "2026-06-22T10:35:00Z"
}
```

#### Evaluate Strategy
Manually trigger a strategy evaluation on the latest market data.

```bash
POST /api/v1/strategy/evaluate
```

Response:
```json
{
  "id": "sig_1234567890",
  "strategy_name": "sma-crossover",
  "version": "v1",
  "symbol": "BTCUSDT",
  "interval": "1m",
  "side": "buy",
  "strength": 75.5,
  "reason": "fast SMA crossed above slow SMA",
  "generated_at": "2026-06-22T10:30:00Z"
}
```

### Backtesting Endpoints

#### Create a Backtest
Run a historical backtest for a strategy over a specified time range.

```bash
POST /api/v1/backtests
Content-Type: application/json

{
  "symbol": "BTCUSDT",
  "interval": "1m",
  "strategy_name": "sma-crossover",
  "from": "2026-06-01T00:00:00Z",
  "to": "2026-06-22T00:00:00Z",
  "execution_fill_mode": "next_open",
  "save_equity_curve": false
}
```

Response:
```json
{
  "id": "bt_1234567890",
  "symbol": "BTCUSDT",
  "interval": "1m",
  "strategy_name": "sma-crossover",
  "from": "2026-06-01T00:00:00Z",
  "to": "2026-06-22T00:00:00Z",
  "execution_fill_mode": "next_open",
  "created_at": "2026-06-22T10:30:00Z",
  "completed_at": "2026-06-22T10:32:15Z",
  "total_return_percent": 15.5,
  "benchmark_return_percent": 12.3,
  "excess_return_percent": 3.2,
  "win_rate": 0.65,
  "profit_factor": 1.8,
  "max_drawdown_percent": 8.5,
  "sharpe_ratio": 1.2,
  "sortino_ratio": 1.5,
  "validation_status": "candidate",
  "trade_count": 42
}
```

#### Get Backtest Details
Retrieve detailed information about a specific backtest run.

```bash
GET /api/v1/backtests/bt_1234567890
```

Response includes the backtest summary plus detailed trade information:
```json
{
  "id": "bt_1234567890",
  "symbol": "BTCUSDT",
  "interval": "1m",
  "strategy_name": "sma-crossover",
  "from": "2026-06-01T00:00:00Z",
  "to": "2026-06-22T00:00:00Z",
  "execution_fill_mode": "next_open",
  "created_at": "2026-06-22T10:30:00Z",
  "completed_at": "2026-06-22T10:32:15Z",
  "total_return_percent": 15.5,
  "benchmark_return_percent": 12.3,
  "excess_return_percent": 3.2,
  "win_rate": 0.65,
  "profit_factor": 1.8,
  "max_drawdown_percent": 8.5,
  "sharpe_ratio": 1.2,
  "sortino_ratio": 1.5,
  "validation_status": "candidate",
  "trade_count": 42,
  "trades": [
    {
      "id": "t_123",
      "entry_time": "2026-06-01T00:30:00Z",
      "entry_price": "25000.00",
      "exit_time": "2026-06-01T02:45:00Z",
      "exit_price": "25200.00",
      "side": "buy",
      "quantity": "0.04",
      "fee": "0.50",
      "pnl": "800.00",
      "pnl_percent": 3.2,
      "holding_time_minutes": 135
    }
  ],
  "round_trips": [
    {
      "entry_time": "2026-06-01T00:30:00Z",
      "exit_time": "2026-06-01T02:45:00Z",
      "entry_price": "25000.00",
      "exit_price": "25200.00",
      "side": "buy",
      "quantity": "0.04",
      "fee": "0.50",
      "pnl": "800.00",
      "pnl_percent": 3.2,
      "holding_time_minutes": 135
    }
  ]
}
```

### Paper Trading Endpoints

#### Get Paper Account
Retrieve the current state of the paper trading account.

```bash
GET /api/v1/paper/account
```

Response:
```json
{
  "base_asset": "BTC",
  "quote_asset": "USDT",
  "base_balance": "0.15",
  "quote_balance": "2500.00",
  "base_balance_usd": "3750.00",
  "quote_balance_usd": "2500.00",
  "total_balance_usd": "6250.00",
  "unrealized_pnl_usd": "125.00",
  "realized_pnl_usd": "375.00",
  "total_pnl_usd": "500.00",
  "total_pnl_percent": 8.7
}
```

#### Reset Paper Account
Reset the paper trading account to initial state.

```bash
POST /api/v1/paper/account/reset
Content-Type: application/json

{
  "base_asset": "BTC",
  "quote_asset": "USDT",
  "base_balance": 0,
  "quote_balance": 10000
}
```

Response:
```json
{
  "base_asset": "BTC",
  "quote_asset": "USDT",
  "base_balance": "0",
  "quote_balance": "10000",
  "base_balance_usd": "0",
  "quote_balance_usd": "10000",
  "total_balance_usd": "10000",
  "unrealized_pnl_usd": "0",
  "realized_pnl_usd": "0",
  "total_pnl_usd": "0",
  "total_pnl_percent": 0
}
```

#### Run Manual Paper Cycle
Execute one manual paper trading cycle using current settings.

```bash
POST /api/v1/paper/cycles
Content-Type: application/json

{
  "symbol": "BTCUSDT",
  "interval": "1m"
}
```

Response:
```json
{
  "signal": {
    "id": "sig_1234567890",
    "strategy_name": "sma-crossover",
    "version": "v1",
    "symbol": "BTCUSDT",
    "interval": "1m",
    "side": "buy",
    "strength": 75.5,
    "reason": "fast SMA crossed above slow SMA",
    "generated_at": "2026-06-22T10:30:00Z"
  },
  "risk_decision": {
    "id": "rd_123",
    "signal_id": "sig_1234567890",
    "approved": true,
    "reason": "within risk limits",
    "created_at": "2026-06-22T10:30:01Z"
  },
  "execution": {
    "id": "ex_123",
    "risk_decision_id": "rd_123",
    "symbol": "BTCUSDT",
    "side": "buy",
    "quantity": "0.04",
    "price": "25050.00",
    "fee": "0.50",
    "timestamp": "2026-06-22T10:30:02Z"
  }
}
```

### System Endpoints

#### Health Check
Basic service health verification.

```bash
GET /health
```

Response:
```json
{
  "status": "ok",
  "timestamp": "2026-06-22T10:30:00Z",
  "service": "sentra",
  "version": "1.0.0"
}
```

#### Readiness Check
Verify that all dependencies are available.

```bash
GET /ready
```

Response:
```json
{
  "status": "ready",
  "timestamp": "2026-06-22T10:30:00Z",
  "checks": {
    "postgres": {
      "status": "ok",
      "latency_ms": 5
    },
    "redis": {
      "status": "ok",
      "latency_ms": 2
    }
  }
}
```

#### Metrics
Prometheus-compatible metrics endpoint.

```bash
GET /metrics
```

Response:
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",endpoint="/api/v1/strategy/settings"} 150
# HELP strategy_evaluations_total Total number of strategy evaluations
# TYPE strategy_evaluations_total counter
strategy_evaluations_total{strategy="sma-crossover",result="buy"} 42
```

## Local Setup

1. Copy `.env.example` to `.env` and adjust values if needed.
2. Start dependencies:

## Local Setup

1. Copy `.env.example` to `.env` and adjust values if needed.
2. Start dependencies:

```powershell
docker compose up -d postgres redis
```

3. Run tests:

```powershell
go test ./...
```

4. Apply migrations:

```powershell
go run ./cmd/migrate
```

5. Run the API:

```powershell
go run ./cmd/api
```

Dashboard:

- `http://localhost:8080/dashboard/`

Health endpoints:

- `GET /health`
- `GET /ready`
- `GET /api/v1/market/candles?symbol=BTCUSDT&interval=1m&from=2026-06-16T00:00:00Z&to=2026-06-16T01:00:00Z&limit=500`
- `GET /api/v1/market/candles/coverage?symbol=BTCUSDT&interval=1m&from=2026-06-16T00:00:00Z&to=2026-06-16T01:00:00Z`
- `GET /api/v1/dashboard/summary?symbol=BTCUSDT&interval=1m`
- `GET /api/v1/signals?symbol=BTCUSDT&limit=100`
- `GET /api/v1/strategy/settings`
- `PUT /api/v1/strategy/settings`
- `POST /api/v1/strategy/evaluate`
- `POST /api/v1/backtests`
- `POST /api/v1/backtests/optimizations`
- `GET /api/v1/backtests?symbol=BTCUSDT&limit=50`
- `GET /api/v1/backtests/{id}`
- `POST /api/v1/strategy/comparisons`
- `GET /api/v1/strategy/comparisons?symbol=BTCUSDT&limit=50`
- `GET /api/v1/strategy/comparisons/{id}`
- `POST /api/v1/strategy/comparisons/{id}/activate`
- `GET /api/v1/strategy/activations?limit=50`
- `GET /api/v1/risk/settings`
- `PUT /api/v1/risk/settings`
- `GET /api/v1/safety/status`
- `PUT /api/v1/safety/status`
- `GET /api/v1/audit/events`
- `GET /api/v1/reports/paper/daily-pnl?symbol=BTCUSDT&limit=30`
- `GET /api/v1/reports/paper/trade-counts?symbol=BTCUSDT&limit=30`
- `GET /api/v1/reports/risk/rejections?symbol=BTCUSDT&limit=30`
- `GET /api/v1/risk-decisions?symbol=BTCUSDT&limit=100`
- `GET /api/v1/paper/account`
- `POST /api/v1/paper/account/reset`
- `POST /api/v1/paper/cycles`
- `GET /api/v1/paper/orders?symbol=BTCUSDT&limit=100`
- `GET /api/v1/paper/trades?symbol=BTCUSDT&limit=100`
- `GET /metrics`
- `GET /api/v1/ops/pipeline-runs?status=failed&limit=100`
- `GET /api/v1/ops/streams`

6. Run the worker:

```powershell
go run ./cmd/worker
```

The collector reconnects on stream disconnects using:

- `MARKET_DATA_MAX_RECONNECTS`
- `MARKET_DATA_RECONNECT_DELAY`

Backfill historical candles from Binance into PostgreSQL:

```powershell
go run ./cmd/backfill -symbol BTCUSDT -interval 1m -from 2026-06-16T00:00:00Z -to 2026-06-16T01:00:00Z -limit 1000
```

Resume a failed or interrupted durable backfill job:

```powershell
go run ./cmd/backfill -resume <backfill-job-id>
```

Run the SMA crossover strategy once against persisted candles:

```powershell
$env:STRATEGY_SYMBOL="BTCUSDT"
$env:STRATEGY_INTERVAL="1m"
$env:STRATEGY_FAST_PERIOD="9"
$env:STRATEGY_SLOW_PERIOD="21"
$env:STRATEGY_LOOKBACK_LIMIT="100"
go run ./cmd/strategy
```

Evaluate the latest strategy signal through the risk engine:

```powershell
$env:RISK_ENABLED="true"
$env:RISK_SYMBOL="BTCUSDT"
$env:RISK_MAX_SIGNAL_STRENGTH="100"
$env:RISK_ALLOW_BUY="true"
$env:RISK_ALLOW_SELL="true"
go run ./cmd/risk
```

Execute the latest approved risk decision in paper mode:

```powershell
$env:EXECUTION_ENABLED="true"
$env:EXECUTION_SYMBOL="BTCUSDT"
$env:EXECUTION_INTERVAL="1m"
$env:EXECUTION_BASE_ASSET="BTC"
$env:EXECUTION_QUOTE_ASSET="USDT"
$env:EXECUTION_QUOTE_ORDER_AMOUNT="100"
$env:EXECUTION_FEE_RATE="0.001"
go run ./cmd/execution
```

Run the automated paper-trading pipeline from market-data stream events:

```powershell
$env:ORCHESTRATION_ENABLED="true"
$env:ORCHESTRATION_CONSUMER_GROUP="paper-pipeline"
$env:ORCHESTRATION_CONSUMER_NAME="worker-1"
$env:RISK_ENABLED="true"
$env:EXECUTION_ENABLED="true"
go run ./cmd/worker
```

The orchestration worker processes closed candle events, runs strategy, risk, and paper execution, and uses `pipeline_runs` idempotency records to avoid duplicate processing.

Observability:

- `/metrics` returns in-process counters for HTTP requests and pipeline outcomes.
- `/api/v1/ops/pipeline-runs` returns idempotency/run history from PostgreSQL.
- `/api/v1/ops/streams` returns Redis Streams group and pending-message counts for configured streams.

API authentication:

- Set `AUTH_ENABLED=true` and `ADMIN_API_KEY=<strong-secret>` to protect `/api/v1/...` routes.
- Send the key as `X-API-Key: <strong-secret>`.
- `/health`, `/ready`, `/metrics`, and `/dashboard/` remain public for uptime, monitoring, and dashboard shell loading.
- The dashboard stores the entered API key in browser `sessionStorage` and sends it only to protected API routes.

Security hardening:

- Production mode requires `AUTH_ENABLED=true`.
- `MAX_REQUEST_BODY_BYTES` limits request body size.
- `RATE_LIMIT_REQUESTS_PER_MINUTE` limits protected API routes per API key/IP.
- API responses include security headers such as `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, and `Content-Security-Policy`.
- Panic responses return a generic JSON error without exposing internal details.

The market data collector is disabled by default. To run the Binance kline collector:

```powershell
$env:MARKET_DATA_ENABLED="true"
$env:MARKET_DATA_SYMBOL="BTCUSDT"
$env:MARKET_DATA_INTERVAL="1m"
go run ./cmd/worker
```

To persist market data stream events into PostgreSQL candles:

```powershell
$env:MARKET_DATA_PERSISTENCE_ENABLED="true"
$env:MARKET_DATA_REDIS_STREAM="stream:market-data"
$env:MARKET_DATA_CONSUMER_GROUP="market-data-persistence"
$env:MARKET_DATA_CONSUMER_NAME="worker-1"
go run ./cmd/worker
```

## Migrations

Migration files live in `migrations/`. The migration runner records applied versions in `schema_migrations`.

## Deployment

Phase 14 adds a production-oriented container path:

- `Dockerfile` builds all Go commands into one runtime image.
- `docker-compose.yml` can run `api`, `worker`, `migrate`, and one-shot commands with the `app` profile.
- `.env.production.example` documents production-safe defaults and required secret placeholders.
- `docs/deployment.md` and `docs/runbook.md` document startup, migrations, health checks, and common operational failures.

Run the application stack with Docker Compose:

```powershell
docker compose up -d postgres redis
docker compose --profile app run --rm migrate
docker compose --profile app up -d api worker
```

## Dashboard

Phase 15 adds a static operations dashboard served by the Go API from `web/dashboard/`.

The dashboard reads existing API endpoints only. It shows summary, candles, signals, risk decisions, paper orders, paper trades, pipeline runs, and Redis Streams state.

Phase 16 adds strategy settings controls:

- Settings are stored in PostgreSQL in `strategy_settings`.
- The dashboard can update `symbol`, `interval`, `fast_period`, `slow_period`, and `lookback_limit`.
- Manual evaluation saves a strategy signal only.
- The dashboard does not expose live-trading controls or live-order execution.

Phase 17 adds an isolated backtesting engine:

- Backtest runs are stored in `backtest_runs`.
- Simulated fills are stored in `backtest_trades`.
- Backtests run over persisted historical candles.
- Backtests do not publish Redis events.
- Backtests do not mutate the paper trading account, paper orders, or paper trades.

Phase 18 improves the backtest data workflow:

- Candle coverage reports available candle count and date range.
- Backtest creation returns structured preflight errors when there are not enough candles.
- Backtest reports include best/worst trade, average win/loss, buy/sell counts, and open-position status.
- The dashboard shows data readiness before running a backtest and can export simulated trades to CSV.

Phase 19 adds persisted advanced risk controls:

- Risk settings are stored in PostgreSQL in `risk_settings`.
- The API and dashboard can update signal strength bounds, quote-size limits, daily trade and loss limits, side permissions, and post-trade cooldown.
- Risk decision snapshots include the active rule values used during evaluation.
- The orchestration worker applies these rules to paper execution only.

Phase 20 adds paper trading control operations:

- `POST /api/v1/paper/account/reset` resets the singleton paper account balances and assets.
- `POST /api/v1/paper/cycles` runs one manual paper cycle using persisted strategy settings, risk settings, latest candles, and paper execution.
- The dashboard Ops tab includes a Paper Control Center for resetting the paper account and running a one-shot cycle.
- The manual cycle publishes the same signal, risk, and execution events as the paper pipeline when configured streams are available.

Phase 21 adds production safety controls:

- Safety status is stored in PostgreSQL in `safety_status`.
- `kill_switch_active=true` blocks paper order creation in both the worker pipeline and manual paper cycles.
- Safety changes are recorded in `audit_events`.
- The dashboard Ops tab includes safety controls and an audit event table.

Phase 22 hardens audit and reporting:

- Sensitive operator actions write audit events, including settings changes, account resets, manual cycles, and backtest creation.
- Audit events support event type, actor, and time range filters.
- Audit events, risk decisions, and paper trades support CSV export with `format=csv`.
- Reporting endpoints summarize daily paper P&L, paper trade counts, and rejected risk decisions by reason.

Phase 23 adds strategy expansion:

- `sma-crossover` remains the default strategy.
- `rsi-mean-reversion` can be selected in persisted strategy settings and backtests.
- `btc-trend-pullback` can be selected in persisted strategy settings and backtests.
- RSI settings include `rsi_period`, `rsi_oversold`, and `rsi_overbought`.
- Manual evaluation, manual paper cycles, event-driven paper orchestration, and backtests use the selected strategy.

Phase 24 adds strategy comparison:

- SMA and RSI can be run over the same historical candle range.
- Results are ranked by return, then lower drawdown, then win rate.
- Comparison headers and ranked result rows are stored in PostgreSQL.
- The dashboard Compare tab can run and review saved comparisons.

Phase 25 adds evidence-based strategy activation:

- A saved comparison can activate its winning strategy into `strategy_settings`.
- Activation requires recent comparison evidence.
- Activation is blocked unless the selected comparison result is a validated `candidate`, beats buy-and-hold, stays within the activation drawdown cap, has completed trades, uses `next_open`, and passes train/test plus walk-forward evidence.
- Activation history is stored in `strategy_activations`.
- Each activation writes an audit event.
- The dashboard Compare tab includes an Activate Winner action and activation history.
- Blocked activations return `409 Conflict` and do not update settings, activation history, or audit logs.

Phase 27 adds failure simulation testing:

- Binance subscription failures and WebSocket interruptions are covered by collector recovery tests.
- Redis publish failures are covered by collector and orchestration failure tests.
- Redis Streams read failures retry instead of stopping the consumer.
- Database write failures mark orchestration runs failed and prevent downstream side effects.
- Duplicate candle events are covered by idempotency tests.
- Retry-after-failure behavior is covered for the paper pipeline.

Phase 28 adds historical market data backfill and optimization:

- Candle storage includes Binance quote volume and trade count metadata.
- Backfill jobs are tracked in PostgreSQL and can resume from `next_open_time`.
- The backfill engine pages through Binance klines in batches instead of fetching only one page.
- Candle batch upserts reduce PostgreSQL write overhead for large imports.
- `1m` candles can be aggregated into `5m`, `15m`, and `1h` candles.
- Backtests and strategy comparisons can now request the full selected historical range instead of being capped at 1000 candles.
- The dashboard Candles view shows historical coverage, backfill jobs, a 2-year backfill action, and timeframe aggregation.

Historical data APIs:

- `POST /api/v1/market/backfills`
- `GET /api/v1/market/backfills?symbol=BTCUSDT&limit=20`
- `GET /api/v1/market/backfills/{id}`
- `POST /api/v1/market/aggregations`

Phase 29 adds backtest diagnostics and strategy validation:

- Backtests now simulate slippage in addition to fees.
- Completed round-trip trades are recorded with entry/exit price, net P&L, holding time, and reasons.
- Equity curve points are recorded with drawdown percentage.
- Backtest runs include profit factor, average trade, average holding time, winning/losing trades, and validation status.
- Strategy comparison ranking now prioritizes validated candidates, then profit factor, lower drawdown, return, and trade count.
- The dashboard Backtests view shows diagnostics, round trips, and sampled equity curve data.

Phase 30 adds backtest position sizing and benchmark comparison:

- Backtests support all-in, fixed quote, and percent-of-equity position sizing.
- Strategy comparisons use the same position sizing assumptions as single backtests.
- Backtest runs and comparison results include buy-and-hold benchmark return and excess return.
- The dashboard Backtests and Compare views expose position sizing controls and benchmark metrics.

Phase 31 starts the strategy research framework:

- Backtest validation now requires positive excess return versus buy-and-hold.
- Candidate runs require profit factor above 1.2, drawdown at or below 30%, positive average completed trade, and sane trade frequency for the selected interval.
- New rejection statuses include `underperforms_benchmark`, `weak_profit_factor`, `negative_average_trade`, and `overtrading`.
- SMA backtests and comparisons can apply research filters: trend SMA, cooldown bars, and minimum holding bars.
- SMA parameter sweeps can run a bounded grid of fast/slow periods and rank results by candidate status, excess return, profit factor, drawdown, and trade count.
- SMA sweeps support train/test validation; the dashboard uses a 70/30 split and shows train/test excess return and validation status.
- SMA sweeps support walk-forward validation; the dashboard tests 4 chronological folds and shows average fold excess return, pass count, and fold validation status.

Phase 32A starts the professional strategy-quality upgrade:

- Backtests and SMA sweeps support ATR risk exits through additive API fields: `atr_exit_enabled`, `atr_period`, `atr_stop_multiplier`, and `atr_take_profit_multiplier`.
- ATR stop/target levels are fixed at entry and evaluated against later candle low/high values.
- Saved backtest runs persist the ATR exit settings via migration `000018_backtest_atr_exits`.
- The dashboard Backtests view exposes ATR exit controls and reports the selected ATR settings in the result panel.

Phase 32B adds the first professional strategy family:

- `btc-trend-pullback` uses `fast_period` as the pullback EMA, `slow_period` as the trend EMA, and `rsi_period` for momentum confirmation.
- The strategy buys only when price recovers above the pullback EMA while pullback EMA is above trend EMA and RSI crosses from `<= 50` to `> 50`.
- It emits a sell signal when price falls below the trend EMA or RSI drops below 45.
- Migration `000019_btc_trend_pullback_strategy` seeds default settings for the new strategy.

Phase 33 improves interactive backtest performance:

- `POST /api/v1/backtests` accepts additive field `save_equity_curve`.
- When `save_equity_curve` is `false`, the API saves the run summary, trades, and round trips but skips per-candle equity point persistence.
- Dashboard backtest runs now send `save_equity_curve: false` by default to keep long historical tests responsive.

Phase 34 hardens backtest correctness:

- `POST /api/v1/backtests` accepts additive field `execution_fill_mode`.
- Supported fill modes are `same_close` and `next_open`; omitted backtest fill mode defaults to `next_open`.
- The backtest engine rejects invalid candle series with gaps, duplicates, unclosed candles, out-of-order times, or invalid OHLC values.
- Backtest run persistence now uses a transaction when the database connection supports PostgreSQL transactions.
- Migration `000020_backtest_execution_fill_mode` persists the selected fill mode on saved runs.

Phase 35 hardens strategy comparison evidence:

- `POST /api/v1/strategy/comparisons` accepts additive fields `train_test_enabled`, `train_ratio`, `walk_forward_enabled`, and `walk_forward_folds`.
- Comparison result rows now include train/test and walk-forward validation metrics.
- Comparison ranking prefers strategies that pass walk-forward validation, then train/test validation, then baseline candidate scoring.
- Migration `000021_strategy_comparison_research_evidence` persists the new evidence fields.
- Strategy activation enforces train/test and walk-forward validation evidence.

Phase 36 hardens research execution timing:

- Strategy comparisons and SMA optimizations accept additive field `execution_fill_mode`.
- New comparison and optimization research defaults to `next_open`.
- Comparison results persist their execution fill mode via migration `000022_strategy_comparison_execution_fill_mode`.
- The dashboard sends `next_open` for strategy comparisons and forwards the selected backtest fill mode to SMA sweeps.
- Strategy activation rejects comparison evidence unless it used `next_open`.

Phase 37 hardens open-position accounting:

- Backtests force-close any remaining long position at the final candle close.
- Forced final exits apply normal sell slippage and fees and produce a round trip with exit reason `end_of_backtest`.
- Ending balance and validation now use realized final-exit proceeds instead of optimistic mark-to-market value.
- The dashboard round-trip table shows exit reason.

Phase 38 starts BTC Trend Pullback v2:

- `btc-trend-pullback` entries now require a fresh RSI midline cross instead of accepting any candle where RSI is already above 50.
- Required candle calculations include both previous and current RSI windows.
- The strategy now applies a default ATR percent volatility filter to avoid very flat or extreme-volatility entries.
- Required candle calculations include the default ATR filter window.
- This is the first incremental strategy-quality pass before adding broader regime filters.

Phase 39 hardens research activation integrity:

- Single backtest requests now default to `next_open` instead of `same_close`.
- Backtest candidate validation rejects `same_close` execution evidence.
- Strategy activation now requires train/test and walk-forward evidence before settings can be promoted.

Phase 40 adds strategy-quality metrics:

- Backtest runs now report expectancy, trades per day, churn ratio, Sharpe ratio, and Sortino ratio.
- Strategy comparison and optimization results inherit these metrics from the underlying backtest run.
- Migration `000023_strategy_quality_metrics` persists the new metrics for saved backtests and comparison results.
- The dashboard Backtests result panel shows the new quality metrics.
