package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sentra/internal/pine"
)

type Settings struct {
	StrategyName   string    `json:"strategy_name"`
	Version        string    `json:"version"`
	Symbol         string    `json:"symbol"`
	Interval       string    `json:"interval"`
	FastPeriod     int       `json:"fast_period"`
	SlowPeriod     int       `json:"slow_period"`
	LookbackLimit  int       `json:"lookback_limit"`
	RSIPeriod      int       `json:"rsi_period"`
	RSIOversold    float64   `json:"rsi_oversold"`
	RSIOverbought  float64   `json:"rsi_overbought"`
	PineStrategyID *string   `json:"pine_strategy_id,omitempty"`
	PineConfig     *string   `json:"pine_config,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SettingsRepository struct {
	db QueryRower
}

func DefaultSettings() Settings {
	return Settings{
		StrategyName:  StrategySMACrossover,
		Version:       "v1",
		Symbol:        "BTCUSDT",
		Interval:      "1m",
		FastPeriod:    9,
		SlowPeriod:    21,
		LookbackLimit: 100,
		RSIPeriod:     14,
		RSIOversold:   30,
		RSIOverbought: 70,
	}
}

func NewSettingsRepository(db QueryRower) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (s Settings) Validate() error {
	if strings.TrimSpace(s.StrategyName) == "" {
		return fmt.Errorf("strategy_name is required")
	}
	if strings.TrimSpace(s.Version) == "" {
		return fmt.Errorf("version is required")
	}
	if strings.TrimSpace(s.Symbol) == "" {
		return fmt.Errorf("symbol is required")
	}
	if strings.TrimSpace(s.Interval) == "" {
		return fmt.Errorf("interval is required")
	}
	switch s.StrategyName {
	case StrategySMACrossover, StrategyBTCTrendPullback:
		if s.FastPeriod <= 0 {
			return fmt.Errorf("fast_period must be positive")
		}
		if s.SlowPeriod <= s.FastPeriod {
			return fmt.Errorf("slow_period must be greater than fast_period")
		}
		if s.LookbackLimit < s.SlowPeriod {
			return fmt.Errorf("lookback_limit must be greater than or equal to slow_period")
		}
		if s.StrategyName == StrategyBTCTrendPullback {
			if s.RSIPeriod <= 0 {
				return fmt.Errorf("rsi_period must be positive")
			}
			if s.LookbackLimit < s.RSIPeriod+2 {
				return fmt.Errorf("lookback_limit must cover current and previous RSI windows")
			}
		}
	case StrategyRSIMeanReversion:
		if s.RSIPeriod <= 0 {
			return fmt.Errorf("rsi_period must be positive")
		}
		if s.RSIOversold <= 0 || s.RSIOverbought <= s.RSIOversold || s.RSIOverbought >= 100 {
			return fmt.Errorf("RSI thresholds are invalid")
		}
		if s.LookbackLimit < s.RSIPeriod+1 {
			return fmt.Errorf("lookback_limit must be greater than rsi_period")
		}
	case StrategyPineCustom:
		if s.PineConfig == nil || strings.TrimSpace(*s.PineConfig) == "" {
			return fmt.Errorf("pine_config is required for custom Pine strategy")
		}
		var cfg pine.IRConfig
		if err := json.Unmarshal([]byte(*s.PineConfig), &cfg); err != nil {
			return fmt.Errorf("invalid pine_config JSON: %w", err)
		}
	default:
		if _, found, err := TemplateExecutionConfig(s.StrategyName); found {
			return err
		}
		return fmt.Errorf("unsupported strategy_name %q", s.StrategyName)
	}
	return nil
}

func (s Settings) RequiredCandles() int {
	switch s.StrategyName {
	case StrategyRSIMeanReversion:
		return s.RSIPeriod + 1
	case StrategyBTCTrendPullback:
		required := s.SlowPeriod + 1
		if s.RSIPeriod+2 > required {
			required = s.RSIPeriod + 2
		}
		if DefaultTrendPullbackATRPeriod+1 > required {
			required = DefaultTrendPullbackATRPeriod + 1
		}
		return required
	case StrategyPineCustom:
		if s.PineConfig == nil || strings.TrimSpace(*s.PineConfig) == "" {
			return 100
		}
		var cfg pine.IRConfig
		if err := json.Unmarshal([]byte(*s.PineConfig), &cfg); err != nil {
			return 100
		}
		return requiredCandlesForPineConfig(cfg)
	default:
		if cfg, found, err := TemplateExecutionConfig(s.StrategyName); found && err == nil {
			return requiredCandlesForPineConfig(cfg)
		}
		return s.SlowPeriod + 1
	}
}

func (s Settings) Normalized() Settings {
	s.StrategyName = strings.TrimSpace(s.StrategyName)
	s.Version = strings.TrimSpace(s.Version)
	s.Symbol = strings.ToUpper(strings.TrimSpace(s.Symbol))
	s.Interval = strings.TrimSpace(s.Interval)
	return s
}

func requiredCandlesForPineConfig(cfg pine.IRConfig) int {
	maxPeriod := 0
	hasExponential := false
	for _, ind := range cfg.Indicators {
		if ind.Type == "rsi" || ind.Type == "ema" {
			hasExponential = true
		}
		for _, p := range ind.Params {
			if int(p) > maxPeriod {
				maxPeriod = int(p)
			}
		}
	}
	if maxPeriod <= 0 {
		maxPeriod = 14
	}
	if hasExponential {
		return maxPeriod + 250
	}
	return maxPeriod + 2
}

func (r *SettingsRepository) Get(ctx context.Context) (Settings, error) {
	query := `
SELECT strategy_name, version, symbol, interval, fast_period, slow_period, lookback_limit,
       rsi_period, rsi_oversold, rsi_overbought, updated_at
FROM strategy_settings
WHERE strategy_name IN ('sma-crossover', 'rsi-mean-reversion', 'btc-trend-pullback')
ORDER BY updated_at DESC
LIMIT 1`

	var settings Settings
	if err := r.db.QueryRow(ctx, query).Scan(
		&settings.StrategyName,
		&settings.Version,
		&settings.Symbol,
		&settings.Interval,
		&settings.FastPeriod,
		&settings.SlowPeriod,
		&settings.LookbackLimit,
		&settings.RSIPeriod,
		&settings.RSIOversold,
		&settings.RSIOverbought,
		&settings.UpdatedAt,
	); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func (r *SettingsRepository) Save(ctx context.Context, settings Settings) (Settings, error) {
	settings = settings.Normalized()
	if err := settings.Validate(); err != nil {
		return Settings{}, err
	}
	query, args := BuildUpsertSettingsSQL(settings)
	var saved Settings
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&saved.StrategyName,
		&saved.Version,
		&saved.Symbol,
		&saved.Interval,
		&saved.FastPeriod,
		&saved.SlowPeriod,
		&saved.LookbackLimit,
		&saved.RSIPeriod,
		&saved.RSIOversold,
		&saved.RSIOverbought,
		&saved.UpdatedAt,
	); err != nil {
		return Settings{}, err
	}
	return saved, nil
}

func BuildUpsertSettingsSQL(settings Settings) (string, []any) {
	return `
INSERT INTO strategy_settings (
    strategy_name,
    version,
    symbol,
    interval,
    fast_period,
    slow_period,
    lookback_limit,
    rsi_period,
    rsi_oversold,
    rsi_overbought,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
ON CONFLICT (strategy_name)
DO UPDATE SET
    version = EXCLUDED.version,
    symbol = EXCLUDED.symbol,
    interval = EXCLUDED.interval,
    fast_period = EXCLUDED.fast_period,
    slow_period = EXCLUDED.slow_period,
    lookback_limit = EXCLUDED.lookback_limit,
    rsi_period = EXCLUDED.rsi_period,
    rsi_oversold = EXCLUDED.rsi_oversold,
    rsi_overbought = EXCLUDED.rsi_overbought,
    updated_at = NOW()
RETURNING strategy_name, version, symbol, interval, fast_period, slow_period, lookback_limit,
          rsi_period, rsi_oversold, rsi_overbought, updated_at`, []any{
			settings.StrategyName,
			settings.Version,
			settings.Symbol,
			settings.Interval,
			settings.FastPeriod,
			settings.SlowPeriod,
			settings.LookbackLimit,
			settings.RSIPeriod,
			settings.RSIOversold,
			settings.RSIOverbought,
		}
}
