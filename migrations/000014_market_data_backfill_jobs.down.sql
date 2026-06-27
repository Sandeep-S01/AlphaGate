DROP INDEX IF EXISTS idx_market_data_backfill_jobs_status_updated_at;
DROP INDEX IF EXISTS idx_market_data_backfill_jobs_symbol_created_at_desc;
DROP TABLE IF EXISTS market_data_backfill_jobs;

ALTER TABLE candles
    DROP COLUMN IF EXISTS trade_count,
    DROP COLUMN IF EXISTS quote_volume;
