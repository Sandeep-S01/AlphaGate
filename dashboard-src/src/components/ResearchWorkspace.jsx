import React, { useState, useEffect, useRef } from 'react';
import { Play, TrendingUp, AlertTriangle, SlidersHorizontal } from 'lucide-react';

const templateStatusLabels = {
  executable_native: 'Native',
  executable_pine: 'Pine Backtest',
  template_only: 'Template Only',
  blocked_by_data: 'Data Blocked'
};

const formatCurrency = (value) => {
  if (value === null || value === undefined || Number.isNaN(Number(value))) return '--';
  return `$${Number(value).toFixed(2)}`;
};

const formatNumber = (value, digits = 2) => {
  if (value === null || value === undefined || Number.isNaN(Number(value))) return '--';
  return Number(value).toFixed(digits);
};

const validationCopy = {
  candidate: {
    label: 'Candidate',
    tone: 'success',
    message: 'Strategy meets hardened validation rules.'
  },
  low_bull_market_capture: {
    label: 'Low Bull Capture',
    tone: 'warning',
    message: 'Benchmark was strongly positive, but this strategy captured too little of that move.'
  },
  underperforms_benchmark: {
    label: 'Benchmark Underperformance',
    tone: 'warning',
    message: 'Strategy return is below the buy-and-hold benchmark for this window.'
  },
  weak_profit_factor: {
    label: 'Weak Profit Factor',
    tone: 'warning',
    message: 'Profit factor is below the hardened candidate threshold.'
  },
  insufficient_sample: {
    label: 'Insufficient Sample',
    tone: 'warning',
    message: 'Completed round trips are below the minimum sample threshold.'
  },
  cost_drag: {
    label: 'Cost Drag',
    tone: 'warning',
    message: 'Average trade is below estimated round-trip cost. Reduce churn, use a higher timeframe, or improve entry quality.'
  },
  overtrading: {
    label: 'Overtrading',
    tone: 'warning',
    message: 'Trade frequency is too high for the selected interval.'
  },
  high_drawdown: {
    label: 'High Drawdown',
    tone: 'error',
    message: 'Maximum drawdown exceeds the hardened candidate threshold.'
  },
  unsafe_execution_timing: {
    label: 'Unsafe Timing',
    tone: 'error',
    message: 'Candidate backtests must use next-open execution timing.'
  }
};

const validationToneClasses = {
  success: 'border-signal-buy/40 bg-signal-buy/10 text-signal-buy',
  warning: 'border-signal-warn/40 bg-signal-warn/10 text-signal-warn',
  error: 'border-signal-sell/40 bg-signal-sell/10 text-signal-sell'
};

const getValidationDisplay = (run) => {
  if (!run?.validation_status) return null;
  const fallbackLabel = run.validation_status.replaceAll('_', ' ').toUpperCase();
  const copy = validationCopy[run.validation_status] || {
    label: fallbackLabel,
    tone: 'warning',
    message: run.validation_reason || 'Backtest did not meet hardened validation rules.'
  };
  return {
    ...copy,
    message: run.validation_reason || copy.message
  };
};

const benchmarkCapturePercent = (run) => {
  const benchmark = Number(run?.benchmark_return_percent);
  const strategyReturn = Number(run?.return_percent);
  if (!Number.isFinite(benchmark) || !Number.isFinite(strategyReturn) || benchmark <= 0) return null;
  return (strategyReturn / benchmark) * 100;
};

const uniquePositiveInts = (values) => Array.from(new Set(
  values
    .map((value) => Number(value))
    .filter((value) => Number.isFinite(value) && value > 0)
    .map((value) => Math.round(value))
)).sort((a, b) => a - b);

const defaultBacktestWindow = () => {
  const toDate = new Date();
  const fromDate = new Date();
  fromDate.setDate(toDate.getDate() - 30);
  return { fromDate, toDate };
};

export default function ResearchWorkspace({ activeSymbol = 'BTCUSDT', showToast = () => {} }) {
  const [strategy, setStrategy] = useState('sma-crossover');
  const [strategyTemplates, setStrategyTemplates] = useState([]);
  const [symbol, setSymbol] = useState(activeSymbol);
  const [interval, setInterval] = useState('1m');
  const [startingBalance, setStartingBalance] = useState(1000);
  const [fastPeriod, setFastPeriod] = useState(9);
  const [slowPeriod, setSlowPeriod] = useState(21);
  const [rsiPeriod, setRsiPeriod] = useState(14);
  const [rsiOversold, setRsiOversold] = useState(30);
  const [rsiOverbought, setRsiOverbought] = useState(70);
  const [cooldownBars, setCooldownBars] = useState(1);
  const [minHoldingBars, setMinHoldingBars] = useState(1);
  const [feeRate, setFeeRate] = useState(0.001);
  const [positionSizingMode, setPositionSizingMode] = useState('percent_equity');
  const [positionSizeValue, setPositionSizeValue] = useState(10);
  const [atrExitEnabled, setAtrExitEnabled] = useState(true);
  const [atrPeriod, setAtrPeriod] = useState(14);
  const [atrStopMultiplier, setAtrStopMultiplier] = useState(2.0);
  const [atrTakeProfitMultiplier, setAtrTakeProfitMultiplier] = useState(3.0);
  const [shortingEnabled, setShortingEnabled] = useState(false);
  const [regimeFilterEnabled, setRegimeFilterEnabled] = useState(false);
  const [regimeFilterPeriod, setRegimeFilterPeriod] = useState(14);
  const [regimeMinATRPercent, setRegimeMinATRPercent] = useState(0);
  const [regimeMaxATRPercent, setRegimeMaxATRPercent] = useState(0);

  const [isLoading, setIsLoading] = useState(false);
  const [isOptimizing, setIsOptimizing] = useState(false);
  const [error, setError] = useState(null);
  const [optimizationError, setOptimizationError] = useState(null);
  const [backtestRun, setBacktestRun] = useState(null);
  const [backtestTrades, setBacktestTrades] = useState([]);
  const [optimizationResult, setOptimizationResult] = useState(null);

  const canvasRef = useRef(null);

  useEffect(() => {
    setSymbol(activeSymbol);
  }, [activeSymbol]);

  useEffect(() => {
    const fetchTemplates = async () => {
      try {
        const res = await fetch('/api/v1/strategies/templates');
        if (res.ok) {
          const body = await res.json();
          setStrategyTemplates(body.data || []);
        }
      } catch (err) {
        console.error('Error fetching strategy templates', err);
      }
    };
    fetchTemplates();
  }, []);

  // Handle Equity Curve drawing on Canvas
  useEffect(() => {
    if (!canvasRef.current || !backtestRun || !backtestRun.equity_curve || backtestRun.equity_curve.length === 0) return;
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Dimensions
    const dpr = window.devicePixelRatio || 1;
    const width = canvas.clientWidth;
    const height = canvas.clientHeight;
    canvas.width = width * dpr;
    canvas.height = height * dpr;
    ctx.scale(dpr, dpr);

    // Clear
    ctx.fillStyle = '#080a12';
    ctx.fillRect(0, 0, width, height);

    const curve = backtestRun.equity_curve;
    const pointsCount = curve.length;
    const rightMargin = 60;
    const bottomMargin = 20;
    const chartWidth = width - rightMargin;
    const chartHeight = height - bottomMargin;

    // Boundaries
    const equities = curve.map(p => p.equity);
    const minEq = Math.min(...equities, startingBalance) * 0.98;
    const maxEq = Math.max(...equities, startingBalance) * 1.02;
    const eqRange = maxEq - minEq || 1;

    // Grid lines
    ctx.strokeStyle = '#1d2533';
    ctx.lineWidth = 0.5;
    ctx.fillStyle = '#d8e1f1';
    ctx.font = '8.5px JetBrains Mono';
    ctx.textAlign = 'left';

    const gridSteps = 4;
    for (let i = 0; i <= gridSteps; i++) {
      const val = minEq + eqRange * (i / gridSteps);
      const y = chartHeight - ((val - minEq) / eqRange) * chartHeight;
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(chartWidth, y);
      ctx.stroke();
      ctx.fillText(`$${val.toFixed(0)}`, chartWidth + 5, y + 3);
    }

    // Draw Equity Line
    ctx.strokeStyle = backtestRun.profit_loss >= 0 ? '#20f2a3' : '#ff5c7a';
    ctx.lineWidth = 1.8;
    ctx.beginPath();

    curve.forEach((p, idx) => {
      const x = (idx / (pointsCount - 1)) * chartWidth;
      const y = chartHeight - ((p.equity - minEq) / eqRange) * chartHeight;
      if (idx === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.stroke();

    // Fill gradient below line
    const gradient = ctx.createLinearGradient(0, 0, 0, chartHeight);
    gradient.addColorStop(0, backtestRun.profit_loss >= 0 ? 'rgba(32, 242, 163, 0.18)' : 'rgba(255, 92, 122, 0.18)');
    gradient.addColorStop(1, 'rgba(8, 10, 18, 0)');
    ctx.fillStyle = gradient;
    ctx.beginPath();
    ctx.moveTo(0, chartHeight);
    curve.forEach((p, idx) => {
      const x = (idx / (pointsCount - 1)) * chartWidth;
      const y = chartHeight - ((p.equity - minEq) / eqRange) * chartHeight;
      ctx.lineTo(x, y);
    });
    ctx.lineTo(chartWidth, chartHeight);
    ctx.closePath();
    ctx.fill();

  }, [backtestRun, startingBalance]);

  const selectedTemplate = strategyTemplates.find((item) => item.id === strategy);
  const selectedTemplateBacktestable = !selectedTemplate || ['executable_pine', 'executable_native'].includes(selectedTemplate.support_status);
  const selectedTemplateStatus = selectedTemplate ? templateStatusLabels[selectedTemplate.support_status] || selectedTemplate.support_status : 'Native';
  const validationDisplay = getValidationDisplay(backtestRun);
  const capturePercent = benchmarkCapturePercent(backtestRun);
  const optimizationRows = optimizationResult?.results?.slice(0, 8) || [];

  const handleStrategyChange = (value) => {
    setStrategy(value);
    const tmpl = strategyTemplates.find((item) => item.id === value);
    if (!tmpl) return;

    const defaults = tmpl.default_settings || {};
    if (defaults.fast_period) setFastPeriod(defaults.fast_period);
    if (defaults.slow_period) setSlowPeriod(defaults.slow_period);
    if (defaults.symbol) setSymbol(defaults.symbol);
    if (defaults.interval) setInterval(defaults.interval);

    const profile = tmpl.execution_profile;
    if (profile) {
      if (profile.recommended_interval) setInterval(profile.recommended_interval);
      if (profile.cooldown_bars) setCooldownBars(profile.cooldown_bars);
      if (profile.min_holding_bars) setMinHoldingBars(profile.min_holding_bars);
      setAtrExitEnabled(Boolean(profile.atr_exit_enabled));
      if (profile.atr_period) setAtrPeriod(profile.atr_period);
      if (profile.atr_stop_multiplier) setAtrStopMultiplier(profile.atr_stop_multiplier);
      if (profile.atr_take_profit_multiplier) setAtrTakeProfitMultiplier(profile.atr_take_profit_multiplier);
      setShortingEnabled(Boolean(profile.shorting_enabled));
      setRegimeFilterEnabled(Boolean(profile.regime_filter_enabled));
      if (profile.regime_filter_period) setRegimeFilterPeriod(profile.regime_filter_period);
      setRegimeMinATRPercent(Number(profile.regime_min_atr_percent || 0));
      setRegimeMaxATRPercent(Number(profile.regime_max_atr_percent || 0));
      if (profile.position_size_percent) {
        setPositionSizingMode('percent_equity');
        setPositionSizeValue(profile.position_size_percent);
      }
    } else {
      setShortingEnabled(false);
      setRegimeFilterEnabled(false);
      setRegimeMinATRPercent(0);
      setRegimeMaxATRPercent(0);
    }

    if (!['executable_pine', 'executable_native'].includes(tmpl.support_status)) {
      const tmplStatus = templateStatusLabels[tmpl.support_status] || tmpl.support_status;
      showToast(`${tmpl.name} is ${tmplStatus}; backtesting is blocked until required data is implemented.`, 'warning');
    }
  };

  const handleExecute = async () => {
    if (!selectedTemplateBacktestable) {
      const blockers = selectedTemplate?.blockers?.join(' ') || 'Required data is not available in the current backtest engine.';
      setError(`${selectedTemplate?.name || strategy} cannot be backtested yet. ${blockers}`);
      showToast('Selected strategy model is not backtestable yet.', 'warning');
      return;
    }

    setIsLoading(true);
    setError(null);
    setOptimizationError(null);
    setBacktestRun(null);
    setBacktestTrades([]);

    const { fromDate, toDate } = defaultBacktestWindow();

    const payload = {
      strategy_name: strategy,
      version: 'v1',
      symbol: symbol,
      interval: interval,
      from: fromDate.toISOString(),
      to: toDate.toISOString(),
      fast_period: Number(fastPeriod),
      slow_period: Number(slowPeriod),
      rsi_period: Number(rsiPeriod),
      rsi_oversold: Number(rsiOversold),
      rsi_overbought: Number(rsiOverbought),
      starting_balance: Number(startingBalance),
      fee_rate: Number(feeRate),
      slippage_rate: 0,
      position_sizing_mode: positionSizingMode,
      position_size_value: Number(positionSizeValue),
      trend_filter_enabled: false,
      trend_period: 50,
      cooldown_bars: Number(cooldownBars),
      min_holding_bars: Number(minHoldingBars),
      atr_exit_enabled: atrExitEnabled,
      atr_period: Number(atrPeriod),
      atr_stop_multiplier: Number(atrStopMultiplier),
      atr_take_profit_multiplier: Number(atrTakeProfitMultiplier),
      regime_filter_enabled: regimeFilterEnabled,
      regime_filter_period: Number(regimeFilterPeriod),
      regime_min_atr_percent: Number(regimeMinATRPercent),
      regime_max_atr_percent: Number(regimeMaxATRPercent),
      execution_fill_mode: 'next_open',
      shorting_enabled: shortingEnabled
    };

    try {
      const res = await fetch('/api/v1/backtests', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      const body = await res.json();
      if (!res.ok) {
        throw new Error(body.error || 'Backtest execution failed');
      }

      setBacktestRun(body.data);
      setBacktestTrades(body.data?.round_trips || []);
      showToast('Backtest simulation completed successfully!', 'success');
    } catch (err) {
      setError(err.message);
      showToast(err.message, 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleOptimize = async () => {
    if (!selectedTemplateBacktestable) {
      const blockers = selectedTemplate?.blockers?.join(' ') || 'Required data is not available in the current optimizer engine.';
      const message = `${selectedTemplate?.name || strategy} cannot be optimized yet. ${blockers}`;
      setOptimizationError(message);
      showToast(message, 'warning');
      return;
    }

    setIsOptimizing(true);
    setOptimizationError(null);
    setOptimizationResult(null);

    const { fromDate, toDate } = defaultBacktestWindow();
    const fastPeriods = strategy === 'btc-trend-pullback'
      ? uniquePositiveInts([5, 9, 12, fastPeriod])
      : uniquePositiveInts([5, 8, 9, 12, 20, fastPeriod]);
    const slowPeriods = strategy === 'btc-trend-pullback'
      ? uniquePositiveInts([21, 50, slowPeriod])
      : uniquePositiveInts([21, 34, 50, 100, 200, slowPeriod]);
    const rsiPeriods = strategy === 'btc-trend-pullback'
      ? uniquePositiveInts([10, 14, rsiPeriod])
      : uniquePositiveInts([7, 10, 14, 21, rsiPeriod]);

    const payload = {
      strategy_name: strategy,
      symbol,
      interval,
      from: fromDate.toISOString(),
      to: toDate.toISOString(),
      fast_periods: fastPeriods,
      slow_periods: slowPeriods,
      rsi_periods: rsiPeriods,
      rsi_oversold_values: [25, 30, Number(rsiOversold)],
      rsi_overbought_values: [70, 75, Number(rsiOverbought)],
      starting_balance: Number(startingBalance),
      fee_rate: Number(feeRate),
      slippage_rate: 0,
      position_sizing_mode: positionSizingMode,
      position_size_value: Number(positionSizeValue),
      trend_filter_enabled: false,
      trend_period: 50,
      cooldown_bars: Number(cooldownBars),
      min_holding_bars: Number(minHoldingBars),
      atr_exit_enabled: atrExitEnabled,
      atr_period: Number(atrPeriod),
      atr_stop_multiplier: Number(atrStopMultiplier),
      atr_take_profit_multiplier: Number(atrTakeProfitMultiplier),
      regime_filter_enabled: regimeFilterEnabled,
      regime_filter_period: Number(regimeFilterPeriod),
      regime_min_atr_percent: Number(regimeMinATRPercent),
      regime_max_atr_percent: Number(regimeMaxATRPercent),
      execution_fill_mode: 'next_open',
      shorting_enabled: shortingEnabled,
      train_test_enabled: true,
      train_ratio: 0.7,
      walk_forward_enabled: true,
      walk_forward_folds: 3,
      max_combinations: 100
    };

    try {
      const res = await fetch('/api/v1/backtests/optimizations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      const body = await res.json();
      if (!res.ok) {
        throw new Error(body.error || 'Parameter optimization failed');
      }

      setOptimizationResult(body.data);
      showToast('Parameter optimization completed.', 'success');
    } catch (err) {
      setOptimizationError(err.message);
      showToast(err.message, 'error');
    } finally {
      setIsOptimizing(false);
    }
  };

  const applyOptimizationCandidate = (row) => {
    setFastPeriod(row.fast_period);
    setSlowPeriod(row.slow_period);
    if (row.rsi_period) setRsiPeriod(row.rsi_period);
    if (row.rsi_oversold) setRsiOversold(row.rsi_oversold);
    if (row.rsi_overbought) setRsiOverbought(row.rsi_overbought);
    showToast(`Applied optimizer candidate ${row.fast_period}/${row.slow_period} RSI ${row.rsi_period || rsiPeriod}.`, 'info');
  };

  return (
    <div className="flex-1 flex h-full overflow-hidden select-none w-full bg-bg-60-1">
      {/* Parameters Panel */}
      <aside className="w-80 border-r border-chrome-border flex flex-col bg-bg-60-3 overflow-y-auto no-scrollbar p-3.5 gap-4">
        <div>
          <div className="text-[10px] uppercase tracking-wider text-chrome-text font-bold border-b border-chrome-border pb-1 mb-3">
            Backtest Parameters
          </div>
          <div className="space-y-3.5 text-[11px]">
            <div>
              <label className="block text-chrome-text/80 mb-1">STRATEGY MODEL</label>
              <select
                value={strategy}
                onChange={(e) => handleStrategyChange(e.target.value)}
                className="w-full bg-bg-60-4 border border-chrome-border p-1.5 rounded text-white outline-none focus:border-signal-brand h-[28px] cursor-pointer"
              >
                <optgroup label="Native Runtime">
                  <option value="sma-crossover">SMA Crossover / Native</option>
                  <option value="rsi-mean-reversion">RSI Mean Reversion / Native</option>
                  <option value="btc-trend-pullback">BTC Trend Pullback / Native</option>
                </optgroup>
                <optgroup label="Predefined Templates">
                  {strategyTemplates.map((tmpl) => (
                    <option key={tmpl.id} value={tmpl.id}>
                      {tmpl.name} / {templateStatusLabels[tmpl.support_status] || tmpl.support_status}
                    </option>
                  ))}
                </optgroup>
              </select>
              {selectedTemplate && (
                <div className={`mt-2 border rounded p-2 text-[10px] leading-relaxed ${
                  selectedTemplateBacktestable
                    ? 'border-signal-info/35 bg-signal-info/10 text-chrome-text'
                    : 'border-signal-warn/35 bg-signal-warn/10 text-signal-warn'
                }`}>
                  <div className="font-bold uppercase">{selectedTemplateStatus}</div>
                  <div>{selectedTemplate.summary}</div>
                  {selectedTemplate.execution_profile && (
                    <div className="mt-2 text-[12px] text-chrome-text">
                      Profile: {selectedTemplate.execution_profile.recommended_interval} · {selectedTemplate.execution_profile.position_size_percent || positionSizeValue}% equity · cooldown {selectedTemplate.execution_profile.cooldown_bars} bars · max {selectedTemplate.execution_profile.max_trades_per_day}/day{selectedTemplate.execution_profile.shorting_enabled ? ' · long/short' : ''}{selectedTemplate.execution_profile.regime_filter_enabled ? ` · ATR regime ${selectedTemplate.execution_profile.regime_min_atr_percent}-${selectedTemplate.execution_profile.regime_max_atr_percent}%` : ''}
                    </div>
                  )}
                  {!selectedTemplateBacktestable && selectedTemplate.blockers?.length > 0 && (
                    <div className="mt-1 text-chrome-text/75">{selectedTemplate.blockers[0]}</div>
                  )}
                </div>
              )}
            </div>

            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="block text-chrome-text/80 mb-1">SYMBOL</label>
                <input
                  type="text"
                  value={symbol}
                  onChange={(e) => setSymbol(e.target.value.toUpperCase())}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px] uppercase"
                />
              </div>
              <div>
                <label className="block text-chrome-text/80 mb-1">INTERVAL</label>
                <select
                  value={interval}
                  onChange={(e) => setInterval(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border p-1.5 rounded text-white outline-none focus:border-signal-brand h-[28px] cursor-pointer"
                >
                  <option value="1m">1 minute</option>
                  <option value="5m">5 minutes</option>
                  <option value="15m">15 minutes</option>
                  <option value="1h">1 hour</option>
                  <option value="1d">1 day</option>
                </select>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-2 border-t border-chrome-border/70 pt-3">
              <div>
                <label className="block text-chrome-text/80 mb-1">FAST PERIOD</label>
                <input
                  type="number"
                  value={fastPeriod}
                  onChange={(e) => setFastPeriod(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
              <div>
                <label className="block text-chrome-text/80 mb-1">SLOW PERIOD</label>
                <input
                  type="number"
                  value={slowPeriod}
                  onChange={(e) => setSlowPeriod(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
            </div>

            <div className="grid grid-cols-3 gap-2">
              <div>
                <label className="block text-chrome-text/80 mb-1">RSI PERIOD</label>
                <input
                  type="number"
                  value={rsiPeriod}
                  onChange={(e) => setRsiPeriod(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
              <div>
                <label className="block text-chrome-text/80 mb-1">OVERSOLD</label>
                <input
                  type="number"
                  value={rsiOversold}
                  onChange={(e) => setRsiOversold(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
              <div>
                <label className="block text-chrome-text/80 mb-1">OVERBOUGHT</label>
                <input
                  type="number"
                  value={rsiOverbought}
                  onChange={(e) => setRsiOverbought(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-2 border-t border-chrome-border/70 pt-3">
              <div>
                <label className="block text-chrome-text/80 mb-1">COOLDOWN (BARS)</label>
                <input
                  type="number"
                  value={cooldownBars}
                  onChange={(e) => setCooldownBars(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
              <div>
                <label className="block text-chrome-text/80 mb-1">MIN HOLD (BARS)</label>
                <input
                  type="number"
                  value={minHoldingBars}
                  onChange={(e) => setMinHoldingBars(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
            </div>

            <div className="border-t border-chrome-border/70 pt-3 space-y-2">
              <div className="flex justify-between items-center">
                <span className="text-chrome-text font-bold">ENABLE ATR EXITS</span>
                <input
                  type="checkbox"
                  checked={atrExitEnabled}
                  onChange={(e) => setAtrExitEnabled(e.target.checked)}
                  className="w-4 h-4 cursor-pointer accent-signal-info"
                />
              </div>
              {atrExitEnabled && (
                <div className="grid grid-cols-2 gap-2 pt-1">
                  <div>
                    <label className="block text-chrome-text/70 mb-1">STOP MULT</label>
                    <input
                      type="number"
                      step="0.1"
                      value={atrStopMultiplier}
                      onChange={(e) => setAtrStopMultiplier(e.target.value)}
                      className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                    />
                  </div>
                  <div>
                    <label className="block text-chrome-text/70 mb-1">LIMIT MULT</label>
                    <input
                      type="number"
                      step="0.1"
                      value={atrTakeProfitMultiplier}
                      onChange={(e) => setAtrTakeProfitMultiplier(e.target.value)}
                      className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                    />
                  </div>
                </div>
              )}
            </div>

            <div className="border-t border-chrome-border/70 pt-3 space-y-3">
              <div>
                <label className="block text-chrome-text/80 mb-1">STARTING BAL ($)</label>
                <input
                  type="number"
                  value={startingBalance}
                  onChange={(e) => setStartingBalance(e.target.value)}
                  className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-chrome-text/80 mb-1">FEE RATE (%)</label>
                  <input
                    type="number"
                    step="0.0001"
                    value={feeRate}
                    onChange={(e) => setFeeRate(e.target.value)}
                    className="w-full bg-bg-60-4 border border-chrome-border px-2 py-1 rounded text-white font-mono-data outline-none focus:border-signal-brand h-[28px]"
                  />
                </div>
                <div>
                  <label className="block text-chrome-text/80 mb-1">SIZE MODE</label>
                  <select
                    value={positionSizingMode}
                    onChange={(e) => setPositionSizingMode(e.target.value)}
                    className="w-full bg-bg-60-4 border border-chrome-border p-1.5 rounded text-white outline-none focus:border-signal-brand h-[28px] cursor-pointer"
                  >
                    <option value="percent_equity">% Equity</option>
                    <option value="fixed_qty">Fixed Qty</option>
                  </select>
                </div>
              </div>
            </div>

            <button
              onClick={handleExecute}
              disabled={isLoading || !selectedTemplateBacktestable}
              className="w-full h-[32px] bg-signal-info hover:bg-signal-info/90 text-bg-60-1 font-bold rounded cursor-pointer btn-active-scale transition-all duration-100 flex justify-center items-center gap-2 disabled:opacity-40 disabled:cursor-not-allowed text-[11px] uppercase tracking-wider mt-2"
            >
              {isLoading ? (
                <>
                  <div className="w-3.5 h-3.5 border-2 border-bg-60-1 border-t-transparent rounded-full animate-spin"></div>
                  <span>Running Simulation...</span>
                </>
              ) : (
                <>
                  <Play size={12} fill="currentColor" />
                  <span>Execute Backtest</span>
                </>
              )}
            </button>

            <button
              onClick={handleOptimize}
              disabled={isOptimizing || isLoading}
              className="w-full h-[32px] bg-bg-60-4 hover:bg-bg-60-5 text-chrome-text border border-chrome-border font-bold rounded cursor-pointer btn-active-scale transition-all duration-100 flex justify-center items-center gap-2 disabled:opacity-40 disabled:cursor-not-allowed text-[11px] uppercase tracking-wider"
            >
              {isOptimizing ? (
                <>
                  <div className="w-3.5 h-3.5 border-2 border-signal-info border-t-transparent rounded-full animate-spin"></div>
                  <span>Optimizing...</span>
                </>
              ) : (
                <>
                  <SlidersHorizontal size={13} />
                  <span>Run Strategy Optimizer</span>
                </>
              )}
            </button>
            <div className="text-[10px] leading-snug text-chrome-text/65">
              Optimizer ranks fast/slow parameter candidates for the selected executable strategy with the same fee, sizing, ATR, regime, shorting, train/test, and walk-forward controls.
            </div>
          </div>
        </div>
      </aside>

      {/* Main Backtest View */}
      <main className="flex-1 flex flex-col min-w-0 bg-bg-60-2 overflow-hidden p-3.5 gap-4">
        {/* Results Banner Header */}
        <div className="border border-chrome-border bg-bg-60-3/60 rounded-lg p-3 flex flex-col gap-1 relative overflow-hidden">
          <div className="absolute top-0 left-0 w-1.5 h-full bg-signal-info"></div>
          <div className="text-[10px] uppercase tracking-wider text-chrome-text font-bold pl-2">
            Simulation Laboratory Diagnostics
          </div>
          <div className="text-[11px] text-chrome-text pl-2 mt-1">
            Run an automated parameter sequence to view capital simulation curve overlays and trade execution logs.
          </div>
        </div>

        {error && (
          <div className="border border-signal-sell bg-signal-sell/15 text-signal-sell p-3 rounded-lg flex items-center gap-2 text-[12px] leading-tight">
            <AlertTriangle size={15} />
            <div>
              <span className="font-bold">Execution Failed:</span> {error}
            </div>
          </div>
        )}

        {optimizationError && (
          <div className="border border-signal-warn bg-signal-warn/15 text-signal-warn p-3 rounded-lg flex items-center gap-2 text-[12px] leading-tight">
            <AlertTriangle size={15} />
            <div>
              <span className="font-bold">Optimization Notice:</span> {optimizationError}
            </div>
          </div>
        )}

        {validationDisplay && (
          <div className={`border px-3 py-2 text-[12px] rounded-lg flex items-start gap-2 ${validationToneClasses[validationDisplay.tone] || validationToneClasses.warning}`}>
            <AlertTriangle size={15} className="mt-0.5 shrink-0" />
            <div className="min-w-0">
              <div className="font-bold uppercase tracking-wide">{validationDisplay.label}</div>
              <div className="text-chrome-text mt-1 leading-snug">
                {validationDisplay.message}
                {backtestRun.validation_status === 'low_bull_market_capture' && capturePercent !== null && (
                  <span className="block mt-1">
                    Captured {formatNumber(capturePercent, 1)}% of benchmark return; hardened minimum is 25.0%.
                  </span>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Equity Canvas Curve */}
        <div className="flex-1 min-h-0 border border-chrome-border bg-bg-60-1 rounded-lg p-3 flex flex-col">
          <div className="flex justify-between items-center text-[10px] text-chrome-text font-bold mb-2 border-b border-chrome-border/70 pb-2">
            <span>EQUITY CURVE VALUATION</span>
            {backtestRun && (
              <span className={`font-bold ${backtestRun.profit_loss >= 0 ? 'text-signal-buy' : 'text-signal-sell'}`}>
                {backtestRun.profit_loss >= 0 ? '▲' : '▼'} {backtestRun.return_percent.toFixed(2)}%
              </span>
            )}
          </div>
          <div className="flex-1 min-h-0 relative">
            {isLoading ? (
              <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 text-chrome-text">
                <div className="w-8 h-8 border-3 border-signal-info border-t-transparent rounded-full animate-spin"></div>
                <div className="text-[11px] tracking-wider uppercase">AGGREGATING DICTIONARY DATA CYCLES...</div>
              </div>
            ) : !backtestRun ? (
              <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 text-chrome-text/70 text-[12px]">
                <TrendingUp size={36} className="opacity-30" />
                <span>Configure settings and run a simulation backtest to generate the equity chart.</span>
              </div>
            ) : (
              <canvas ref={canvasRef} className="w-full h-full" />
            )}
          </div>
        </div>

        {/* Key Metrics Grid */}
        <div className="grid grid-cols-5 gap-3 text-[11px]">
          <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
            <span className="text-chrome-text/60 block font-sans">PROFIT FACTOR</span>
            <span className={`font-mono-data text-[16px] font-bold block mt-1 ${backtestRun?.profit_factor >= 1.2 ? 'text-signal-buy' : 'text-white'}`}>
              {backtestRun ? backtestRun.profit_factor.toFixed(2) : '--'}
            </span>
          </div>
          <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
            <span className="text-chrome-text/60 block font-sans">SHARPE RATIO</span>
            <span className={`font-mono-data text-[16px] font-bold block mt-1 ${backtestRun?.sharpe_ratio >= 1.5 ? 'text-signal-buy' : 'text-white'}`}>
              {backtestRun ? backtestRun.sharpe_ratio.toFixed(2) : '--'}
            </span>
          </div>
          <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
            <span className="text-chrome-text/60 block font-sans">WIN RATE</span>
            <span className="font-mono-data text-[16px] font-bold block mt-1 text-white">
              {backtestRun ? `${(backtestRun.win_rate * 100).toFixed(1)}%` : '--'}
            </span>
          </div>
          <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
            <span className="text-chrome-text/60 block font-sans">MAX DRAWDOWN</span>
            <span className="font-mono-data text-[16px] font-bold block mt-1 text-signal-sell">
              {backtestRun ? `${(backtestRun.max_drawdown * 100).toFixed(2)}%` : '--'}
            </span>
          </div>
          <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
            <span className="text-chrome-text/60 block font-sans">TOTAL TRADES</span>
            <span className="font-mono-data text-[16px] font-bold block mt-1 text-white">
              {backtestRun ? backtestRun.total_trades : '--'}
            </span>
          </div>
        </div>

        {backtestRun && (
          <div className="grid grid-cols-5 gap-3 text-[11px]">
            <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
              <span className="text-chrome-text/60 block font-sans">GROSS PNL</span>
              <span className="font-mono-data text-[14px] font-bold block mt-1 text-white">
                {formatCurrency(backtestRun.gross_profit_loss)}
              </span>
            </div>
            <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
              <span className="text-chrome-text/60 block font-sans">FEES</span>
              <span className="font-mono-data text-[14px] font-bold block mt-1 text-signal-warn">
                {formatCurrency(backtestRun.total_fees)}
              </span>
            </div>
            <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
              <span className="text-chrome-text/60 block font-sans">SLIPPAGE COST</span>
              <span className="font-mono-data text-[14px] font-bold block mt-1 text-signal-warn">
                {formatCurrency(backtestRun.estimated_slippage_cost)}
              </span>
            </div>
            <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
              <span className="text-chrome-text/60 block font-sans">ROUND TRIP COST</span>
              <span className="font-mono-data text-[14px] font-bold block mt-1 text-white">
                {formatNumber(backtestRun.round_trip_cost_percent)}%
              </span>
            </div>
            <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
              <span className="text-chrome-text/60 block font-sans">BREAK EVEN MOVE</span>
              <span className="font-mono-data text-[14px] font-bold block mt-1 text-white">
                {formatNumber(backtestRun.break_even_move_percent)}%
              </span>
            </div>
          </div>
        )}

        {backtestRun && (
          <div className="grid grid-cols-4 gap-3 text-[11px]">
            <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
              <span className="text-chrome-text/60 block font-sans">BENCHMARK RETURN</span>
              <span className={`font-mono-data text-[14px] font-bold block mt-1 ${backtestRun.benchmark_return_percent >= 0 ? 'text-signal-buy' : 'text-signal-sell'}`}>
                {formatNumber(backtestRun.benchmark_return_percent)}%
              </span>
            </div>
            <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
              <span className="text-chrome-text/60 block font-sans">EXCESS RETURN</span>
              <span className={`font-mono-data text-[14px] font-bold block mt-1 ${backtestRun.excess_return_percent >= 0 ? 'text-signal-buy' : 'text-signal-sell'}`}>
                {formatNumber(backtestRun.excess_return_percent)}%
              </span>
            </div>
            <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
              <span className="text-chrome-text/60 block font-sans">BENCHMARK CAPTURE</span>
              <span className={`font-mono-data text-[14px] font-bold block mt-1 ${capturePercent !== null && capturePercent >= 25 ? 'text-signal-buy' : 'text-signal-warn'}`}>
                {capturePercent === null ? '--' : `${formatNumber(capturePercent, 1)}%`}
              </span>
            </div>
            <div className="border border-chrome-border bg-bg-60-3 p-3 rounded-lg">
              <span className="text-chrome-text/60 block font-sans">VALIDATION</span>
              <span className={`font-mono-data text-[12px] font-bold block mt-1 uppercase ${validationDisplay?.tone === 'success' ? 'text-signal-buy' : validationDisplay?.tone === 'error' ? 'text-signal-sell' : 'text-signal-warn'}`}>
                {validationDisplay?.label || '--'}
              </span>
            </div>
          </div>
        )}

        {(isOptimizing || optimizationResult) && (
          <div className="border border-chrome-border bg-bg-60-1 rounded-lg flex flex-col min-h-[180px]">
            <div className="p-2 border-b border-chrome-border flex items-center justify-between gap-3">
              <div>
                <div className="text-[10px] uppercase tracking-wider text-chrome-text font-bold">
                  Parameter Optimization
                </div>
                <div className="text-[10px] text-chrome-text/65 mt-0.5">
                  Selected strategy fast/slow grid, ranked after benchmark validation and robustness checks.
                </div>
              </div>
              {optimizationResult && (
                <div className="text-[10px] text-chrome-text/75 font-mono-data text-right">
                  {optimizationResult.total_combinations} combos · {optimizationResult.train_test_enabled ? '70/30 split' : 'no split'} · {optimizationResult.walk_forward_enabled ? `${optimizationResult.walk_forward_folds} WF folds` : 'no WF'}
                </div>
              )}
            </div>
            <div className="flex-1 overflow-auto p-2 font-mono-data text-[10px]">
              {isOptimizing ? (
                <div className="h-full min-h-[120px] flex flex-col items-center justify-center gap-2 text-chrome-text">
                  <div className="w-7 h-7 border-2 border-signal-info border-t-transparent rounded-full animate-spin"></div>
                  <div className="uppercase tracking-wider">Running train/test and walk-forward candidate ranking...</div>
                </div>
              ) : optimizationRows.length === 0 ? (
                <div className="h-full min-h-[120px] flex items-center justify-center text-chrome-text/60 italic">
                  No optimizer candidates returned for the selected grid.
                </div>
              ) : (
                <table className="w-full border-collapse">
                  <thead>
                    <tr className="text-left text-[10px] text-chrome-text border-b border-chrome-border/80">
                      <th className="p-1 font-bold">Rank</th>
                      <th className="p-1 font-bold">Fast/Slow</th>
                      <th className="p-1 font-bold">RSI</th>
                      <th className="p-1 font-bold">Return</th>
                      <th className="p-1 font-bold">Excess</th>
                      <th className="p-1 font-bold">Capture</th>
                      <th className="p-1 font-bold">PF</th>
                      <th className="p-1 font-bold">Max DD</th>
                      <th className="p-1 font-bold">Validation</th>
                      <th className="p-1 font-bold text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody>
                    {optimizationRows.map((row) => {
                      const rowValidation = getValidationDisplay(row);
                      const rowCapture = benchmarkCapturePercent(row);
                      return (
                        <tr key={`${row.strategy_name}-${row.fast_period}-${row.slow_period}`} className="border-b border-chrome-border/35 hover:bg-bg-60-4/60">
                          <td className="p-1 text-chrome-text">{row.rank}</td>
                          <td className="p-1 text-white font-bold">{row.fast_period}/{row.slow_period}</td>
                          <td className="p-1 text-chrome-text">{row.rsi_period ? `${row.rsi_period} ${formatNumber(row.rsi_oversold, 0)}/${formatNumber(row.rsi_overbought, 0)}` : '--'}</td>
                          <td className={`p-1 font-bold ${row.return_percent >= 0 ? 'text-signal-buy' : 'text-signal-sell'}`}>
                            {formatNumber(row.return_percent)}%
                          </td>
                          <td className={`p-1 ${row.excess_return_percent >= 0 ? 'text-signal-buy' : 'text-signal-sell'}`}>
                            {formatNumber(row.excess_return_percent)}%
                          </td>
                          <td className={`p-1 ${rowCapture !== null && rowCapture >= 25 ? 'text-signal-buy' : 'text-signal-warn'}`}>
                            {rowCapture === null ? '--' : `${formatNumber(rowCapture, 1)}%`}
                          </td>
                          <td className="p-1 text-white">{formatNumber(row.profit_factor)}</td>
                          <td className="p-1 text-signal-sell">{formatNumber(Number(row.max_drawdown) * 100)}%</td>
                          <td className={`p-1 uppercase ${rowValidation?.tone === 'success' ? 'text-signal-buy' : rowValidation?.tone === 'error' ? 'text-signal-sell' : 'text-signal-warn'}`}>
                            {rowValidation?.label || row.validation_status || '--'}
                          </td>
                          <td className="p-1 text-right">
                            <button
                              type="button"
                              onClick={() => applyOptimizationCandidate(row)}
                              className="px-2 h-[22px] rounded border border-chrome-border bg-bg-60-4 hover:bg-bg-60-5 text-chrome-text font-bold uppercase tracking-wide"
                              aria-label={`Apply optimizer candidate fast ${row.fast_period} slow ${row.slow_period}`}
                            >
                              Apply
                            </button>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              )}
            </div>
          </div>
        )}

        {/* Backtest Trades list */}
        <div className="h-[180px] flex flex-col min-h-0 border border-chrome-border bg-bg-60-1 rounded-lg">
          <div className="p-2 border-b border-chrome-border text-[10px] uppercase tracking-wider text-chrome-text font-bold">
            Simulated Trade Logs
          </div>
          <div className="flex-1 overflow-y-auto p-2 font-mono-data text-[10px]">
            <table className="w-full border-collapse">
              <thead>
                <tr className="text-left text-[10px] text-chrome-text border-b border-chrome-border/80">
                  <th className="p-1 font-bold">Exit Time</th>
                  <th className="p-1 font-bold">Duration</th>
                  <th className="p-1 font-bold">Direction</th>
                  <th className="p-1 font-bold">PnL (%)</th>
                  <th className="p-1 font-bold">Exit Reason</th>
                </tr>
              </thead>
              <tbody>
                {backtestTrades.length === 0 ? (
                  <tr>
                    <td colSpan="5" className="p-4 text-center text-chrome-text/60 italic">
                      {isLoading ? 'Processing cycle records...' : 'No simulation executions recorded'}
                    </td>
                  </tr>
                ) : (
                  backtestTrades.map((t, idx) => (
                    <tr key={idx} className="border-b border-chrome-border/35 hover:bg-bg-60-4/60">
                      <td className="p-1">{new Date(t.exit_time || t.exitTime).toLocaleString()}</td>
                      <td className="p-1 text-chrome-text">{(t.holding_seconds / 60).toFixed(0)}m</td>
                      <td className="p-1 text-white font-bold">LONG</td>
                      <td className={`p-1 font-bold ${t.profit_percent >= 0 ? 'text-signal-buy' : 'text-signal-sell'}`}>
                        {(t.profit_percent * 100).toFixed(2)}%
                      </td>
                      <td className="p-1 text-chrome-text">{t.exit_reason}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </main>
    </div>
  );
}
