package risk

import (
	"time"

	"sentra/internal/strategy"
)

type DecisionStatus string

const (
	DecisionApproved DecisionStatus = "approved"
	DecisionRejected DecisionStatus = "rejected"
)

type Config struct {
	Enabled                     bool
	MaxSignalStrength           float64
	MinSignalStrength           float64
	MaxQuoteAmount              float64
	MaxOrderQuoteAmount         float64
	MaxPositionQuoteAmount      float64
	MaxTotalExposureQuoteAmount float64
	MaxOpenPositions            int
	MaxDailyLoss                float64
	MaxDailyTrades              int
	AllowBuy                    bool
	AllowSell                   bool
	AllowedSymbols              []string
	Cooldown                    time.Duration
	Now                         func() time.Time `json:"-"`
}

type Settings struct {
	Enabled                     bool          `json:"enabled"`
	MaxSignalStrength           float64       `json:"max_signal_strength"`
	MinSignalStrength           float64       `json:"min_signal_strength"`
	MaxQuoteAmount              float64       `json:"max_quote_amount"`
	MaxOrderQuoteAmount         float64       `json:"max_order_quote_amount"`
	MaxPositionQuoteAmount      float64       `json:"max_position_quote_amount"`
	MaxTotalExposureQuoteAmount float64       `json:"max_total_exposure_quote_amount"`
	MaxOpenPositions            int           `json:"max_open_positions"`
	MaxDailyLoss                float64       `json:"max_daily_loss"`
	MaxDailyTrades              int           `json:"max_daily_trades"`
	AllowBuy                    bool          `json:"allow_buy"`
	AllowSell                   bool          `json:"allow_sell"`
	AllowedSymbols              []string      `json:"allowed_symbols"`
	Cooldown                    time.Duration `json:"-"`
	CooldownSeconds             int64         `json:"cooldown_seconds"`
	UpdatedAt                   time.Time     `json:"updated_at"`
}

type Context struct {
	QuoteAmount              float64
	Price                    float64
	BaseBalance              float64
	DailyTrades              int
	DailyLoss                float64
	LastTradeAt              time.Time
	OpenPositions            int
	PositionQuoteAmount      float64
	TotalExposureQuoteAmount float64
}

type Decision struct {
	ID           string
	SignalID     string
	Symbol       string
	SignalSide   strategy.Side
	Decision     DecisionStatus
	Reason       string
	EvaluatedAt  time.Time
	RiskSnapshot string
}

type RejectionSummary struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}
