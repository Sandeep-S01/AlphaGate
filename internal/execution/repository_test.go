package execution

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"sentra/internal/strategy"
)

func TestBuildInsertOrderSQLUsesOrderFields(t *testing.T) {
	order := Order{
		RiskDecisionID:    "risk-1",
		ClientOrderID:     "client-1",
		StrategyName:      "sma-crossover",
		Symbol:            "BTCUSDT",
		Side:              strategy.SideBuy,
		Quantity:          0.001,
		RequestedQuantity: 0.001,
		FilledQuantity:    0.001,
		Price:             50000,
		AverageFillPrice:  50000,
		QuoteAmount:       50,
		Fee:               0.05,
		Status:            OrderStatusFilled,
		CreatedAt:         time.Unix(10, 0).UTC(),
	}

	query, args := BuildInsertOrderSQL(order)

	if !strings.Contains(query, "INSERT INTO paper_orders") {
		t.Fatalf("expected paper_orders insert, got %s", query)
	}
	if !strings.Contains(query, "RETURNING id") {
		t.Fatalf("expected RETURNING id, got %s", query)
	}
	if len(args) != 18 {
		t.Fatalf("expected 18 args, got %d", len(args))
	}
	if args[0] != "risk-1" || args[1] != "client-1" || args[5] != string(strategy.SideBuy) || args[13] != string(OrderStatusFilled) {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestBuildInsertOrderEventSQLUsesEventFields(t *testing.T) {
	query, args := BuildInsertOrderEventSQL(OrderEvent{
		OrderID:   "order-1",
		Status:    OrderStatusSubmitted,
		Reason:    "submitted to paper exchange",
		CreatedAt: time.Unix(10, 0).UTC(),
	})

	if !strings.Contains(query, "INSERT INTO paper_order_events") {
		t.Fatalf("expected paper_order_events insert, got %s", query)
	}
	if len(args) != 4 || args[0] != "order-1" || args[1] != string(OrderStatusSubmitted) {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestBuildListOrdersSQLIncludesLifecycleFields(t *testing.T) {
	query, _ := BuildListOrdersSQL(Query{Symbol: "BTCUSDT", Limit: 10})

	expectedColumns := []string{
		"client_order_id",
		"exchange_order_id",
		"strategy_name",
		"requested_quantity",
		"filled_quantity",
		"average_fill_price",
		"failure_reason",
		"submitted_at",
		"updated_at",
	}
	for _, column := range expectedColumns {
		if !strings.Contains(query, column) {
			t.Fatalf("expected list orders query to include %s, got %s", column, query)
		}
	}
}

func TestBuildDailyStatsQueryUsesPaperTrades(t *testing.T) {
	query := BuildDailyStatsSQL()

	if !strings.Contains(query, "FROM paper_trades") {
		t.Fatalf("expected paper_trades query, got %s", query)
	}
	if !strings.Contains(query, "quantity * price") {
		t.Fatalf("expected notional calculation, got %s", query)
	}
}

func TestBuildResetAccountSQLUpdatesAssetsAndBalances(t *testing.T) {
	query, args := BuildResetAccountSQL(Account{
		BaseAsset:    "BTC",
		QuoteAsset:   "USDT",
		BaseBalance:  1.25,
		QuoteBalance: 5000,
	})

	if !strings.Contains(query, "UPDATE paper_accounts") {
		t.Fatalf("expected paper_accounts update, got %s", query)
	}
	if !strings.Contains(query, "base_asset = $1") || !strings.Contains(query, "quote_asset = $2") {
		t.Fatalf("expected asset fields in reset query, got %s", query)
	}
	if len(args) != 4 || args[0] != "BTC" || args[1] != "USDT" || args[2] != 1.25 || args[3] != 5000.0 {
		t.Fatalf("unexpected reset args: %#v", args)
	}
}

func TestBuildDailyPnLSQLGroupsTradesByDay(t *testing.T) {
	query, args := BuildDailyPnLSQL(Query{Symbol: "BTCUSDT", Limit: 30})

	if !strings.Contains(query, "DATE_TRUNC('day', created_at)") {
		t.Fatalf("expected daily grouping, got %s", query)
	}
	if !strings.Contains(query, "quantity * price") {
		t.Fatalf("expected notional calculation, got %s", query)
	}
	if len(args) != 2 || args[0] != "BTCUSDT" || args[1] != 30 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestRepositoryWrapsPostgresDisconnectDuringExecutionSave(t *testing.T) {
	dbErr := errors.New("postgres disconnect")
	repository := NewRepository(&fakeExecutionDB{rowErrs: []error{dbErr}})

	_, _, err := repository.Save(context.Background(), Result{Order: Order{Symbol: "BTCUSDT"}, Trade: Trade{Symbol: "BTCUSDT"}})
	if err == nil || !strings.Contains(err.Error(), "insert paper order") || !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped order insert disconnect, got %v", err)
	}

	repository = NewRepository(&fakeExecutionDB{rowErrs: []error{nil, dbErr}})
	_, _, err = repository.Save(context.Background(), Result{Order: Order{Symbol: "BTCUSDT"}, Trade: Trade{Symbol: "BTCUSDT"}})
	if err == nil || !strings.Contains(err.Error(), "insert paper trade") || !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped trade insert disconnect, got %v", err)
	}
}

func TestRepositorySaveSkipsTradeForFailedOrder(t *testing.T) {
	db := &fakeExecutionDB{}
	repository := NewRepository(db)

	orderID, tradeID, err := repository.Save(context.Background(), Result{
		Order: Order{
			RiskDecisionID: "risk-1",
			Symbol:         "BTCUSDT",
			Side:           strategy.SideSell,
			Status:         OrderStatusFailed,
			FailureReason:  "insufficient base balance",
			CreatedAt:      time.Unix(10, 0).UTC(),
			UpdatedAt:      time.Unix(10, 0).UTC(),
		},
		OrderEvents: []OrderEvent{
			{Status: OrderStatusCreated, CreatedAt: time.Unix(10, 0).UTC()},
			{Status: OrderStatusFailed, Reason: "insufficient base balance", CreatedAt: time.Unix(10, 0).UTC()},
		},
	})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if orderID == "" {
		t.Fatal("expected order id")
	}
	if tradeID != "" {
		t.Fatalf("expected no trade id for failed order, got %q", tradeID)
	}
	if len(db.queries) != 3 {
		t.Fatalf("expected order insert and two event inserts only, got %d queries", len(db.queries))
	}
	if strings.Contains(strings.Join(db.queries, "\n"), "paper_trades") {
		t.Fatalf("did not expect failed order to insert trade, got queries: %s", strings.Join(db.queries, "\n"))
	}
}

func TestAccountRepositoryWrapsPostgresDisconnect(t *testing.T) {
	dbErr := errors.New("postgres disconnect")
	accounts := NewAccountRepository(&fakeExecutionDB{rowErrs: []error{dbErr}})

	if _, err := accounts.Get(context.Background()); err == nil || !strings.Contains(err.Error(), "query paper account") || !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped account query disconnect, got %v", err)
	}

	accounts = NewAccountRepository(&fakeExecutionDB{rowErrs: []error{dbErr}})
	if err := accounts.Save(context.Background(), Account{BaseAsset: "BTC", QuoteAsset: "USDT"}); err == nil || !strings.Contains(err.Error(), "save paper account") || !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped account save disconnect, got %v", err)
	}
}

type fakeExecutionDB struct {
	rowErrs []error
	queries []string
}

func (f *fakeExecutionDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.queries = append(f.queries, sql)
	var err error
	if len(f.rowErrs) > 0 {
		err = f.rowErrs[0]
		f.rowErrs = f.rowErrs[1:]
	}
	return fakeExecutionRow{err: err}
}

func (f *fakeExecutionDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

type fakeExecutionRow struct {
	err error
}

func (f fakeExecutionRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	for _, target := range dest {
		if value, ok := target.(*string); ok {
			*value = "id"
		}
	}
	return nil
}
