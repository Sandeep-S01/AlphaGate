package strategy

import (
	"fmt"
	"strings"

	"sentra/internal/pine"
)

type TemplateSupportStatus string

const (
	TemplateExecutableNative TemplateSupportStatus = "executable_native"
	TemplateExecutablePine   TemplateSupportStatus = "executable_pine"
	TemplateTemplateOnly     TemplateSupportStatus = "template_only"
	TemplateBlockedByData    TemplateSupportStatus = "blocked_by_data"
)

type StrategyTemplate struct {
	ID               string                   `json:"id"`
	Name             string                   `json:"name"`
	Category         string                   `json:"category"`
	Market           string                   `json:"market"`
	TimeHorizon      string                   `json:"time_horizon"`
	Summary          string                   `json:"summary"`
	SupportStatus    TemplateSupportStatus    `json:"support_status"`
	NativeStrategy   string                   `json:"native_strategy,omitempty"`
	RequiredData     []string                 `json:"required_data"`
	Indicators       []string                 `json:"indicators"`
	EntryRules       []string                 `json:"entry_rules"`
	ExitRules        []string                 `json:"exit_rules"`
	RiskRules        []string                 `json:"risk_rules"`
	DefaultSettings  Settings                 `json:"default_settings"`
	ExecutionProfile TemplateExecutionProfile `json:"execution_profile"`
	PineCode         string                   `json:"pine_code,omitempty"`
	Blockers         []string                 `json:"blockers,omitempty"`
}

type TemplateExecutionProfile struct {
	RecommendedInterval     string  `json:"recommended_interval"`
	CooldownBars            int     `json:"cooldown_bars"`
	MinHoldingBars          int     `json:"min_holding_bars"`
	ATRExitEnabled          bool    `json:"atr_exit_enabled"`
	ATRPeriod               int     `json:"atr_period"`
	ATRStopMultiplier       float64 `json:"atr_stop_multiplier"`
	ATRTakeProfitMultiplier float64 `json:"atr_take_profit_multiplier"`
	MaxTradesPerDay         float64 `json:"max_trades_per_day"`
	ShortingEnabled         bool    `json:"shorting_enabled"`
	RegimeFilterEnabled     bool    `json:"regime_filter_enabled"`
	RegimeFilterPeriod      int     `json:"regime_filter_period"`
	RegimeMinATRPercent     float64 `json:"regime_min_atr_percent"`
	RegimeMaxATRPercent     float64 `json:"regime_max_atr_percent"`
	PositionSizePercent     float64 `json:"position_size_percent"`
}

func PredefinedTemplates() []StrategyTemplate {
	templates := predefinedTemplates()
	out := make([]StrategyTemplate, len(templates))
	copy(out, templates)
	return out
}

func GetPredefinedTemplate(id string) (StrategyTemplate, bool) {
	for _, tmpl := range predefinedTemplates() {
		if tmpl.ID == id {
			return tmpl, true
		}
	}
	return StrategyTemplate{}, false
}

func TemplateExecutionConfig(id string) (pine.IRConfig, bool, error) {
	tmpl, ok := GetPredefinedTemplate(strings.TrimSpace(id))
	if !ok {
		return pine.IRConfig{}, false, nil
	}
	if tmpl.SupportStatus != TemplateExecutablePine {
		return pine.IRConfig{}, true, fmt.Errorf("strategy template %q is not executable: support status is %q", id, tmpl.SupportStatus)
	}
	if strings.TrimSpace(tmpl.PineCode) == "" {
		return pine.IRConfig{}, true, fmt.Errorf("strategy template %q does not include executable Pine code", id)
	}
	res := pine.NewParser(tmpl.PineCode).Parse()
	if len(res.Errors) > 0 {
		return pine.IRConfig{}, true, fmt.Errorf("strategy template %q Pine code is invalid: %s", id, strings.Join(res.Errors, "; "))
	}
	return res.Config, true, nil
}

func IsExecutableTemplate(id string) bool {
	tmpl, ok := GetPredefinedTemplate(strings.TrimSpace(id))
	return ok && tmpl.SupportStatus == TemplateExecutablePine
}

func predefinedTemplates() []StrategyTemplate {
	return []StrategyTemplate{
		{
			ID:            StrategyTrendFollowingMTF,
			Name:          "Trend Following (Multi-Timeframe)",
			Category:      "Trend",
			Market:        "All Markets",
			TimeHorizon:   "Swing",
			Summary:       "Captures persistent directional moves using moving-average trend filters, trend strength, breakout confirmation, and ATR risk controls.",
			SupportStatus: TemplateExecutablePine,
			RequiredData:  []string{"OHLCV", "20-period high/low"},
			Indicators:    []string{"EMA(20)", "SMA(50)", "ADX(14)", "ATR(14)", "Volume SMA(20)"},
			EntryRules: []string{
				"Close is above EMA(20) and SMA(50).",
				"ADX(14) is above 25.",
				"Price breaks above the prior 20-period high.",
				"Volume is above 1.5x the 20-period average.",
			},
			ExitRules: []string{
				"Close falls below EMA(20) or SMA(50).",
				"ADX(14) falls below 20.",
				"Trailing stop is hit at roughly 2x ATR from entry.",
			},
			RiskRules: []string{
				"Target 1% to 1.5% account risk per trade.",
				"Stop new risk after 3% daily loss.",
				"Reduce size by 50% at 10% drawdown and stop at 15% drawdown.",
			},
			DefaultSettings: defaultTemplateSettings(StrategyTrendFollowingMTF, 20, 50, 14, 30, 70, 260),
			ExecutionProfile: TemplateExecutionProfile{
				RecommendedInterval:     "15m",
				CooldownBars:            8,
				MinHoldingBars:          4,
				ATRExitEnabled:          true,
				ATRPeriod:               14,
				ATRStopMultiplier:       2,
				ATRTakeProfitMultiplier: 3,
				MaxTradesPerDay:         12,
			},
			PineCode: `ema20 = ta.ema(close, 20)
sma50 = ta.sma(close, 50)
rsi14 = ta.rsi(close, 14)
buy = close > ema20 and close > sma50 and rsi14 > 55
sell = close < ema20 or close < sma50
if buy
    strategy.entry("LONG", strategy.long)
if sell
    strategy.close("LONG")`,
			Blockers: []string{"Backtest uses an OHLCV-compatible starter variant; ADX and highest-high breakout semantics require a future native evaluator for full document fidelity."},
		},
		{
			ID:            StrategyMomentumBreakoutVolume,
			Name:          "Momentum Breakout (Volume-Confirmed)",
			Category:      "Breakout",
			Market:        "All Markets",
			TimeHorizon:   "Swing",
			Summary:       "Targets expansion after consolidation when price clears resistance with strong volume and non-overbought momentum.",
			SupportStatus: TemplateExecutablePine,
			RequiredData:  []string{"OHLCV", "20-period highs/lows", "resistance context"},
			Indicators:    []string{"RSI(14)", "Bollinger Bands(20,2)", "Volume SMA(20)", "ATR(14)"},
			EntryRules: []string{
				"Close breaks above 20-period resistance.",
				"Volume is above 2x average.",
				"RSI(14) is between 50 and 70.",
				"Bollinger Band width is above 6%.",
			},
			ExitRules: []string{
				"Close falls below the entry candle low.",
				"RSI(14) exceeds 75.",
				"Target 2R profit or exit after 5 days without 1R progress.",
			},
			RiskRules: []string{
				"Risk 1% per trade with volatility adjustment.",
				"Daily loss cap of 2.5%.",
				"Drawdown circuit breaker at 12%.",
			},
			DefaultSettings: defaultTemplateSettings(StrategyMomentumBreakoutVolume, 20, 50, 14, 50, 75, 120),
			ExecutionProfile: TemplateExecutionProfile{
				RecommendedInterval:     "15m",
				CooldownBars:            8,
				MinHoldingBars:          4,
				ATRExitEnabled:          true,
				ATRPeriod:               14,
				ATRStopMultiplier:       2,
				ATRTakeProfitMultiplier: 3,
				MaxTradesPerDay:         12,
			},
			PineCode: `rsi14 = ta.rsi(close, 14)
vol20 = ta.sma(volume, 20)
basis = ta.sma(close, 20)
buy = close > basis and volume > vol20 and rsi14 > 50 and rsi14 < 70
sell = close < basis or rsi14 > 75
if buy
    strategy.entry("LONG", strategy.long)
if sell
    strategy.close("LONG")`,
			Blockers: []string{"Backtest uses an OHLCV-compatible starter variant; prior resistance, Bollinger width, time stops, and 2R exits require future native support for full document fidelity."},
		},
		{
			ID:            StrategyAdaptiveMeanReversion,
			Name:          "Adaptive Mean Reversion",
			Category:      "Mean Reversion",
			Market:        "Crypto, Tech Stocks",
			TimeHorizon:   "Intraday to Swing",
			Summary:       "Buys statistically extreme oversold moves in broader uptrends and exits into the mean.",
			SupportStatus: TemplateExecutablePine,
			RequiredData:  []string{"OHLCV"},
			Indicators:    []string{"Bollinger Bands(20,2.5)", "RSI(2)", "SMA(200)", "ATR(14)", "Volume SMA(20)"},
			EntryRules: []string{
				"Close is below the lower Bollinger Band.",
				"RSI(2) is below 10.",
				"Close remains above SMA(200).",
				"Volume spike is above 1.5x average.",
			},
			ExitRules: []string{
				"Close touches the middle Bollinger Band.",
				"RSI(2) exceeds 50.",
				"Exit after 3 days or on ATR-based stop below entry-day low.",
			},
			RiskRules: []string{
				"Risk 0.75% per trade.",
				"Daily loss cap of 2%.",
				"Pause new signals around severe volatility or 12% drawdown.",
			},
			DefaultSettings: defaultTemplateSettings(StrategyAdaptiveMeanReversion, 20, 200, 2, 10, 50, 260),
			ExecutionProfile: TemplateExecutionProfile{
				RecommendedInterval:     "15m",
				CooldownBars:            8,
				MinHoldingBars:          4,
				ATRExitEnabled:          true,
				ATRPeriod:               14,
				ATRStopMultiplier:       2,
				ATRTakeProfitMultiplier: 3,
				MaxTradesPerDay:         12,
			},
			PineCode: `rsi2 = ta.rsi(close, 2)
sma200 = ta.sma(close, 200)
basis = ta.sma(close, 20)
buy = close < basis and close > sma200 and rsi2 < 10
sell = close >= basis or rsi2 > 50
if buy
    strategy.entry("LONG", strategy.long)
if sell
    strategy.close("LONG")`,
			Blockers: []string{"Backtest uses an OHLCV-compatible starter variant; lower-band and time-stop logic should move into a future native evaluator for full document fidelity."},
		},
		{
			ID:            StrategyStatArbPairs,
			Name:          "Statistical Arbitrage (Pairs Trading)",
			Category:      "Market Neutral",
			Market:        "Pairs / Portfolios",
			TimeHorizon:   "Swing",
			Summary:       "Trades temporary divergence between cointegrated assets using spread z-score and dollar-neutral legs.",
			SupportStatus: TemplateBlockedByData,
			RequiredData:  []string{"Synchronized OHLCV for two symbols", "correlation matrix", "cointegration test inputs"},
			Indicators:    []string{"Spread ratio", "Z-score", "ADF / Engle-Granger test", "half-life"},
			EntryRules: []string{
				"Open spread when z-score exceeds +/-2.0.",
				"Only trade pairs with confirmed cointegration p-value below 0.05.",
				"Require mean-reversion half-life below 20 days.",
			},
			ExitRules: []string{
				"Exit when z-score returns inside +/-0.5.",
				"Stop if z-score extends beyond +/-3.5.",
				"Exit after holding period exceeds two half-lives.",
			},
			RiskRules: []string{
				"Use dollar-neutral equal notional legs.",
				"Risk 1% of spread notional.",
				"Recalculate cointegration monthly and halt at 15% drawdown.",
			},
			DefaultSettings: defaultTemplateSettings(StrategyStatArbPairs, 20, 60, 14, 30, 70, 120),
			Blockers:        []string{"Current evaluator accepts one candle stream only.", "Current execution engine does not support multi-leg spread orders.", "Cointegration testing is not implemented."},
		},
		{
			ID:            StrategyVWAPReversion,
			Name:          "VWAP Reversion",
			Category:      "Mean Reversion",
			Market:        "Indian Equities",
			TimeHorizon:   "Intraday",
			Summary:       "Fades intraday institutional-flow deviations from VWAP after opening volatility settles.",
			SupportStatus: TemplateTemplateOnly,
			RequiredData:  []string{"Intraday OHLCV", "session boundaries", "market calendar"},
			Indicators:    []string{"Session VWAP", "VWAP standard deviation", "RSI(14)", "Volume SMA"},
			EntryRules: []string{
				"After 10:15 local session time, close is below VWAP minus 0.3x VWAP standard deviation.",
				"RSI(14) is below 40.",
				"Volume is above 1.2x average.",
			},
			ExitRules: []string{
				"Exit when price crosses above VWAP.",
				"Exit when RSI(14) exceeds 60.",
				"Exit before market close.",
			},
			RiskRules: []string{
				"Risk 0.5% per trade.",
				"Daily loss cap of 2%.",
				"Drawdown protection at 10%.",
			},
			DefaultSettings: defaultTemplateSettings(StrategyVWAPReversion, 20, 50, 14, 40, 60, 120),
			Blockers:        []string{"Session VWAP and exchange-session calendar helpers are not implemented yet."},
		},
		{
			ID:            StrategyCryptoMarketMaking,
			Name:          "Market Making (Crypto-Focused)",
			Category:      "Market Making",
			Market:        "Crypto",
			TimeHorizon:   "High Frequency",
			Summary:       "Quotes both sides of liquid crypto books to capture spread while controlling inventory and adverse selection.",
			SupportStatus: TemplateBlockedByData,
			RequiredData:  []string{"L1/L2 order book", "trade flow", "funding rate", "inventory state", "latency telemetry"},
			Indicators:    []string{"ATR(14)", "order book imbalance", "funding rate", "inventory skew"},
			EntryRules: []string{
				"Place bid/ask quotes when spread exceeds 0.02%.",
				"Inventory skew is below 50% of max position.",
				"ATR/price is below 2% and funding is near neutral.",
			},
			ExitRules: []string{
				"Cancel or reduce quotes when inventory skew exceeds 70%.",
				"Cancel when volatility expands above 3%.",
				"Cancel under adverse funding or latency above threshold.",
			},
			RiskRules: []string{
				"Limit single-asset inventory to 20% of capital.",
				"Risk 0.1% per quote cycle.",
				"Daily loss cap of 1% and drawdown halt at 8%.",
			},
			DefaultSettings: defaultTemplateSettings(StrategyCryptoMarketMaking, 9, 21, 14, 30, 70, 100),
			Blockers:        []string{"Requires L2 order book reconstruction.", "Requires inventory-aware quote engine.", "Current paper engine is candle-signal based, not quote based."},
		},
		{
			ID:            StrategyFundingRateArbitrage,
			Name:          "Funding Rate Arbitrage",
			Category:      "Market Neutral",
			Market:        "Crypto Perpetuals",
			TimeHorizon:   "8h Funding Cycle",
			Summary:       "Captures perpetual funding payments with hedged spot/perp or inverse spot/perp positions.",
			SupportStatus: TemplateBlockedByData,
			RequiredData:  []string{"spot price", "perpetual price", "funding rates", "borrow rates", "exchange risk state"},
			Indicators:    []string{"funding rate", "spot-perp basis", "borrow rate"},
			EntryRules: []string{
				"Enter when funding rate is above 0.1% per 8h or below -0.1%.",
				"Confirm spot-perp basis exceeds minimum threshold.",
				"Only trade when capital is available through the next funding payment.",
			},
			ExitRules: []string{
				"Close hedge after funding payment capture.",
				"Exit on funding reversal or exchange-risk degradation.",
			},
			RiskRules: []string{
				"Use equal notional hedged legs.",
				"Risk 0.5% for exchange/default/basis exposure.",
				"Daily loss cap of 1% and drawdown halt at 5%.",
			},
			DefaultSettings: defaultTemplateSettings(StrategyFundingRateArbitrage, 9, 21, 14, 30, 70, 100),
			Blockers:        []string{"Funding-rate ingestion is not implemented.", "Spot/perp multi-leg execution is not implemented.", "Borrow-rate and basis-risk models are missing."},
		},
		{
			ID:            StrategyGridTrading,
			Name:          "Grid Trading",
			Category:      "Range",
			Market:        "Crypto",
			TimeHorizon:   "Intraday",
			Summary:       "Builds a ladder of buy and sell orders across a defined range to harvest oscillations in sideways markets.",
			SupportStatus: TemplateTemplateOnly,
			RequiredData:  []string{"OHLCV", "account balances", "persistent open-order state"},
			Indicators:    []string{"ATR(14)", "Bollinger Bands", "range bounds"},
			EntryRules: []string{
				"Define upper/lower range, commonly +/-10% from current price.",
				"Create 10 to 20 equidistant levels.",
				"Place buy orders below current price and sell orders above.",
			},
			ExitRules: []string{
				"When a buy fills, place the matching sell one grid level higher.",
				"When a sell fills, place the matching buy one grid level lower.",
				"Recenter if price drifts more than 15% from grid center.",
			},
			RiskRules: []string{
				"Allocate fixed capital per grid level.",
				"Risk 0.5% per level.",
				"Stop all grids at 20% drawdown or strong breakout.",
			},
			DefaultSettings: defaultTemplateSettings(StrategyGridTrading, 10, 20, 14, 30, 70, 120),
			Blockers:        []string{"Current evaluator returns one candle signal and does not manage standing order ladders."},
		},
		{
			ID:            StrategyMultiFactorMomentum,
			Name:          "Multi-Factor Momentum",
			Category:      "Momentum",
			Market:        "Crypto / All Markets",
			TimeHorizon:   "Swing",
			Summary:       "Combines trend, momentum, volatility, and volume filters into a higher-confidence directional signal.",
			SupportStatus: TemplateExecutablePine,
			RequiredData:  []string{"OHLCV"},
			Indicators:    []string{"SMA(50)", "SMA(200)", "RSI(14)", "MACD(12,26,9)", "ATR(14)", "Volume SMA(20)"},
			EntryRules: []string{
				"Close is above SMA(50).",
				"SMA(50) is above SMA(200).",
				"RSI(14) is above 50 and below 70.",
				"MACD crossover/crossunder confirms fresh momentum acceleration.",
				"Risk-on continuation can enter established uptrends while MACD remains bullish.",
				"Volume is above average.",
			},
			ExitRules: []string{
				"Exit when core trend, RSI momentum, or MACD confirmation fails.",
				"Use ATR trailing stop in execution/backtesting settings.",
				"Use 10-day time stop when implemented natively.",
			},
			RiskRules: []string{
				"Risk up to 2% per trade before portfolio constraints.",
				"Daily loss cap of 4%.",
				"Drawdown protection at 18%.",
			},
			DefaultSettings: defaultTemplateSettings(StrategyMultiFactorMomentum, 50, 200, 14, 50, 70, 260),
			ExecutionProfile: TemplateExecutionProfile{
				RecommendedInterval:     "1h",
				CooldownBars:            72,
				MinHoldingBars:          24,
				ATRExitEnabled:          true,
				ATRPeriod:               14,
				ATRStopMultiplier:       2.5,
				ATRTakeProfitMultiplier: 4,
				MaxTradesPerDay:         0.5,
				ShortingEnabled:         true,
				RegimeFilterEnabled:     true,
				RegimeFilterPeriod:      14,
				RegimeMinATRPercent:     0.25,
				RegimeMaxATRPercent:     4,
				PositionSizePercent:     10,
			},
			PineCode: `sma50 = ta.sma(close, 50)
sma200 = ta.sma(close, 200)
rsi14 = ta.rsi(close, 14)
vol20 = ta.sma(volume, 20)
macdLine = ta.macd(close, 12, 26, 9)
longSignal = close > sma50 and sma50 > sma200 and volume > vol20 and ((rsi14 > 50 and rsi14 < 70 and ta.crossover(macdLine, macdLine.signal)) or (rsi14 > 55 and rsi14 < 78 and macdLine > macdLine.signal))
shortSignal = close < sma50 and sma50 < sma200 and rsi14 < 50 and rsi14 > 30 and volume > vol20 and ta.crossunder(macdLine, macdLine.signal)
longExit = close < sma50 or rsi14 < 48 or macdLine < macdLine.signal
shortExit = close > sma50 or rsi14 > 52 or macdLine > macdLine.signal
if longSignal
    strategy.entry("LONG", strategy.long)
if shortSignal
    strategy.entry("SHORT", strategy.short)
if longExit
    strategy.close("LONG")
if shortExit
    strategy.close("SHORT")`,
			Blockers: []string{"ATR trailing stop is represented by the backtest/risk settings rather than inline Pine."},
		},
		{
			ID:            StrategySmartMoneyOrderFlow,
			Name:          "Smart Money / Order Flow",
			Category:      "Order Flow",
			Market:        "Crypto / Futures",
			TimeHorizon:   "Intraday",
			Summary:       "Uses institutional flow footprints such as CVD divergence, bid/ask imbalance, liquidation cascades, open interest, and funding extremes.",
			SupportStatus: TemplateBlockedByData,
			RequiredData:  []string{"tick data", "CVD", "L2 bid/ask depth", "liquidation feed", "open interest", "funding rate"},
			Indicators:    []string{"CVD", "bid/ask ratio", "liquidations", "open interest", "funding"},
			EntryRules: []string{
				"CVD turns positive after bullish divergence.",
				"Large bid limits appear at support.",
				"Liquidation cascade completes with negative funding and OI drop.",
			},
			ExitRules: []string{
				"Exit on bearish CVD divergence.",
				"Exit near large offers at resistance.",
				"Exit under extremely positive funding or failed absorption.",
			},
			RiskRules: []string{
				"Risk 1.5% per confirmed flow trade.",
				"Daily loss cap of 3%.",
				"Drawdown halt at 15%.",
			},
			DefaultSettings: defaultTemplateSettings(StrategySmartMoneyOrderFlow, 9, 21, 14, 30, 70, 100),
			Blockers:        []string{"CVD, liquidation, open-interest, and order-book feeds are not available in the current data model."},
		},
	}
}

func defaultTemplateSettings(strategyName string, fastPeriod, slowPeriod, rsiPeriod int, rsiOversold, rsiOverbought float64, lookbackLimit int) Settings {
	return Settings{
		StrategyName:  strategyName,
		Version:       "v1",
		Symbol:        "BTCUSDT",
		Interval:      "1m",
		FastPeriod:    fastPeriod,
		SlowPeriod:    slowPeriod,
		LookbackLimit: lookbackLimit,
		RSIPeriod:     rsiPeriod,
		RSIOversold:   rsiOversold,
		RSIOverbought: rsiOverbought,
	}
}
