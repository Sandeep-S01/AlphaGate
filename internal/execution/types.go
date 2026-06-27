package execution

import (
	"time"

	"sentra/internal/strategy"
)

type OrderStatus string

const (
	OrderStatusCreated         OrderStatus = "created"
	OrderStatusSubmitted       OrderStatus = "submitted"
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	OrderStatusFilled          OrderStatus = "filled"
	OrderStatusCancelled       OrderStatus = "cancelled"
	OrderStatusFailed          OrderStatus = "failed"
)

type Config struct {
	Enabled          bool
	Symbol           string
	BaseAsset        string
	QuoteAsset       string
	QuoteOrderAmount float64
	FeeRate          float64
	SlippageRate     float64
	FillRatio        float64
}

type Account struct {
	BaseAsset    string    `json:"base_asset"`
	QuoteAsset   string    `json:"quote_asset"`
	BaseBalance  float64   `json:"base_balance"`
	QuoteBalance float64   `json:"quote_balance"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Order struct {
	ID                string
	RiskDecisionID    string
	ClientOrderID     string
	ExchangeOrderID   string
	StrategyName      string
	Symbol            string
	Side              strategy.Side
	Quantity          float64
	RequestedQuantity float64
	FilledQuantity    float64
	Price             float64
	AverageFillPrice  float64
	QuoteAmount       float64
	Fee               float64
	Status            OrderStatus
	FailureReason     string
	SubmittedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Trade struct {
	ID        string
	OrderID   string
	Symbol    string
	Side      strategy.Side
	Quantity  float64
	Price     float64
	Fee       float64
	CreatedAt time.Time
}

type Result struct {
	Order       Order
	Trade       Trade
	OrderEvents []OrderEvent
}

type OrderEvent struct {
	ID        string
	OrderID   string
	Status    OrderStatus
	Reason    string
	CreatedAt time.Time
}

type DailyStats struct {
	TradeCount  int
	DailyLoss   float64
	LastTradeAt time.Time
}

type DailyPnL struct {
	Day        time.Time `json:"day"`
	NetPnL     float64   `json:"net_pnl"`
	TradeCount int       `json:"trade_count"`
}
