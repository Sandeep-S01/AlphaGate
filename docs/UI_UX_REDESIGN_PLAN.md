# Sentra — UI/UX Redesign Plan

> **Date**: 2026-06-23
> **Status**: Design document

---

## 1. Design Principles

| Principle | Description |
|-----------|-------------|
| **Dark-First** | Trading platforms are dark-themed. Dark reduces eye strain during long monitoring sessions and makes highlighted data (green P&L, red losses) pop naturally. |
| **Information Density** | Traders need to see a lot of data at once — balances, prices, signals, open positions. Every pixel should earn its place. |
| **Progressive Disclosure** | Show the most critical info immediately (P&L, latest price, kill switch). Layer details behind tabs, expandable sections, and drill-downs. |
| **Real-Time First** | The UI assumes live data. Every view has loading → streaming states. Stale data is visually distinct from fresh data. |
| **Keyboard-Friendly** | All actions accessible via keyboard. Power users should not need a mouse for common operations. |
| **Consistent Visual Language** | One spacing scale, one color palette, one typography system. Every element follows the same rules. |

---

## 2. Visual Identity

### 2.1 Color Palette

```
Background (darkest)   → #0a0e17    (main background)
Background (card)      → #131722    (card surfaces)
Background (hover)     → #1a1f2e    (hover/active states)
Border                 → #2a2e3e    (card borders, dividers)
Text (primary)         → #e0e3eb    (headings, key data)
Text (secondary)       → #78828a    (labels, descriptions)
Text (muted)           → #4a535c    (disabled, placeholders)

Accent (cyan)          → #00bcd4    (primary actions, active tabs, links)
Accent (cyan hover)    → #26c6da
Accent (blue)          → #2196f3    (info, secondary actions)

Signal (buy)           → #0ecb81    (buy signals, positive P&L)
Signal (sell)          → #f6465d    (sell signals, negative P&L)
Signal (hold)          → #f0b90b    (hold, neutral, warnings)

Danger                 → #ff4444    (kill switch, errors, critical alerts)
Success                → #00c853    (healthy, completed, synced)
Warning                → #ff9100    (degraded, pending, cooldown)

Chart (up)             → #0ecb81    (green candles)
Chart (down)           → #f6465d    (red candles)
Chart (volume)         → rgba(255,255,255,0.08)
```

### 2.2 Typography

```
Font Family: Inter (UI) + JetBrains Mono (numbers/data)
Scale:
  - 11px  →  meta, timestamps, secondary labels
  - 12px  →  tab labels, table cells, badges
  - 13px  →  body text, form labels, descriptions
  - 14px  →  card titles, navigation items
  - 16px  →  section headings
  - 20px  →  page titles
  - 24px+ →  hero values (big P&L, balance display)
```

### 2.3 Spacing (4px grid)

```
 4px  →  inner padding, icon gaps
 8px  →  tight spacing, badge padding
12px  →  form element padding
16px  →  card padding, section gaps
20px  →  content to border
24px  →  section margins
32px  →  page margins, large section gaps
```

### 2.4 Shadows & Elevation

```
Elevation 1: 0 2px 4px rgba(0,0,0,0.2)   →  cards, panels
Elevation 2: 0 4px 12px rgba(0,0,0,0.3)  →  dropdowns, modals
Elevation 3: 0 8px 24px rgba(0,0,0,0.4)  →  tooltips, notifications
```

---

## 3. Layout

```
┌────────────────────────────────────────────────────────────────────────┐
│  TOP BAR                                                                │
│  Logo | System Status | Symbol/Interval | API Key | Clock              │
├──────┬─────────────────────────────────────────────────────────────────┤
│      │                                                                  │
│  S   │  MAIN CONTENT AREA                                              │
│  I   │                                                                  │
│  D   │  ┌─────────────────────────────────────────────────────────┐   │
│  E   │  │  TAB NAVIGATION (Overview | Strategy | Backtests | ...)  │   │
│  B   │  └─────────────────────────────────────────────────────────┘   │
│  A   │                                                                  │
│  R   │  Active tab panel content                                       │
│      │                                                                  │
│  📊  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  Ov  │  │ Health   │ │ Price    │ │ Equity   │ │ Trades   │          │
│  📈  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
│  St  │                                                                  │
│  🔬  │  ┌─────────────────────────────────────────────────────────┐   │
│  Bt  │  │ Candlestick Chart (Overview tab)                        │   │
│  📊  │  │                                                         │   │
│  Cm  │  └─────────────────────────────────────────────────────────┘   │
│  📝  │                                                                  │
│  Ps  │  ┌──────────────┐ ┌────────────────────────────────┐           │
│  📍  │  │ Latest Signal│ │ Latest Risk Decision           │           │
│  Si  │  └──────────────┘ └────────────────────────────────┘           │
│  ⚠️  │                                                                  │
│  Rk  │                                                                  │
│  📋  │                                                                  │
│  Or  │                                                                  │
│  🔄  │                                                                  │
│  Tr  │                                                                  │
│  ⚙️  │                                                                  │
│  Op  │                                                                  │
│      │                                                                  │
└──────┴─────────────────────────────────────────────────────────────────┘
```

### 3.1 Sidebar Navigation

- Fixed width: 56px (collapsed icons) / 200px (expanded)
- Icons + labels, active state highlight
- Tooltip on hover for collapsed state
- Bottom section: theme toggle, collapse toggle

### 3.2 Top Bar

- Fixed height: 48px
- Left: Logo + system status indicator (green/amber/red dot)
- Center: Symbol + Interval selector
- Right: API Key indicator (green if set), live clock, connection status

### 3.3 Main Content Area

- Scrollable, fills remaining viewport height
- Tab navigation below top bar
- Active panel fills the area

---

## 4. View Designs (12 Views)

### 4.1 Overview (Dashboard Home)

```
┌─── Status Cards ──────────────────────────────────────────────────────┐
│ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────────┐│
│ │ Health   │ │ Price    │ │ 24h P&L  │ │ Equity   │ │ Trades Today││
│ │ ● OK     │ │ $62,450  │ │ +$1,230  │ │ $12,450  │ │ 3           ││
│ └──────────┘ └──────────┘ └──────────┘ └──────────┘ └─────────────┘│
├─── Chart ────────────────────────────────────────────────────────────┤
│ ┌────────────────────────────────────────────────────────────────┐  │
│ │ Candlestick Chart (BTCUSDT 1m)                                │  │
│ │                    📈                                          │  │
│ │            SMA ▲ ▲ ▲                                          │  │
│ │        ██ ██  ██ ██ ██ ██                                     │  │
│ │     ██ ██ ██ ██ ██ ██ ██ ██                                   │  │
│ │  ██ ██ ██ ██ ██ ██ ██ ██ ██                                   │  │
│ │  Volume bars below                                             │  │
│ └────────────────────────────────────────────────────────────────┘  │
├─── Latest Activity ──────────────────────────────────────────────────┤
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐│
│ │ Latest Signal │ │ Risk Decision│ │ Latest Order │ │ Latest Trade ││
│ │ Side: BUY    │ │ Approved     │ │ Filled       │ │ 0.002 BTC    ││
│ │ Strength: 45 │ │ Reason: OK   │ │ 0.002 BTC    │ │ $62,450      ││
│ └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘│
└──────────────────────────────────────────────────────────────────────┘
```

### 4.2 Strategy

```
┌─── Strategy Configuration ───────────────┐ ┌─── Evaluation Result ─────┐
│ Strategy     [SMA Crossover ▼]           │ │ Side     [BUY]            │
│ Version      [v1           ]             │ │ Strength  45.2            │
│ Symbol       [BTCUSDT      ]             │ │ Reason    Fast SMA        │
│ Interval     [1m           ]             │ │           crossed above   │
│ Fast Period  [9            ]             │ │ Generated 2026-06-23      │
│ Slow Period  [21           ]             │ │           10:30:00 UTC    │
│ Lookback     [100          ]             │ └───────────────────────────┘
│ RSI Period   [14           ]             │ ┌───────────────────────────┐
│ RSI Oversold [30           ]             │ │ Chart: indicators overlay │
│ RSI Overbought [70         ]             │ │ SMA ▲ RSI ▲               │
│ ┌───────────────────────────┐            │ └───────────────────────────┘
│ │ [💾 Save Settings]        │            │
│ │ [▶ Evaluate Now]          │            │
│ └───────────────────────────┘            │
└──────────────────────────────────────────┘
```

### 4.3 Backtests

```
┌─── Backtest Form (Left Panel) ──────┐ ┌─── Results (Right Panel) ─────┐
│ Strategy   [SMA ▼]                  │ │ ┌─── Key Metrics ──────────┐  │
│ Symbol     [BTCUSDT]                │ │ │ Return     +12.4%        │  │
│ Interval   [1m]                     │ │ │ Win Rate    58.3%        │  │
│ From       [2026-06-16 00:00]       │ │ │ Max DD     -8.2%        │  │
│ To         [2026-06-23 00:00]       │ │ │ Sharpe      1.24        │  │
│ Fee Rate   [0.001]                  │ │ │ Trades      142         │  │
│ Position   [Percent Equity ▼]       │ │ └────────────────────────┘  │
│ Size       [10%]                    │ │ ┌─── Equity Curve ─────────┐  │
│ Fill Mode  [Next Open ▼]            │ │ │   📈 chart               │  │
│ ┌─────────────────────────┐         │ │ └────────────────────────┘  │
│ │ [▶ Run Backtest]        │         │ │ ┌─── Round Trips Table ────┐  │
│ │ [🔬 Run Optimization]   │         │ │ │ # │Entry│Exit│PnL│Hold│  │
│ └─────────────────────────┘         │ │ │ 1 │62k  │63k │+2%│1h  │  │
│ ┌─── Coverage ────────────┐         │ │ └────────────────────────┘  │
│ │ Available: 10,000   ✅  │         │ └──────────────────────────────┘
│ │ Required:  200          │         │
│ └─────────────────────────┘         │
└─────────────────────────────────────┘
```

### 4.4 Compare

Side-by-side strategy comparison table with:
- Strategy name, parameters
- Return %, win rate, max DD, profit factor, Sharpe, Sortino
- Trade count, avg holding time
- Sortable columns, color-coded best/worst values
- Radar chart comparison (optional)
- Train/Test/Walk-forward results

### 4.5 Pine Script

```
┌─── Code Editor ──────────────────────────┐ ┌─── Validation ───────────┐
│ ┌────────────────────────────────────────┐│ │ Status: ✅ Valid         │
│ │ //@version=5                          ││ │ Indicators found: 3     │
│ │ strategy("My Strategy", overlay=true)  ││ │ ├ SMA(20)               │
│ │ smaFast = ta.sma(close, 9)            ││ │ ├ SMA(50)               │
│ │ smaSlow = ta.sma(close, 21)           ││ │ └ RSI(14)               │
│ │ // Plot                               ││ │                         │
│ │ plot(smaFast, "Fast", color.blue)     ││ │ Rules parsed: 2         │
│ │ plot(smaSlow, "Slow", color.red)      ││ │ ┌─── Saved Strategies ─┐│
│ │ if ta.crossover(smaFast, smaSlow)     ││ │ │ BTC Trend v2    ▶   ││
│ │     strategy.entry("Long", ...)       ││ │ │ SMA Grid v1     ▶   ││
│ └────────────────────────────────────────┘│ └─────────────────────┘│
│ [Validate] [Save Strategy] [Test]        │ └──────────────────────────┘
└──────────────────────────────────────────┘
```

### 4.6 Candles (Market Data Explorer)

```
┌─── Controls ─────────────────────────────────────────────────────────┐
│ Symbol [BTCUSDT] Interval [1m ▼] From [...] To [...] [Search]        │
├─── Candle Table ─────────────────────────────────────────────────────┤
│ Open Time     │ Open │ High │ Low  │ Close │ Volume │ Trades        │
│ 2026-06-23T10:│62450 │62500 │62420 │ 62480 │ 12.5   │ 342           │
│ 2026-06-23T10:│62480 │62520 │62410 │ 62430 │ 8.2    │ 215           │
│ ...                                                                  │
├─── Quick Chart (inline mini candlestick) ────────────────────────────┤
└──────────────────────────────────────────────────────────────────────┘
```

### 4.7 Signals

- Table with strategy, side badge (green/red/gold), strength bar, reason, timestamp
- Filter by strategy, side, date range
- Pagination

### 4.8 Risk

- Settings panel: enable/disable, signal strength range, daily limits, cooldown
- Decision history table with status badges
- Rejection reasons breakdown (pie/bar chart)

### 4.9 Orders

- Table: ID, symbol, side, quantity, price, status, created
- Status badges with color coding
- Filter by status, symbol, side

### 4.10 Trades

- Table: ID, order, symbol, side, qty, price, fee, time
- Real-time updates via WebSocket
- Daily PnL summary bar at top

### 4.11 Ops (Operations)

- Stream stats (name, groups, pending count)
- Pipeline runs (id, status, key, timestamps)
- Safety controls (kill switch toggle, reason input, last updated)
- Audit log

### 4.12 Settings (NEW)

- API key management
- Theme toggle (dark/light — future)
- Notification preferences (future)
- System configuration display

---

## 5. Component Architecture

```
App
├── TopBar
│   ├── Logo
│   ├── SystemStatus (Health + Ready indicators)
│   ├── SymbolSelector
│   ├── IntervalSelector
│   ├── APIKeyIndicator
│   └── LiveClock
├── Sidebar
│   ├── NavItem[] (icon + label, active state)
│   └── SidebarFooter (collapse, theme, version)
├── MainContent
│   ├── TabBar (horizontal tab navigation for sub-views)
│   └── ViewPanel (active view)
│       ├── OverviewView
│       │   ├── StatusCard[]
│       │   ├── CandlestickChart (canvas)
│       │   │   ├── PriceAxis
│       │   │   ├── TimeAxis
│       │   │   ├── OHLCV Candles
│       │   │   ├── Volume Bars
│       │   │   ├── SMA Overlay
│       │   │   └── Crosshair
│       │   └── ActivityPanel[]
│       │       ├── SignalPanel
│       │       ├── RiskPanel
│       │       ├── OrderPanel
│       │       └── TradePanel
│       ├── StrategyView
│       │   ├── StrategyForm
│       │   ├── EvalResult
│       │   └── IndicatorChart
│       ├── BacktestView
│       │   ├── BacktestForm
│       │   ├── CoverageInfo
│       │   ├── BacktestResult
│       │   │   ├── KeyMetrics
│       │   │   ├── EquityCurve (canvas)
│       │   │   └── RoundTripsTable
│       │   └── OptimizationResults
│       ├── ComparisonView
│       │   ├── ComparisonForm
│       │   └── ComparisonTable
│       ├── PineScriptView
│       │   ├── CodeEditor (textarea with basic syntax)
│       │   ├── ValidationResult
│       │   └── SavedStrategies
│       ├── CandlesView → Table + MiniChart
│       ├── SignalsView → FilterableTable
│       ├── RiskView → Settings + History
│       ├── OrdersView → FilterableTable
│       ├── TradesView → FilterableTable + PnL
│       └── OpsView → Streams + Pipeline + Safety + Audit
└── NotificationToast (global, auto-dismiss)
```

---

## 6. Interaction Design

### 6.1 Micro-Interactions

| Element | Interaction |
|---------|-------------|
| Sidebar nav | Hover → glow left border, active → cyan accent + left border |
| Status cards | Hover → slight scale up, content becomes more opaque |
| Tables | Row hover → background highlight |
| Badges | Pulse animation on state change (new signal, fill) |
| Buttons | Click → scale 0.98 → release → scale 1.0 |
| Chart crosshair | Smooth follow, price/time labels on axes |
| Notifications | Slide in from top-right, auto-dismiss after 5s |

### 6.2 Loading States

- Initial load: skeleton screens (pulsing gray rectangles)
- Data refresh: subtle shimmer on updating elements
- API errors: toast notification, element shows stale data with warning indicator
- Empty state: illustrated empty box with helpful message

### 6.3 Real-Time Updates

- WebSocket connection on page load
- Connection status indicator in top bar
- Auto-reconnect with exponential backoff
- Stale data indicator if WS disconnects > 30s
- DOM updates are batched to avoid layout thrashing

---

## 7. Responsive Design

| Breakpoint | Layout |
|------------|--------|
| >1400px | Full layout with expanded sidebar |
| 1024-1400px | Full layout, collapsed sidebar (icons only) |
| 768-1024px | Single column, sidebar as bottom tab bar |
| <768px | Single column, full-screen views, hamburger menu |

---

## 8. Accessibility

- All interactive elements focusable and keyboard-accessible
- Tab order follows visual order
- Color contrast ratios meet WCAG AA (4.5:1 for text, 3:1 for large text)
- Color is never the only indicator — badges include text labels
- ARIA labels on all interactive controls
- Reduced motion media query for animations

---

## 9. Implementation Strategy

### Phase 1: Foundation (This PR)
- [x] Design system (colors, typography, spacing)
- [x] Layout (sidebar, topbar, main content)
- [x] Sidebar navigation with all 11 views
- [x] TopBar with system status, controls, clock
- [x] Complete CSS design system
- [x] Overview view with status cards + chart
- [x] All view panels (Strategy, Backtests, Compare, etc.)
- [x] Canvas-based candlestick chart
- [x] Data fetching layer
- [x] WebSocket real-time feed

### Phase 2: Polish
- [ ] Dark/light theme toggle
- [ ] Mobile responsive
- [ ] Drag-reorderable dashboard widgets
- [ ] Saved layout presets
- [ ] Export backtest results (CSV/PNG)
- [ ] Keyboard shortcuts cheat sheet

### Phase 3: Advanced
- [ ] Multi-chart layouts (compare symbols side-by-side)
- [ ] Advanced order book visualization
- [ ] Heatmap calendar (daily returns)
- [ ] Strategy performance waterfall
- [ ] Portfolio allocation pie chart

---

## 10. File Structure

```
web/dashboard/              # New design replaces old files
├── index.html              # Main HTML shell
├── styles.css              # Complete design system + component styles
└── app.js                  # Application JS (component-based, modular)

web/dashboard-old/          # Old files preserved for reference
├── index.html
├── app.js
└── styles.css
```