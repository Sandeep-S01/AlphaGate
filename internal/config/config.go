package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

const defaultPostgresDSN = "postgres://sentra:sentra_password@localhost:5432/sentra?sslmode=disable"

type Config struct {
	App           AppConfig           `validate:"required"`
	HTTP          HTTPConfig          `validate:"required"`
	Postgres      PostgresConfig      `validate:"required"`
	Redis         RedisConfig         `validate:"required"`
	Logging       LoggingConfig       `validate:"required"`
	Binance       BinanceConfig       `validate:"required"`
	MarketData    MarketDataConfig    `validate:"required"`
	Strategy      StrategyConfig      `validate:"required"`
	Risk          RiskConfig          `validate:"required"`
	Execution     ExecutionConfig     `validate:"required"`
	Orchestration OrchestrationConfig `validate:"required"`
	Auth          AuthConfig          `validate:"required"`
	Security      SecurityConfig      `validate:"required"`
}

type AppConfig struct {
	Env             string        `validate:"required,oneof=local development test staging production"`
	ShutdownTimeout time.Duration `validate:"gt=0"`
}

type HTTPConfig struct {
	Addr string `validate:"required,hostname_port"`
}

type PostgresConfig struct {
	DSN               string `validate:"required,uri"`
	MaxConns          int    `validate:"gte=0"`
	MinConns          int    `validate:"gte=0"`
	MaxConnLifetime   string `validate:"omitempty"` // duration string
	MaxConnIdleTime   string `validate:"omitempty"` // duration string
	HealthCheckPeriod string `validate:"omitempty"` // duration string
}

type RedisConfig struct {
	Addr     string `validate:"required,hostname_port"`
	Password string
	DB       int `validate:"min=0"`
}

type LoggingConfig struct {
	Level string `validate:"required,oneof=debug info warn error"`
}

type BinanceConfig struct {
	RESTBaseURL string `validate:"required,url"`
	WSBaseURL   string `validate:"required,wsurl"`
}

type MarketDataConfig struct {
	Enabled            bool
	PersistenceEnabled bool
	Symbol             string        `validate:"required,uppercase,min=3,max=20"`
	Interval           string        `validate:"required,interval"`
	RedisStream        string        `validate:"required"`
	ConsumerGroup      string        `validate:"required"`
	ConsumerName       string        `validate:"required"`
	MaxReconnects      int           `validate:"min=0"`
	ReconnectDelay     time.Duration `validate:"gt=0"`
}

type StrategyConfig struct {
	Enabled       bool
	Name          string  `validate:"required"`
	Version       string  `validate:"required"`
	Symbol        string  `validate:"required,uppercase,min=3,max=20"`
	Interval      string  `validate:"required,interval"`
	FastPeriod    int     `validate:"required,gt=0"`
	SlowPeriod    int     `validate:"required,gt=0,fieldslowperiod"`
	LookbackLimit int     `validate:"required,gt=0"`
	RSIPeriod     int     `validate:"required,gte=0,lte=100"`
	RSIOversold   float64 `validate:"required,gte=0,lte=100"`
	RSIOverbought float64 `validate:"required,gte=0,lte=100"`
	SignalStream  string  `validate:"required"`
}

type RiskConfig struct {
	Enabled           bool
	MaxSignalStrength float64 `validate:"required,gte=0,lte=100"`
	AllowBuy          bool
	AllowSell         bool
	Symbol            string `validate:"required,uppercase,min=3,max=20"`
	DecisionStream    string `validate:"required"`
}

type ExecutionConfig struct {
	Enabled          bool
	Symbol           string  `validate:"required,uppercase,min=3,max=20"`
	Interval         string  `validate:"required,interval"`
	BaseAsset        string  `validate:"required,uppercase,min=2,max=10"`
	QuoteAsset       string  `validate:"required,uppercase,min=2,max=10"`
	QuoteOrderAmount float64 `validate:"required,gt=0"`
	FeeRate          float64 `validate:"required,gte=0,lte=1"`
	Stream           string  `validate:"required"`
}

type OrchestrationConfig struct {
	Enabled       bool
	ConsumerGroup string `validate:"required"`
	ConsumerName  string `validate:"required"`
}

type AuthConfig struct {
	Enabled     bool
	AdminAPIKey string `validate:"ifemptyfield=Enabled"`
}

type SecurityConfig struct {
	MaxRequestBodyBytes        int64 `validate:"gte=0"`
	RateLimitRequestsPerMinute int   `validate:"gte=0"`
	// CORS settings
	CORSEnabled          bool
	CORSAllowedOrigins   []string `validate:"omitempty,dive,url"`
	CORSAllowedMethods   []string `validate:"omitempty,dive,oneof=GET POST PUT DELETE PATCH OPTIONS"`
	CORSAllowedHeaders   []string `validate:"omitempty,dive"`
	CORSExposedHeaders   []string `validate:"omitempty,dive"`
	CORSAllowCredentials bool
	CORSMaxAge           int `validate:"gte=0"`
	// Security headers configurability
	DisableSecurityHeaders bool
	CustomSecurityHeaders  map[string]string `validate:"omitempty,dive,keys,required,endkeys,required"`
}

func Load() (Config, error) {
	redisDB, err := intFromEnv("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}

	marketDataEnabled, err := boolFromEnv("MARKET_DATA_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	marketDataPersistenceEnabled, err := boolFromEnv("MARKET_DATA_PERSISTENCE_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	marketDataMaxReconnects, err := intFromEnv("MARKET_DATA_MAX_RECONNECTS", 100)
	if err != nil {
		return Config{}, err
	}
	marketDataReconnectDelay, err := durationFromEnv("MARKET_DATA_RECONNECT_DELAY", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	strategyEnabled, err := boolFromEnv("STRATEGY_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	strategyFastPeriod, err := intFromEnv("STRATEGY_FAST_PERIOD", 9)
	if err != nil {
		return Config{}, err
	}
	strategySlowPeriod, err := intFromEnv("STRATEGY_SLOW_PERIOD", 21)
	if err != nil {
		return Config{}, err
	}
	strategyLookbackLimit, err := intFromEnv("STRATEGY_LOOKBACK_LIMIT", 100)
	if err != nil {
		return Config{}, err
	}
	strategyRSIPeriod, err := intFromEnv("STRATEGY_RSI_PERIOD", 14)
	if err != nil {
		return Config{}, err
	}
	strategyRSIOversold, err := floatFromEnv("STRATEGY_RSI_OVERSOLD", 30)
	if err != nil {
		return Config{}, err
	}
	strategyRSIOverbought, err := floatFromEnv("STRATEGY_RSI_OVERBOUGHT", 70)
	if err != nil {
		return Config{}, err
	}
	riskEnabled, err := boolFromEnv("RISK_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	riskMaxSignalStrength, err := floatFromEnv("RISK_MAX_SIGNAL_STRENGTH", 100)
	if err != nil {
		return Config{}, err
	}
	riskAllowBuy, err := boolFromEnv("RISK_ALLOW_BUY", true)
	if err != nil {
		return Config{}, err
	}
	riskAllowSell, err := boolFromEnv("RISK_ALLOW_SELL", true)
	if err != nil {
		return Config{}, err
	}
	executionEnabled, err := boolFromEnv("EXECUTION_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	executionQuoteOrderAmount, err := floatFromEnv("EXECUTION_QUOTE_ORDER_AMOUNT", 100)
	if err != nil {
		return Config{}, err
	}
	executionFeeRate, err := floatFromEnv("EXECUTION_FEE_RATE", 0.001)
	if err != nil {
		return Config{}, err
	}
	orchestrationEnabled, err := boolFromEnv("ORCHESTRATION_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	authEnabled, err := boolFromEnv("AUTH_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	adminAPIKey := os.Getenv("ADMIN_API_KEY")
	if authEnabled && adminAPIKey == "" {
		return Config{}, fmt.Errorf("ADMIN_API_KEY is required when AUTH_ENABLED is true")
	}
	if appEnv := stringFromEnv("APP_ENV", "local"); appEnv == "production" && !authEnabled {
		return Config{}, fmt.Errorf("AUTH_ENABLED must be true in production")
	}
	maxRequestBodyBytes, err := int64FromEnv("MAX_REQUEST_BODY_BYTES", 1_048_576)
	if err != nil {
		return Config{}, err
	}
	rateLimitRequestsPerMinute, err := intFromEnv("RATE_LIMIT_REQUESTS_PER_MINUTE", 120)
	if err != nil {
		return Config{}, err
	}
	postgresMaxConns, err := intFromEnv("POSTGRES_MAX_CONNS", 10)
	if err != nil {
		return Config{}, err
	}
	postgresMinConns, err := intFromEnv("POSTGRES_MIN_CONNS", 1)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := durationFromEnv("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		App: AppConfig{
			Env:             stringFromEnv("APP_ENV", "local"),
			ShutdownTimeout: shutdownTimeout,
		},
		HTTP: HTTPConfig{
			Addr: stringFromEnv("HTTP_ADDR", ":8080"),
		},
		Postgres: PostgresConfig{
			DSN:               stringFromEnv("POSTGRES_DSN", defaultPostgresDSN),
			MaxConns:          postgresMaxConns,
			MinConns:          postgresMinConns,
			MaxConnLifetime:   stringFromEnv("POSTGRES_MAX_CONN_LIFETIME", "1h"),
			MaxConnIdleTime:   stringFromEnv("POSTGRES_MAX_CONN_IDLE_TIME", "30m"),
			HealthCheckPeriod: stringFromEnv("POSTGRES_HEALTH_CHECK_PERIOD", "1m"),
		},
		Redis: RedisConfig{
			Addr:     stringFromEnv("REDIS_ADDR", "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
		Logging: LoggingConfig{
			Level: stringFromEnv("LOG_LEVEL", "info"),
		},
		Binance: BinanceConfig{
			RESTBaseURL: stringFromEnv("BINANCE_REST_BASE_URL", "https://api.binance.com"),
			WSBaseURL:   stringFromEnv("BINANCE_WS_BASE_URL", "wss://stream.binance.com:9443/ws"),
		},
		MarketData: MarketDataConfig{
			Enabled:            marketDataEnabled,
			PersistenceEnabled: marketDataPersistenceEnabled,
			Symbol:             stringFromEnv("MARKET_DATA_SYMBOL", "BTCUSDT"),
			Interval:           stringFromEnv("MARKET_DATA_INTERVAL", "1m"),
			RedisStream:        stringFromEnv("MARKET_DATA_REDIS_STREAM", "stream:market-data"),
			ConsumerGroup:      stringFromEnv("MARKET_DATA_CONSUMER_GROUP", "market-data-persistence"),
			ConsumerName:       stringFromEnv("MARKET_DATA_CONSUMER_NAME", "worker-1"),
			MaxReconnects:      marketDataMaxReconnects,
			ReconnectDelay:     marketDataReconnectDelay,
		},
		Strategy: StrategyConfig{
			Enabled:       strategyEnabled,
			Name:          stringFromEnv("STRATEGY_NAME", "sma-crossover"),
			Version:       stringFromEnv("STRATEGY_VERSION", "v1"),
			Symbol:        stringFromEnv("STRATEGY_SYMBOL", "BTCUSDT"),
			Interval:      stringFromEnv("STRATEGY_INTERVAL", "1m"),
			FastPeriod:    strategyFastPeriod,
			SlowPeriod:    strategySlowPeriod,
			LookbackLimit: strategyLookbackLimit,
			RSIPeriod:     strategyRSIPeriod,
			RSIOversold:   strategyRSIOversold,
			RSIOverbought: strategyRSIOverbought,
			SignalStream:  stringFromEnv("STRATEGY_SIGNAL_STREAM", "stream:strategy-signals"),
		},
		Risk: RiskConfig{
			Enabled:           riskEnabled,
			MaxSignalStrength: riskMaxSignalStrength,
			AllowBuy:          riskAllowBuy,
			AllowSell:         riskAllowSell,
			Symbol:            stringFromEnv("RISK_SYMBOL", "BTCUSDT"),
			DecisionStream:    stringFromEnv("RISK_DECISION_STREAM", "stream:risk-decisions"),
		},
		Execution: ExecutionConfig{
			Enabled:          executionEnabled,
			Symbol:           stringFromEnv("EXECUTION_SYMBOL", "BTCUSDT"),
			Interval:         stringFromEnv("EXECUTION_INTERVAL", "1m"),
			BaseAsset:        stringFromEnv("EXECUTION_BASE_ASSET", "BTC"),
			QuoteAsset:       stringFromEnv("EXECUTION_QUOTE_ASSET", "USDT"),
			QuoteOrderAmount: executionQuoteOrderAmount,
			FeeRate:          executionFeeRate,
			Stream:           stringFromEnv("EXECUTION_STREAM", "stream:execution-results"),
		},
		Orchestration: OrchestrationConfig{
			Enabled:       orchestrationEnabled,
			ConsumerGroup: stringFromEnv("ORCHESTRATION_CONSUMER_GROUP", "paper-pipeline"),
			ConsumerName:  stringFromEnv("ORCHESTRATION_CONSUMER_NAME", "worker-1"),
		},
		Auth: AuthConfig{
			Enabled:     authEnabled,
			AdminAPIKey: adminAPIKey,
		},
		Security: SecurityConfig{
			MaxRequestBodyBytes:        maxRequestBodyBytes,
			RateLimitRequestsPerMinute: rateLimitRequestsPerMinute,
		},
	}

	// Validate configuration using struct tags
	validate := validator.New()
	// Register custom validation functions
	_ = validate.RegisterValidation("hostname_port", validateHostnamePort)
	_ = validate.RegisterValidation("uri", validateURI)
	_ = validate.RegisterValidation("wsurl", validateWSURL)
	_ = validate.RegisterValidation("interval", validateInterval)
	_ = validate.RegisterValidation("fieldslowperiod", validateSlowPeriod)
	_ = validate.RegisterValidation("ifemptyfield", validateIfEmptyField)

	if err := validate.Struct(cfg); err != nil {
		return Config{}, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

func validateHostnamePort(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}
	// Basic hostname:port validation - hostname can contain letters, numbers, hyphens, dots
	// Port should be 1-65535
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return false
	}
	hostname := parts[0]
	portStr := parts[1]

	// Empty host is valid for bind addresses such as ":8080".
	if hostname != "" {
		for _, r := range hostname {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '.') {
				return false
			}
		}
	}

	// Validate port
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return false
	}
	return true
}

func validateURI(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}
	// Basic URI validation - should start with a scheme
	return strings.Contains(value, "://")
}

func validateWSURL(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}
	// WebSocket URL validation - should start with ws:// or wss://
	return strings.HasPrefix(value, "ws://") || strings.HasPrefix(value, "wss://")
}

func validateInterval(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}
	// Basic interval validation - number followed by time unit (s, m, h, d)
	if len(value) < 2 {
		return false
	}
	unit := value[len(value)-1:]
	if _, err := strconv.Atoi(value[:len(value)-1]); err != nil {
		return false
	}
	return unit == "s" || unit == "m" || unit == "h" || unit == "d"
}

func validateSlowPeriod(fl validator.FieldLevel) bool {
	fastPeriod := fl.Parent().FieldByName("FastPeriod").Int()
	slowPeriod := fl.Field().Int()
	return slowPeriod > fastPeriod
}

func validateIfEmptyField(fl validator.FieldLevel) bool {
	// This validates that a field is empty if another field is false
	// We'll enable this field if the referenced field is true
	enabledField := fl.Parent().FieldByName("Enabled")
	if enabledField.Kind() == reflect.Bool && enabledField.Bool() {
		// If enabled, the field should not be empty
		return fl.Field().String() != ""
	}
	// If not enabled, field can be empty
	return true
}

func stringFromEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func intFromEnv(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}

func int64FromEnv(key string, fallback int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}

func boolFromEnv(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}

func floatFromEnv(key string, fallback float64) (float64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}

// Helper functions for config parsing
func csvToStrings(csv string) []string {
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func stringToMap(s string) map[string]string {
	if s == "" {
		return nil
	}
	// Format: key1=value1,key2=value2
	pairs := strings.Split(s, ",")
	result := make(map[string]string)
	for _, pair := range pairs {
		if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}
