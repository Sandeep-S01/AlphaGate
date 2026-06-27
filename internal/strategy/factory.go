package strategy

import (
	"encoding/json"
	"fmt"

	"sentra/internal/pine"
)

func NewEvaluatorFromSettings(settings Settings) (Evaluator, error) {
	settings = settings.Normalized()
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	switch settings.StrategyName {
	case StrategySMACrossover:
		return NewSMACrossover(SMAConfig{
			Name:       settings.StrategyName,
			Version:    settings.Version,
			Symbol:     settings.Symbol,
			Interval:   settings.Interval,
			FastPeriod: settings.FastPeriod,
			SlowPeriod: settings.SlowPeriod,
		}), nil
	case StrategyRSIMeanReversion:
		return NewRSIMeanReversion(RSIConfig{
			Name:       settings.StrategyName,
			Version:    settings.Version,
			Symbol:     settings.Symbol,
			Interval:   settings.Interval,
			Period:     settings.RSIPeriod,
			Oversold:   settings.RSIOversold,
			Overbought: settings.RSIOverbought,
		}), nil
	case StrategyBTCTrendPullback:
		return NewBTCTrendPullback(TrendPullbackConfig{
			Name:             settings.StrategyName,
			Version:          settings.Version,
			Symbol:           settings.Symbol,
			Interval:         settings.Interval,
			PullbackPeriod:   settings.FastPeriod,
			TrendPeriod:      settings.SlowPeriod,
			RSIPeriod:        settings.RSIPeriod,
			VolatilityFilter: true,
			ATRPeriod:        DefaultTrendPullbackATRPeriod,
			MinATRPercent:    DefaultTrendPullbackMinATRPercent,
			MaxATRPercent:    DefaultTrendPullbackMaxATRPercent,
		}), nil
	case StrategyPineCustom:
		var cfg pine.IRConfig
		if settings.PineConfig != nil {
			if err := json.Unmarshal([]byte(*settings.PineConfig), &cfg); err != nil {
				return nil, fmt.Errorf("unmarshal pine config: %w", err)
			}
		}
		return NewPineEvaluator(settings.StrategyName, settings.Version, settings.Symbol, settings.Interval, cfg), nil
	default:
		if cfg, found, err := TemplateExecutionConfig(settings.StrategyName); found {
			if err != nil {
				return nil, err
			}
			return NewPineEvaluator(settings.StrategyName, settings.Version, settings.Symbol, settings.Interval, cfg), nil
		}
		return nil, fmt.Errorf("unsupported strategy_name %q", settings.StrategyName)
	}
}
