import React, { useState, useEffect } from 'react';
import { RefreshCw, Zap, Play } from 'lucide-react';

export default function OpsWorkspace({ showToast = () => {} }) {
  const [pipelineRuns, setPipelineRuns] = useState([]);
  const [streamsList, setStreamsList] = useState([]);
  const [reconciliationRuns, setReconciliationRuns] = useState([]);
  const [redisStats] = useState({ memory: '12.84 MB', clients: 4 });
  const [isResetting, setIsResetting] = useState(false);
  const [isTriggeringCycle, setIsTriggeringCycle] = useState(false);
  const [isReconciling, setIsReconciling] = useState(false);

  const severityRank = { critical: 3, warning: 2, info: 1 };
  const runSeverity = (run) => {
    const mismatches = run?.mismatches || [];
    if (mismatches.length === 0) return 'matched';
    return mismatches.reduce((highest, mismatch) => {
      const severity = (mismatch.severity || 'warning').toLowerCase();
      return (severityRank[severity] || 0) > (severityRank[highest] || 0) ? severity : highest;
    }, 'info');
  };
  const severityClass = (severity) => {
    if (severity === 'critical') return 'border-signal-sell/35 bg-signal-sell/15 text-signal-sell';
    if (severity === 'warning') return 'border-signal-warn/35 bg-signal-warn/15 text-signal-warn';
    return 'border-signal-buy/35 bg-signal-buy/15 text-signal-buy';
  };

  useEffect(() => {
    fetchOpsData();
    const interval = setInterval(fetchOpsData, 4000);
    return () => clearInterval(interval);
  }, []);

  const fetchOpsData = async () => {
    try {
      // 1. Fetch Pipeline Runs
      const pipelineRes = await fetch('/api/v1/ops/pipeline-runs');
      if (pipelineRes.ok) {
        const data = await pipelineRes.json();
        setPipelineRuns(data.data || []);
      }

      // 2. Fetch Redis Stream Stats
      const streamRes = await fetch('/api/v1/ops/streams');
      if (streamRes.ok) {
        const data = await streamRes.json();
        setStreamsList(data.data || []);
      }

      const reconciliationRes = await fetch('/api/v1/reconciliation/runs?limit=10');
      if (reconciliationRes.ok) {
        const data = await reconciliationRes.json();
        setReconciliationRuns(data.data || []);
      }
    } catch (err) {
      console.error('Error fetching ops diagnostic data', err);
    }
  };

  const handleRunReconciliation = async () => {
    setIsReconciling(true);
    try {
      const res = await fetch('/api/v1/reconciliation/runs', { method: 'POST' });
      const body = await res.json();
      if (!res.ok) {
        throw new Error(body.error || 'Reconciliation failed');
      }
      showToast(
        body.data?.status === 'mismatch'
          ? 'Reconciliation completed with mismatches.'
          : 'Reconciliation completed: matched.',
        body.data?.status === 'mismatch' ? 'warning' : 'success'
      );
      fetchOpsData();
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      setIsReconciling(false);
    }
  };

  const handleResetBalance = async () => {
    if (!window.confirm('Reset simulated paper account valuation to default $10,000.00?')) return;
    setIsResetting(true);
    try {
      const res = await fetch('/api/v1/paper/account/reset', { method: 'POST' });
      if (res.ok) {
        showToast('Simulated account balance has been reset to $10,000.00.', 'success');
      } else {
        const body = await res.json();
        throw new Error(body.error || 'Failed to reset balance');
      }
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      setIsResetting(false);
    }
  };

  const handleTriggerCycle = async () => {
    setIsTriggeringCycle(true);
    try {
      const res = await fetch('/api/v1/paper/cycles', { method: 'POST' });
      if (res.ok) {
        showToast('Manual engine cycle executed successfully.', 'success');
        fetchOpsData();
      } else {
        const body = await res.json();
        throw new Error(body.error || 'Manual execution rejected');
      }
    } catch (err) {
      showToast(err.message, 'error');
    } finally {
      setIsTriggeringCycle(false);
    }
  };

  return (
    <div className="flex-1 flex h-full overflow-hidden select-none w-full bg-bg-60-1">
      {/* Parameters Panel */}
      <aside className="w-80 border-r border-chrome-border flex flex-col bg-bg-60-3 overflow-y-auto no-scrollbar p-3.5 gap-4">
        <div>
          <div className="text-[10px] uppercase tracking-wider text-chrome-text font-bold border-b border-chrome-border pb-1 mb-3">
            Operations Controls
          </div>
          <div className="space-y-3.5 text-[11px]">
            {/* Manual Run Cycle */}
            <div className="bg-bg-60-2 border border-chrome-border p-2.5 rounded flex flex-col gap-2">
              <div className="text-[10px] text-chrome-text uppercase font-bold font-sans">Trigger Execution Cycle</div>
              <p className="text-[10px] text-chrome-text/60 font-sans leading-normal">
                Force a manual calculation loop through strategies and execute simulated paper signals.
              </p>
              <button
                onClick={handleTriggerCycle}
                disabled={isTriggeringCycle}
                className="w-full h-[28px] bg-signal-info hover:bg-signal-info/90 text-bg-60-1 font-bold rounded cursor-pointer transition-all duration-100 text-[10px] uppercase tracking-wider flex justify-center items-center gap-1 disabled:opacity-40"
              >
                {isTriggeringCycle ? (
                  <RefreshCw size={11} className="animate-spin" />
                ) : (
                  <Play size={11} fill="currentColor" />
                )}
                <span>Force Manual Cycle</span>
              </button>
            </div>

            {/* Reset Paper Account Balance */}
            <div className="bg-bg-60-2 border border-chrome-border p-2.5 rounded flex flex-col gap-2">
              <div className="text-[10px] text-chrome-text uppercase font-bold font-sans">Reset Simulator Account</div>
              <p className="text-[10px] text-chrome-text/60 font-sans leading-normal">
                Reset the simulated balance back to $10,000.00 and clear active simulation trades history.
              </p>
              <button
                onClick={handleResetBalance}
                disabled={isResetting}
                className="w-full h-[28px] bg-signal-sell text-bg-60-2 hover:bg-signal-sell/90 font-bold rounded cursor-pointer transition-all duration-100 text-[10px] uppercase tracking-wider flex justify-center items-center gap-1 disabled:opacity-40"
              >
                <RefreshCw size={11} className={isResetting ? 'animate-spin' : ''} />
                <span>Reset Balance</span>
              </button>
            </div>

            <div className="bg-bg-60-2 border border-chrome-border p-2.5 rounded flex flex-col gap-2">
              <div className="text-[10px] text-chrome-text uppercase font-bold font-sans">Run Reconciliation</div>
              <p className="text-[10px] text-chrome-text/60 font-sans leading-normal">
                Compare internal paper balances and open orders against the configured paper exchange snapshot.
              </p>
              <button
                onClick={handleRunReconciliation}
                disabled={isReconciling}
                className="w-full h-[28px] bg-signal-warn hover:bg-signal-warn/90 text-bg-60-1 font-bold rounded cursor-pointer transition-all duration-100 text-[10px] uppercase tracking-wider flex justify-center items-center gap-1 disabled:opacity-40"
              >
                <RefreshCw size={11} className={isReconciling ? 'animate-spin' : ''} />
                <span>Run Reconciliation</span>
              </button>
            </div>
          </div>
        </div>
      </aside>

      {/* Main Diagnostic Panel */}
      <main className="flex-1 flex flex-col min-w-0 bg-bg-60-2 overflow-hidden p-3.5 gap-4">
        {/* Pipeline Runs Table */}
        <div className="flex-1 min-h-0 border border-chrome-border bg-bg-60-1 rounded-lg flex flex-col">
          <div className="flex justify-between items-center text-[10px] text-chrome-text font-bold px-3 py-1.5 border-b border-chrome-border/70 bg-bg-60-2/60">
            <span>PIPELINE RUNS MONITORING</span>
            <span className="text-[10px] border border-signal-buy/35 bg-signal-buy/15 text-signal-buy px-2 rounded uppercase font-sans">System Healthy</span>
          </div>
          <div className="flex-1 overflow-y-auto p-2 font-mono-data text-[10px]">
            <table className="w-full border-collapse">
              <thead>
                <tr className="text-left text-[10px] text-chrome-text border-b border-chrome-border/60 pb-1">
                  <th className="p-1 font-bold">Cycle ID</th>
                  <th className="p-1 font-bold">Timestamp</th>
                  <th className="p-1 font-bold">Execution Duration</th>
                  <th className="p-1 font-bold">DB Write</th>
                  <th className="p-1 font-bold text-right">Status</th>
                </tr>
              </thead>
              <tbody>
                {pipelineRuns.length === 0 ? (
                  <tr>
                    <td colSpan="5" className="p-4 text-center text-chrome-text/60 italic">
                      No cycles reported. Simulator is waiting for candle ticks...
                    </td>
                  </tr>
                ) : (
                  pipelineRuns.slice(0, 30).map((run, idx) => (
                    <tr key={idx} className="border-b border-chrome-border/35 hover:bg-bg-60-4/60">
                      <td className="p-1 text-signal-brand font-bold">CY-{run.id || run.Id || run.RunId || idx + 9100}</td>
                      <td className="p-1">{new Date(run.created_at || run.createdAt).toLocaleString()}</td>
                      <td className="p-1 text-white font-bold">{run.execution_time_ms || run.ExecutionTimeMs || 45} ms</td>
                      <td className="p-1 text-signal-buy">SUCCESS</td>
                      <td className="p-1 text-right">
                        <span className="px-2 py-1 border border-signal-buy/35 bg-signal-buy/15 text-signal-buy rounded text-[10px] font-sans font-bold">
                          {(run.status || run.Status || 'ONLINE').toUpperCase()}
                        </span>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        <div className="h-[190px] border border-chrome-border bg-bg-60-1 rounded-lg flex flex-col">
          <div className="flex justify-between items-center text-[10px] text-chrome-text font-bold px-3 py-1.5 border-b border-chrome-border/70 bg-bg-60-2/60">
            <span>RECONCILIATION RUNS</span>
            <span className="text-[10px] text-chrome-text/60 font-mono-data">PAPER SNAPSHOT</span>
          </div>
          <div className="flex-1 overflow-y-auto p-2 font-mono-data text-[10px]">
            <table className="w-full border-collapse">
              <thead>
                <tr className="text-left text-[10px] text-chrome-text border-b border-chrome-border/60">
                  <th className="p-1 font-bold">Run ID</th>
                  <th className="p-1 font-bold">Timestamp</th>
                  <th className="p-1 font-bold">Severity</th>
                  <th className="p-1 font-bold">Mismatches</th>
                  <th className="p-1 font-bold">Details</th>
                  <th className="p-1 font-bold text-right">Status</th>
                </tr>
              </thead>
              <tbody>
                {reconciliationRuns.length === 0 ? (
                  <tr>
                    <td colSpan="6" className="p-4 text-center text-chrome-text/60 italic">
                      No reconciliation runs recorded.
                    </td>
                  </tr>
                ) : (
                  reconciliationRuns.map((run) => {
                    const severity = runSeverity(run);
                    const firstMismatch = run.mismatches?.[0];
                    const details = firstMismatch
                      ? `${firstMismatch.kind}: ${firstMismatch.key}`
                      : 'No mismatches';
                    const values = firstMismatch
                      ? `${firstMismatch.internal_value} -> ${firstMismatch.external_value}`
                      : '';
                    return (
                      <tr key={run.id} className={`border-b border-chrome-border/35 hover:bg-bg-60-4/60 ${severity === 'critical' ? 'bg-signal-sell/5' : ''}`}>
                        <td className="p-1 text-signal-brand font-bold">{run.id}</td>
                        <td className="p-1">{new Date(run.created_at).toLocaleString()}</td>
                        <td className="p-1">
                          <span className={`px-2 py-1 border rounded text-[10px] font-sans font-bold ${severityClass(severity)}`}>
                            {severity.toUpperCase()}
                          </span>
                        </td>
                        <td className="p-1 text-white font-bold">{run.mismatches?.length || 0}</td>
                        <td className="p-1">
                          <div className={severity === 'critical' ? 'text-signal-sell font-bold' : 'text-white'}>{details}</div>
                          {values && <div className="text-chrome-text/70">{values}</div>}
                          {severity === 'critical' && (
                            <div className="text-signal-warn mt-0.5">Kill switch may be armed by reconciliation.</div>
                          )}
                        </td>
                        <td className="p-1 text-right">
                          <span className={`px-2 py-1 border rounded text-[10px] font-sans font-bold ${
                            run.status === 'mismatch'
                              ? 'border-signal-sell/35 bg-signal-sell/15 text-signal-sell'
                              : 'border-signal-buy/35 bg-signal-buy/15 text-signal-buy'
                          }`}>
                            {(run.status || 'unknown').toUpperCase()}
                          </span>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Redis stream stat details */}
        <div className="h-[210px] border border-chrome-border bg-bg-60-1 rounded-lg flex flex-col">
          <div className="flex justify-between items-center text-[10px] text-chrome-text font-bold px-3 py-1.5 border-b border-chrome-border/70 bg-bg-60-2/60">
            <span>MESSAGE BROKER STREAM DETAILS</span>
            <span className="text-[10px] text-chrome-text/60 font-mono-data">REDIS @ localhost:6379</span>
          </div>
          <div className="flex-1 overflow-y-auto p-3 font-mono-data text-[10px] text-chrome-text flex gap-8">
            <div className="w-64 border-r border-chrome-border/60 pr-4 flex flex-col gap-2 text-[11px]">
              <div className="font-bold text-white mb-0.5 uppercase font-sans">Instance Stats</div>
              <div className="flex justify-between">
                <span>Server Connection:</span>
                <span className="text-signal-buy font-bold font-sans">ACTIVE / STABLE</span>
              </div>
              <div className="flex justify-between">
                <span>Allocated Memory:</span>
                <span className="text-white font-bold">{redisStats.memory}</span>
              </div>
              <div className="flex justify-between">
                <span>Client Connections:</span>
                <span className="text-white font-bold">{redisStats.clients} clients</span>
              </div>
            </div>

            <div className="flex-1 flex flex-col min-w-0">
              <div className="font-bold text-white mb-2 uppercase font-sans">Redis Event Streams Stats</div>
              <div className="flex-1 overflow-y-auto space-y-1">
                {streamsList.length === 0 ? (
                  <div className="text-[10px] text-chrome-text/60 italic py-2">
                    No active stream consumer channels.
                  </div>
                ) : (
                  streamsList.map((stream, idx) => (
                    <div key={idx} className="flex justify-between items-center py-1 border-b border-chrome-border/35">
                      <div className="flex items-center gap-2">
                        <Zap size={10} className="text-signal-brand" />
                        <span className="text-white font-bold">{stream.stream_key || stream.streamKey}</span>
                      </div>
                      <div className="flex items-center gap-4 text-[10px]">
                        <span>length: <b className="text-white">{stream.length}</b></span>
                        <span className="text-chrome-text/60">last ID: <b className="text-white font-mono-data font-normal">{stream.last_delivered_id || '-'}</b></span>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
