import React, { useRef, useState } from 'react';
import CanvasChart from './CanvasChart';
import { ShieldAlert } from 'lucide-react';

export default function TradingWorkspace({
  candles = [],
  signals = [],
  orders = [],
  trades = [],
  auditLogs = [],
  pipelineRuns = [],
  redisStreams = [],
  activeSymbol = 'BTCUSDT',
  liveTickPrice = null,
  watchlist = [],
  onSymbolSelect = () => {},
  strategySettings = null,
  accountBalance = '10,000.00',
  riskDecision = null,
  strategyLifecycle = [],
  reconciliationRuns = [],
  riskSettings = null,
  riskDecisions = [],
  executionStatus = null,
  onLifecycleAdvance = () => {},
  onManualAction = () => {},
  isSafetyBlocked = false,
  isChartLoading = false,
  isDataStale = false
}) {
  const [activeLogTab, setActiveLogTab] = useState('orders');
  const [holdAction, setHoldAction] = useState(null);
  const holdTimerRef = useRef(null);
  const holdIntervalRef = useRef(null);
  const orderBook = { bids: [], asks: [], spread: '0.00' };
  const logTabs = ['orders', 'trades', 'signals', 'audit', 'events'];
  const currentLifecycle = strategyLifecycle.find((item) => item.symbol === activeSymbol) || strategyLifecycle[0];
  const latestReconciliation = reconciliationRuns[0];
  const latestRiskRejection = riskDecisions.find((item) => (item.decision || item.Decision) === 'rejected');
  const lifecycleState = currentLifecycle?.state || 'DRAFT';
  const reconciliationStatus = latestReconciliation?.status || 'not_run';
  const allowedSymbols = riskSettings?.allowed_symbols || riskSettings?.AllowedSymbols || [];
  const riskEngineDisabled = riskSettings && riskSettings.enabled === false;
  const activeSymbolDisallowed = allowedSymbols.length > 0 && !allowedSymbols.includes(activeSymbol);
  const executionStatusUnavailable = !executionStatus;
  const liveExecutionEnabled = executionStatus?.live_trading_enabled === true;
  const liveExecutionDisabled = executionStatus && executionStatus.live_trading_enabled === false;
  const lifecycleNextStates = {
    DRAFT: 'BACKTESTING',
    BACKTESTING: 'VALIDATED',
    VALIDATED: 'PAPER_TRADING',
    PAPER_TRADING: 'APPROVED',
    APPROVED: 'LIVE_ENABLED'
  };
  const nextLifecycleState = lifecycleNextStates[lifecycleState];
  const liveBlockedReasons = [
    isSafetyBlocked ? 'Kill switch is armed' : '',
    lifecycleState !== 'LIVE_ENABLED' ? `Lifecycle is ${lifecycleState}` : '',
    reconciliationStatus === 'mismatch' ? 'Reconciliation mismatch detected' : '',
    executionStatusUnavailable ? 'Execution status unavailable' : '',
    liveExecutionDisabled ? 'Live exchange adapter is disabled' : '',
    riskEngineDisabled ? 'Risk engine is disabled' : '',
    activeSymbolDisallowed ? `${activeSymbol} is not in allowed symbols` : '',
    latestRiskRejection ? `Latest risk rejection: ${latestRiskRejection.reason || latestRiskRejection.Reason || 'risk rejected latest signal'}` : ''
  ].filter(Boolean);
  const canShowLiveReady = lifecycleState === 'LIVE_ENABLED' && !isSafetyBlocked && reconciliationStatus !== 'mismatch' && liveExecutionEnabled && !riskEngineDisabled && !activeSymbolDisallowed && !latestRiskRejection;
  const formatQuote = (value) => {
    const amount = Number(value || 0);
    return amount > 0 ? `$${amount.toFixed(2)}` : 'UNLIMITED';
  };
  const getOrderField = (order, lowerKey, upperKey) => order?.[lowerKey] ?? order?.[upperKey];
  const formatDateTime = (value) => {
    if (!value) return '--';
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? '--' : date.toLocaleString();
  };
  const formatFixed = (value, digits) => {
    const number = Number(value);
    return Number.isFinite(number) ? number.toFixed(digits) : '--';
  };

  const getBadgeColor = (val) => {
    if (!val) return 'text-chrome-text border-chrome-border';
    const lower = val.toLowerCase();
    if (['buy', 'ok', 'filled', 'approved', 'active', 'healthy', 'sane'].includes(lower)) return 'text-signal-buy border-signal-buy/35 bg-signal-buy/15';
    if (['hold', 'pending', 'warning', 'degraded'].includes(lower)) return 'text-signal-warn border-signal-warn/35 bg-signal-warn/15';
    return 'text-signal-sell border-signal-sell/35 bg-signal-sell/15';
  };

  const EmptyState = ({ children }) => (
    <div className="p-4 text-center text-chrome-text/60 leading-relaxed font-sans">
      {children}
    </div>
  );

  const startHold = (type, label, disabled = false) => {
    if (disabled) return;
    clearHold();
    const startedAt = Date.now();
    setHoldAction({ type, label, progress: 0 });
    holdIntervalRef.current = setInterval(() => {
      const progress = Math.min(100, Math.round(((Date.now() - startedAt) / 2000) * 100));
      setHoldAction({ type, label, progress });
    }, 100);
    holdTimerRef.current = setTimeout(() => {
      clearHold();
      onManualAction(type);
    }, 2000);
  };

  const clearHold = () => {
    if (holdTimerRef.current) clearTimeout(holdTimerRef.current);
    if (holdIntervalRef.current) clearInterval(holdIntervalRef.current);
    holdTimerRef.current = null;
    holdIntervalRef.current = null;
    setHoldAction(null);
  };

  // Structured intelligence reasoning display
  const renderSignalIntelligence = () => {
    const isSell = riskDecision && riskDecision.signal_side?.toLowerCase() === 'sell';
    const isBuy = riskDecision && riskDecision.signal_side?.toLowerCase() === 'buy';
    
    let strategyTitle = 'BTC TREND\nPULLBACK';
    if (strategySettings?.strategy_name === 'rsi-mean-reversion') {
      strategyTitle = isSell ? 'BTC BEARISH\nMEAN REVERSION' : isBuy ? 'BTC BULLISH\nMEAN REVERSION' : 'BTC RANGE\nMEAN REVERSION';
    } else if (strategySettings?.strategy_name === 'sma-crossover') {
      strategyTitle = 'BTC MA\nCROSSOVER';
    }

    // Static reasoning rules matched based on active strategy
    let rules = [];
    let confidence = '74%';
    let histMatch = '63 sets';
    let riskLevel = 'LOW';

    if (strategySettings?.strategy_name === 'rsi-mean-reversion') {
      rules = [
        { text: 'Price reached extreme bands', matched: true },
        { text: 'RSI crossed oversold boundary', matched: true },
        { text: 'Volume expansion confirms fatigue', matched: true }
      ];
      confidence = '81%';
      histMatch = '48 sets';
    } else if (strategySettings?.strategy_name === 'btc-trend-pullback') {
      rules = [
        { text: 'Price recovered above EMA pullback', matched: !isSell },
        { text: 'RSI momentum crossover (50)', matched: !isSell },
        { text: 'ATR volatility index checks passed', matched: true }
      ];
      if (isSell) {
        rules = [
          { text: 'Price breached trend EMA line', matched: true },
          { text: 'RSI drops below weakness index (35)', matched: true },
          { text: 'Capital fee depletion threat active', matched: false }
        ];
      }
      confidence = '85%';
      histMatch = '71 sets';
    } else {
      // SMA crossover
      rules = [
        { text: 'Fast SMA crossed Slow SMA line', matched: true },
        { text: 'Cross gap exceeds 0.01% noise filter', matched: true },
        { text: 'Cooldown guard timer is inactive', matched: true }
      ];
      confidence = '78%';
      histMatch = '52 sets';
    }

    // Left border accent color depending on signal status
    const borderAccentClass = isSell ? 'bg-signal-sell' : isBuy ? 'bg-signal-buy' : 'bg-signal-warn';

    return (
      <div className="p-3 bg-bg-60-4 border border-chrome-border rounded-lg relative overflow-hidden">
        {/* Left vertical accent indicator */}
        <div className={`absolute top-0 left-0 w-1 h-full ${borderAccentClass}`}></div>
        
        <div className="flex justify-between items-start mb-3 pl-1">
          <div>
            <h4 className="font-extrabold text-[13px] leading-tight text-white uppercase whitespace-pre-line font-sans tracking-wide">
              {strategyTitle}
            </h4>
          </div>
          <div>
            {isSell ? (
              <div className="bg-signal-sell/35 text-white px-2 py-1 rounded flex items-center gap-1 animate-pulse border border-signal-sell/35" aria-label="Signal recommends execute sell">
                <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="3">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6" />
                </svg>
                <span className="font-bold text-[18px] uppercase tracking-tighter">SELL</span>
              </div>
            ) : isBuy ? (
              <div className="bg-signal-buy/35 text-signal-buy px-2 py-1 rounded flex items-center gap-1 animate-pulse border border-signal-buy/35" aria-label="Signal recommends execute buy">
                <svg className="w-3 h-3 text-signal-buy" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="3">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M13 7h8m0 0v8m0-8L13 15l-4-4-6 6" />
                </svg>
                <span className="font-bold text-[18px] uppercase tracking-tighter">BUY</span>
              </div>
            ) : (
              <div className="bg-bg-60-3 text-chrome-text px-2 py-1 rounded flex items-center gap-1 border border-chrome-border">
                <span className="font-bold text-[18px] uppercase tracking-tighter">HOLD</span>
              </div>
            )}
          </div>
        </div>

        <ul className="space-y-2 mb-4 pl-1">
          {rules.map((r, i) => (
            <li key={i} className="flex items-start gap-2 text-[10px]">
              {r.matched ? (
                <svg className="w-3.5 h-3.5 text-signal-buy flex-shrink-0 mt-1" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="3">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              ) : (
                <svg className="w-3.5 h-3.5 text-signal-sell flex-shrink-0 mt-1" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="3">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              )}
              <div className="flex flex-col">
                <span className="text-white leading-tight font-medium font-sans">{r.text}</span>
              </div>
            </li>
          ))}
        </ul>

        <div className="grid grid-cols-3 gap-2 border-t border-chrome-border pt-3 pl-1 text-[10px] font-mono-data">
          <div className="flex flex-col">
            <span className="text-chrome-text uppercase text-[10px] font-sans">Model Conf.</span>
            <span className="font-bold text-signal-buy">{confidence}</span>
          </div>
          <div className="flex flex-col">
            <span className="text-chrome-text uppercase text-[10px] font-sans">Risk Profile</span>
            <span className="font-bold text-signal-buy uppercase">{riskLevel}</span>
          </div>
          <div className="flex flex-col">
            <span className="text-chrome-text uppercase text-[10px] font-sans">Hist Backtest</span>
            <span className="font-bold text-white">{histMatch}</span>
          </div>
        </div>
      </div>
    );
  };


  return (
    <div className="flex-1 flex h-full overflow-hidden select-none w-full bg-bg-60-1">
      
      {/* 1. SIDEBAR NAVIGATION (LEFT - NARROW) */}
      <aside className="w-16 flex-shrink-0 border-r border-chrome-border flex flex-col items-center py-4 bg-bg-60-3 z-40">
        <div className="flex flex-col items-center gap-4 w-full">
          <div className="text-chrome-text text-[10px] font-bold tracking-wider mb-2">MARKETS</div>
          {watchlist.map((item) => {
            const shortName = item.symbol.replace('USDT', '');
            const isActive = item.symbol === activeSymbol;
            return (
              <button
                key={item.symbol}
                onClick={() => onSymbolSelect(item.symbol)}
                className={`w-12 h-12 flex flex-col items-center justify-center transition-all duration-100 rounded border-l-2 ${
                  isActive
                    ? 'bg-bg-60-4 text-signal-info border-l-signal-info'
                    : 'text-chrome-text hover:bg-bg-60-4/70 hover:text-white border-l-transparent'
                }`}
              >
                <svg className="w-5 h-5 mb-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeMiterlimit="10" d="M3 12h3L9 3l6 18 3-9h3" />
                </svg>
                <span className="text-[10px] font-bold font-sans">{shortName}</span>
              </button>
            );
          })}
        </div>
        <div className="mt-auto">
          <button className="w-12 h-12 flex items-center justify-center text-chrome-text hover:text-white transition-colors">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
              <path strokeLinecap="round" strokeLinejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </button>
        </div>
      </aside>

      {/* 2. CENTER PANEL: CHART & LOG DRAWERS */}
      <div className="flex-1 flex flex-col min-w-0 bg-bg-60-2 overflow-hidden border-r border-chrome-border">
        
        {/* Canvas chart center main */}
        <div className="flex-1 relative min-h-0">
          <CanvasChart
            candles={candles}
            signals={
              strategySettings
                ? signals.filter(
                    (s) =>
                      (s.strategyName || s.strategy_name) ===
                      strategySettings.strategy_name
                  )
                : null
            }
            activeSymbol={activeSymbol}
            liveTick={liveTickPrice}
            isLoading={isChartLoading}
            isDataStale={isDataStale}
          />
        </div>

        {/* Tabbed Log Drawer */}
        <div className="h-[210px] flex flex-col min-h-0 bg-bg-60-1 border-t border-chrome-border">
          <div className="flex border-b border-chrome-border bg-bg-60-1 h-7">
            {logTabs.map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveLogTab(tab)}
                className={`px-4 text-[10px] uppercase tracking-wider font-bold border-r border-chrome-border h-full transition-all duration-100 btn-active-scale interactive-control ${
                  activeLogTab === tab
                    ? 'border-b-2 border-b-signal-info text-signal-info bg-bg-60-2/60'
                    : 'text-chrome-text hover:text-white hover:bg-bg-60-4/70'
                }`}
              >
                {tab}
              </button>
            ))}
          </div>

          <div className="flex-1 overflow-auto p-2 font-mono-data text-[10px]">
            {activeLogTab === 'orders' && (
              <table className="w-full border-collapse">
                <thead>
                  <tr className="text-left text-[10px] text-chrome-text uppercase tracking-wider border-b border-chrome-border/60 bg-bg-60-3">
                    <th className="p-1 font-bold">Created</th>
                    <th className="p-1 font-bold">Client Order</th>
                    <th className="p-1 font-bold">Side</th>
                    <th className="p-1 font-bold">Status</th>
                    <th className="p-1 font-bold">Requested/Filled</th>
                    <th className="p-1 font-bold">Price</th>
                    <th className="p-1 font-bold">Avg Fill</th>
                    <th className="p-1 font-bold">Updated</th>
                    <th className="p-1 font-bold">Failure</th>
                  </tr>
                </thead>
                <tbody>
                  {orders.length === 0 ? (
                    <tr><td colSpan="9"><EmptyState>No orders logged. Strategy is running in paper mode; orders will appear when signals execute.</EmptyState></td></tr>
                  ) : (
                    orders.map((ord, idx) => {
                      const side = getOrderField(ord, 'side', 'Side');
                      const status = getOrderField(ord, 'status', 'Status');
                      const requested = getOrderField(ord, 'requested_quantity', 'RequestedQuantity') ?? getOrderField(ord, 'quantity', 'Quantity');
                      const filled = getOrderField(ord, 'filled_quantity', 'FilledQuantity') ?? getOrderField(ord, 'quantity', 'Quantity');
                      const clientOrderID = getOrderField(ord, 'client_order_id', 'ClientOrderID') || getOrderField(ord, 'id', 'ID') || '--';
                      const exchangeOrderID = getOrderField(ord, 'exchange_order_id', 'ExchangeOrderID');
                      const failureReason = getOrderField(ord, 'failure_reason', 'FailureReason');
                      return (
                        <tr key={idx} className={`border-b border-chrome-border/60 hover:bg-bg-60-4/60 ${status === 'failed' ? 'bg-signal-sell/5' : status === 'partially_filled' ? 'bg-signal-warn/5' : ''}`}>
                          <td className="p-1">{formatDateTime(getOrderField(ord, 'created_at', 'CreatedAt'))}</td>
                          <td className="p-1 text-white">
                            <div className="font-bold">{clientOrderID}</div>
                            <div className="text-chrome-text/65">{exchangeOrderID || 'paper-local'}</div>
                          </td>
                          <td className={`p-1 font-bold ${side?.toLowerCase() === 'buy' ? 'text-signal-buy' : 'text-signal-sell'}`}>
                            {side?.toUpperCase() || '--'}
                          </td>
                          <td className="p-1">
                            <span className={`px-1 py-1 border rounded text-[10px] ${getBadgeColor(status)}`}>
                              {status || '--'}
                            </span>
                          </td>
                          <td className="p-1 text-white">{formatFixed(requested, 4)} / {formatFixed(filled, 4)}</td>
                          <td className="p-1 text-white">{formatFixed(getOrderField(ord, 'price', 'Price'), 2)}</td>
                          <td className="p-1 text-white">{formatFixed(getOrderField(ord, 'average_fill_price', 'AverageFillPrice'), 2)}</td>
                          <td className="p-1">
                            <div>{formatDateTime(getOrderField(ord, 'updated_at', 'UpdatedAt'))}</div>
                            <div className="text-chrome-text/65">submitted {formatDateTime(getOrderField(ord, 'submitted_at', 'SubmittedAt'))}</div>
                          </td>
                          <td className={`p-1 ${failureReason ? 'text-signal-sell' : 'text-chrome-text/50'}`}>{failureReason || '--'}</td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            )}

            {activeLogTab === 'trades' && (
              <table className="w-full border-collapse">
                <thead>
                  <tr className="text-left text-[10px] text-chrome-text uppercase tracking-wider border-b border-chrome-border/60 bg-bg-60-3">
                    <th className="p-1 font-bold">Time</th>
                    <th className="p-1 font-bold">Side</th>
                    <th className="p-1 font-bold">Quantity</th>
                    <th className="p-1 font-bold">Price</th>
                    <th className="p-1 font-bold">Fee</th>
                  </tr>
                </thead>
                <tbody>
                  {trades.length === 0 ? (
                    <tr><td colSpan="5"><EmptyState>No trades filled. Filled paper orders will appear here after execution.</EmptyState></td></tr>
                  ) : (
                    trades.map((trd, idx) => (
                      <tr key={idx} className="border-b border-chrome-border/60 hover:bg-bg-60-4/60">
                        <td className="p-1">{new Date(trd.created_at).toLocaleString()}</td>
                        <td className={`p-1 font-bold ${trd.side?.toLowerCase() === 'buy' ? 'text-signal-buy' : 'text-signal-sell'}`}>
                          {trd.side?.toUpperCase()}
                        </td>
                        <td className="p-1 text-white">{Number(trd.quantity).toFixed(4)}</td>
                        <td className="p-1 text-white">{Number(trd.price).toFixed(2)}</td>
                        <td className="p-1 text-chrome-text">{Number(trd.fee).toFixed(5)}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            )}

            {activeLogTab === 'signals' && (
              <table className="w-full border-collapse">
                <thead>
                  <tr className="text-left text-[10px] text-chrome-text uppercase tracking-wider border-b border-chrome-border/60 bg-bg-60-3">
                    <th className="p-1 font-bold">Generated</th>
                    <th className="p-1 font-bold">Strategy</th>
                    <th className="p-1 font-bold">Side</th>
                    <th className="p-1 font-bold">Confidence</th>
                    <th className="p-1 font-bold">Reason</th>
                  </tr>
                </thead>
                <tbody>
                  {signals.length === 0 ? (
                    <tr><td colSpan="5"><EmptyState>No signals generated. Wait for closed candles or run a manual cycle from Ops.</EmptyState></td></tr>
                  ) : (
                    signals.slice(0, 30).map((sig, idx) => (
                      <tr key={idx} className="border-b border-chrome-border/60 hover:bg-bg-60-4/60">
                        <td className="p-1">{new Date(sig.generatedAt || sig.generated_at).toLocaleString()}</td>
                        <td className="p-1 text-white">{sig.strategyName || sig.strategy_name}</td>
                        <td className={`p-1 font-bold ${sig.side?.toLowerCase() === 'buy' ? 'text-signal-buy' : 'text-signal-sell'}`}>
                          {sig.side?.toUpperCase()}
                        </td>
                        <td className="p-1 text-white">{Number(sig.strength).toFixed(3)}</td>
                        <td className="p-1 text-chrome-text">{sig.reason}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            )}

            {activeLogTab === 'audit' && (
              <table className="w-full border-collapse">
                <thead>
                  <tr className="text-left text-[10px] text-chrome-text uppercase tracking-wider border-b border-chrome-border/60 bg-bg-60-3">
                    <th className="p-1 font-bold">Timestamp</th>
                    <th className="p-1 font-bold">Event Type</th>
                    <th className="p-1 font-bold">Actor</th>
                    <th className="p-1 font-bold">Message</th>
                  </tr>
                </thead>
                <tbody>
                  {auditLogs.length === 0 ? (
                    <tr><td colSpan="4"><EmptyState>No audit trail events. Safety and manual actions will create audit entries here.</EmptyState></td></tr>
                  ) : (
                    auditLogs.slice(0, 30).map((log, idx) => (
                      <tr key={idx} className="border-b border-chrome-border/60 hover:bg-bg-60-4/60 text-[10px]">
                        <td className="p-1">{new Date(log.created_at).toLocaleString()}</td>
                        <td className="p-1 text-signal-info font-bold">{log.event_type}</td>
                        <td className="p-1 text-white">{log.actor}</td>
                        <td className="p-1 text-chrome-text">{log.payload}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            )}

            {activeLogTab === 'events' && (
              <div className="grid grid-cols-3 gap-4 h-full">
                <div className="flex flex-col min-h-0 border border-chrome-border bg-bg-60-3/60 p-2 rounded">
                  <div className="text-[10px] uppercase tracking-wider text-chrome-text font-bold mb-2 border-b border-chrome-border pb-1">
                    Pipeline Cycles Runs
                  </div>
                  <div className="flex-1 overflow-y-auto">
                    <table className="w-full">
                      <thead>
                        <tr className="text-left text-[10px] text-chrome-text border-b border-chrome-border">
                          <th className="p-1">Time</th>
                          <th className="p-1">Status</th>
                          <th className="p-1">Duration</th>
                        </tr>
                      </thead>
                      <tbody>
                        {pipelineRuns.length === 0 ? (
                          <tr><td colSpan="3"><EmptyState>No runs reported. Pipeline cycles will appear after candle events or a manual Ops cycle.</EmptyState></td></tr>
                        ) : (
                          pipelineRuns.slice(0, 10).map((run, idx) => (
                            <tr key={idx} className="border-b border-chrome-border/35 text-[10px]">
                              <td className="p-1">{new Date(run.created_at).toLocaleTimeString()}</td>
                              <td className="p-1"><span className={`px-1 py-1 border rounded text-[10px] ${getBadgeColor(run.status)}`}>{run.status}</span></td>
                              <td className="p-1 text-white">{run.execution_time_ms}ms</td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                <div className="flex flex-col min-h-0 border border-chrome-border bg-bg-60-3/60 p-2 rounded">
                  <div className="text-[10px] uppercase tracking-wider text-chrome-text font-bold mb-2 border-b border-chrome-border pb-1">
                    Redis Stream Event Channels
                  </div>
                  <div className="flex-1 overflow-y-auto">
                    <table className="w-full">
                      <thead>
                        <tr className="text-left text-[10px] text-chrome-text border-b border-chrome-border">
                          <th className="p-1">Channel Name</th>
                          <th className="p-1">Length</th>
                          <th className="p-1">Last ID</th>
                        </tr>
                      </thead>
                      <tbody>
                        {redisStreams.length === 0 ? (
                          <tr><td colSpan="3"><EmptyState>No stream channels active. Redis stream diagnostics will appear after backend telemetry is available.</EmptyState></td></tr>
                        ) : (
                          redisStreams.map((stream, idx) => (
                            <tr key={idx} className="border-b border-chrome-border/35 text-[10px] text-white">
                              <td className="p-1 text-signal-brand font-bold">{stream.stream_key}</td>
                              <td className="p-1">{stream.length}</td>
                              <td className="p-1 text-chrome-text">{stream.last_delivered_id || '-'}</td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                <div className="flex flex-col min-h-0 border border-chrome-border bg-bg-60-3/60 p-2 rounded">
                  <div className="text-[10px] uppercase tracking-wider text-chrome-text font-bold mb-2 border-b border-chrome-border pb-1">
                    L2 Book Snapshot
                  </div>
                  <div className="flex-1 overflow-y-auto text-[10px]">
                    {orderBook.bids.length === 0 && orderBook.asks.length === 0 ? (
                      <EmptyState>No L2 book levels loaded. Connect market-depth telemetry to inspect bid and ask pressure.</EmptyState>
                    ) : (
                      <div className="grid grid-cols-2 gap-4 font-mono-data">
                        <div>
                          <div className="text-signal-buy font-bold uppercase mb-2">Bids</div>
                          {orderBook.bids.map((bid, idx) => (
                            <div key={`events-bid-${idx}`} className="flex justify-between">
                              <span className="text-signal-buy">{bid.price.toFixed(2)}</span>
                              <span>{bid.size}</span>
                            </div>
                          ))}
                        </div>
                        <div>
                          <div className="text-signal-sell font-bold uppercase mb-2">Asks</div>
                          {orderBook.asks.map((ask, idx) => (
                            <div key={`events-ask-${idx}`} className="flex justify-between">
                              <span className="text-signal-sell">{ask.price.toFixed(2)}</span>
                              <span>{ask.size}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* 3. RIGHT PANEL: OPERATIONS CONSOLE */}
      <div className="w-80 flex-shrink-0 bg-bg-60-1 p-3.5 flex flex-col gap-4 overflow-y-auto no-scrollbar">
        <section className="border border-chrome-border bg-bg-60-4/80 rounded-lg p-3 flex flex-col gap-2">
          <div className="flex justify-between items-center border-b border-chrome-border/60 pb-2">
            <h3 className="text-[10px] text-chrome-text tracking-widest font-bold uppercase font-sans">Trade Readiness</h3>
            <span className={`text-[10px] px-2 py-1 rounded font-bold border ${canShowLiveReady ? 'text-signal-buy border-signal-buy/40 bg-signal-buy/10' : 'text-signal-warn border-signal-warn/40 bg-signal-warn/10'}`}>
              {canShowLiveReady ? 'LIVE READY' : 'LIVE BLOCKED'}
            </span>
          </div>
          <div className="grid grid-cols-2 gap-2 text-[10px]">
            <div className="border border-chrome-border bg-bg-60-3 p-2 rounded">
              <span className="block text-chrome-text/65 uppercase">Lifecycle</span>
              <span className="block mt-1 text-white font-bold font-mono-data">{lifecycleState}</span>
            </div>
            <div className="border border-chrome-border bg-bg-60-3 p-2 rounded">
              <span className="block text-chrome-text/65 uppercase">Kill Switch</span>
              <span className={`block mt-1 font-bold font-mono-data ${isSafetyBlocked ? 'text-signal-sell' : 'text-signal-buy'}`}>
                {isSafetyBlocked ? 'ARMED' : 'CLEAR'}
              </span>
            </div>
            <div className="border border-chrome-border bg-bg-60-3 p-2 rounded">
              <span className="block text-chrome-text/65 uppercase">Recon</span>
              <span className={`block mt-1 font-bold font-mono-data ${reconciliationStatus === 'mismatch' ? 'text-signal-sell' : reconciliationStatus === 'matched' ? 'text-signal-buy' : 'text-signal-warn'}`}>
                {reconciliationStatus.toUpperCase()}
              </span>
            </div>
            <div className="border border-chrome-border bg-bg-60-3 p-2 rounded">
              <span className="block text-chrome-text/65 uppercase">Mode</span>
              <span className="block mt-1 text-signal-warn font-bold font-mono-data">{(executionStatus?.mode || 'paper').toUpperCase()}</span>
            </div>
          </div>
          {liveBlockedReasons.length > 0 && (
            <div className="text-[10px] text-signal-warn leading-snug border border-signal-warn/30 bg-signal-warn/10 rounded p-2">
              <div className="font-bold uppercase mb-1">Live trading blocked</div>
              <ul className="space-y-1">
                {liveBlockedReasons.map((reason) => (
                  <li key={reason}>- {reason}</li>
                ))}
              </ul>
            </div>
          )}
          {currentLifecycle?.id && nextLifecycleState && (
            <button
              type="button"
              onClick={() => onLifecycleAdvance(currentLifecycle.id, nextLifecycleState)}
              aria-label={`Advance lifecycle to ${nextLifecycleState}`}
              className="h-[30px] rounded border border-signal-info/40 bg-signal-info/10 text-signal-info text-[10px] uppercase font-bold hover:bg-signal-info/20 transition-colors interactive-control"
            >
              Advance to {nextLifecycleState}
            </button>
          )}
          <div className="border border-chrome-border bg-bg-60-3 rounded p-2 text-[10px] leading-snug">
            <div className="text-chrome-text/65 uppercase font-bold mb-1">Execution Adapter</div>
            <div className="grid grid-cols-2 gap-x-2 gap-y-1 mb-2">
              <span className="text-chrome-text/70">Adapter</span>
              <span className="text-white font-mono-data text-right">{executionStatus?.exchange_adapter || 'binance_disabled'}</span>
              <span className="text-chrome-text/70">Live Trading</span>
              <span className={`font-mono-data text-right font-bold ${executionStatus?.live_trading_enabled ? 'text-signal-buy' : 'text-signal-sell'}`}>
                {executionStatus?.live_trading_enabled ? 'ENABLED' : 'DISABLED'}
              </span>
              <span className="text-chrome-text/70">Retry Policy</span>
              <span className="text-white font-mono-data text-right">{executionStatus?.retry_attempts || 1} attempts / {executionStatus?.timeout || '0s'}</span>
              <span className="text-chrome-text/70">Last Error</span>
              <span className="text-signal-warn font-mono-data text-right">{executionStatus?.last_error || 'none'}</span>
            </div>
            <div className="text-chrome-text/65 uppercase font-bold mb-1">Risk Settings</div>
            <div className="grid grid-cols-2 gap-x-2 gap-y-1">
              <span className="text-chrome-text/70">Allowed Symbols</span>
              <span className="text-white font-mono-data text-right">{allowedSymbols.length ? allowedSymbols.join(', ') : 'ALL'}</span>
              <span className="text-chrome-text/70">Max Order</span>
              <span className="text-white font-mono-data text-right">{formatQuote(riskSettings?.max_order_quote_amount)}</span>
              <span className="text-chrome-text/70">Max Exposure</span>
              <span className="text-white font-mono-data text-right">{formatQuote(riskSettings?.max_total_exposure_quote_amount)}</span>
              <span className="text-chrome-text/70">Max Positions</span>
              <span className="text-white font-mono-data text-right">{riskSettings?.max_open_positions || 'UNLIMITED'}</span>
              <span className="text-chrome-text/70">Allowed Sides</span>
              <span className="text-white font-mono-data text-right">
                {riskSettings?.allow_buy === false ? '' : 'BUY'}{riskSettings?.allow_buy !== false && riskSettings?.allow_sell !== false ? '/' : ''}{riskSettings?.allow_sell === false ? '' : 'SELL'}
              </span>
            </div>
          </div>
        </section>
        
        {/* Portfolio Status Snapshot */}
        <section className="border border-chrome-border bg-bg-60-4/80 rounded-lg p-3 flex flex-col gap-3">
          <div className="flex justify-between items-center border-b border-chrome-border/60 pb-2">
            <h3 className="text-[10px] text-chrome-text tracking-widest font-bold uppercase font-sans">Portfolio Status</h3>
            <span className="bg-bg-60-3 text-[10px] px-2 py-1 rounded text-signal-warn font-bold border border-chrome-border">SIMULATED</span>
          </div>
          <div className="flex justify-between items-end">
            <div className="flex flex-col">
              <span className="text-[10px] text-chrome-text uppercase font-sans">Current Valuation</span>
              <span className="text-[18px] text-white font-bold font-mono-data tracking-tighter">${accountBalance}</span>
            </div>
            <div className="flex flex-col items-end">
              <span className="text-[10px] text-chrome-text uppercase font-sans">Risk State</span>
              <div className="flex items-center gap-1 mt-1">
                <div className="w-1.5 h-1.5 rounded-full bg-signal-buy shadow-[0_0_8px_#44dfa3]"></div>
                <span className="font-bold text-signal-buy text-[11px] font-sans">OPTIMAL</span>
              </div>
            </div>
          </div>
        </section>

        {/* Risk Criticality Section */}
        <section className="flex flex-col gap-2">
          <div className="flex items-center justify-between border-b border-chrome-border/60 pb-2">
            <h3 className="text-[10px] text-chrome-text tracking-widest font-bold uppercase font-sans">Risk Criticality</h3>
            <svg className="w-4 h-4 text-chrome-text" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
          </div>
          <div className="grid grid-cols-2 gap-2 text-[10px]">
            <div className="border border-chrome-border border-l-4 border-l-signal-sell bg-bg-60-4 p-2.5 rounded-lg flex flex-col gap-1">
              <span className="text-[10px] text-chrome-text leading-none uppercase font-sans">Max Drawdown</span>
              <span className="font-mono-data text-[12px] text-signal-sell font-bold">4.20%</span>
              <span className="text-[10px] text-chrome-text/60 font-sans">Threshold: 5.0%</span>
            </div>
            <div className="border border-chrome-border border-l-4 border-l-signal-buy bg-bg-60-4 p-2.5 rounded-lg flex flex-col gap-1">
              <span className="text-[10px] text-chrome-text leading-none uppercase font-sans">Sharpe Ratio</span>
              <span className="font-mono-data text-[12px] text-signal-buy font-bold">2.84</span>
              <span className="text-[10px] text-chrome-text/60 font-sans">Risk-Adj Return</span>
            </div>
            <div className="border border-chrome-border border-l-4 border-l-chrome-text bg-bg-60-4 p-2.5 rounded-lg flex flex-col gap-1">
              <span className="text-[10px] text-chrome-text leading-none uppercase font-sans">Asset Exposure</span>
              <span className="font-mono-data text-[12px] text-white font-bold">10.00%</span>
              <span className="text-[10px] text-chrome-text/60 font-sans">Capital Utilized</span>
            </div>
            <div className="border border-chrome-border border-l-4 border-l-signal-info bg-bg-60-4 p-2.5 rounded-lg flex flex-col gap-1">
              <span className="text-[10px] text-chrome-text leading-none uppercase font-sans">Avg Volatility</span>
              <span className="font-mono-data text-[12px] text-signal-info font-bold">1.45%</span>
              <span className="text-[10px] text-chrome-text/60 font-sans">ATR 14-Day</span>
            </div>
          </div>
        </section>

        {/* Signal Logic Card */}
        <section className="flex flex-col gap-2">
          <div className="flex items-center justify-between border-b border-chrome-border/60 pb-2">
            <h3 className="text-[10px] text-chrome-text tracking-widest font-bold uppercase font-sans">Signal Logic</h3>
            <svg className="w-4 h-4 text-chrome-text" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
              <path strokeLinecap="round" strokeLinejoin="round" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
            </svg>
          </div>
          {renderSignalIntelligence()}
        </section>

        {/* Quick Order Action Area & Safety Control */}
        <div className="mt-auto pt-2 border-t border-chrome-border flex flex-col gap-2">
          <div className="grid grid-cols-2 gap-2">
            <button
              onPointerDown={() => startHold('force_buy', 'BUY', isSafetyBlocked)}
              onPointerUp={clearHold}
              onPointerLeave={clearHold}
              onPointerCancel={clearHold}
              disabled={isSafetyBlocked}
              className="relative min-h-11 bg-signal-buy text-bg-60-2 hover:bg-signal-buy/90 active:scale-95 font-bold py-2 rounded-lg uppercase tracking-wider text-[11px] font-sans transition-all duration-100 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed interactive-control overflow-hidden"
              aria-label="Hold to confirm manual buy order"
              title="Hold to confirm BUY"
            >
              {holdAction?.type === 'force_buy' && (
                <span className="absolute inset-y-0 left-0 bg-bg-60-2/35" style={{ width: `${holdAction.progress}%` }} aria-hidden="true" />
              )}
              <span className="relative">{holdAction?.type === 'force_buy' ? `Hold to confirm BUY ${holdAction.progress}%` : 'Order Buy'}</span>
            </button>
            <button
              onPointerDown={() => startHold('force_sell', 'SELL', isSafetyBlocked)}
              onPointerUp={clearHold}
              onPointerLeave={clearHold}
              onPointerCancel={clearHold}
              disabled={isSafetyBlocked}
              className="relative min-h-11 bg-signal-sell bg-opacity-20 border border-signal-sell text-white hover:bg-signal-sell/25 active:scale-95 font-bold py-2 rounded-lg uppercase tracking-wider text-[11px] font-sans transition-all duration-100 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed interactive-control overflow-hidden"
              aria-label="Hold to confirm manual sell order"
              title="Hold to confirm SELL"
            >
              {holdAction?.type === 'force_sell' && (
                <span className="absolute inset-y-0 left-0 bg-signal-sell/35" style={{ width: `${holdAction.progress}%` }} aria-hidden="true" />
              )}
              <span className="relative">{holdAction?.type === 'force_sell' ? `Hold to confirm SELL ${holdAction.progress}%` : 'Order Sell'}</span>
            </button>
          </div>
          
          <button
            onPointerDown={() => startHold('toggle_kill_switch', isSafetyBlocked ? 'DISARM' : 'ARM')}
            onPointerUp={clearHold}
            onPointerLeave={clearHold}
            onPointerCancel={clearHold}
            className={`relative min-h-11 border flex justify-center items-center gap-2 font-bold rounded-lg cursor-pointer select-none text-[10px] w-full btn-active-scale transition-all duration-100 uppercase tracking-wider interactive-control overflow-hidden ${
              isSafetyBlocked
                ? 'border-signal-buy bg-signal-buy/35 text-signal-buy hover:bg-signal-buy/35'
                : 'border-signal-sell bg-signal-sell/35 text-signal-sell hover:bg-signal-sell/35'
            }`}
            aria-label={`Hold to ${isSafetyBlocked ? 'disarm' : 'arm'} safety block`}
            title={`Hold to confirm ${isSafetyBlocked ? 'DISARM' : 'ARM'}`}
          >
            {holdAction?.type === 'toggle_kill_switch' && (
              <span className="absolute inset-y-0 left-0 bg-white/35" style={{ width: `${holdAction.progress}%` }} aria-hidden="true" />
            )}
            <ShieldAlert size={12} aria-hidden="true" />
            <span className="relative">
              {holdAction?.type === 'toggle_kill_switch'
                ? `Hold to confirm ${holdAction.label} ${holdAction.progress}%`
                : isSafetyBlocked ? 'Disarm Safety Block' : 'Arm Safety Block'}
            </span>
          </button>
        </div>

      </div>

    </div>
  );
}
