package execution

import (
	"context"
	"fmt"
	"time"

	"sentra/internal/strategy"
)

type ExchangeOrderRequest struct {
	ClientOrderID string
	Symbol        string
	Side          strategy.Side
	Quantity      float64
	QuoteAmount   float64
}

type ExchangeOrderStatus struct {
	ExchangeOrderID  string
	Status           OrderStatus
	FilledQuantity   float64
	AverageFillPrice float64
	Fee              float64
	UpdatedAt        time.Time
}

type ExchangeBalance struct {
	Asset string
	Free  float64
}

type ExchangePosition struct {
	Symbol        string
	Quantity      float64
	QuoteExposure float64
}

type ExchangeClient interface {
	SubmitOrder(ctx context.Context, request ExchangeOrderRequest) (ExchangeOrderStatus, error)
	GetOrderStatus(ctx context.Context, symbol string, clientOrderID string) (ExchangeOrderStatus, error)
	CancelOrder(ctx context.Context, symbol string, clientOrderID string) (ExchangeOrderStatus, error)
	ListBalances(ctx context.Context) ([]ExchangeBalance, error)
	ListPositions(ctx context.Context) ([]ExchangePosition, error)
}

type DisabledBinanceClient struct{}

func NewDisabledBinanceClient() *DisabledBinanceClient {
	return &DisabledBinanceClient{}
}

func (c *DisabledBinanceClient) SubmitOrder(ctx context.Context, request ExchangeOrderRequest) (ExchangeOrderStatus, error) {
	return ExchangeOrderStatus{}, fmt.Errorf("binance live trading is disabled")
}

func (c *DisabledBinanceClient) GetOrderStatus(ctx context.Context, symbol string, clientOrderID string) (ExchangeOrderStatus, error) {
	return ExchangeOrderStatus{}, fmt.Errorf("binance live trading is disabled")
}

func (c *DisabledBinanceClient) CancelOrder(ctx context.Context, symbol string, clientOrderID string) (ExchangeOrderStatus, error) {
	return ExchangeOrderStatus{}, fmt.Errorf("binance live trading is disabled")
}

func (c *DisabledBinanceClient) ListBalances(ctx context.Context) ([]ExchangeBalance, error) {
	return nil, fmt.Errorf("binance live trading is disabled")
}

func (c *DisabledBinanceClient) ListPositions(ctx context.Context) ([]ExchangePosition, error) {
	return nil, fmt.Errorf("binance live trading is disabled")
}

type RetryExchangeClient struct {
	client   ExchangeClient
	attempts int
	timeout  time.Duration
}

func NewRetryExchangeClient(client ExchangeClient, attempts int, timeout time.Duration) *RetryExchangeClient {
	if attempts <= 0 {
		attempts = 1
	}
	return &RetryExchangeClient{client: client, attempts: attempts, timeout: timeout}
}

func (c *RetryExchangeClient) SubmitOrder(ctx context.Context, request ExchangeOrderRequest) (ExchangeOrderStatus, error) {
	var lastErr error
	for attempt := 0; attempt < c.attempts; attempt++ {
		callCtx := ctx
		cancel := func() {}
		if c.timeout > 0 {
			callCtx, cancel = context.WithTimeout(ctx, c.timeout)
		}
		status, err := c.client.SubmitOrder(callCtx, request)
		cancel()
		if err == nil {
			return status, nil
		}
		lastErr = err
	}
	return ExchangeOrderStatus{}, fmt.Errorf("submit exchange order failed after %d attempts: %w", c.attempts, lastErr)
}

func (c *RetryExchangeClient) GetOrderStatus(ctx context.Context, symbol string, clientOrderID string) (ExchangeOrderStatus, error) {
	return c.client.GetOrderStatus(ctx, symbol, clientOrderID)
}

func (c *RetryExchangeClient) CancelOrder(ctx context.Context, symbol string, clientOrderID string) (ExchangeOrderStatus, error) {
	return c.client.CancelOrder(ctx, symbol, clientOrderID)
}

func (c *RetryExchangeClient) ListBalances(ctx context.Context) ([]ExchangeBalance, error) {
	return c.client.ListBalances(ctx)
}

func (c *RetryExchangeClient) ListPositions(ctx context.Context) ([]ExchangePosition, error) {
	return c.client.ListPositions(ctx)
}
