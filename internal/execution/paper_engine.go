package execution

import (
	"fmt"
	"time"

	"sentra/internal/risk"
	"sentra/internal/strategy"
)

type PaperEngine struct {
	cfg Config
}

func NewPaperEngine(cfg Config) *PaperEngine {
	return &PaperEngine{cfg: cfg}
}

func (e *PaperEngine) Execute(decision risk.Decision, price float64, account Account) (Result, Account, error) {
	if !e.cfg.Enabled {
		return Result{}, account, fmt.Errorf("paper execution is disabled")
	}
	if decision.Decision != risk.DecisionApproved {
		return Result{}, account, fmt.Errorf("risk decision is not approved")
	}
	if price <= 0 {
		return Result{}, account, fmt.Errorf("price must be positive")
	}
	if e.cfg.QuoteOrderAmount <= 0 {
		return Result{}, account, fmt.Errorf("quote order amount must be positive")
	}

	switch decision.SignalSide {
	case strategy.SideBuy:
		return e.executeBuy(decision, price, account)
	case strategy.SideSell:
		return e.executeSell(decision, price, account)
	default:
		return Result{}, account, fmt.Errorf("unsupported signal side %q", decision.SignalSide)
	}
}

func (e *PaperEngine) executeBuy(decision risk.Decision, price float64, account Account) (Result, Account, error) {
	quoteAmount := e.cfg.QuoteOrderAmount
	fillRatio := e.fillRatio()
	filledQuoteAmount := quoteAmount * fillRatio
	if account.QuoteBalance < filledQuoteAmount {
		return Result{}, account, fmt.Errorf("insufficient quote balance")
	}

	fee := filledQuoteAmount * e.cfg.FeeRate
	quantity := (filledQuoteAmount - fee) / price
	account.QuoteBalance -= filledQuoteAmount
	account.BaseBalance += quantity
	account.BaseAsset = e.cfg.BaseAsset
	account.QuoteAsset = e.cfg.QuoteAsset
	account.UpdatedAt = time.Now().UTC()

	return e.result(decision, strategy.SideBuy, quantity, price, quoteAmount, filledQuoteAmount, fee, fillRatio), account, nil
}

func (e *PaperEngine) executeSell(decision risk.Decision, price float64, account Account) (Result, Account, error) {
	quoteAmount := e.cfg.QuoteOrderAmount
	fillRatio := e.fillRatio()
	filledQuoteAmount := quoteAmount * fillRatio
	quantity := filledQuoteAmount / price
	if account.BaseBalance < quantity {
		return Result{}, account, fmt.Errorf("insufficient base balance")
	}

	fee := filledQuoteAmount * e.cfg.FeeRate
	account.BaseBalance -= quantity
	account.QuoteBalance += filledQuoteAmount - fee
	account.BaseAsset = e.cfg.BaseAsset
	account.QuoteAsset = e.cfg.QuoteAsset
	account.UpdatedAt = time.Now().UTC()

	return e.result(decision, strategy.SideSell, quantity, price, quoteAmount, filledQuoteAmount, fee, fillRatio), account, nil
}

func (e *PaperEngine) fillRatio() float64 {
	if e.cfg.FillRatio <= 0 || e.cfg.FillRatio > 1 {
		return 1
	}
	return e.cfg.FillRatio
}

func (e *PaperEngine) result(decision risk.Decision, side strategy.Side, quantity float64, price float64, quoteAmount float64, filledQuoteAmount float64, fee float64, fillRatio float64) Result {
	now := time.Now().UTC()
	submittedAt := now
	status := OrderStatusFilled
	if fillRatio < 1 {
		status = OrderStatusPartiallyFilled
	}
	requestedQuantity := quoteAmount / price
	order := Order{
		RiskDecisionID:    decision.ID,
		ClientOrderID:     fmt.Sprintf("paper-%s-%d", decision.ID, now.UnixNano()),
		ExchangeOrderID:   "",
		StrategyName:      "",
		Symbol:            decision.Symbol,
		Side:              side,
		Quantity:          quantity,
		RequestedQuantity: requestedQuantity,
		FilledQuantity:    quantity,
		Price:             price,
		AverageFillPrice:  price,
		QuoteAmount:       filledQuoteAmount,
		Fee:               fee,
		Status:            status,
		SubmittedAt:       &submittedAt,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	return Result{
		Order: order,
		Trade: Trade{
			Symbol:    decision.Symbol,
			Side:      side,
			Quantity:  quantity,
			Price:     price,
			Fee:       fee,
			CreatedAt: now,
		},
		OrderEvents: []OrderEvent{
			{Status: OrderStatusCreated, CreatedAt: now},
			{Status: OrderStatusSubmitted, CreatedAt: now},
			{Status: status, CreatedAt: now},
		},
	}
}
