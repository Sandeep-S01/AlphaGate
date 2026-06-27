package backtest

import (
	"fmt"
	"strings"
	"time"

	"sentra/internal/strategy"
)

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

const (
	PositionSizingAllIn         = "all_in"
	PositionSizingFixedQuote    = "fixed_quote"
	PositionSizingPercentEquity = "percent_equity"
)

const (
	ExecutionFillModeSameClose = "same_close"
	ExecutionFillModeNextOpen  = "next_open"
)

type Request struct {
	StrategyName            string    `json:"strategy_name"`
	Version                 string    `json:"version"`
	Symbol                  string    `json:"symbol"`
	Interval                string    `json:"interval"`
	From                    time.Time `json:"from"`
	To                      time.Time `json:"to"`
	FastPeriod              int       `json:"fast_period"`
	SlowPeriod              int       `json:"slow_period"`
	RSIPeriod               int       `json:"rsi_period"`
	RSIOversold             float64   `json:"rsi_oversold"`
	RSIOverbought           float64   `json:"rsi_overbought"`
	StartingBalance         float64   `json:"starting_balance"`
	FeeRate                 float64   `json:"fee_rate"`
	SlippageRate            float64   `json:"slippage_rate"`
	PositionSizingMode      string    `json:"position_sizing_mode"`
	PositionSizeValue       float64   `json:"position_size_value"`
	TrendFilterEnabled      bool      `json:"trend_filter_enabled"`
	TrendPeriod             int       `json:"trend_period"`
	CooldownBars            int       `json:"cooldown_bars"`
	MinHoldingBars          int       `json:"min_holding_bars"`
	ATRExitEnabled          bool      `json:"atr_exit_enabled"`
	ATRPeriod               int       `json:"atr_period"`
	ATRStopMultiplier       float64   `json:"atr_stop_multiplier"`
	ATRTakeProfitMultiplier float64   `json:"atr_take_profit_multiplier"`
	RegimeFilterEnabled     bool      `json:"regime_filter_enabled"`
	RegimeFilterPeriod      int       `json:"regime_filter_period"`
	RegimeMinATRPercent     float64   `json:"regime_min_atr_percent"`
	RegimeMaxATRPercent     float64   `json:"regime_max_atr_percent"`
	SaveEquityCurve         *bool     `json:"save_equity_curve,omitempty"`
	ExecutionFillMode       string    `json:"execution_fill_mode"`
	ShortingEnabled         bool      `json:"shorting_enabled"`
	PineStrategyID          *string   `json:"pine_strategy_id,omitempty"`
	PineConfig              *string   `json:"pine_config,omitempty"`
}

type Run struct {
	ID                      string        `json:"id"`
	StrategyName            string        `json:"strategy_name"`
	Version                 string        `json:"version"`
	Symbol                  string        `json:"symbol"`
	Interval                string        `json:"interval"`
	From                    time.Time     `json:"from"`
	To                      time.Time     `json:"to"`
	FastPeriod              int           `json:"fast_period"`
	SlowPeriod              int           `json:"slow_period"`
	RSIPeriod               int           `json:"rsi_period"`
	RSIOversold             float64       `json:"rsi_oversold"`
	RSIOverbought           float64       `json:"rsi_overbought"`
	StartingBalance         float64       `json:"starting_balance"`
	EndingBalance           float64       `json:"ending_balance"`
	ProfitLoss              float64       `json:"profit_loss"`
	GrossProfitLoss         float64       `json:"gross_profit_loss"`
	TotalFees               float64       `json:"total_fees"`
	EstimatedSlippageCost   float64       `json:"estimated_slippage_cost"`
	RoundTripCostPercent    float64       `json:"round_trip_cost_percent"`
	BreakEvenMovePercent    float64       `json:"break_even_move_percent"`
	ReturnPercent           float64       `json:"return_percent"`
	WinRate                 float64       `json:"win_rate"`
	MaxDrawdown             float64       `json:"max_drawdown"`
	TotalTrades             int           `json:"total_trades"`
	BuyCount                int           `json:"buy_count"`
	SellCount               int           `json:"sell_count"`
	BestTrade               float64       `json:"best_trade"`
	WorstTrade              float64       `json:"worst_trade"`
	AverageWin              float64       `json:"average_win"`
	AverageLoss             float64       `json:"average_loss"`
	OpenPosition            bool          `json:"open_position"`
	FeeRate                 float64       `json:"fee_rate"`
	SlippageRate            float64       `json:"slippage_rate"`
	PositionSizingMode      string        `json:"position_sizing_mode"`
	PositionSizeValue       float64       `json:"position_size_value"`
	TrendFilterEnabled      bool          `json:"trend_filter_enabled"`
	TrendPeriod             int           `json:"trend_period"`
	CooldownBars            int           `json:"cooldown_bars"`
	MinHoldingBars          int           `json:"min_holding_bars"`
	ATRExitEnabled          bool          `json:"atr_exit_enabled"`
	ATRPeriod               int           `json:"atr_period"`
	ATRStopMultiplier       float64       `json:"atr_stop_multiplier"`
	ATRTakeProfitMultiplier float64       `json:"atr_take_profit_multiplier"`
	RegimeFilterEnabled     bool          `json:"regime_filter_enabled"`
	RegimeFilterPeriod      int           `json:"regime_filter_period"`
	RegimeMinATRPercent     float64       `json:"regime_min_atr_percent"`
	RegimeMaxATRPercent     float64       `json:"regime_max_atr_percent"`
	ShortingEnabled         bool          `json:"shorting_enabled"`
	WinningTrades           int           `json:"winning_trades"`
	LosingTrades            int           `json:"losing_trades"`
	ProfitFactor            float64       `json:"profit_factor"`
	AverageTrade            float64       `json:"average_trade"`
	AverageHoldingSeconds   float64       `json:"average_holding_seconds"`
	Expectancy              float64       `json:"expectancy"`
	TradesPerDay            float64       `json:"trades_per_day"`
	ChurnRatio              float64       `json:"churn_ratio"`
	SharpeRatio             float64       `json:"sharpe_ratio"`
	SortinoRatio            float64       `json:"sortino_ratio"`
	BenchmarkEndingBalance  float64       `json:"benchmark_ending_balance"`
	BenchmarkProfitLoss     float64       `json:"benchmark_profit_loss"`
	BenchmarkReturnPercent  float64       `json:"benchmark_return_percent"`
	ExcessReturnPercent     float64       `json:"excess_return_percent"`
	ValidationStatus        string        `json:"validation_status"`
	ValidationReason        string        `json:"validation_reason"`
	ExecutionFillMode       string        `json:"execution_fill_mode"`
	CreatedAt               time.Time     `json:"created_at"`
	RoundTrips              []RoundTrip   `json:"round_trips,omitempty"`
	EquityCurve             []EquityPoint `json:"equity_curve,omitempty"`
	PineStrategyID          *string       `json:"pine_strategy_id,omitempty"`
	PineConfig              *string       `json:"pine_config,omitempty"`
}

type Trade struct {
	ID          string    `json:"id"`
	RunID       string    `json:"run_id"`
	Symbol      string    `json:"symbol"`
	Side        Side      `json:"side"`
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	QuoteAmount float64   `json:"quote_amount"`
	Fee         float64   `json:"fee"`
	Equity      float64   `json:"equity"`
	ExecutedAt  time.Time `json:"executed_at"`
}

type RoundTrip struct {
	ID              string    `json:"id"`
	RunID           string    `json:"run_id"`
	Symbol          string    `json:"symbol"`
	EntryTime       time.Time `json:"entry_time"`
	ExitTime        time.Time `json:"exit_time"`
	EntryPrice      float64   `json:"entry_price"`
	ExitPrice       float64   `json:"exit_price"`
	Quantity        float64   `json:"quantity"`
	GrossProfitLoss float64   `json:"gross_profit_loss"`
	Fees            float64   `json:"fees"`
	NetProfitLoss   float64   `json:"net_profit_loss"`
	ProfitPercent   float64   `json:"profit_percent"`
	HoldingSeconds  int64     `json:"holding_seconds"`
	EntryReason     string    `json:"entry_reason"`
	ExitReason      string    `json:"exit_reason"`
}

type EquityPoint struct {
	ID              string    `json:"id"`
	RunID           string    `json:"run_id"`
	Time            time.Time `json:"time"`
	Equity          float64   `json:"equity"`
	DrawdownPercent float64   `json:"drawdown_percent"`
}

type Query struct {
	Symbol string
	Limit  int
}

func DefaultRequest() Request {
	return Request{
		StrategyName:       strategy.StrategySMACrossover,
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		FastPeriod:         9,
		SlowPeriod:         21,
		RSIPeriod:          14,
		RSIOversold:        30,
		RSIOverbought:      70,
		StartingBalance:    1000,
		FeeRate:            0.001,
		SlippageRate:       0,
		PositionSizingMode: PositionSizingPercentEquity,
		PositionSizeValue:  10,
		CooldownBars:       1,
		MinHoldingBars:     1,
		ATRPeriod:          14,
	}
}

func (r Request) Normalize() Request {
	r.StrategyName = strings.TrimSpace(r.StrategyName)
	if r.StrategyName == "" {
		r.StrategyName = "sma-crossover"
	}
	r.Version = strings.TrimSpace(r.Version)
	if r.Version == "" {
		r.Version = "v1"
	}
	r.Symbol = strings.ToUpper(strings.TrimSpace(r.Symbol))
	r.Interval = strings.TrimSpace(r.Interval)
	r.PositionSizingMode = strings.TrimSpace(r.PositionSizingMode)
	if r.PositionSizingMode == "" {
		r.PositionSizingMode = PositionSizingPercentEquity
	}
	r.ExecutionFillMode = strings.TrimSpace(r.ExecutionFillMode)
	if r.ExecutionFillMode == "" {
		r.ExecutionFillMode = ExecutionFillModeNextOpen
	}
	if r.PositionSizingMode == PositionSizingPercentEquity && r.PositionSizeValue <= 0 {
		r.PositionSizeValue = 10
	}
	if r.PositionSizingMode == PositionSizingAllIn && r.PositionSizeValue <= 0 {
		r.PositionSizeValue = 100
	}
	// Safety guardrails: enforce minimum cooldown and holding bars to prevent
	// same-candle round trips that cause fee-compounding capital destruction.
	// A value of 1 means "wait at least 1 bar" — the absolute minimum safeguard.
	if r.CooldownBars <= 0 {
		r.CooldownBars = 1
	}
	if r.MinHoldingBars <= 0 {
		r.MinHoldingBars = 1
	}
	if r.TrendFilterEnabled && r.TrendPeriod <= 0 {
		r.TrendPeriod = 200
	}
	if r.RSIPeriod <= 0 {
		r.RSIPeriod = 14
	}
	if r.RSIOversold <= 0 {
		r.RSIOversold = 30
	}
	if r.RSIOverbought <= 0 {
		r.RSIOverbought = 70
	}
	if r.ATRExitEnabled && r.ATRPeriod <= 0 {
		r.ATRPeriod = 14
	}
	if r.RegimeFilterEnabled && r.RegimeFilterPeriod <= 0 {
		r.RegimeFilterPeriod = 14
	}
	if r.StrategyName == strategy.StrategyRSIMeanReversion {
		if r.FastPeriod <= 0 {
			r.FastPeriod = 9
		}
		if r.SlowPeriod <= r.FastPeriod {
			r.SlowPeriod = 21
		}
	}
	if r.SaveEquityCurve == nil {
		enabled := true
		r.SaveEquityCurve = &enabled
	}
	return r
}

func (r Request) Validate() error {
	if strings.TrimSpace(r.Symbol) == "" {
		return fmt.Errorf("symbol is required")
	}
	if strings.TrimSpace(r.Interval) == "" {
		return fmt.Errorf("interval is required")
	}
	if r.From.IsZero() || r.To.IsZero() {
		return fmt.Errorf("from and to are required")
	}
	if !r.To.After(r.From) {
		return fmt.Errorf("to must be after from")
	}
	switch r.StrategyName {
	case strategy.StrategySMACrossover, strategy.StrategyBTCTrendPullback:
		if r.FastPeriod <= 0 {
			return fmt.Errorf("fast_period must be positive")
		}
		if r.SlowPeriod <= r.FastPeriod {
			return fmt.Errorf("slow_period must be greater than fast_period")
		}
		if r.StrategyName == strategy.StrategyBTCTrendPullback && r.RSIPeriod <= 0 {
			return fmt.Errorf("rsi_period must be positive")
		}
	case strategy.StrategyRSIMeanReversion:
		if r.RSIPeriod <= 0 {
			return fmt.Errorf("rsi_period must be positive")
		}
		if r.RSIOversold <= 0 || r.RSIOverbought <= r.RSIOversold || r.RSIOverbought >= 100 {
			return fmt.Errorf("RSI thresholds are invalid")
		}
	case strategy.StrategyPineCustom:
		if r.PineConfig == nil || strings.TrimSpace(*r.PineConfig) == "" {
			return fmt.Errorf("pine_config is required for custom Pine strategy")
		}
	default:
		if _, found, err := strategy.TemplateExecutionConfig(r.StrategyName); found {
			return err
		}
		return fmt.Errorf("unsupported strategy_name %q", r.StrategyName)
	}
	if r.StartingBalance <= 0 {
		return fmt.Errorf("starting_balance must be positive")
	}
	if r.FeeRate < 0 {
		return fmt.Errorf("fee_rate cannot be negative")
	}
	if r.SlippageRate < 0 {
		return fmt.Errorf("slippage_rate cannot be negative")
	}
	switch r.PositionSizingMode {
	case PositionSizingAllIn:
	case PositionSizingFixedQuote:
		if r.PositionSizeValue <= 0 {
			return fmt.Errorf("position_size_value must be positive for fixed_quote")
		}
	case PositionSizingPercentEquity:
		if r.PositionSizeValue <= 0 || r.PositionSizeValue > 100 {
			return fmt.Errorf("position_size_value must be between 0 and 100 for percent_equity")
		}
	default:
		return fmt.Errorf("unsupported position_sizing_mode %q", r.PositionSizingMode)
	}
	switch r.ExecutionFillMode {
	case ExecutionFillModeSameClose, ExecutionFillModeNextOpen:
	default:
		return fmt.Errorf("unsupported execution_fill_mode %q", r.ExecutionFillMode)
	}
	if r.TrendFilterEnabled && r.TrendPeriod <= 0 {
		return fmt.Errorf("trend_period must be positive when trend filter is enabled")
	}
	if r.CooldownBars < 0 {
		return fmt.Errorf("cooldown_bars cannot be negative")
	}
	if r.MinHoldingBars < 0 {
		return fmt.Errorf("min_holding_bars cannot be negative")
	}
	if r.ATRExitEnabled {
		if r.ATRPeriod <= 0 {
			return fmt.Errorf("atr_period must be positive when ATR exit is enabled")
		}
		if r.ATRStopMultiplier <= 0 && r.ATRTakeProfitMultiplier <= 0 {
			return fmt.Errorf("at least one ATR exit multiplier must be positive")
		}
		if r.ATRStopMultiplier < 0 {
			return fmt.Errorf("atr_stop_multiplier cannot be negative")
		}
		if r.ATRTakeProfitMultiplier < 0 {
			return fmt.Errorf("atr_take_profit_multiplier cannot be negative")
		}
	}
	if r.RegimeFilterEnabled {
		if r.RegimeFilterPeriod <= 0 {
			return fmt.Errorf("regime_filter_period must be positive when regime filter is enabled")
		}
		if r.RegimeMinATRPercent < 0 {
			return fmt.Errorf("regime_min_atr_percent cannot be negative")
		}
		if r.RegimeMaxATRPercent < 0 {
			return fmt.Errorf("regime_max_atr_percent cannot be negative")
		}
		if r.RegimeMinATRPercent <= 0 && r.RegimeMaxATRPercent <= 0 {
			return fmt.Errorf("at least one regime ATR percent bound must be positive")
		}
		if r.RegimeMaxATRPercent > 0 && r.RegimeMinATRPercent > r.RegimeMaxATRPercent {
			return fmt.Errorf("regime_min_atr_percent must be less than or equal to regime_max_atr_percent")
		}
	}
	return nil
}

func (r Request) requiredCandlesWithFilters(required int) int {
	if r.TrendFilterEnabled && r.TrendPeriod > required {
		required = r.TrendPeriod
	}
	if r.ATRExitEnabled && r.ATRPeriod+1 > required {
		required = r.ATRPeriod + 1
	}
	if r.RegimeFilterEnabled && r.RegimeFilterPeriod+1 > required {
		required = r.RegimeFilterPeriod + 1
	}
	return required
}

func (r Request) RequiredCandles() int {
	switch r.StrategyName {
	case strategy.StrategyRSIMeanReversion:
		required := r.RSIPeriod + 1
		return r.requiredCandlesWithFilters(required)
	case strategy.StrategyBTCTrendPullback:
		required := r.SlowPeriod + 1
		if r.RSIPeriod+2 > required {
			required = r.RSIPeriod + 2
		}
		if strategy.DefaultTrendPullbackATRPeriod+1 > required {
			required = strategy.DefaultTrendPullbackATRPeriod + 1
		}
		return r.requiredCandlesWithFilters(required)
	case strategy.StrategyPineCustom:
		settings := r.StrategySettings()
		required := settings.RequiredCandles()
		return r.requiredCandlesWithFilters(required)
	default:
		if _, found, err := strategy.TemplateExecutionConfig(r.StrategyName); found && err == nil {
			settings := strategy.Settings{
				StrategyName:  r.StrategyName,
				Version:       r.Version,
				Symbol:        r.Symbol,
				Interval:      r.Interval,
				FastPeriod:    r.FastPeriod,
				SlowPeriod:    r.SlowPeriod,
				RSIPeriod:     r.RSIPeriod,
				RSIOversold:   r.RSIOversold,
				RSIOverbought: r.RSIOverbought,
			}
			required := settings.RequiredCandles()
			return r.requiredCandlesWithFilters(required)
		}
		required := r.SlowPeriod + 1
		return r.requiredCandlesWithFilters(required)
	}
}

func (r Request) StrategySettings() strategy.Settings {
	return strategy.Settings{
		StrategyName:   r.StrategyName,
		Version:        r.Version,
		Symbol:         r.Symbol,
		Interval:       r.Interval,
		FastPeriod:     r.FastPeriod,
		SlowPeriod:     r.SlowPeriod,
		LookbackLimit:  r.RequiredCandles(),
		RSIPeriod:      r.RSIPeriod,
		RSIOversold:    r.RSIOversold,
		RSIOverbought:  r.RSIOverbought,
		PineStrategyID: r.PineStrategyID,
		PineConfig:     r.PineConfig,
	}
}
