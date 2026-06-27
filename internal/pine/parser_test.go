package pine

import (
	"encoding/json"
	"testing"
)

func TestParserBasicEMA(t *testing.T) {
	code := `
	// EMA Crossover Strategy
	ema50 = ta.ema(close, 50)
	ema200 = ta.ema(close, 200)
	buy = ta.crossover(ema50, ema200)
	sell = ta.crossunder(ema50, ema200)
	if buy
		strategy.entry("LONG", strategy.long)
	if sell
		strategy.close("LONG")
	`

	parser := NewParser(code)
	res := parser.Parse()

	if len(res.Errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", res.Errors)
	}

	// Verify indicators
	ema50, ok := res.Config.Indicators["ema50"]
	if !ok {
		t.Fatalf("expected ema50 indicator")
	}
	if ema50.Type != "ema" || ema50.Source != "close" || len(ema50.Params) != 1 || ema50.Params[0] != 50 {
		t.Errorf("ema50 def invalid: %+v", ema50)
	}

	ema200, ok := res.Config.Indicators["ema200"]
	if !ok {
		t.Fatalf("expected ema200 indicator")
	}
	if ema200.Type != "ema" || ema200.Source != "close" || len(ema200.Params) != 1 || ema200.Params[0] != 200 {
		t.Errorf("ema200 def invalid: %+v", ema200)
	}

	// Verify conditions
	buyCond, ok := res.Config.Conditions["buy"]
	if !ok {
		t.Fatalf("expected buy condition")
	}
	if buyCond.Op != "crossover" || len(buyCond.Args) != 2 || buyCond.Args[0].Val != "ema50" || buyCond.Args[1].Val != "ema200" {
		t.Errorf("buy condition invalid: %+v", buyCond)
	}

	// Verify rules
	if len(res.Config.Rules) != 2 {
		t.Fatalf("expected 2 execution rules, got %d", len(res.Config.Rules))
	}
	rule1 := res.Config.Rules[0]
	if rule1.Condition != "buy" || rule1.Action != "entry" || rule1.ID != "LONG" || rule1.Direction != "long" {
		t.Errorf("rule1 invalid: %+v", rule1)
	}
	rule2 := res.Config.Rules[1]
	if rule2.Condition != "sell" || rule2.Action != "close" || rule2.ID != "LONG" || rule2.Direction != "long" {
		t.Errorf("rule2 invalid: %+v", rule2)
	}
}

func TestParserCloseInfersShortDirectionFromID(t *testing.T) {
	code := `
	shortExit = close > ema50
	if shortExit
		strategy.close("SHORT")
	`

	parser := NewParser(code)
	res := parser.Parse()
	if len(res.Errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", res.Errors)
	}
	if len(res.Config.Rules) != 1 {
		t.Fatalf("expected 1 execution rule, got %d", len(res.Config.Rules))
	}
	rule := res.Config.Rules[0]
	if rule.Condition != "shortExit" || rule.Action != "close" || rule.ID != "SHORT" || rule.Direction != "short" {
		t.Fatalf("expected short close rule, got %+v", rule)
	}
}

func TestParserNestedIndicators(t *testing.T) {
	code := `
	rsiEma = ta.ema(ta.rsi(close, 14), 9)
	`
	parser := NewParser(code)
	res := parser.Parse()

	if len(res.Errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", res.Errors)
	}

	// Should have flattened the nested RSI
	if len(res.Config.Indicators) != 2 {
		t.Errorf("expected 2 indicators, got %d: %+v", len(res.Config.Indicators), res.Config.Indicators)
	}

	// Let's print out the parsed JSON for inspection
	b, _ := json.MarshalIndent(res.Config, "", "  ")
	t.Logf("Parsed Config:\n%s", string(b))
}

func TestParserPrecedenceAndExpressions(t *testing.T) {
	code := `
	cond1 = (close > 50000) and (rsi < 30) or (volume > 1000)
	`
	parser := NewParser(code)
	res := parser.Parse()

	if len(res.Errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", res.Errors)
	}

	cond, ok := res.Config.Conditions["cond1"]
	if !ok {
		t.Fatalf("expected cond1 condition")
	}

	// 'or' has lower precedence than 'and', so top-level operator should be 'or'
	if cond.Op != "or" {
		t.Errorf("expected top-level operator to be 'or', got %q", cond.Op)
	}
	if len(cond.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(cond.Args))
	}

	// Left of 'or' should be 'and'
	left := cond.Args[0]
	if left.Op != "and" {
		t.Errorf("expected left arg operator to be 'and', got %q", left.Op)
	}

	// Right of 'or' should be volume comparison
	right := cond.Args[1]
	if right.Op != ">" || right.Args[0].Val != "volume" || right.Args[1].Val != "1000" {
		t.Errorf("expected right arg to be volume > 1000, got %+v", right)
	}
}

func TestParserWarnings(t *testing.T) {
	code := `
	ema50 = ta.ema(close, 50)
	buy = close > 50000
	`
	parser := NewParser(code)
	res := parser.Parse()

	if len(res.Errors) > 0 {
		t.Fatalf("unexpected parser errors: %v", res.Errors)
	}

	// Should have warnings:
	// 1. Missing entry rule
	// 2. Missing close rule
	// 3. Hardcoded constant 50000
	if len(res.Warnings) < 3 {
		t.Errorf("expected at least 3 warnings, got %d: %v", len(res.Warnings), res.Warnings)
	}
}

func TestParserComplexStrategy(t *testing.T) {
	code := `//@version=6
strategy(
     title        = "BTC Multi-Confluence Strategy [4H]",
     shorttitle   = "BTC-MCS",
     overlay      = true,
     default_qty_type  = strategy.percent_of_equity,
     default_qty_value = 25,
     commission_type   = strategy.commission.percent,
     commission_value  = 0.1,
     slippage          = 2,
     initial_capital   = 10000
     )

// ─── INPUTS ───────────────────────────────────────────────────────────────────
emaFastLen   = input.int(50,  "Fast EMA",        group="Trend Filter")
emaSlowLen   = input.int(200, "Slow EMA",        group="Trend Filter")
rsiLen       = input.int(14,  "RSI Length",      group="Momentum")
rsiBullEntry = input.int(52,  "RSI Bull Entry >" , group="Momentum")
rsiBearEntry = input.int(48,  "RSI Bear Entry <" , group="Momentum")
atrLen       = input.int(14,  "ATR Length",      group="Risk")
atrSLMult    = input.float(2.0,"ATR SL Multiplier", step=0.1, group="Risk")
atrTPMult    = input.float(4.0,"ATR TP Multiplier", step=0.1, group="Risk")
volFilter    = input.bool(true,"Volatility Regime Filter", group="Filter")
volLen       = input.int(20,  "Volatility MA Length",      group="Filter")
minBarsBetween = input.int(10,"Min Bars Between Trades",   group="Filter")
tradeDir     = input.string("Both", "Trade Direction", options=["Long Only","Short Only","Both"])

// ─── CORE CALCULATIONS ────────────────────────────────────────────────────────
emaFast = ta.ema(close, emaFastLen)
emaSlow = ta.ema(close, emaSlowLen)
rsi     = ta.rsi(close, rsiLen)
atr     = ta.atr(atrLen)

// Volatility regime filter: only trade when ATR is above its own MA
// This avoids choppy, compressed, low-vol markets
atrMa   = ta.sma(atr, volLen)
volOk   = volFilter ? atr > atrMa : true

// Trend direction
bullTrend = emaFast > emaSlow
bearTrend = emaFast < emaSlow

// Price above/below fast EMA as additional confirmation
priceAboveFast = close > emaFast
priceBelowFast = close < emaFast

// ─── BAR COUNTER (prevent overtrading) ────────────────────────────────────────
var int lastTradeBar = 0
barsSinceTrade = bar_index - lastTradeBar
cooldownOk = barsSinceTrade >= minBarsBetween

// ─── ENTRY CONDITIONS ─────────────────────────────────────────────────────────
longCondition  = bullTrend
              and priceAboveFast
              and rsi > rsiBullEntry
              and rsi < 70           // not overbought
              and volOk
              and cooldownOk
              and (tradeDir == "Long Only" or tradeDir == "Both")

shortCondition = bearTrend
              and priceBelowFast
              and rsi < rsiBearEntry
              and rsi > 30           // not oversold
              and volOk
              and cooldownOk
              and (tradeDir == "Short Only" or tradeDir == "Both")

// ─── DYNAMIC SL/TP ────────────────────────────────────────────────────────────
longSL  = close - atr * atrSLMult
longTP  = close + atr * atrTPMult
shortSL = close + atr * atrSLMult
shortTP = close - atr * atrTPMult

// ─── ENTRIES ──────────────────────────────────────────────────────────────────
if longCondition and strategy.position_size == 0
    strategy.entry("Long", strategy.long)
    strategy.exit("Long Exit", "Long", stop=longSL, limit=longTP)
    lastTradeBar := bar_index

if shortCondition and strategy.position_size == 0
    strategy.entry("Short", strategy.short)
    strategy.exit("Short Exit", "Short", stop=shortSL, limit=shortTP)
    lastTradeBar := bar_index

// ─── PLOTS ────────────────────────────────────────────────────────────────────
plot(emaFast, "EMA 50",  color=color.new(color.orange, 0), linewidth=1)
plot(emaSlow, "EMA 200", color=color.new(color.blue,   0), linewidth=2)

// SL/TP lines for active position
var float activeSL = na
var float activeTP = na

if strategy.position_size > 0
    activeSL := longSL
    activeTP := longTP
else if strategy.position_size < 0
    activeSL := shortSL
    activeTP := shortTP
else
    activeSL := na
    activeTP := na

plot(activeSL, "Stop Loss",   color=color.new(color.red,   0), style=plot.style_linebr, linewidth=1)
plot(activeTP, "Take Profit", color=color.new(color.green, 0), style=plot.style_linebr, linewidth=1)

// Signal markers
plotshape(longCondition  and strategy.position_size == 0, "Long",  shape.triangleup,   location.belowbar, color.green, size=size.small)
plotshape(shortCondition and strategy.position_size == 0, "Short", shape.triangledown, location.abovebar, color.red,   size=size.small)

// Background: trend color
bgcolor(bullTrend ? color.new(color.green, 95) : color.new(color.red, 95))

// ─── INFO TABLE ───────────────────────────────────────────────────────────────
var table infoTable = table.new(position.top_right, 2, 6, border_width=1)
if barstate.islast
    table.cell(infoTable, 0, 0, "Position",  text_color=color.white, bgcolor=color.gray)
    table.cell(infoTable, 1, 0, strategy.position_size > 0 ? "LONG" : strategy.position_size < 0 ? "SHORT" : "FLAT",
         text_color=strategy.position_size > 0 ? color.green : strategy.position_size < 0 ? color.red : color.white,
         bgcolor=color.gray)
    table.cell(infoTable, 0, 1, "RSI",       text_color=color.white, bgcolor=color.gray)
    table.cell(infoTable, 1, 1, str.tostring(math.round(rsi, 1)), text_color=color.white, bgcolor=color.gray)
    table.cell(infoTable, 0, 2, "ATR",       text_color=color.white, bgcolor=color.gray)
    table.cell(infoTable, 1, 2, str.tostring(math.round(atr, 0)), text_color=color.white, bgcolor=color.gray)
    table.cell(infoTable, 0, 3, "Vol Filter",text_color=color.white, bgcolor=color.gray)
    table.cell(infoTable, 1, 3, volOk ? "PASS" : "FAIL", text_color=volOk ? color.green : color.red, bgcolor=color.gray)
    table.cell(infoTable, 0, 4, "Trend",     text_color=color.white, bgcolor=color.gray)
    table.cell(infoTable, 1, 4, bullTrend ? "BULL" : "BEAR", text_color=bullTrend ? color.green : color.red, bgcolor=color.gray)
    table.cell(infoTable, 0, 5, "Net P&L",   text_color=color.white, bgcolor=color.gray)
    table.cell(infoTable, 1, 5, str.tostring(math.round(strategy.netprofit, 2)), text_color=strategy.netprofit > 0 ? color.green : color.red, bgcolor=color.gray)
`
	parser := NewParser(code)
	res := parser.Parse()

	if len(res.Errors) > 0 {
		t.Logf("Errors: %v", res.Errors)
		t.Fatalf("unexpected compile errors in complex strategy")
	}
}
