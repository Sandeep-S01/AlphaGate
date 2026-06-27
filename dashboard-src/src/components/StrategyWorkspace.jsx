import React, { useState, useEffect } from 'react';
import { Play, Code, FileText, Plus, Save } from 'lucide-react';

const supportLabels = {
  executable_native: 'Native',
  executable_pine: 'Pine',
  template_only: 'Template',
  blocked_by_data: 'Data Blocked'
};

const supportClasses = {
  executable_native: 'text-signal-buy border-signal-buy/60 bg-signal-buy/10',
  executable_pine: 'text-signal-info border-signal-info/60 bg-signal-info/10',
  template_only: 'text-signal-warn border-signal-warn/60 bg-signal-warn/10',
  blocked_by_data: 'text-signal-sell border-signal-sell/60 bg-signal-sell/10'
};

export default function StrategyWorkspace({ showToast = () => {} }) {
  const [strategiesList, setStrategiesList] = useState([]);
  const [templatesList, setTemplatesList] = useState([]);
  const [catalogMode, setCatalogMode] = useState('templates');
  const [activeStrategy, setActiveStrategy] = useState(null);
  const [activeTemplate, setActiveTemplate] = useState(null);
  const [templateSaveBlocked, setTemplateSaveBlocked] = useState(false);
  const [strategyName, setStrategyName] = useState('My EMA Cross Strategy');
  const [pineCode, setPineCode] = useState(`//@version=5\nindicator("EMA Crossover Strategy", overlay=true)\n\nfast = ta.ema(close, 9)\nslow = ta.ema(close, 21)\n\nbuy = ta.crossover(fast, slow)\nsell = ta.crossunder(fast, slow)\n\nif buy\n    strategy.entry("Buy", strategy.long)\nif sell\n    strategy.close("Buy")`);
  
  const [isCompiling, setIsCompiling] = useState(false);
  const [logs, setLogs] = useState([
    { type: 'info', message: 'Initializing strategy compilation workspace...' }
  ]);
  const [, setValidationResult] = useState(null);

  // Load compiled strategies list on init
  useEffect(() => {
    fetchStrategies();
    fetchTemplates();
  }, []);

  const fetchStrategies = async () => {
    try {
      const res = await fetch('/api/v1/strategies/pine');
      if (res.ok) {
        const data = await res.json();
        setStrategiesList(data.data || []);
      }
    } catch (err) {
      console.error('Error fetching strategies', err);
    }
  };

  const fetchTemplates = async () => {
    try {
      const res = await fetch('/api/v1/strategies/templates');
      if (res.ok) {
        const data = await res.json();
        setTemplatesList(data.data || []);
      }
    } catch (err) {
      console.error('Error fetching strategy templates', err);
    }
  };

  const handleValidate = async () => {
    setIsCompiling(true);
    setValidationResult(null);
    setLogs((prev) => [...prev, { type: 'info', message: 'Parsing Pine Script grammar syntax...' }]);

    try {
      const res = await fetch('/api/v1/strategies/pine/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pine_code: pineCode })
      });
      const data = await res.json();
      setValidationResult(data);

      if (data.valid) {
        const parsedIndicators = data.indicators?.map(i => i.name).join(', ') || 'None';
        const parsedRules = data.rules?.length || 0;
        setLogs((prev) => [
          ...prev,
          { type: 'success', message: 'Compilation completed successfully.' },
          { type: 'info', message: `Parsed Indicators: [ ${parsedIndicators} ]` },
          { type: 'info', message: `Execution Rules defined: [ ${parsedRules} rules ]` }
        ]);
        showToast('Strategy validation completed: OK', 'success');
      } else {
        const errList = data.errors || ['Unknown compilation token fault'];
        setLogs((prev) => [
          ...prev,
          ...errList.map(e => ({ type: 'error', message: `Error: ${e}` }))
        ]);
        showToast('Compilation diagnostics failed.', 'error');
      }
    } catch (err) {
      setLogs((prev) => [...prev, { type: 'error', message: `Build failed: ${err.message}` }]);
      showToast('Validation connection failure.', 'error');
    } finally {
      setIsCompiling(false);
    }
  };

  const handleSave = async () => {
    if (!strategyName.trim()) {
      showToast('Strategy name is required', 'error');
      return;
    }
    if (templateSaveBlocked || !pineCode.trim()) {
      showToast('This template has no executable Pine code yet.', 'warning');
      setLogs((prev) => [...prev, { type: 'warning', message: 'Save blocked: selected template is a reference profile only.' }]);
      return;
    }

    setIsCompiling(true);
    setLogs((prev) => [...prev, { type: 'info', message: `Saving strategy: ${strategyName}...` }]);

    try {
      const res = await fetch('/api/v1/strategies/pine', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: strategyName, pine_code: pineCode })
      });
      const data = await res.json();

      if (res.ok) {
        setLogs((prev) => [...prev, { type: 'success', message: `Strategy saved successfully inside database as: ${strategyName}` }]);
        showToast('Pine Script strategy stored.', 'success');
        fetchStrategies();
      } else {
        const errList = data.errors || [data.error || 'Syntax parsing failed'];
        setLogs((prev) => [
          ...prev,
          ...errList.map(e => ({ type: 'error', message: `Error: ${e}` }))
        ]);
        showToast('Failed to save strategy: syntax errors present.', 'error');
      }
    } catch (err) {
      setLogs((prev) => [...prev, { type: 'error', message: `Save error: ${err.message}` }]);
    } finally {
      setIsCompiling(false);
    }
  };

  const handleSelectStrategy = (strat) => {
    setActiveStrategy(strat.id);
    setActiveTemplate(null);
    setTemplateSaveBlocked(false);
    setStrategyName(strat.name);
    setPineCode(strat.pine_code || strat.pineCode);
    setValidationResult(null);
    setLogs([
      { type: 'info', message: `Scope shifted to: ${strat.name}` }
    ]);
  };

  const handleSelectTemplate = (tmpl) => {
    const code = tmpl.pine_code || tmpl.pineCode || '';
    setActiveTemplate(tmpl.id);
    setActiveStrategy(null);
    setStrategyName(tmpl.name);
    setPineCode(code || `// ${tmpl.name}\n// Reference template only. Required data: ${(tmpl.required_data || []).join(', ')}`);
    setTemplateSaveBlocked(!code);
    setValidationResult(null);

    const ruleLines = [
      { type: 'info', message: `Template loaded: ${tmpl.name}` },
      { type: tmpl.support_status === 'blocked_by_data' ? 'warning' : 'info', message: `Support: ${supportLabels[tmpl.support_status] || tmpl.support_status}` },
      { type: 'info', message: `Required Data: ${(tmpl.required_data || []).join(', ')}` },
      ...(tmpl.entry_rules || []).map((rule) => ({ type: 'info', message: `Entry: ${rule}` })),
      ...(tmpl.exit_rules || []).map((rule) => ({ type: 'info', message: `Exit: ${rule}` })),
      ...(tmpl.risk_rules || []).map((rule) => ({ type: 'info', message: `Risk: ${rule}` })),
      ...(tmpl.blockers || []).map((blocker) => ({ type: 'warning', message: `Blocked/Note: ${blocker}` }))
    ];
    setLogs(ruleLines);
    showToast(code ? 'Template loaded into Pine editor.' : 'Reference template loaded.', code ? 'info' : 'warning');
  };

  const handleNewStrategy = () => {
    setActiveStrategy(null);
    setActiveTemplate(null);
    setTemplateSaveBlocked(false);
    setStrategyName('New Strategy');
    setPineCode(`//@version=5\nindicator("Custom Logic Strategy", overlay=true)\n\n// Write rules here`);
    setValidationResult(null);
    setLogs([
      { type: 'info', message: 'New blank compiler profile opened.' }
    ]);
  };

  return (
    <div className="flex-1 flex h-full overflow-hidden select-none w-full bg-bg-60-1">
      {/* Strategies List Panel */}
      <aside className="w-64 border-r border-chrome-border flex flex-col bg-bg-60-3 overflow-y-auto no-scrollbar p-3.5 gap-4">
        <div>
          <div className="flex justify-between items-center border-b border-chrome-border pb-1 mb-3">
            <span className="text-[10px] uppercase tracking-wider text-chrome-text font-bold">Strategy Catalog</span>
            <button
              onClick={handleNewStrategy}
              className="text-signal-info hover:text-white transition-colors cursor-pointer text-[10px] flex items-center gap-1 font-bold font-sans uppercase"
            >
              <Plus size={10} />
              <span>New</span>
            </button>
          </div>
          <div className="grid grid-cols-2 gap-1 mb-3">
            {['templates', 'saved'].map((mode) => (
              <button
                key={mode}
                onClick={() => setCatalogMode(mode)}
                className={`h-[26px] border text-[10px] uppercase font-bold tracking-wider rounded transition-colors ${
                  catalogMode === mode
                    ? 'border-signal-info text-white bg-signal-info/15'
                    : 'border-chrome-border text-chrome-text hover:text-white hover:border-chrome-text'
                }`}
              >
                {mode === 'templates' ? 'Templates' : 'Saved'}
              </button>
            ))}
          </div>
          <div className="space-y-1">
            {catalogMode === 'templates' && templatesList.length === 0 ? (
              <div className="text-[10px] text-chrome-text/70 text-center py-4 italic font-sans">
                No predefined templates available.
              </div>
            ) : catalogMode === 'templates' ? (
              templatesList.map((tmpl) => {
                const isActive = tmpl.id === activeTemplate;
                const badgeClass = supportClasses[tmpl.support_status] || 'text-chrome-text border-chrome-border';
                return (
                  <button
                    key={tmpl.id}
                    onClick={() => handleSelectTemplate(tmpl)}
                    className={`w-full text-left p-2 rounded transition-all duration-100 flex flex-col gap-1 border-l-2 text-[11px] ${
                      isActive
                        ? 'bg-bg-60-4 text-white border-l-signal-info'
                        : 'text-chrome-text hover:bg-bg-60-4/70 hover:text-white border-l-transparent'
                    }`}
                  >
                    <span className="flex items-center justify-between gap-2">
                      <span className="truncate font-sans font-bold">{tmpl.name}</span>
                      <span className={`shrink-0 border rounded px-1 py-0.5 text-[10px] uppercase ${badgeClass}`}>
                        {supportLabels[tmpl.support_status] || tmpl.support_status}
                      </span>
                    </span>
                    <span className="text-[10px] text-chrome-text/70 truncate">
                      {tmpl.market} / {tmpl.category}
                    </span>
                    {tmpl.execution_profile && (
                      <span className="text-[10px] text-chrome-text/70">
                        Profile: {tmpl.execution_profile.recommended_interval} · cooldown {tmpl.execution_profile.cooldown_bars} · max {tmpl.execution_profile.max_trades_per_day}/day
                      </span>
                    )}
                  </button>
                );
              })
            ) : strategiesList.length === 0 ? (
              <div className="text-[10px] text-chrome-text/70 text-center py-4 italic font-sans">
                No custom strategies compiled.
              </div>
            ) : (
              strategiesList.map((strat) => {
                const isActive = strat.id === activeStrategy;
                return (
                  <button
                    key={strat.id}
                    onClick={() => handleSelectStrategy(strat)}
                    className={`w-full text-left p-2 rounded transition-all duration-100 flex items-center gap-2 border-l-2 text-[11px] ${
                      isActive
                        ? 'bg-bg-60-4 text-signal-info border-l-signal-info'
                        : 'text-chrome-text hover:bg-bg-60-4/70 hover:text-white border-l-transparent'
                    }`}
                  >
                    <FileText size={11} className={isActive ? 'text-signal-info' : 'text-chrome-text/60'} />
                    <span className="truncate font-sans font-bold">{strat.name}</span>
                  </button>
                );
              })
            )}
          </div>
        </div>
      </aside>

      {/* Editor & Log Panel */}
      <main className="flex-1 flex flex-col min-w-0 bg-bg-60-2 overflow-hidden p-3.5 gap-4">
        {/* Strategy Title Area */}
        <div className="flex justify-between items-center border border-chrome-border bg-bg-60-3/60 rounded-lg p-3 relative overflow-hidden">
          <div className="absolute top-0 left-0 w-1.5 h-full bg-signal-brand"></div>
          <div className="flex items-center gap-2 pl-2">
            <Code size={14} className="text-signal-brand" />
            <input
              type="text"
              value={strategyName}
              onChange={(e) => setStrategyName(e.target.value)}
              className="bg-transparent border-b border-transparent hover:border-chrome-border focus:border-signal-brand font-bold text-white text-[12px] outline-none w-64 h-[24px]"
            />
          </div>
          <div className="flex gap-2">
            <button
              onClick={handleValidate}
              disabled={isCompiling || templateSaveBlocked}
              title={templateSaveBlocked ? 'Reference template only; no executable Pine code is available yet.' : 'Validate Pine code'}
              className="h-[28px] border border-chrome-border hover:border-white text-chrome-text hover:text-white font-bold px-3 rounded flex items-center gap-2 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed text-[11px] uppercase tracking-wider font-sans"
            >
              <Play size={10} fill="currentColor" />
              <span>Validate Code</span>
            </button>
            <button
              onClick={handleSave}
              disabled={isCompiling || templateSaveBlocked}
              title={templateSaveBlocked ? 'Reference template only; no executable Pine code is available yet.' : 'Compile and save Pine strategy'}
              className="h-[28px] bg-signal-brand hover:bg-signal-brand/90 text-bg-60-1 font-bold px-3 rounded flex items-center gap-2 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed text-[11px] uppercase tracking-wider font-sans transition-colors"
            >
              <Save size={10} />
              <span>{templateSaveBlocked ? 'Reference Only' : 'Compile & Save'}</span>
            </button>
          </div>
        </div>

        {/* Script Editor Container */}
        <div className="flex-1 min-h-0 border border-chrome-border bg-bg-60-1 rounded-lg flex flex-col relative overflow-hidden">
          <div className="flex justify-between items-center text-[10px] text-chrome-text font-bold px-3 py-1.5 border-b border-chrome-border/70 bg-bg-60-2/60 select-none">
            <span>PINE SCRIPT CORE COMPILER</span>
            <span className="font-mono-data text-[10px] text-chrome-text/60">PINE v5.0</span>
          </div>
          <div className="flex-1 min-h-0 relative flex">
            {/* Simple Line Numbers */}
            <div className="w-10 bg-bg-60-2 border-r border-chrome-border/70 text-right p-3 select-none font-mono-data text-[11px] text-chrome-text/60 leading-relaxed overflow-hidden">
              {Array.from({ length: Math.max(pineCode.split('\n').length, 1) }).map((_, i) => (
                <div key={i}>{i + 1}</div>
              ))}
            </div>
            {/* Editor Textarea */}
            <textarea
              spellCheck="false"
              value={pineCode}
              onChange={(e) => setPineCode(e.target.value)}
              className="flex-1 h-full bg-transparent border-none p-3 text-white font-mono-data text-[11px] leading-relaxed outline-none resize-none overflow-y-auto"
            />
          </div>
        </div>

        {/* Compiler Diagnostics console */}
        <div className="h-[180px] flex flex-col min-h-0 border border-chrome-border bg-bg-60-3/35 rounded-lg">
          <div className="p-2 border-b border-chrome-border text-[10px] uppercase tracking-wider text-chrome-text font-bold">
            Compiler Diagnostic Outputs
          </div>
          <div className="flex-1 overflow-y-auto p-3 font-mono-data text-[11px] leading-relaxed space-y-1 text-chrome-text">
            {logs.map((l, idx) => {
              const isErr = l.type === 'error';
              const isOk = l.type === 'success';
              const isWarn = l.type === 'warning';
              return (
                <div 
                  key={idx} 
                  className={isErr ? 'text-signal-sell' : isOk ? 'text-signal-buy' : isWarn ? 'text-signal-warn' : 'text-chrome-text/75'}
                >
                  [{l.type.toUpperCase()}] {l.message}
                </div>
              );
            })}
          </div>
        </div>
      </main>
    </div>
  );
}
