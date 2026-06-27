# Market Data

Sentra stores backtesting data as PostgreSQL candles. Binance is used as the historical source, but backtests and strategy comparisons read only from the local `candles` table.

## Historical Backfill

Use the CLI for large operator jobs:

```powershell
go run ./cmd/backfill -symbol BTCUSDT -interval 1m -from 2024-06-18T00:00:00Z -to 2026-06-18T00:00:00Z -limit 1000
```

Each run creates a row in `market_data_backfill_jobs`. Progress is saved after every fetched batch using `next_open_time`.

The backfill service uses bounded retries for transient Binance/network failures and pauses between paged requests to avoid tight API-call loops.

If the process stops, resume it:

```powershell
go run ./cmd/backfill -resume <backfill-job-id>
```

## Storage

The `candles` table is unique by:

```text
exchange + symbol + interval + open_time
```

Backfill writes use upsert behavior, so duplicate Binance candles update the existing row safely.

Stored Binance candle metadata includes:

- base volume
- quote volume
- trade count

## Aggregation

Fetch `1m` candles from Binance as the base dataset. Generate higher timeframes locally:

- `5m`
- `15m`
- `1h`

Aggregation rules:

- open = first source candle open
- high = max source high
- low = min source low
- close = last source candle close
- volume = sum
- quote volume = sum
- trade count = sum
- buckets align to clock boundaries, such as `00/05/10` for `5m`, `00/15/30/45` for `15m`, and top-of-hour UTC boundaries for `1h`
- target interval rows are replaced for the requested range before regenerated rows are inserted

## API

Queue a default two-year BTCUSDT 1m backfill:

```http
POST /api/v1/market/backfills
```

List jobs:

```http
GET /api/v1/market/backfills?symbol=BTCUSDT&limit=20
```

Run aggregation:

```http
POST /api/v1/market/aggregations
```

```json
{
  "symbol": "BTCUSDT",
  "source_interval": "1m",
  "target_intervals": ["5m", "15m", "1h"],
  "from": "2024-06-18T00:00:00Z",
  "to": "2026-06-18T00:00:00Z"
}
```

## Dashboard

Open `http://localhost:8080/dashboard/` and use the Candles view.

It shows:

- available candle count by timeframe
- first and last candle times
- latest backfill jobs
- 2-year backfill trigger
- timeframe aggregation trigger
