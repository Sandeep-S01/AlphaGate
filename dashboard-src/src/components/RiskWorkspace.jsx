import React, { useState, useEffect } from 'react';
import { ShieldAlert, CheckCircle, Save } from 'lucide-react';

export default function RiskWorkspace({ showToast = () => {} }) {
  const [enabled, setEnabled] = useState(true);
  const [minSignalStrength, setMinSignalStrength] = useState(0.20);
  const [maxSignalStrength, setMaxSignalStrength] = useState(0.95);
  const [maxQuoteAmount, setMaxQuoteAmount] = useState(500.0);
  const [maxOrderQuoteAmount, setMaxOrderQuoteAmount] = useState(0);
  const [maxPositionQuoteAmount, setMaxPositionQuoteAmount] = useState(0);
  const [maxTotalExposureQuoteAmount, setMaxTotalExposureQuoteAmount] = useState(0);
  const [maxOpenPositions, setMaxOpenPositions] = useState(0);
  const [maxDailyLoss, setMaxDailyLoss] = useState(150.0);
  const [maxDailyTrades, setMaxDailyTrades] = useState(10);
  const [allowBuy, setAllowBuy] = useState(true);
  const [allowSell, setAllowSell] = useState(true);
  const [allowedSymbols, setAllowedSymbols] = useState('');
  const [cooldownSeconds, setCooldownSeconds] = useState(60);

  const [isSaving, setIsSaving] = useState(false);
  const [decisions, setDecisions] = useState([]);

  useEffect(() => {
    fetchSettings();
    fetchDecisions();
    const interval = setInterval(fetchDecisions, 5000);
    return () => clearInterval(interval);
  }, []);

  const fetchSettings = async () => {
    try {
      const res = await fetch('/api/v1/risk/settings');
      if (res.ok) {
        const data = await res.json();
        const settings = data.data || {};
        setEnabled(settings.enabled ?? true);
        setMinSignalStrength(settings.min_signal_strength ?? 0.20);
        setMaxSignalStrength(settings.max_signal_strength ?? 0.95);
        setMaxQuoteAmount(settings.max_quote_amount ?? 500.0);
        setMaxOrderQuoteAmount(settings.max_order_quote_amount ?? 0);
        setMaxPositionQuoteAmount(settings.max_position_quote_amount ?? 0);
        setMaxTotalExposureQuoteAmount(settings.max_total_exposure_quote_amount ?? 0);
        setMaxOpenPositions(settings.max_open_positions ?? 0);
        setMaxDailyLoss(settings.max_daily_loss ?? 150.0);
        setMaxDailyTrades(settings.max_daily_trades ?? 10);
        setAllowBuy(settings.allow_buy ?? true);
        setAllowSell(settings.allow_sell ?? true);
        setAllowedSymbols((settings.allowed_symbols || []).join(', '));
        setCooldownSeconds(settings.cooldown_seconds ?? 60);
      }
    } catch (err) {
      console.error('Error loading risk settings', err);
    }
  };

  const fetchDecisions = async () => {
    try {
      const res = await fetch('/api/v1/risk-decisions');
      if (res.ok) {
        const data = await res.json();
        setDecisions(data.data || []);
      }
    } catch (err) {
      console.error('Error loading decisions', err);
    }
  };

  const handleSaveSettings = async () => {
    setIsSaving(true);
    const payload = {
      enabled,
      min_signal_strength: Number(minSignalStrength),
      max_signal_strength: Number(maxSignalStrength),
      max_quote_amount: Number(maxQuoteAmount),
      max_order_quote_amount: Number(maxOrderQuoteAmount),
      max_position_quote_amount: Number(maxPositionQuoteAmount),
      max_total_exposure_quote_amount: Number(maxTotalExposureQuoteAmount),
      max_open_positions: Number(maxOpenPositions),
      max_daily_loss: Number(maxDailyLoss),
      max_daily_trades: Number(maxDailyTrades),
      allow_buy: allowBuy,
      allow_sell: allowSell,
      allowed_symbols: allowedSymbols.split(',').map((item) => item.trim().toUpperCase()).filter(Boolean),
      cooldown_seconds: Number(cooldownSeconds)
    };

    try {
      const res = await fetch('/api/v1/risk/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (res.ok) {
        showToast('System Risk boundary parameters updated successfully.', 'success');
      } else {
        const body = await res.json();
        throw new Error(body.error || 'Failed to update settings');
      }
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="flex-1 flex h-full overflow-hidden select-none w-full bg-bg-60-1">
      {/* Parameters Panel */}
      <aside className="w-80 border-r border-chrome-border flex flex-col bg-bg-60-3 overflow-y-auto no-scrollbar p-3.5 gap-4">
        <div>
          <div className="text-[10px] uppercase tracking-wider text-chrome-text font-bold border-b border-chrome-border pb-1 mb-3">
            Risk Boundaries Editor
          </div>
          <div className="space-y-3.5 text-[11px]">
            <div className="flex items-center justify-between bg-bg-60-4 p-2.5 rounded border border-chrome-border">
              <span className="text-white font-bold font-sans uppercase">ACTIVATE RISK ENGINE</span>
              <input
                type="checkbox"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
                className="w-4 h-4 cursor-pointer accent-signal-info"
              />
            </div>

            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="block text-chrome-text/80 mb-1 font-sans">MIN SIGNAL COEF.</label>
                <input
                  type="number"
                  step="0.05"
                  value={minSignalStrength}
                  onChange={(e) => setMinSignalStrength(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
              <div>
                <label className="block text-chrome-text/80 mb-1 font-sans">MAX SIGNAL COEF.</label>
                <input
                  type="number"
                  step="0.05"
                  value={maxSignalStrength}
                  onChange={(e) => setMaxSignalStrength(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
            </div>

            <div className="border-t border-chrome-border/70 pt-3">
              <label htmlFor="risk-max-quote" className="block text-chrome-text/80 mb-1 font-sans">MAX QUOTE VALUE ($)</label>
              <input
                id="risk-max-quote"
                type="number"
                value={maxQuoteAmount}
                onChange={(e) => setMaxQuoteAmount(e.target.value)}
                className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
              />
            </div>

            <div className="border-t border-chrome-border/70 pt-3">
              <label htmlFor="risk-allowed-symbols" className="block text-chrome-text/80 mb-1 font-sans">ALLOWED SYMBOLS</label>
              <input
                id="risk-allowed-symbols"
                aria-label="Allowed symbols"
                type="text"
                value={allowedSymbols}
                onChange={(e) => setAllowedSymbols(e.target.value)}
                placeholder="BTCUSDT, ETHUSDT"
                className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px] uppercase"
              />
            </div>

            <div className="grid grid-cols-2 gap-2 border-t border-chrome-border/70 pt-3">
              <div>
                <label htmlFor="risk-max-order" className="block text-chrome-text/80 mb-1 font-sans">MAX ORDER ($)</label>
                <input
                  id="risk-max-order"
                  aria-label="Max order quote amount"
                  type="number"
                  value={maxOrderQuoteAmount}
                  onChange={(e) => setMaxOrderQuoteAmount(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
              <div>
                <label htmlFor="risk-max-position" className="block text-chrome-text/80 mb-1 font-sans">MAX POSITION ($)</label>
                <input
                  id="risk-max-position"
                  aria-label="Max position quote amount"
                  type="number"
                  value={maxPositionQuoteAmount}
                  onChange={(e) => setMaxPositionQuoteAmount(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-2">
              <div>
                <label htmlFor="risk-max-exposure" className="block text-chrome-text/80 mb-1 font-sans">MAX EXPOSURE ($)</label>
                <input
                  id="risk-max-exposure"
                  aria-label="Max total exposure quote amount"
                  type="number"
                  value={maxTotalExposureQuoteAmount}
                  onChange={(e) => setMaxTotalExposureQuoteAmount(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
              <div>
                <label htmlFor="risk-max-open-positions" className="block text-chrome-text/80 mb-1 font-sans">OPEN POSITIONS</label>
                <input
                  id="risk-max-open-positions"
                  aria-label="Max open positions"
                  type="number"
                  value={maxOpenPositions}
                  onChange={(e) => setMaxOpenPositions(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-2 border-t border-chrome-border/70 pt-3">
              <div>
                <label className="block text-chrome-text/80 mb-1 font-sans">DAILY LOSS CAP ($)</label>
                <input
                  type="number"
                  value={maxDailyLoss}
                  onChange={(e) => setMaxDailyLoss(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
              <div>
                <label className="block text-chrome-text/80 mb-1 font-sans">DAILY TRADES CAP</label>
                <input
                  type="number"
                  value={maxDailyTrades}
                  onChange={(e) => setMaxDailyTrades(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
            </div>

            <div className="border-t border-chrome-border/70 pt-3 grid grid-cols-2 gap-3 bg-bg-60-2/60 p-2.5 rounded border border-chrome-border">
              <div className="flex justify-between items-center">
                <span className="text-chrome-text font-bold font-sans">ALLOW BUY</span>
                <input
                  type="checkbox"
                  checked={allowBuy}
                  onChange={(e) => setAllowBuy(e.target.checked)}
                  className="w-3.5 h-3.5 cursor-pointer accent-signal-buy"
                />
              </div>
              <div className="flex justify-between items-center">
                <span className="text-chrome-text font-bold font-sans">ALLOW SELL</span>
                <input
                  type="checkbox"
                  checked={allowSell}
                  onChange={(e) => setAllowSell(e.target.checked)}
                  className="w-3.5 h-3.5 cursor-pointer accent-signal-sell"
                />
              </div>
            </div>

            <div className="border-t border-chrome-border/70 pt-3">
              <label className="block text-chrome-text/80 mb-1 font-sans">COOLDOWN GUARD PERIOD (SEC)</label>
              <input
                type="number"
                value={cooldownSeconds}
                onChange={(e) => setCooldownSeconds(e.target.value)}
                className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
              />
            </div>

            <button
              onClick={handleSaveSettings}
              disabled={isSaving}
              className="w-full h-[32px] bg-signal-info hover:bg-signal-info/90 text-bg-60-1 font-bold rounded cursor-pointer btn-active-scale transition-all duration-100 flex justify-center items-center gap-2 disabled:opacity-40 disabled:cursor-not-allowed text-[11px] uppercase tracking-wider mt-2"
            >
              {isSaving ? (
                <>
                  <div className="w-3.5 h-3.5 border-2 border-bg-60-1 border-t-transparent rounded-full animate-spin"></div>
                  <span>Updating boundaries...</span>
                </>
              ) : (
                <>
                  <Save size={12} />
                  <span>Save Parameters</span>
                </>
              )}
            </button>
          </div>
        </div>
      </aside>

      {/* Main Diagnostics Pane */}
      <main className="flex-1 flex flex-col min-w-0 bg-bg-60-2 overflow-hidden p-3.5 gap-4">
        {/* Risk Status Indicator */}
        <div className="border border-chrome-border bg-bg-60-3/60 rounded-lg p-3 flex flex-col gap-1 relative overflow-hidden">
          <div className={`absolute top-0 left-0 w-1.5 h-full ${enabled ? 'bg-signal-buy' : 'bg-signal-warn'}`}></div>
          <div className="text-[10px] uppercase tracking-wider text-chrome-text font-bold pl-2">
            System Risk Threshold Boundaries
          </div>
          <div className="text-[11px] text-white pl-2 mt-1 font-bold flex items-center gap-2 font-sans">
            {enabled ? (
              <>
                <CheckCircle size={12} className="text-signal-buy" />
                <span className="text-signal-buy">ACTIVE ENFORCEMENT STATE</span>
              </>
            ) : (
              <>
                <ShieldAlert size={12} className="text-signal-warn" />
                <span className="text-signal-warn">RISK BLOCK BYPASSED - DANGER MODE</span>
              </>
            )}
          </div>
        </div>

        {/* Diagnostic boundary cards */}
        <div className="grid grid-cols-4 gap-3 text-[10px]">
          <div className="border border-chrome-border border-l-4 border-l-signal-sell bg-bg-60-1 p-2.5 rounded-lg flex flex-col gap-1">
            <span className="text-[10px] text-chrome-text uppercase font-sans">Daily Loss Limit</span>
            <span className="font-mono-data text-[12px] text-signal-sell font-bold">${maxDailyLoss}</span>
            <span className="text-[10px] text-chrome-text/60 font-sans">Drawdown Boundary Check</span>
          </div>
          <div className="border border-chrome-border border-l-4 border-l-signal-buy bg-bg-60-1 p-2.5 rounded-lg flex flex-col gap-1">
            <span className="text-[10px] text-chrome-text uppercase font-sans">Max Daily Trades</span>
            <span className="font-mono-data text-[12px] text-signal-buy font-bold">{maxDailyTrades} trades</span>
            <span className="text-[10px] text-chrome-text/60 font-sans">Frequency Throttle</span>
          </div>
          <div className="border border-chrome-border border-l-4 border-l-chrome-text bg-bg-60-1 p-2.5 rounded-lg flex flex-col gap-1">
            <span className="text-[10px] text-chrome-text uppercase font-sans">Max Trade Size</span>
            <span className="font-mono-data text-[12px] text-white font-bold">${maxQuoteAmount}</span>
            <span className="text-[10px] text-chrome-text/60 font-sans">Leverage Constraint</span>
          </div>
          <div className="border border-chrome-border border-l-4 border-l-signal-info bg-bg-60-1 p-2.5 rounded-lg flex flex-col gap-1">
            <span className="text-[10px] text-chrome-text uppercase font-sans">Signal Range</span>
            <span className="font-mono-data text-[12px] text-signal-info font-bold">{minSignalStrength} - {maxSignalStrength}</span>
            <span className="text-[10px] text-chrome-text/60 font-sans">Probability Limits</span>
          </div>
        </div>
        <div className="grid grid-cols-4 gap-3 text-[10px]">
          <div className="border border-chrome-border bg-bg-60-1 p-2.5 rounded-lg flex flex-col gap-1">
            <span className="text-[10px] text-chrome-text uppercase font-sans">Allowed Symbols</span>
            <span className="font-mono-data text-[12px] text-white font-bold truncate">{allowedSymbols || 'ALL'}</span>
            <span className="text-[10px] text-chrome-text/60 font-sans">Symbol Whitelist</span>
          </div>
          <div className="border border-chrome-border bg-bg-60-1 p-2.5 rounded-lg flex flex-col gap-1">
            <span className="text-[10px] text-chrome-text uppercase font-sans">Max Order</span>
            <span className="font-mono-data text-[12px] text-white font-bold">${maxOrderQuoteAmount}</span>
            <span className="text-[10px] text-chrome-text/60 font-sans">Per-Order Notional</span>
          </div>
          <div className="border border-chrome-border bg-bg-60-1 p-2.5 rounded-lg flex flex-col gap-1">
            <span className="text-[10px] text-chrome-text uppercase font-sans">Max Exposure</span>
            <span className="font-mono-data text-[12px] text-white font-bold">${maxTotalExposureQuoteAmount}</span>
            <span className="text-[10px] text-chrome-text/60 font-sans">Portfolio Notional</span>
          </div>
          <div className="border border-chrome-border bg-bg-60-1 p-2.5 rounded-lg flex flex-col gap-1">
            <span className="text-[10px] text-chrome-text uppercase font-sans">Open Positions</span>
            <span className="font-mono-data text-[12px] text-white font-bold">{maxOpenPositions || 'UNLIMITED'}</span>
            <span className="text-[10px] text-chrome-text/60 font-sans">Concurrent Limit</span>
          </div>
        </div>

        {/* Decision Trails Log */}
        <div className="flex-1 min-h-0 border border-chrome-border bg-bg-60-1 rounded-lg flex flex-col">
          <div className="p-2 border-b border-chrome-border text-[10px] uppercase tracking-wider text-chrome-text font-bold">
            Enforced Risk Evaluation Decision Trail
          </div>
          <div className="flex-1 overflow-y-auto p-2 font-mono-data text-[10px]">
            <table className="w-full border-collapse">
              <thead>
                <tr className="text-left text-[10px] text-chrome-text border-b border-chrome-border/80">
                  <th className="p-1 font-bold">Time</th>
                  <th className="p-1 font-bold">Asset</th>
                  <th className="p-1 font-bold">Signal Side</th>
                  <th className="p-1 font-bold">Evaluation Status</th>
                  <th className="p-1 font-bold">Reasoning Parameters</th>
                </tr>
              </thead>
              <tbody>
                {decisions.length === 0 ? (
                  <tr>
                    <td colSpan="5" className="p-4 text-center text-chrome-text/60 italic">
                      No decision logs active. Trading simulator ticks are normal.
                    </td>
                  </tr>
                ) : (
                  decisions.map((d, idx) => {
                    const isOk = d.Decision?.toLowerCase() === 'approved' || d.decision?.toLowerCase() === 'approved';
                    const sideClass = (d.SignalSide || d.signal_side)?.toLowerCase() === 'buy' ? 'text-signal-buy' : 'text-signal-sell';
                    
                    return (
                      <tr key={idx} className="border-b border-chrome-border/35 hover:bg-bg-60-4/60">
                        <td className="p-1">{new Date(d.EvaluatedAt || d.evaluated_at).toLocaleString()}</td>
                        <td className="p-1 text-white">{d.Symbol || d.symbol}</td>
                        <td className={`p-1 font-bold ${sideClass}`}>{(d.SignalSide || d.signal_side)?.toUpperCase()}</td>
                        <td className="p-1">
                          <span className={`px-1 py-1 border rounded text-[10px] font-sans font-bold ${
                            isOk ? 'text-signal-buy border-signal-buy/35 bg-signal-buy/15' : 'text-signal-sell border-signal-sell/35 bg-signal-sell/15'
                          }`}>
                            {(d.Decision || d.decision)?.toUpperCase()}
                          </span>
                        </td>
                        <td className="p-1 text-chrome-text">{d.Reason || d.reason}</td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>
      </main>
    </div>
  );
}
