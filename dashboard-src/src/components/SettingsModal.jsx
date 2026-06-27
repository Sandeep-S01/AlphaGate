import React, { useEffect, useRef, useState } from 'react';
import { X, Shield, Settings, Server } from 'lucide-react';

export default function SettingsModal({ isOpen, onClose, onSave, currentSettings }) {
  const dialogRef = useRef(null);
  const [apiKey, setApiKey] = useState(currentSettings?.apiKey ?? 'AQ.Ab8RN6LovyaG6ou... [Active]');
  const [tradeSize, setTradeSize] = useState(currentSettings?.tradeSize ?? 0.15);
  const [slippage, setSlippage] = useState(currentSettings?.slippage ?? 0.05);
  const [refreshInterval, setRefreshInterval] = useState(currentSettings?.refreshInterval ?? 1500);
  const [backtestSaveEnabled, setBacktestSaveEnabled] = useState(currentSettings?.backtestSaveEnabled ?? true);

  useEffect(() => {
    if (!isOpen || !dialogRef.current) return;
    const focusable = dialogRef.current.querySelectorAll('button, input, select, textarea, [tabindex]:not([tabindex="-1"])');
    focusable[0]?.focus();

    const onKeyDown = (event) => {
      if (event.key === 'Escape') {
        onClose();
        return;
      }
      if (event.key !== 'Tab' || focusable.length === 0) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const handleSave = () => {
    onSave({
      apiKey,
      tradeSize: Number(tradeSize),
      slippage: Number(slippage),
      refreshInterval: Number(refreshInterval),
      backtestSaveEnabled
    });
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-bg-60-2/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div
        ref={dialogRef}
        className="bg-bg-60-3 border border-chrome-border rounded-lg w-full max-w-[450px] shadow-[0_12px_40px_rgba(0,0,0,0.8)] overflow-hidden flex flex-col"
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-title"
      >
        {/* Header */}
        <div className="flex justify-between items-center px-4 py-3 border-b border-chrome-border bg-bg-60-2">
          <div className="flex items-center gap-2 text-white font-bold text-[12px] uppercase tracking-wider font-sans">
            <Settings size={14} className="text-signal-info" aria-hidden="true" />
            <span id="settings-title">Terminal Configurations</span>
          </div>
          <button 
            onClick={onClose}
            className="text-chrome-text hover:text-white transition-colors cursor-pointer interactive-control"
            aria-label="Close settings"
          >
            <X size={16} aria-hidden="true" />
          </button>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4 text-[11px]">
          {/* Section: API Access */}
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-white font-bold uppercase tracking-wide font-sans">
              <Server size={12} className="text-signal-brand" aria-hidden="true" />
              <span>STITCH GATEWAY AUTHORIZATION</span>
            </div>
            <div className="bg-bg-60-2 border border-chrome-border rounded p-2.5">
              <label htmlFor="settings-api-key" className="block text-chrome-text/80 mb-1 font-sans">X-GOOG-API-KEY</label>
              <input
                id="settings-api-key"
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                className="w-full bg-bg-60-1 border border-chrome-border p-1.5 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
              />
            </div>
          </div>

          {/* Section: Default Parameters */}
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-white font-bold uppercase tracking-wide font-sans">
              <Shield size={12} className="text-signal-info" aria-hidden="true" />
              <span>SIMULATOR DEFAULT PRESETS</span>
            </div>
            <div className="bg-bg-60-2 border border-chrome-border rounded p-2.5 grid grid-cols-2 gap-3">
              <div>
                <label htmlFor="settings-trade-size" className="block text-chrome-text/80 mb-1 font-sans">DEFAULT TRADE SIZE (BTC)</label>
                <input
                  id="settings-trade-size"
                  type="number"
                  step="0.01"
                  value={tradeSize}
                  onChange={(e) => setTradeSize(e.target.value)}
                  className="w-full bg-bg-60-1 border border-chrome-border p-1.5 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
              <div>
                <label htmlFor="settings-slippage" className="block text-chrome-text/80 mb-1 font-sans">SLIPPAGE RATE (%)</label>
                <input
                  id="settings-slippage"
                  type="number"
                  step="0.01"
                  value={slippage}
                  onChange={(e) => setSlippage(e.target.value)}
                  className="w-full bg-bg-60-1 border border-chrome-border p-1.5 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
              <div className="col-span-2">
                <label htmlFor="settings-refresh-interval" className="block text-chrome-text/80 mb-1 font-sans">UI TICK REFRESH SPEED (MS)</label>
                <select
                  id="settings-refresh-interval"
                  value={refreshInterval}
                  onChange={(e) => setRefreshInterval(e.target.value)}
                  className="w-full bg-bg-60-1 border border-chrome-border p-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px] cursor-pointer"
                >
                  <option value="500">500 ms (Fastest)</option>
                  <option value="1000">1000 ms (Recommended)</option>
                  <option value="1500">1500 ms (Normal)</option>
                  <option value="3000">3000 ms (Eco Mode)</option>
                </select>
              </div>
            </div>
          </div>

          {/* Section: Logging & Saving Options */}
          <div className="flex items-center justify-between bg-bg-60-2 border border-chrome-border p-2.5 rounded">
            <span className="text-white font-bold font-sans uppercase">Persist Simulation runs</span>
            <input
              aria-label="Persist simulation runs"
              type="checkbox"
              checked={backtestSaveEnabled}
              onChange={(e) => setBacktestSaveEnabled(e.target.checked)}
              className="w-4 h-4 cursor-pointer accent-signal-info"
            />
          </div>
        </div>

        {/* Footer actions */}
        <div className="flex justify-end gap-2 px-4 py-3 border-t border-chrome-border bg-bg-60-2">
          <button
            onClick={onClose}
            className="px-3 py-2 rounded border border-chrome-border hover:bg-bg-60-4 text-chrome-text hover:text-white font-bold cursor-pointer transition-colors interactive-control"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            className="px-4 py-2 rounded bg-signal-info hover:bg-signal-info/90 text-bg-60-1 font-bold cursor-pointer transition-colors interactive-control"
          >
            Save Preset Settings
          </button>
        </div>
      </div>
    </div>
  );
}
