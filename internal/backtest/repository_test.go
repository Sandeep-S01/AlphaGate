package backtest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestBuildInsertRunSQLUsesRunFields(t *testing.T) {
	run := Run{
		StrategyName:            "sma-crossover",
		Version:                 "v1",
		Symbol:                  "BTCUSDT",
		Interval:                "1m",
		From:                    time.Unix(10, 0).UTC(),
		To:                      time.Unix(100, 0).UTC(),
		FastPeriod:              9,
		SlowPeriod:              21,
		RSIPeriod:               14,
		RSIOversold:             30,
		RSIOverbought:           70,
		StartingBalance:         1000,
		EndingBalance:           1100,
		GrossProfitLoss:         30,
		TotalFees:               2,
		EstimatedSlippageCost:   1.5,
		RoundTripCostPercent:    0.3,
		BreakEvenMovePercent:    0.3,
		TotalTrades:             2,
		WinningTrades:           1,
		LosingTrades:            1,
		ProfitFactor:            1.25,
		AverageTrade:            10,
		Expectancy:              10,
		TradesPerDay:            2.5,
		ChurnRatio:              4.5,
		SharpeRatio:             1.1,
		SortinoRatio:            1.4,
		SlippageRate:            0.0005,
		PositionSizingMode:      PositionSizingFixedQuote,
		PositionSizeValue:       250,
		TrendFilterEnabled:      true,
		TrendPeriod:             200,
		CooldownBars:            20,
		MinHoldingBars:          10,
		ATRExitEnabled:          true,
		ATRPeriod:               14,
		ATRStopMultiplier:       1.5,
		ATRTakeProfitMultiplier: 3,
		RegimeFilterEnabled:     true,
		RegimeFilterPeriod:      14,
		RegimeMinATRPercent:     0.25,
		RegimeMaxATRPercent:     4,
		BenchmarkEndingBalance:  1200,
		BenchmarkProfitLoss:     200,
		BenchmarkReturnPercent:  20,
		ExcessReturnPercent:     -5,
		ValidationStatus:        "insufficient_sample",
		ValidationReason:        "completed trades below 100",
		ExecutionFillMode:       ExecutionFillModeNextOpen,
	}

	query, args := BuildInsertRunSQL(run)

	if !strings.Contains(query, "INSERT INTO backtest_runs") {
		t.Fatalf("expected backtest_runs insert, got %s", query)
	}
	if !strings.Contains(query, "RETURNING id") {
		t.Fatalf("expected RETURNING id, got %s", query)
	}
	if !strings.Contains(query, "execution_fill_mode") {
		t.Fatalf("expected execution_fill_mode in insert, got %s", query)
	}
	if !strings.Contains(query, "expectancy") || !strings.Contains(query, "sharpe_ratio") {
		t.Fatalf("expected strategy quality metrics in insert, got %s", query)
	}
	if !strings.Contains(query, "gross_profit_loss") || !strings.Contains(query, "total_fees") || !strings.Contains(query, "estimated_slippage_cost") {
		t.Fatalf("expected cost diagnostics in insert, got %s", query)
	}
	if !strings.Contains(query, "regime_filter_enabled") || !strings.Contains(query, "regime_min_atr_percent") {
		t.Fatalf("expected regime filter settings in insert, got %s", query)
	}
	if len(args) != 67 {
		t.Fatalf("expected 67 args, got %d", len(args))
	}
	if args[0] != "sma-crossover" || args[2] != "BTCUSDT" || args[6] != 9 || args[8] != 14 {
		t.Fatalf("unexpected args: %+v", args)
	}
	if args[14] != 30.0 || args[15] != 2.0 || args[16] != 1.5 || args[17] != 0.3 || args[18] != 0.3 {
		t.Fatalf("unexpected cost diagnostic args: %+v", args)
	}
	if args[32] != PositionSizingFixedQuote || args[33] != 250.0 || args[34] != true || args[35] != 200 || args[36] != 20 || args[37] != 10 || args[38] != true || args[39] != 14 || args[40] != 1.5 || args[41] != 3.0 || args[42] != true || args[43] != 14 || args[44] != 0.25 || args[45] != 4.0 || args[46] != false || args[57] != 1200.0 || args[60] != -5.0 {
		t.Fatalf("unexpected sizing/benchmark args: %+v", args)
	}
	if args[52] != 10.0 || args[53] != 2.5 || args[54] != 4.5 || args[55] != 1.1 || args[56] != 1.4 {
		t.Fatalf("unexpected strategy quality metric args: %+v", args)
	}
	if args[63] != ExecutionFillModeNextOpen {
		t.Fatalf("expected execution fill mode arg, got %+v", args)
	}
}

func TestBuildGetRunSQLQualifiesRunID(t *testing.T) {
	query := BuildGetRunSQL()
	if !strings.Contains(query, "WHERE r.id = $1") {
		t.Fatalf("expected qualified run id predicate, got %s", query)
	}
	if strings.Contains(query, "WHERE id = $1") {
		t.Fatalf("expected no ambiguous id predicate, got %s", query)
	}
}

func TestBuildInsertRoundTripSQLUsesRoundTripFields(t *testing.T) {
	roundTrip := RoundTrip{
		RunID:           "run-1",
		Symbol:          "BTCUSDT",
		EntryTime:       time.Unix(10, 0).UTC(),
		ExitTime:        time.Unix(70, 0).UTC(),
		EntryPrice:      100,
		ExitPrice:       110,
		Quantity:        1,
		GrossProfitLoss: 10,
		Fees:            0.2,
		NetProfitLoss:   9.8,
		ProfitPercent:   9.8,
		HoldingSeconds:  60,
		EntryReason:     "strategy buy signal",
		ExitReason:      "strategy sell signal",
	}

	query, args := BuildInsertRoundTripSQL(roundTrip)

	if !strings.Contains(query, "INSERT INTO backtest_round_trips") {
		t.Fatalf("expected round trip insert, got %s", query)
	}
	if len(args) != 14 {
		t.Fatalf("expected 14 args, got %d", len(args))
	}
	if args[0] != "run-1" || args[1] != "BTCUSDT" || args[4] != 100.0 {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestBuildInsertEquityPointSQLUsesEquityFields(t *testing.T) {
	point := EquityPoint{
		RunID:           "run-1",
		Time:            time.Unix(10, 0).UTC(),
		Equity:          990,
		DrawdownPercent: 1,
	}

	query, args := BuildInsertEquityPointSQL(point)

	if !strings.Contains(query, "INSERT INTO backtest_equity_points") {
		t.Fatalf("expected equity point insert, got %s", query)
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}
	if args[0] != "run-1" || args[2] != 990.0 {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestBuildInsertTradeSQLUsesTradeFields(t *testing.T) {
	trade := Trade{
		RunID:       "run-1",
		Symbol:      "BTCUSDT",
		Side:        "buy",
		Quantity:    0.01,
		Price:       50000,
		QuoteAmount: 500,
		Fee:         0.5,
		Equity:      999.5,
		ExecutedAt:  time.Unix(20, 0).UTC(),
	}

	query, args := BuildInsertTradeSQL(trade)

	if !strings.Contains(query, "INSERT INTO backtest_trades") {
		t.Fatalf("expected backtest_trades insert, got %s", query)
	}
	if len(args) != 9 {
		t.Fatalf("expected 9 args, got %d", len(args))
	}
	if args[0] != "run-1" || args[2] != "buy" || args[4] != 50000.0 {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestEquityPointsForSaveSkipsEquityCurveWhenDisabled(t *testing.T) {
	run := Run{
		StrategyName: "sma-crossover",
		Symbol:       "BTCUSDT",
		EquityCurve: []EquityPoint{
			{Time: time.Unix(10, 0).UTC(), Equity: 1000},
			{Time: time.Unix(20, 0).UTC(), Equity: 990},
		},
	}

	points := equityPointsForSave(run, SaveOptions{SaveEquityCurve: false})
	if len(points) != 0 {
		t.Fatalf("expected no equity points, got %+v", points)
	}
}

func TestRepositorySaveWithOptionsCommitsTransaction(t *testing.T) {
	tx := &fakeBacktestTx{}
	db := &fakeBacktestDB{tx: tx}
	repo := NewRepository(db)

	_, err := repo.SaveWithOptions(context.Background(), minimalRun(), nil, SaveOptions{SaveEquityCurve: false})
	if err != nil {
		t.Fatalf("SaveWithOptions returned error: %v", err)
	}

	if !db.began {
		t.Fatal("expected repository to begin a transaction")
	}
	if !tx.committed {
		t.Fatal("expected repository to commit transaction")
	}
	if tx.rolledBack {
		t.Fatal("did not expect rollback after successful commit")
	}
	if db.directQueryRows != 0 {
		t.Fatalf("expected inserts to use transaction, got %d direct calls", db.directQueryRows)
	}
}

func TestRepositorySaveWithOptionsRollsBackTransactionOnChildInsertFailure(t *testing.T) {
	tx := &fakeBacktestTx{failQueryRowAfter: 1}
	db := &fakeBacktestDB{tx: tx}
	repo := NewRepository(db)

	_, err := repo.SaveWithOptions(context.Background(), minimalRun(), []Trade{{
		Symbol:      "BTCUSDT",
		Side:        SideBuy,
		Quantity:    1,
		Price:       100,
		QuoteAmount: 100,
		ExecutedAt:  time.Unix(20, 0).UTC(),
	}}, SaveOptions{SaveEquityCurve: false})
	if err == nil {
		t.Fatal("expected save error")
	}

	if !db.began {
		t.Fatal("expected repository to begin a transaction")
	}
	if tx.committed {
		t.Fatal("did not expect commit after insert failure")
	}
	if !tx.rolledBack {
		t.Fatal("expected rollback after insert failure")
	}
}

func minimalRun() Run {
	return Run{
		StrategyName:       "sma-crossover",
		Version:            "v1",
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		From:               time.Unix(10, 0).UTC(),
		To:                 time.Unix(100, 0).UTC(),
		FastPeriod:         2,
		SlowPeriod:         3,
		RSIPeriod:          14,
		RSIOversold:        30,
		RSIOverbought:      70,
		StartingBalance:    1000,
		EndingBalance:      1000,
		PositionSizingMode: PositionSizingAllIn,
		ExecutionFillMode:  ExecutionFillModeSameClose,
		ValidationStatus:   "insufficient_sample",
		ValidationReason:   "completed trades below 100",
	}
}

type fakeBacktestDB struct {
	tx              *fakeBacktestTx
	began           bool
	directQueryRows int
}

func (f *fakeBacktestDB) Begin(ctx context.Context) (pgx.Tx, error) {
	f.began = true
	return f.tx, nil
}

func (f *fakeBacktestDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.directQueryRows++
	return fakeBacktestRow{}
}

func (f *fakeBacktestDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

type fakeBacktestTx struct {
	queryRows         int
	failQueryRowAfter int
	committed         bool
	rolledBack        bool
}

func (f *fakeBacktestTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return f, nil
}

func (f *fakeBacktestTx) Commit(ctx context.Context) error {
	f.committed = true
	return nil
}

func (f *fakeBacktestTx) Rollback(ctx context.Context) error {
	if !f.committed {
		f.rolledBack = true
	}
	return nil
}

func (f *fakeBacktestTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.queryRows++
	if f.failQueryRowAfter > 0 && f.queryRows > f.failQueryRowAfter {
		return fakeBacktestRow{err: errors.New("insert failed")}
	}
	return fakeBacktestRow{}
}

func (f *fakeBacktestTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeBacktestTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f *fakeBacktestTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}

func (f *fakeBacktestTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (f *fakeBacktestTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (f *fakeBacktestTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeBacktestTx) Conn() *pgx.Conn {
	return nil
}

type fakeBacktestRow struct {
	err error
}

func (f fakeBacktestRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	for _, target := range dest {
		switch value := target.(type) {
		case *string:
			*value = "id"
		case *time.Time:
			*value = time.Unix(100, 0).UTC()
		}
	}
	return nil
}
