import React, { useState, useEffect, useRef } from 'react';
import TradingWorkspace from './components/TradingWorkspace';
import ResearchWorkspace from './components/ResearchWorkspace';
import StrategyWorkspace from './components/StrategyWorkspace';
import RiskWorkspace from './components/RiskWorkspace';
import OpsWorkspace from './components/OpsWorkspace';
import SettingsModal from './components/SettingsModal';
import { Search, Settings, CheckCircle, AlertTriangle, Info } from 'lucide-react';

export default function App() {
  const [activeWorkspace, setActiveWorkspace] = useState('Trading');
  const [activeSymbol, setActiveSymbol] = useState('BTCUSDT');
  const [liveTickPrice, setLiveTickPrice] = useState(null);
  const [liveTickDirection, setLiveTickDirection] = useState(null);
  const [liveTickChange, setLiveTickChange] = useState(null);
  const [lastTickAt, setLastTickAt] = useState(null);
  const [isDataStale, setIsDataStale] = useState(false);
  const [isChartPriming, setIsChartPriming] = useState(true);
  const [backendConnected, setBackendConnected] = useState(false);
  const [isSafetyBlocked, setIsSafetyBlocked] = useState(false);
  const [toast, setToast] = useState(null);
  
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [terminalSettings, setTerminalSettings] = useState({
    apiKey: 'AQ.Ab8RN6LovyaG6ou... [Active]',
    tradeSize: 0.15,
    slippage: 0.05,
    refreshInterval: 1500,
    backtestSaveEnabled: true
  });

  // Core Data States
  const [candles, setCandles] = useState([]);
  const [signals, setSignals] = useState([]);
  const [orders, setOrders] = useState([]);
  const [trades, setTrades] = useState([]);
  const [auditLogs, setAuditLogs] = useState([]);
  const [pipelineRuns, setPipelineRuns] = useState([]);
  const [redisStreams, setRedisStreams] = useState([]);
  const [strategyLifecycle, setStrategyLifecycle] = useState([]);
  const [reconciliationRuns, setReconciliationRuns] = useState([]);
  const [riskSettings, setRiskSettings] = useState(null);
  const [riskDecisions, setRiskDecisions] = useState([]);
  const [executionStatus, setExecutionStatus] = useState(null);
  const [strategySettings, setStrategySettings] = useState({ strategy_name: 'btc-trend-pullback' });
  const [watchlist, setWatchlist] = useState([]);

  // Command Palette States
  const [isPaletteOpen, setIsPaletteOpen] = useState(false);
  const [paletteQuery, setPaletteQuery] = useState('');
  const [selectedPaletteIdx, setSelectedPaletteIdx] = useState(0);

  const commandPaletteInputRef = useRef(null);

  const commands = [
    { text: 'Switch Workspace to [ Trading ]', action: () => setActiveWorkspace('Trading'), shortcut: 'view trading' },
    { text: 'Switch Workspace to [ Research ]', action: () => setActiveWorkspace('Research'), shortcut: 'view research' },
    { text: 'Switch Workspace to [ Strategy ]', action: () => setActiveWorkspace('Strategy'), shortcut: 'view strategy' },
    { text: 'Switch Workspace to [ Risk ]', action: () => setActiveWorkspace('Risk'), shortcut: 'view risk' },
    { text: 'Switch Workspace to [ Ops ]', action: () => setActiveWorkspace('Ops'), shortcut: 'view ops' },
    { text: 'Arm Execution Safety Block', action: () => triggerKillSwitchAction(true), shortcut: 'action arm' },
    { text: 'Disarm Execution Safety Block', action: () => triggerKillSwitchAction(false), shortcut: 'action disarm' },
    { text: 'Force Manual Position Exit', action: () => executeManualAction('force_sell'), shortcut: 'action exit' },
    { text: 'Focus Active Symbol Input', action: () => document.getElementById('symbol-input-box')?.focus(), shortcut: 'focus symbol' }
  ];

  // Helper for structured reasoning alerts
  const showToast = (message, type = 'info') => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 4000);
  };

  // Switch safety execution block status
  const triggerKillSwitchAction = (armed) => {
    setIsSafetyBlocked(armed);
    const text = armed ? 'Execution Block ARMED. All orders rejected.' : 'Execution Block DISARMED. Resuming simulator.';
    showToast(text, armed ? 'error' : 'success');
    
    // Add event log directly
    const log = {
      created_at: new Date().toISOString(),
      event_type: 'SAFETY_BLOCK_CHANGED',
      actor: 'operator',
      payload: armed ? 'Execution safety block ARMED - emergency mode' : 'Execution safety block DISARMED - resuming trades'
    };
    setAuditLogs((prev) => [log, ...prev]);
  };

  const advanceStrategyLifecycle = async (id, nextState) => {
    if (!id || !nextState) {
      showToast('Lifecycle advance rejected: missing lifecycle target.', 'error');
      return;
    }
    try {
      const res = await fetch(`/api/v1/strategy/lifecycle/${id}/advance`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          state: nextState,
          reason: `Operator advanced lifecycle to ${nextState}`,
          updated_by: 'operator'
        })
      });
      const body = await res.json();
      if (!res.ok) {
        throw new Error(body.error || 'Lifecycle advance failed');
      }
      if (body.data) {
        setStrategyLifecycle((current) => {
          const exists = current.some((item) => item.id === body.data.id);
          if (!exists) return [body.data, ...current];
          return current.map((item) => item.id === body.data.id ? body.data : item);
        });
      }
      showToast(`Strategy lifecycle advanced to ${nextState}.`, 'success');
    } catch (err) {
      showToast(err.message, 'error');
    }
  };

  // Manual Console actions override
  const executeManualAction = (type) => {
    if (isSafetyBlocked && type !== 'toggle_kill_switch') {
      showToast('Action rejected: Emergency execution block is active.', 'error');
      return;
    }
    if ((type === 'force_buy' || type === 'force_sell') && !liveTickPrice) {
      showToast('Action rejected: no backend market price is available.', 'error');
      return;
    }

    if (type === 'force_buy') {
      const buyPrice = liveTickPrice;
      const size = 0.15;
      const order = {
        created_at: new Date().toISOString(),
        side: 'buy',
        status: 'filled',
        quantity: size,
        price: buyPrice
      };
      const trade = {
        created_at: new Date().toISOString(),
        side: 'buy',
        quantity: size,
        price: buyPrice,
        fee: (buyPrice * size * 0.001)
      };
      setOrders((prev) => [order, ...prev]);
      setTrades((prev) => [trade, ...prev]);
      showToast(`MANUAL ORDER FILLED: BUY 0.15 BTC @ ${buyPrice.toFixed(2)}`, 'success');
    } else if (type === 'force_sell') {
      if (trades.length === 0) {
        showToast('No active positions to exit.', 'error');
        return;
      }
      const sellPrice = liveTickPrice;
      const size = 0.15;
      const order = {
        created_at: new Date().toISOString(),
        side: 'sell',
        status: 'filled',
        quantity: size,
        price: sellPrice
      };
      const trade = {
        created_at: new Date().toISOString(),
        side: 'sell',
        quantity: size,
        price: sellPrice,
        fee: (sellPrice * size * 0.001)
      };
      setOrders((prev) => [order, ...prev]);
      setTrades((prev) => [trade, ...prev]);
      showToast(`MANUAL ORDER FILLED: SELL 0.15 BTC @ ${sellPrice.toFixed(2)}`, 'success');
    } else if (type === 'toggle_kill_switch') {
      triggerKillSwitchAction(!isSafetyBlocked);
    }
  };

  // API fetches
  const loadBackendData = async () => {
    try {
      const candlesRes = await fetch(`/api/v1/market/candles?symbol=${activeSymbol}&interval=1m&limit=100`);
      if (candlesRes.ok) {
        setBackendConnected(true);
        const body = await candlesRes.json();
        if (body.data && body.data.length > 0) {
          const sorted = body.data.map(c => ({
            openTime: c.OpenTime || c.open_time,
            open: Number(c.Open || c.open),
            high: Number(c.High || c.high),
            low: Number(c.Low || c.low),
            close: Number(c.Close || c.close),
            volume: Number(c.Volume || c.volume)
          })).sort((a,b) => new Date(a.openTime).getTime() - new Date(b.openTime).getTime());
          setCandles(sorted);
          if (sorted.length > 0) {
            const latest = sorted[sorted.length - 1];
            const previous = sorted.length > 1 ? sorted[sorted.length - 2] : latest;
            const change = previous.close ? ((latest.close - previous.close) / previous.close * 100) : 0;
            const direction = latest.close >= previous.close ? 'up' : 'down';
            const formattedChange = `${change >= 0 ? '+' : ''}${change.toFixed(2)}%`;
            setLiveTickPrice(latest.close);
            setLiveTickDirection(direction);
            setLiveTickChange(formattedChange);
            setLastTickAt(Date.now());
            setIsDataStale(false);
            setWatchlist([{
              symbol: activeSymbol,
              price: latest.close,
              change: formattedChange,
              direction,
              history: sorted.slice(-24).map(c => c.close)
            }]);
          }
        } else {
          setCandles([]);
          setLiveTickPrice(null);
          setLiveTickDirection(null);
          setLiveTickChange(null);
          setIsDataStale(true);
          setWatchlist([]);
        }
      }
    } catch {
      setBackendConnected(false);
      setCandles([]);
      setLiveTickPrice(null);
      setLiveTickDirection(null);
      setLiveTickChange(null);
      setIsDataStale(true);
      setWatchlist([]);
    }

    try {
      const settingsRes = await fetch('/api/v1/strategy/settings');
      if (settingsRes.ok) {
        setBackendConnected(true);
        const body = await settingsRes.json();
        if (body.data) setStrategySettings(body.data);
      }

      const [ordersRes, tradesRes, signalsRes, auditRes, runsRes, streamsRes, safetyRes, lifecycleRes, reconciliationRes, riskSettingsRes, riskDecisionsRes, executionStatusRes] = await Promise.all([
        fetch(`/api/v1/paper/orders?symbol=${activeSymbol}&limit=20`),
        fetch(`/api/v1/paper/trades?symbol=${activeSymbol}&limit=20`),
        fetch(`/api/v1/signals?symbol=${activeSymbol}&limit=30`),
        fetch('/api/v1/audit/events?limit=30'),
        fetch('/api/v1/ops/pipeline-runs?limit=20'),
        fetch('/api/v1/ops/streams'),
        fetch('/api/v1/safety/status'),
        fetch(`/api/v1/strategy/lifecycle?symbol=${activeSymbol}&limit=10`),
        fetch('/api/v1/reconciliation/runs?limit=5'),
        fetch('/api/v1/risk/settings'),
        fetch(`/api/v1/risk-decisions?symbol=${activeSymbol}&limit=10`),
        fetch('/api/v1/execution/status')
      ]);

      if (ordersRes.ok) {
        const body = await ordersRes.json();
        if (body.data) setOrders(body.data);
      }
      if (tradesRes.ok) {
        const body = await tradesRes.json();
        if (body.data) setTrades(body.data);
      }
      if (signalsRes.ok) {
        const body = await signalsRes.json();
        if (body.data) {
          const formatted = body.data.map(s => ({
            generatedAt: s.GeneratedAt || s.generated_at,
            strategyName: s.StrategyName || s.strategy_name,
            side: s.Side || s.side,
            strength: s.Strength || s.strength,
            reason: s.Reason || s.reason
          }));
          setSignals(formatted);
        }
      }
      if (auditRes.ok) {
        const body = await auditRes.json();
        if (body.data) setAuditLogs(body.data);
      }
      if (runsRes.ok) {
        const body = await runsRes.json();
        if (body.data) setPipelineRuns(body.data);
      }
      if (streamsRes.ok) {
        const body = await streamsRes.json();
        if (body.data) setRedisStreams(body.data);
      }
      if (safetyRes.ok) {
        const body = await safetyRes.json();
        if (body.data) setIsSafetyBlocked(Boolean(body.data.kill_switch_active));
      }
      if (lifecycleRes.ok) {
        const body = await lifecycleRes.json();
        if (body.data) setStrategyLifecycle(body.data);
      }
      if (reconciliationRes.ok) {
        const body = await reconciliationRes.json();
        if (body.data) setReconciliationRuns(body.data);
      }
      if (riskSettingsRes.ok) {
        const body = await riskSettingsRes.json();
        if (body.data) setRiskSettings(body.data);
      }
      if (riskDecisionsRes.ok) {
        const body = await riskDecisionsRes.json();
        if (body.data) setRiskDecisions(body.data);
      }
      if (executionStatusRes.ok) {
        const body = await executionStatusRes.json();
        if (body.data) setExecutionStatus(body.data);
      }
    } catch {
      setBackendConnected(false);
      setOrders([]);
      setTrades([]);
      setSignals([]);
      setAuditLogs([]);
      setPipelineRuns([]);
      setRedisStreams([]);
      setStrategyLifecycle([]);
      setReconciliationRuns([]);
      setRiskSettings(null);
      setRiskDecisions([]);
      setExecutionStatus(null);
    }
  };

  // Fetch data on initialization and when symbol/settings change
  useEffect(() => {
    loadBackendData();
    // loadBackendData intentionally runs when activeSymbol changes; its setter dependencies are stable.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeSymbol]);

  useEffect(() => {
    const timeout = setTimeout(() => setIsChartPriming(false), 300);
    return () => clearTimeout(timeout);
  }, []);

  useEffect(() => {
    const interval = setInterval(() => {
      if (!lastTickAt) {
        setIsDataStale(true);
        return;
      }
      setIsDataStale(Date.now() - lastTickAt > 6000);
    }, 1000);
    return () => clearInterval(interval);
  }, [lastTickAt]);

  // Global Keyboard listener for Command Palette (CTRL+K) and Hotkeys
  useEffect(() => {
    const handleKeyDown = (e) => {
      const key = e.key.toUpperCase();
      
      // Ctrl+K or Cmd+K
      if ((e.ctrlKey || e.metaKey) && key === 'K') {
        e.preventDefault();
        setIsPaletteOpen((prev) => !prev);
        setPaletteQuery('');
        setSelectedPaletteIdx(0);
        return;
      }

      if (isPaletteOpen) {
        const filtered = getFilteredCommands();
        if (key === 'ESCAPE') {
          setIsPaletteOpen(false);
        } else if (key === 'ARROWDOWN') {
          e.preventDefault();
          setSelectedPaletteIdx((prev) => (prev + 1) % Math.max(filtered.length, 1));
        } else if (key === 'ARROWUP') {
          e.preventDefault();
          setSelectedPaletteIdx((prev) => (prev - 1 + filtered.length) % Math.max(filtered.length, 1));
        } else if (key === 'ENTER') {
          e.preventDefault();
          if (filtered[selectedPaletteIdx]) {
            filtered[selectedPaletteIdx].action();
            setIsPaletteOpen(false);
          }
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
    // Commands are rebuilt from current render state; listener is refreshed by the state values below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isPaletteOpen, selectedPaletteIdx, liveTickPrice, isSafetyBlocked]);

  // Focus input when palette opens
  useEffect(() => {
    if (isPaletteOpen && commandPaletteInputRef.current) {
      commandPaletteInputRef.current.focus();
    }
  }, [isPaletteOpen]);

  const getFilteredCommands = () => {
    return commands.filter(c =>
      c.text.toLowerCase().includes(paletteQuery.toLowerCase()) ||
      c.shortcut.toLowerCase().includes(paletteQuery.toLowerCase())
    );
  };

  const StatusIndicator = ({ label, status }) => {
    const statusText = status === 'online' ? 'online' : status === 'stale' ? 'stale' : 'offline';
    const colorClass = status === 'online'
      ? 'bg-signal-buy shadow-[0_0_4px_#44dfa3]'
      : status === 'stale'
        ? 'bg-signal-warn shadow-[0_0_4px_#ffdada]'
        : 'bg-signal-sell shadow-[0_0_4px_#f85149]';

    return (
      <div
        className="flex items-center gap-2"
        role="status"
        aria-label={`${label} ${statusText}`}
        title={`${label}: ${statusText}`}
      >
        <span>{label}</span>
        <span className={`w-2 h-2 rounded-full ${colorClass}`} aria-hidden="true" />
      </div>
    );
  };

  return (
    <div className="h-screen w-screen flex flex-col overflow-hidden bg-bg-60-1 text-chrome-text select-none">
      
      {/* Toast Alert overlay */}
      {toast && (
        <div
          className={`fixed bottom-4 right-4 z-50 px-3 py-2 border rounded shadow-lg flex items-center gap-2 text-[11px] animate-slide-in ${
          toast.type === 'success' ? 'bg-signal-buy/35 border-signal-buy text-signal-buy' :
          toast.type === 'warning' ? 'bg-signal-warn/35 border-signal-warn text-signal-warn' :
          toast.type === 'error' ? 'bg-signal-sell/35 border-signal-sell text-signal-sell' :
          'bg-signal-info/35 border-signal-info text-signal-info'
        }`}
          role="status"
          aria-live={toast.type === 'error' ? 'assertive' : 'polite'}
        >
          {toast.type === 'success' ? <CheckCircle size={13} aria-hidden="true" /> : toast.type === 'error' ? <AlertTriangle size={13} aria-hidden="true" /> : <Info size={13} aria-hidden="true" />}
          <span>{toast.message}</span>
        </div>
      )}

      {/* TOPBAR (48px fixed) */}
      <header className="h-[48px] min-h-[48px] border-b border-chrome-border px-4 flex justify-between items-center bg-bg-60-1">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2.5" aria-label="AlphaGate Console">
            <img src="/logo-mark.svg" alt="" aria-hidden="true" className="h-8 w-8 flex-shrink-0" />
            <div className="flex flex-col leading-none">
              <span className="font-extrabold text-[19px] text-white font-sans">AlphaGate</span>
              <span className="text-[10px] text-chrome-text/75 font-mono-data uppercase tracking-wider">Console</span>
            </div>
          </div>
          
          <div
            className={`flex items-center gap-3 bg-bg-60-4 rounded px-3 py-1 border border-chrome-border transition-colors h-[28px] ${
              isDataStale ? 'text-chrome-text/70' : ''
            }`}
            aria-label={`${activeSymbol} market context${isDataStale ? ', data feed stale' : ''}`}
            title={isDataStale ? 'Data feed interrupted. Check API connection or refresh cadence.' : `${activeSymbol} live market context`}
          >
            <span className="text-[11px] font-bold text-white font-mono-data">{activeSymbol}</span>
            <span className="text-[10px] text-chrome-text" aria-hidden="true">▼</span>
            <div className="flex items-center gap-2 border-l border-chrome-border pl-3 ml-1">
              <span className={`text-[11px] font-bold font-mono-data ${isDataStale ? 'text-chrome-text' : 'text-white'}`}>
                {liveTickPrice == null ? '-' : liveTickPrice.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
              </span>
              <span className={`text-[10px] font-bold ${liveTickDirection === 'up' ? 'text-signal-buy' : 'text-signal-sell'}`}>
                {liveTickChange || '-'}
              </span>
              {isDataStale && (
                <span className="text-[10px] text-signal-warn font-bold uppercase" title="Data feed interrupted. Check API connection or refresh cadence.">
                  Stale
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Workspace Switcher navigation */}
        <nav className="flex gap-8 items-center h-full">
          {['Trading', 'Research', 'Strategy', 'Risk', 'Ops'].map((tab) => (
            <button
              key={tab}
              onClick={() => {
                setActiveWorkspace(tab);
                showToast(`Workspace switched to ${tab}`, 'info');
              }}
              className={`h-full px-1 text-[11px] uppercase tracking-wider font-bold border-b-2 transition-all duration-100 flex items-center ${
                activeWorkspace === tab
                  ? 'border-b-signal-info text-signal-info'
                  : 'border-b-transparent text-chrome-text hover:text-white'
              }`}
            >
              {tab}
            </button>
          ))}
        </nav>

        {/* Status Indicators & Key palette tag */}
        <div className="flex items-center gap-4">
          <button
            onClick={() => setIsSettingsOpen(true)}
            className="text-chrome-text hover:text-white transition-colors cursor-pointer flex items-center justify-center p-1 hover:bg-bg-60-4 rounded interactive-control"
            title="Configure Settings"
            aria-label="Configure settings"
          >
            <Settings size={15} aria-hidden="true" />
          </button>

          <div className="flex items-center gap-2 border border-chrome-border px-2 py-1 rounded text-[10px] font-mono-data bg-bg-60-3">
            <span className="text-chrome-text">PALETTE:</span>
            <span className="text-signal-brand font-bold">CTRL+K</span>
          </div>

          <div className="flex items-center gap-3 text-[10px] font-bold text-chrome-text">
            <StatusIndicator label="API" status={backendConnected ? (isDataStale ? 'stale' : 'online') : 'offline'} />
            <StatusIndicator label="DB" status={backendConnected ? 'online' : 'offline'} />
            <StatusIndicator label="REDIS" status={backendConnected ? 'online' : 'offline'} />
          </div>
        </div>
      </header>

      {/* WORKSPACE AREA CONTAINER */}
      <main className="flex-1 min-h-0 bg-bg-60-2 relative">
        
        {/* Render fully active Trading workspace */}
        {activeWorkspace === 'Trading' && (
          <TradingWorkspace
            candles={candles}
            signals={signals}
            orders={orders}
            trades={trades}
            auditLogs={auditLogs}
            pipelineRuns={pipelineRuns}
            redisStreams={redisStreams}
            activeSymbol={activeSymbol}
            liveTickPrice={liveTickPrice}
            watchlist={watchlist}
            onSymbolSelect={(sym) => {
              setActiveSymbol(sym);
              showToast(`Active scope: ${sym}`, 'info');
            }}
            strategySettings={strategySettings}
            accountBalance="0.00"
            riskDecision={signals[0] ? { signal_side: signals[0].side } : null}
            strategyLifecycle={strategyLifecycle}
            reconciliationRuns={reconciliationRuns}
            riskSettings={riskSettings}
            riskDecisions={riskDecisions}
            executionStatus={executionStatus}
            onLifecycleAdvance={advanceStrategyLifecycle}
            onManualAction={executeManualAction}
            isSafetyBlocked={isSafetyBlocked}
            isChartLoading={isChartPriming || (backendConnected && candles.length === 0)}
            isDataStale={isDataStale}
          />
        )}        {/* 2. RESEARCH WORKSPACE */}
        {activeWorkspace === 'Research' && (
          <ResearchWorkspace activeSymbol={activeSymbol} showToast={showToast} />
        )}

        {/* 3. STRATEGY WORKSPACE */}
        {activeWorkspace === 'Strategy' && (
          <StrategyWorkspace showToast={showToast} />
        )}

        {/* 4. RISK WORKSPACE */}
        {activeWorkspace === 'Risk' && (
          <RiskWorkspace showToast={showToast} />
        )}

        {/* 5. OPS WORKSPACE */}
        {activeWorkspace === 'Ops' && (
          <OpsWorkspace showToast={showToast} />
        )}

      </main>

      {/* COMMAND PALETTE DIALOG OVERLAY */}
      {isPaletteOpen && (
        <div className="fixed inset-0 bg-bg-60-1/85 backdrop-blur-[2px] z-50 flex items-center justify-center p-4">
          <div className="bg-bg-60-2 border border-chrome-border rounded w-full max-w-[500px] shadow-[0_12px_40px_rgba(0,0,0,0.8)] overflow-hidden flex flex-col max-h-[350px]">
            <div className="flex items-center gap-3 px-4 py-3 border-b border-chrome-border bg-bg-60-1">
              <Search size={15} className="text-signal-brand" />
              <input
                ref={commandPaletteInputRef}
                type="text"
                value={paletteQuery}
                onChange={(e) => {
                  setPaletteQuery(e.target.value);
                  setSelectedPaletteIdx(0);
                }}
                className="bg-transparent border-none outline-none text-[13px] text-white flex-1 placeholder-chrome-text/80 font-sans"
                placeholder="Search commands (e.g. 'view ops')..."
                spellCheck="false"
                autoComplete="off"
                aria-label="Search commands"
              />
              <span className="text-[10px] border border-chrome-border/60 px-1 py-1 rounded font-mono text-chrome-text/60">ESC</span>
            </div>
            
            <div className="flex-1 overflow-y-auto p-2 space-y-0.5">
              {getFilteredCommands().length === 0 ? (
                <div className="p-4 text-center text-chrome-text/80 text-[11px]">No commands match search query</div>
              ) : (
                getFilteredCommands().map((cmd, idx) => {
                  const isSelected = idx === selectedPaletteIdx;
                  return (
                    <button
                      key={idx}
                      type="button"
                      onClick={() => {
                        cmd.action();
                        setIsPaletteOpen(false);
                      }}
                      className={`w-full flex justify-between items-center px-3 py-2 rounded cursor-pointer text-[11px] font-medium transition-all duration-100 interactive-control ${
                        isSelected ? 'bg-signal-brand/35 text-signal-brand' : 'text-chrome-text hover:bg-bg-60-4/60'
                      }`}
                      aria-current={isSelected ? 'true' : undefined}
                    >
                      <span>{cmd.text}</span>
                      <span className={`text-[10px] font-mono-data ${isSelected ? 'text-signal-brand/60' : 'text-chrome-text/60'}`}>
                        {cmd.shortcut}
                      </span>
                    </button>
                  );
                })
              )}
            </div>
          </div>
        </div>
      )}

      {/* Settings Configuration Modal */}
      <SettingsModal
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
        onSave={(updated) => {
          setTerminalSettings(updated);
          showToast('Preset configurations updated successfully.', 'success');
        }}
        currentSettings={terminalSettings}
      />

    </div>
  );
}
