package config

import (
	"testing"
	"time"
)

func TestLoadUsesDefaultsWhenEnvironmentIsEmpty(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("POSTGRES_DSN", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("BINANCE_REST_BASE_URL", "")
	t.Setenv("BINANCE_WS_BASE_URL", "")
	t.Setenv("MARKET_DATA_ENABLED", "")
	t.Setenv("MARKET_DATA_SYMBOL", "")
	t.Setenv("MARKET_DATA_INTERVAL", "")
	t.Setenv("MARKET_DATA_REDIS_STREAM", "")
	t.Setenv("MARKET_DATA_MAX_RECONNECTS", "")
	t.Setenv("MARKET_DATA_RECONNECT_DELAY", "")
	t.Setenv("STRATEGY_ENABLED", "")
	t.Setenv("STRATEGY_NAME", "")
	t.Setenv("STRATEGY_VERSION", "")
	t.Setenv("STRATEGY_SYMBOL", "")
	t.Setenv("STRATEGY_INTERVAL", "")
	t.Setenv("STRATEGY_FAST_PERIOD", "")
	t.Setenv("STRATEGY_SLOW_PERIOD", "")
	t.Setenv("STRATEGY_LOOKBACK_LIMIT", "")
	t.Setenv("STRATEGY_RSI_PERIOD", "")
	t.Setenv("STRATEGY_RSI_OVERSOLD", "")
	t.Setenv("STRATEGY_RSI_OVERBOUGHT", "")
	t.Setenv("STRATEGY_SIGNAL_STREAM", "")
	t.Setenv("EXECUTION_ENABLED", "")
	t.Setenv("EXECUTION_SYMBOL", "")
	t.Setenv("EXECUTION_INTERVAL", "")
	t.Setenv("EXECUTION_BASE_ASSET", "")
	t.Setenv("EXECUTION_QUOTE_ASSET", "")
	t.Setenv("EXECUTION_QUOTE_ORDER_AMOUNT", "")
	t.Setenv("EXECUTION_FEE_RATE", "")
	t.Setenv("EXECUTION_STREAM", "")
	t.Setenv("ORCHESTRATION_ENABLED", "")
	t.Setenv("ORCHESTRATION_CONSUMER_GROUP", "")
	t.Setenv("ORCHESTRATION_CONSUMER_NAME", "")
	t.Setenv("AUTH_ENABLED", "")
	t.Setenv("ADMIN_API_KEY", "")
	t.Setenv("MAX_REQUEST_BODY_BYTES", "")
	t.Setenv("RATE_LIMIT_REQUESTS_PER_MINUTE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.App.Env != "local" {
		t.Fatalf("expected default env local, got %q", cfg.App.Env)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("expected default HTTP addr :8080, got %q", cfg.HTTP.Addr)
	}
	if cfg.Postgres.DSN == "" {
		t.Fatal("expected default Postgres DSN")
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Fatalf("expected default Redis addr localhost:6379, got %q", cfg.Redis.Addr)
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("expected default log level info, got %q", cfg.Logging.Level)
	}
	if cfg.App.ShutdownTimeout != 10*time.Second {
		t.Fatalf("expected default shutdown timeout 10s, got %s", cfg.App.ShutdownTimeout)
	}
	if cfg.Binance.RESTBaseURL != "https://api.binance.com" {
		t.Fatalf("expected default Binance REST URL, got %q", cfg.Binance.RESTBaseURL)
	}
	if cfg.Binance.WSBaseURL != "wss://stream.binance.com:9443/ws" {
		t.Fatalf("expected default Binance WS URL, got %q", cfg.Binance.WSBaseURL)
	}
	if cfg.MarketData.Enabled {
		t.Fatal("expected market data collector disabled by default")
	}
	if cfg.MarketData.Symbol != "BTCUSDT" {
		t.Fatalf("expected default market symbol BTCUSDT, got %q", cfg.MarketData.Symbol)
	}
	if cfg.MarketData.Interval != "1m" {
		t.Fatalf("expected default market interval 1m, got %q", cfg.MarketData.Interval)
	}
	if cfg.MarketData.RedisStream != "stream:market-data" {
		t.Fatalf("expected default market data stream, got %q", cfg.MarketData.RedisStream)
	}
	if cfg.MarketData.MaxReconnects != 100 {
		t.Fatalf("expected default max reconnects 100, got %d", cfg.MarketData.MaxReconnects)
	}
	if cfg.MarketData.ReconnectDelay != 5*time.Second {
		t.Fatalf("expected default reconnect delay 5s, got %s", cfg.MarketData.ReconnectDelay)
	}
	if cfg.Strategy.Enabled {
		t.Fatal("expected strategy disabled by default")
	}
	if cfg.Strategy.Name != "sma-crossover" || cfg.Strategy.Version != "v1" {
		t.Fatalf("unexpected default strategy identity: %+v", cfg.Strategy)
	}
	if cfg.Strategy.Symbol != "BTCUSDT" || cfg.Strategy.Interval != "1m" {
		t.Fatalf("unexpected default strategy market: %+v", cfg.Strategy)
	}
	if cfg.Strategy.FastPeriod != 9 || cfg.Strategy.SlowPeriod != 21 {
		t.Fatalf("unexpected default strategy periods: %+v", cfg.Strategy)
	}
	if cfg.Strategy.LookbackLimit != 100 {
		t.Fatalf("expected default lookback 100, got %d", cfg.Strategy.LookbackLimit)
	}
	if cfg.Strategy.RSIPeriod != 14 || cfg.Strategy.RSIOversold != 30 || cfg.Strategy.RSIOverbought != 70 {
		t.Fatalf("unexpected default RSI settings: %+v", cfg.Strategy)
	}
	if cfg.Strategy.SignalStream != "stream:strategy-signals" {
		t.Fatalf("expected default signal stream, got %q", cfg.Strategy.SignalStream)
	}
	if cfg.Execution.Enabled {
		t.Fatal("expected execution disabled by default")
	}
	if cfg.Execution.Symbol != "BTCUSDT" || cfg.Execution.Interval != "1m" {
		t.Fatalf("unexpected execution market defaults: %+v", cfg.Execution)
	}
	if cfg.Execution.BaseAsset != "BTC" || cfg.Execution.QuoteAsset != "USDT" {
		t.Fatalf("unexpected execution asset defaults: %+v", cfg.Execution)
	}
	if cfg.Execution.QuoteOrderAmount != 100 {
		t.Fatalf("expected default quote order amount 100, got %f", cfg.Execution.QuoteOrderAmount)
	}
	if cfg.Execution.FeeRate != 0.001 {
		t.Fatalf("expected default fee rate 0.001, got %f", cfg.Execution.FeeRate)
	}
	if cfg.Execution.Stream != "stream:execution-results" {
		t.Fatalf("expected default execution stream, got %q", cfg.Execution.Stream)
	}
	if cfg.Orchestration.Enabled {
		t.Fatal("expected orchestration disabled by default")
	}
	if cfg.Orchestration.ConsumerGroup != "paper-pipeline" {
		t.Fatalf("expected default orchestration group paper-pipeline, got %q", cfg.Orchestration.ConsumerGroup)
	}
	if cfg.Orchestration.ConsumerName != "worker-1" {
		t.Fatalf("expected default orchestration consumer worker-1, got %q", cfg.Orchestration.ConsumerName)
	}
	if cfg.Auth.Enabled {
		t.Fatal("expected auth disabled by default")
	}
	if cfg.Auth.AdminAPIKey != "" {
		t.Fatal("expected default admin API key to be empty")
	}
	if cfg.Security.MaxRequestBodyBytes != 1_048_576 {
		t.Fatalf("expected default max body 1048576, got %d", cfg.Security.MaxRequestBodyBytes)
	}
	if cfg.Security.RateLimitRequestsPerMinute != 120 {
		t.Fatalf("expected default rate limit 120, got %d", cfg.Security.RateLimitRequestsPerMinute)
	}
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("POSTGRES_DSN", "postgres://sentra:secret@db:5432/sentra?sslmode=disable")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("REDIS_PASSWORD", "redis-secret")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SHUTDOWN_TIMEOUT", "15s")
	t.Setenv("BINANCE_REST_BASE_URL", "https://api1.binance.com")
	t.Setenv("BINANCE_WS_BASE_URL", "wss://stream.binance.com:9443/ws")
	t.Setenv("MARKET_DATA_ENABLED", "true")
	t.Setenv("MARKET_DATA_SYMBOL", "ETHUSDT")
	t.Setenv("MARKET_DATA_INTERVAL", "5m")
	t.Setenv("MARKET_DATA_REDIS_STREAM", "stream:test-market-data")
	t.Setenv("MARKET_DATA_MAX_RECONNECTS", "7")
	t.Setenv("MARKET_DATA_RECONNECT_DELAY", "2s")
	t.Setenv("STRATEGY_ENABLED", "true")
	t.Setenv("STRATEGY_NAME", "sma-test")
	t.Setenv("STRATEGY_VERSION", "v2")
	t.Setenv("STRATEGY_SYMBOL", "ETHUSDT")
	t.Setenv("STRATEGY_INTERVAL", "5m")
	t.Setenv("STRATEGY_FAST_PERIOD", "3")
	t.Setenv("STRATEGY_SLOW_PERIOD", "8")
	t.Setenv("STRATEGY_LOOKBACK_LIMIT", "50")
	t.Setenv("STRATEGY_RSI_PERIOD", "10")
	t.Setenv("STRATEGY_RSI_OVERSOLD", "25")
	t.Setenv("STRATEGY_RSI_OVERBOUGHT", "75")
	t.Setenv("STRATEGY_SIGNAL_STREAM", "stream:test-signals")
	t.Setenv("EXECUTION_ENABLED", "true")
	t.Setenv("EXECUTION_SYMBOL", "ETHUSDT")
	t.Setenv("EXECUTION_INTERVAL", "5m")
	t.Setenv("EXECUTION_BASE_ASSET", "ETH")
	t.Setenv("EXECUTION_QUOTE_ASSET", "USDT")
	t.Setenv("EXECUTION_QUOTE_ORDER_AMOUNT", "250")
	t.Setenv("EXECUTION_FEE_RATE", "0.002")
	t.Setenv("EXECUTION_STREAM", "stream:test-execution")
	t.Setenv("ORCHESTRATION_ENABLED", "true")
	t.Setenv("ORCHESTRATION_CONSUMER_GROUP", "test-pipeline")
	t.Setenv("ORCHESTRATION_CONSUMER_NAME", "test-worker")
	t.Setenv("AUTH_ENABLED", "true")
	t.Setenv("ADMIN_API_KEY", "secret-admin-key")
	t.Setenv("MAX_REQUEST_BODY_BYTES", "4096")
	t.Setenv("RATE_LIMIT_REQUESTS_PER_MINUTE", "10")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.App.Env != "test" {
		t.Fatalf("expected env override test, got %q", cfg.App.Env)
	}
	if cfg.HTTP.Addr != ":9090" {
		t.Fatalf("expected HTTP addr override :9090, got %q", cfg.HTTP.Addr)
	}
	if cfg.Postgres.DSN != "postgres://sentra:secret@db:5432/sentra?sslmode=disable" {
		t.Fatalf("unexpected postgres DSN %q", cfg.Postgres.DSN)
	}
	if cfg.Redis.Addr != "redis:6379" {
		t.Fatalf("expected Redis addr override redis:6379, got %q", cfg.Redis.Addr)
	}
	if cfg.Redis.Password != "redis-secret" {
		t.Fatal("expected Redis password override")
	}
	if cfg.Redis.DB != 2 {
		t.Fatalf("expected Redis DB override 2, got %d", cfg.Redis.DB)
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("expected log level override debug, got %q", cfg.Logging.Level)
	}
	if cfg.App.ShutdownTimeout != 15*time.Second {
		t.Fatalf("expected shutdown timeout override 15s, got %s", cfg.App.ShutdownTimeout)
	}
	if cfg.Binance.RESTBaseURL != "https://api1.binance.com" {
		t.Fatalf("expected Binance REST URL override, got %q", cfg.Binance.RESTBaseURL)
	}
	if !cfg.MarketData.Enabled {
		t.Fatal("expected market data collector enabled")
	}
	if cfg.MarketData.Symbol != "ETHUSDT" {
		t.Fatalf("expected market symbol override ETHUSDT, got %q", cfg.MarketData.Symbol)
	}
	if cfg.MarketData.Interval != "5m" {
		t.Fatalf("expected market interval override 5m, got %q", cfg.MarketData.Interval)
	}
	if cfg.MarketData.RedisStream != "stream:test-market-data" {
		t.Fatalf("expected market data stream override, got %q", cfg.MarketData.RedisStream)
	}
	if cfg.MarketData.MaxReconnects != 7 {
		t.Fatalf("expected max reconnects override 7, got %d", cfg.MarketData.MaxReconnects)
	}
	if cfg.MarketData.ReconnectDelay != 2*time.Second {
		t.Fatalf("expected reconnect delay override 2s, got %s", cfg.MarketData.ReconnectDelay)
	}
	if !cfg.Strategy.Enabled {
		t.Fatal("expected strategy enabled")
	}
	if cfg.Strategy.Name != "sma-test" || cfg.Strategy.Version != "v2" {
		t.Fatalf("unexpected strategy identity override: %+v", cfg.Strategy)
	}
	if cfg.Strategy.Symbol != "ETHUSDT" || cfg.Strategy.Interval != "5m" {
		t.Fatalf("unexpected strategy market override: %+v", cfg.Strategy)
	}
	if cfg.Strategy.FastPeriod != 3 || cfg.Strategy.SlowPeriod != 8 || cfg.Strategy.LookbackLimit != 50 {
		t.Fatalf("unexpected strategy numeric override: %+v", cfg.Strategy)
	}
	if cfg.Strategy.RSIPeriod != 10 || cfg.Strategy.RSIOversold != 25 || cfg.Strategy.RSIOverbought != 75 {
		t.Fatalf("unexpected RSI override: %+v", cfg.Strategy)
	}
	if cfg.Strategy.SignalStream != "stream:test-signals" {
		t.Fatalf("expected signal stream override, got %q", cfg.Strategy.SignalStream)
	}
	if !cfg.Execution.Enabled {
		t.Fatal("expected execution enabled")
	}
	if cfg.Execution.Symbol != "ETHUSDT" || cfg.Execution.Interval != "5m" {
		t.Fatalf("unexpected execution market override: %+v", cfg.Execution)
	}
	if cfg.Execution.BaseAsset != "ETH" || cfg.Execution.QuoteAsset != "USDT" {
		t.Fatalf("unexpected execution asset override: %+v", cfg.Execution)
	}
	if cfg.Execution.QuoteOrderAmount != 250 || cfg.Execution.FeeRate != 0.002 {
		t.Fatalf("unexpected execution numeric override: %+v", cfg.Execution)
	}
	if cfg.Execution.Stream != "stream:test-execution" {
		t.Fatalf("expected execution stream override, got %q", cfg.Execution.Stream)
	}
	if !cfg.Orchestration.Enabled {
		t.Fatal("expected orchestration enabled")
	}
	if cfg.Orchestration.ConsumerGroup != "test-pipeline" || cfg.Orchestration.ConsumerName != "test-worker" {
		t.Fatalf("unexpected orchestration overrides: %+v", cfg.Orchestration)
	}
	if !cfg.Auth.Enabled {
		t.Fatal("expected auth enabled")
	}
	if cfg.Auth.AdminAPIKey != "secret-admin-key" {
		t.Fatalf("expected admin API key override, got %q", cfg.Auth.AdminAPIKey)
	}
	if cfg.Security.MaxRequestBodyBytes != 4096 {
		t.Fatalf("expected max body override 4096, got %d", cfg.Security.MaxRequestBodyBytes)
	}
	if cfg.Security.RateLimitRequestsPerMinute != 10 {
		t.Fatalf("expected rate limit override 10, got %d", cfg.Security.RateLimitRequestsPerMinute)
	}
}

func TestLoadRejectsEnabledAuthWithoutAPIKey(t *testing.T) {
	t.Setenv("AUTH_ENABLED", "true")
	t.Setenv("ADMIN_API_KEY", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected enabled auth without admin API key to fail")
	}
}

func TestLoadRejectsProductionWithoutAuth(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_ENABLED", "false")

	_, err := Load()
	if err == nil {
		t.Fatal("expected production without auth to fail")
	}
}

func TestLoadRejectsInvalidRedisDB(t *testing.T) {
	t.Setenv("REDIS_DB", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected invalid Redis DB to return an error")
	}
}

func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
	t.Setenv("SHUTDOWN_TIMEOUT", "slow")

	_, err := Load()
	if err == nil {
		t.Fatal("expected invalid shutdown timeout to return an error")
	}
}

func TestLoadRejectsInvalidMarketDataEnabled(t *testing.T) {
	t.Setenv("MARKET_DATA_ENABLED", "sometimes")

	_, err := Load()
	if err == nil {
		t.Fatal("expected invalid market data enabled flag to return an error")
	}
}
