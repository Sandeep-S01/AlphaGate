package risk

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sentra/internal/strategy"
)

type Evaluator struct {
	cfg Config
}

func NewEvaluator(cfg Config) *Evaluator {
	return &Evaluator{cfg: cfg}
}

func (e *Evaluator) Evaluate(signal strategy.Signal) Decision {
	return e.EvaluateWithContext(signal, Context{})
}

func (e *Evaluator) EvaluateWithContext(signal strategy.Signal, ctx Context) Decision {
	status := DecisionApproved
	reason := "approved by risk rules"
	now := time.Now().UTC()
	if e.cfg.Now != nil {
		now = e.cfg.Now().UTC()
	}

	switch {
	case !e.cfg.Enabled:
		status = DecisionRejected
		reason = "risk engine is disabled"
	case len(e.cfg.AllowedSymbols) > 0 && !symbolAllowed(signal.Symbol, e.cfg.AllowedSymbols):
		status = DecisionRejected
		reason = fmt.Sprintf("symbol %s is not allowed", signal.Symbol)
	case signal.Side == strategy.SideHold:
		status = DecisionRejected
		reason = "hold signal is not executable"
	case signal.Side == strategy.SideBuy && !e.cfg.AllowBuy:
		status = DecisionRejected
		reason = "buy signals are disabled"
	case signal.Side == strategy.SideSell && !e.cfg.AllowSell:
		status = DecisionRejected
		reason = "sell signals are disabled"
	case e.cfg.MinSignalStrength > 0 && signal.Strength < e.cfg.MinSignalStrength:
		status = DecisionRejected
		reason = fmt.Sprintf("signal strength %.6f is below min %.6f", signal.Strength, e.cfg.MinSignalStrength)
	case e.cfg.MaxSignalStrength > 0 && signal.Strength > e.cfg.MaxSignalStrength:
		status = DecisionRejected
		reason = fmt.Sprintf("signal strength %.6f exceeds max %.6f", signal.Strength, e.cfg.MaxSignalStrength)
	case e.cfg.MaxQuoteAmount > 0 && ctx.QuoteAmount > e.cfg.MaxQuoteAmount:
		status = DecisionRejected
		reason = fmt.Sprintf("quote amount %.6f exceeds max %.6f", ctx.QuoteAmount, e.cfg.MaxQuoteAmount)
	case e.cfg.MaxOrderQuoteAmount > 0 && ctx.QuoteAmount > e.cfg.MaxOrderQuoteAmount:
		status = DecisionRejected
		reason = fmt.Sprintf("order quote amount %.6f exceeds max %.6f", ctx.QuoteAmount, e.cfg.MaxOrderQuoteAmount)
	case signal.Side == strategy.SideBuy && ctx.QuoteAmount > 0 && ctx.QuoteBalance < ctx.QuoteAmount:
		status = DecisionRejected
		reason = fmt.Sprintf("quote balance %.6f is below required %.6f", ctx.QuoteBalance, ctx.QuoteAmount)
	case signal.Side == strategy.SideSell && ctx.Price > 0 && ctx.QuoteAmount > 0 && ctx.BaseBalance < ctx.QuoteAmount/ctx.Price:
		status = DecisionRejected
		reason = fmt.Sprintf("base balance %.6f is below required %.6f", ctx.BaseBalance, ctx.QuoteAmount/ctx.Price)
	case e.cfg.MaxOpenPositions > 0 && ctx.OpenPositions >= e.cfg.MaxOpenPositions:
		status = DecisionRejected
		reason = fmt.Sprintf("open positions %d reached max %d", ctx.OpenPositions, e.cfg.MaxOpenPositions)
	case e.cfg.MaxPositionQuoteAmount > 0 && ctx.PositionQuoteAmount+ctx.QuoteAmount > e.cfg.MaxPositionQuoteAmount:
		status = DecisionRejected
		reason = fmt.Sprintf("position quote exposure %.6f would exceed max %.6f", ctx.PositionQuoteAmount+ctx.QuoteAmount, e.cfg.MaxPositionQuoteAmount)
	case e.cfg.MaxTotalExposureQuoteAmount > 0 && ctx.TotalExposureQuoteAmount+ctx.QuoteAmount > e.cfg.MaxTotalExposureQuoteAmount:
		status = DecisionRejected
		reason = fmt.Sprintf("total quote exposure %.6f would exceed max %.6f", ctx.TotalExposureQuoteAmount+ctx.QuoteAmount, e.cfg.MaxTotalExposureQuoteAmount)
	case e.cfg.MaxDailyTrades > 0 && ctx.DailyTrades >= e.cfg.MaxDailyTrades:
		status = DecisionRejected
		reason = fmt.Sprintf("daily trade count %d reached max %d", ctx.DailyTrades, e.cfg.MaxDailyTrades)
	case e.cfg.MaxDailyLoss > 0 && ctx.DailyLoss >= e.cfg.MaxDailyLoss:
		status = DecisionRejected
		reason = fmt.Sprintf("daily loss %.6f reached max %.6f", ctx.DailyLoss, e.cfg.MaxDailyLoss)
	case e.cfg.Cooldown > 0 && !ctx.LastTradeAt.IsZero() && now.Sub(ctx.LastTradeAt) < e.cfg.Cooldown:
		status = DecisionRejected
		reason = "trade cooldown is active"
	}

	snapshot, _ := json.Marshal(snapshotConfig{
		Enabled:                     e.cfg.Enabled,
		MaxSignalStrength:           e.cfg.MaxSignalStrength,
		MinSignalStrength:           e.cfg.MinSignalStrength,
		MaxQuoteAmount:              e.cfg.MaxQuoteAmount,
		MaxOrderQuoteAmount:         e.cfg.MaxOrderQuoteAmount,
		MaxPositionQuoteAmount:      e.cfg.MaxPositionQuoteAmount,
		MaxTotalExposureQuoteAmount: e.cfg.MaxTotalExposureQuoteAmount,
		MaxOpenPositions:            e.cfg.MaxOpenPositions,
		MaxDailyLoss:                e.cfg.MaxDailyLoss,
		MaxDailyTrades:              e.cfg.MaxDailyTrades,
		AllowBuy:                    e.cfg.AllowBuy,
		AllowSell:                   e.cfg.AllowSell,
		AllowedSymbols:              e.cfg.AllowedSymbols,
		CooldownSeconds:             int64(e.cfg.Cooldown.Seconds()),
	})
	return Decision{
		SignalID:     signal.ID,
		Symbol:       signal.Symbol,
		SignalSide:   signal.Side,
		Decision:     status,
		Reason:       reason,
		EvaluatedAt:  now,
		RiskSnapshot: string(snapshot),
	}
}

func symbolAllowed(symbol string, allowed []string) bool {
	for _, candidate := range allowed {
		if strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(symbol)) {
			return true
		}
	}
	return false
}

type snapshotConfig struct {
	Enabled                     bool     `json:"enabled"`
	MaxSignalStrength           float64  `json:"max_signal_strength"`
	MinSignalStrength           float64  `json:"min_signal_strength"`
	MaxQuoteAmount              float64  `json:"max_quote_amount"`
	MaxOrderQuoteAmount         float64  `json:"max_order_quote_amount"`
	MaxPositionQuoteAmount      float64  `json:"max_position_quote_amount"`
	MaxTotalExposureQuoteAmount float64  `json:"max_total_exposure_quote_amount"`
	MaxOpenPositions            int      `json:"max_open_positions"`
	MaxDailyLoss                float64  `json:"max_daily_loss"`
	MaxDailyTrades              int      `json:"max_daily_trades"`
	AllowBuy                    bool     `json:"allow_buy"`
	AllowSell                   bool     `json:"allow_sell"`
	AllowedSymbols              []string `json:"allowed_symbols"`
	CooldownSeconds             int64    `json:"cooldown_seconds"`
}
