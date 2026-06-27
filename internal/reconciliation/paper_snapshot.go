package reconciliation

import (
	"context"

	"sentra/internal/execution"
)

type PaperAccountReader interface {
	Get(ctx context.Context) (execution.Account, error)
}

type PaperOrderReader interface {
	ListOrders(ctx context.Context, query execution.Query) ([]execution.Order, error)
}

type PaperSnapshotReader struct {
	accounts PaperAccountReader
	orders   PaperOrderReader
	symbol   string
}

func NewPaperSnapshotReader(accounts PaperAccountReader, orders PaperOrderReader, symbol string) *PaperSnapshotReader {
	return &PaperSnapshotReader{accounts: accounts, orders: orders, symbol: symbol}
}

func (r *PaperSnapshotReader) Snapshot(ctx context.Context) (Snapshot, error) {
	account, err := r.accounts.Get(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	orders, err := r.orders.ListOrders(ctx, execution.Query{Symbol: r.symbol, Limit: 1000})
	if err != nil {
		return Snapshot{}, err
	}
	snapshot := Snapshot{
		Balances: []Balance{
			{Asset: account.BaseAsset, Free: account.BaseBalance},
			{Asset: account.QuoteAsset, Free: account.QuoteBalance},
		},
	}
	if account.BaseBalance != 0 {
		snapshot.Positions = append(snapshot.Positions, Position{
			Symbol:   r.symbol,
			Quantity: account.BaseBalance,
		})
	}
	for _, order := range orders {
		if isOpenOrder(order.Status) {
			snapshot.Orders = append(snapshot.Orders, Order{
				ClientOrderID: order.ClientOrderID,
				Symbol:        order.Symbol,
				Status:        string(order.Status),
			})
		}
	}
	return snapshot, nil
}

func isOpenOrder(status execution.OrderStatus) bool {
	switch status {
	case execution.OrderStatusCreated, execution.OrderStatusSubmitted, execution.OrderStatusPartiallyFilled:
		return true
	default:
		return false
	}
}
