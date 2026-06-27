package execution

import (
	"context"
	"fmt"
	"strconv"

	"sentra/internal/marketdata"
	"sentra/internal/risk"
)

type DecisionReader interface {
	LatestApproved(ctx context.Context, symbol string) (risk.Decision, error)
}

type PriceReader interface {
	LatestPrice(ctx context.Context, symbol string, interval string) (float64, error)
}

type AccountStore interface {
	Get(ctx context.Context) (Account, error)
	Save(ctx context.Context, account Account) error
}

type Store interface {
	Save(ctx context.Context, result Result) (string, string, error)
}

type Publisher interface {
	PublishExecution(ctx context.Context, stream string, result Result) error
}

type RunnerConfig struct {
	Symbol          string
	Interval        string
	ExecutionStream string
	Engine          *PaperEngine
}

type Runner struct {
	decisions DecisionReader
	prices    PriceReader
	accounts  AccountStore
	store     Store
	publisher Publisher
	cfg       RunnerConfig
}

func NewRunner(decisions DecisionReader, prices PriceReader, accounts AccountStore, store Store, publisher Publisher, cfg RunnerConfig) *Runner {
	return &Runner{
		decisions: decisions,
		prices:    prices,
		accounts:  accounts,
		store:     store,
		publisher: publisher,
		cfg:       cfg,
	}
}

func (r *Runner) RunOnce(ctx context.Context) (Result, error) {
	decision, err := r.decisions.LatestApproved(ctx, r.cfg.Symbol)
	if err != nil {
		return Result{}, fmt.Errorf("read latest approved decision: %w", err)
	}
	price, err := r.prices.LatestPrice(ctx, r.cfg.Symbol, r.cfg.Interval)
	if err != nil {
		return Result{}, fmt.Errorf("read latest price: %w", err)
	}
	account, err := r.accounts.Get(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("read paper account: %w", err)
	}

	result, updatedAccount, err := r.cfg.Engine.Execute(decision, price, account)
	if err != nil {
		return Result{}, err
	}

	orderID, tradeID, err := r.store.Save(ctx, result)
	if err != nil {
		return Result{}, fmt.Errorf("save execution: %w", err)
	}
	result.Order.ID = orderID
	result.Trade.ID = tradeID
	result.Trade.OrderID = orderID

	if err := r.accounts.Save(ctx, updatedAccount); err != nil {
		return Result{}, fmt.Errorf("save paper account: %w", err)
	}

	if r.publisher != nil && r.cfg.ExecutionStream != "" {
		if err := r.publisher.PublishExecution(ctx, r.cfg.ExecutionStream, result); err != nil {
			return Result{}, fmt.Errorf("publish execution: %w", err)
		}
	}

	return result, nil
}

type MarketPriceReader struct {
	candles interface {
		List(ctx context.Context, query marketdata.CandleQuery) ([]marketdata.Candle, error)
	}
}

func NewMarketPriceReader(candles interface {
	List(ctx context.Context, query marketdata.CandleQuery) ([]marketdata.Candle, error)
}) *MarketPriceReader {
	return &MarketPriceReader{candles: candles}
}

func (r *MarketPriceReader) LatestPrice(ctx context.Context, symbol string, interval string) (float64, error) {
	candles, err := r.candles.List(ctx, marketdata.CandleQuery{
		Symbol:   symbol,
		Interval: interval,
		Limit:    1,
		Desc:     true,
	})
	if err != nil {
		return 0, err
	}
	if len(candles) == 0 {
		return 0, fmt.Errorf("no candles available")
	}
	price, err := strconv.ParseFloat(candles[len(candles)-1].Close, 64)
	if err != nil {
		return 0, fmt.Errorf("parse latest candle close: %w", err)
	}
	return price, nil
}
