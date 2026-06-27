package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type SettingsRepository struct {
	db QueryRower
}

func DefaultSettings() Settings {
	return Settings{
		Enabled:                     true,
		MaxSignalStrength:           100,
		MinSignalStrength:           0,
		MaxQuoteAmount:              0,
		MaxOrderQuoteAmount:         0,
		MaxPositionQuoteAmount:      0,
		MaxTotalExposureQuoteAmount: 0,
		MaxOpenPositions:            0,
		MaxDailyLoss:                0,
		MaxDailyTrades:              0,
		AllowBuy:                    true,
		AllowSell:                   true,
		AllowedSymbols:              []string{},
		Cooldown:                    0,
		CooldownSeconds:             0,
	}
}

func NewSettingsRepository(db QueryRower) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (s Settings) Normalize() Settings {
	if s.Cooldown == 0 && s.CooldownSeconds > 0 {
		s.Cooldown = time.Duration(s.CooldownSeconds) * time.Second
	}
	if s.Cooldown > 0 {
		s.CooldownSeconds = int64(s.Cooldown.Seconds())
	}
	return s
}

func (s Settings) Validate() error {
	if s.MaxSignalStrength < 0 {
		return fmt.Errorf("max_signal_strength cannot be negative")
	}
	if s.MinSignalStrength < 0 {
		return fmt.Errorf("min_signal_strength cannot be negative")
	}
	if s.MaxSignalStrength > 0 && s.MinSignalStrength > s.MaxSignalStrength {
		return fmt.Errorf("min_signal_strength cannot exceed max_signal_strength")
	}
	if s.MaxQuoteAmount < 0 {
		return fmt.Errorf("max_quote_amount cannot be negative")
	}
	if s.MaxOrderQuoteAmount < 0 {
		return fmt.Errorf("max_order_quote_amount cannot be negative")
	}
	if s.MaxPositionQuoteAmount < 0 {
		return fmt.Errorf("max_position_quote_amount cannot be negative")
	}
	if s.MaxTotalExposureQuoteAmount < 0 {
		return fmt.Errorf("max_total_exposure_quote_amount cannot be negative")
	}
	if s.MaxOpenPositions < 0 {
		return fmt.Errorf("max_open_positions cannot be negative")
	}
	if s.MaxDailyLoss < 0 {
		return fmt.Errorf("max_daily_loss cannot be negative")
	}
	if s.MaxDailyTrades < 0 {
		return fmt.Errorf("max_daily_trades cannot be negative")
	}
	if s.Cooldown < 0 || s.CooldownSeconds < 0 {
		return fmt.Errorf("cooldown_seconds cannot be negative")
	}
	return nil
}

func (s Settings) Config() Config {
	s = s.Normalize()
	return Config{
		Enabled:                     s.Enabled,
		MaxSignalStrength:           s.MaxSignalStrength,
		MinSignalStrength:           s.MinSignalStrength,
		MaxQuoteAmount:              s.MaxQuoteAmount,
		MaxOrderQuoteAmount:         s.MaxOrderQuoteAmount,
		MaxPositionQuoteAmount:      s.MaxPositionQuoteAmount,
		MaxTotalExposureQuoteAmount: s.MaxTotalExposureQuoteAmount,
		MaxOpenPositions:            s.MaxOpenPositions,
		MaxDailyLoss:                s.MaxDailyLoss,
		MaxDailyTrades:              s.MaxDailyTrades,
		AllowBuy:                    s.AllowBuy,
		AllowSell:                   s.AllowSell,
		AllowedSymbols:              s.AllowedSymbols,
		Cooldown:                    s.Cooldown,
	}
}

func (r *SettingsRepository) Get(ctx context.Context) (Settings, error) {
	query := `
SELECT enabled, max_signal_strength, min_signal_strength, max_quote_amount, max_daily_loss,
       max_daily_trades, allow_buy, allow_sell, cooldown_seconds, updated_at,
       COALESCE(allowed_symbols::text, '[]'),
       max_order_quote_amount, max_position_quote_amount, max_total_exposure_quote_amount, max_open_positions
FROM risk_settings
WHERE id = 1`
	var settings Settings
	var allowedSymbolsJSON string
	if err := r.db.QueryRow(ctx, query).Scan(
		&settings.Enabled,
		&settings.MaxSignalStrength,
		&settings.MinSignalStrength,
		&settings.MaxQuoteAmount,
		&settings.MaxDailyLoss,
		&settings.MaxDailyTrades,
		&settings.AllowBuy,
		&settings.AllowSell,
		&settings.CooldownSeconds,
		&settings.UpdatedAt,
		&allowedSymbolsJSON,
		&settings.MaxOrderQuoteAmount,
		&settings.MaxPositionQuoteAmount,
		&settings.MaxTotalExposureQuoteAmount,
		&settings.MaxOpenPositions,
	); err != nil {
		return Settings{}, err
	}
	_ = json.Unmarshal([]byte(allowedSymbolsJSON), &settings.AllowedSymbols)
	return settings.Normalize(), nil
}

func (r *SettingsRepository) Save(ctx context.Context, settings Settings) (Settings, error) {
	settings = settings.Normalize()
	if err := settings.Validate(); err != nil {
		return Settings{}, err
	}
	query, args := BuildUpsertSettingsSQL(settings)
	var saved Settings
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&saved.Enabled,
		&saved.MaxSignalStrength,
		&saved.MinSignalStrength,
		&saved.MaxQuoteAmount,
		&saved.MaxDailyLoss,
		&saved.MaxDailyTrades,
		&saved.AllowBuy,
		&saved.AllowSell,
		&saved.CooldownSeconds,
		&saved.UpdatedAt,
	); err != nil {
		return Settings{}, err
	}
	return saved.Normalize(), nil
}

func BuildUpsertSettingsSQL(settings Settings) (string, []any) {
	settings = settings.Normalize()
	allowedSymbolsJSON, _ := json.Marshal(settings.AllowedSymbols)
	return `
INSERT INTO risk_settings (
    id,
    enabled,
    max_quote_amount,
    max_signal_strength,
    min_signal_strength,
    max_daily_trades,
    max_daily_loss,
    allow_buy,
    cooldown_seconds,
    allow_sell,
    allowed_symbols,
    max_order_quote_amount,
    max_position_quote_amount,
    max_total_exposure_quote_amount,
    max_open_positions,
    updated_at
) VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13, $14, NOW())
ON CONFLICT (id)
DO UPDATE SET
    enabled = EXCLUDED.enabled,
    max_quote_amount = EXCLUDED.max_quote_amount,
    max_signal_strength = EXCLUDED.max_signal_strength,
    min_signal_strength = EXCLUDED.min_signal_strength,
    max_daily_trades = EXCLUDED.max_daily_trades,
    max_daily_loss = EXCLUDED.max_daily_loss,
    allow_buy = EXCLUDED.allow_buy,
    cooldown_seconds = EXCLUDED.cooldown_seconds,
    allow_sell = EXCLUDED.allow_sell,
    allowed_symbols = EXCLUDED.allowed_symbols,
    max_order_quote_amount = EXCLUDED.max_order_quote_amount,
    max_position_quote_amount = EXCLUDED.max_position_quote_amount,
    max_total_exposure_quote_amount = EXCLUDED.max_total_exposure_quote_amount,
    max_open_positions = EXCLUDED.max_open_positions,
    updated_at = NOW()
RETURNING enabled, max_signal_strength, min_signal_strength, max_quote_amount, max_daily_loss,
          max_daily_trades, allow_buy, allow_sell, cooldown_seconds, updated_at`, []any{
			settings.Enabled,
			settings.MaxQuoteAmount,
			settings.MaxSignalStrength,
			settings.MinSignalStrength,
			settings.MaxDailyTrades,
			settings.MaxDailyLoss,
			settings.AllowBuy,
			int64(settings.Cooldown.Seconds()),
			settings.AllowSell,
			string(allowedSymbolsJSON),
			settings.MaxOrderQuoteAmount,
			settings.MaxPositionQuoteAmount,
			settings.MaxTotalExposureQuoteAmount,
			settings.MaxOpenPositions,
		}
}
