package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sentra/internal/activation"
	"sentra/internal/audit"
	"sentra/internal/backtest"
	"sentra/internal/execution"
	"sentra/internal/marketdata"
	"sentra/internal/observability"
	"sentra/internal/orchestration"
	"sentra/internal/pine"
	"sentra/internal/reconciliation"
	"sentra/internal/risk"
	"sentra/internal/safety"
	"sentra/internal/strategy"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Dependencies struct {
	Postgres             Pinger
	Redis                Pinger
	Candles              CandleReader
	Signals              SignalReader
	Backfills            BackfillStore
	BackfillRunner       BackfillRunner
	Aggregator           AggregationRunner
	RiskDecisions        RiskDecisionReader
	PaperAccount         PaperAccountReader
	Orders               OrderReader
	Trades               TradeReader
	ExecutionStatus      execution.Status
	Metrics              MetricsReader
	PipelineRuns         PipelineRunReader
	Streams              StreamStatsReader
	Dashboard            http.FileSystem
	StrategySettings     StrategySettingsStore
	SignalStore          SignalStore
	Backtests            BacktestStore
	StrategyComparisons  StrategyComparisonStore
	StrategyActivations  activationStore
	ReconciliationRuns   ReconciliationStore
	ReconciliationRunner ReconciliationRunner
	RiskSettings         RiskSettingsStore
	PaperCycleRunner     PaperCycleRunner
	Safety               SafetyStore
	Audit                AuditStore
	Reports              ReportStore
	Auth                 AuthConfig
	Security             SecurityConfig
	PineRepository       *pine.Repository
}

type CandleReader interface {
	List(ctx context.Context, query marketdata.CandleQuery) ([]marketdata.Candle, error)
	Coverage(ctx context.Context, query marketdata.CandleQuery) (marketdata.Coverage, error)
}

type BackfillStore interface {
	Create(ctx context.Context, job marketdata.BackfillJob) (marketdata.BackfillJob, error)
	Get(ctx context.Context, id string) (marketdata.BackfillJob, error)
	List(ctx context.Context, query marketdata.BackfillJobQuery) ([]marketdata.BackfillJob, error)
	Save(ctx context.Context, job marketdata.BackfillJob) (marketdata.BackfillJob, error)
}

type BackfillRunner interface {
	Resume(ctx context.Context, jobID string) (marketdata.BackfillResult, error)
}

type AggregationRunner interface {
	Aggregate(ctx context.Context, request marketdata.AggregationRequest) (marketdata.AggregationResult, error)
}

type SignalReader interface {
	Latest(ctx context.Context, symbol string) (strategy.Signal, error)
	List(ctx context.Context, query strategy.SignalQuery) ([]strategy.Signal, error)
}

type SignalStore interface {
	Save(ctx context.Context, signal strategy.Signal) (string, error)
}

type StrategySettingsStore interface {
	Get(ctx context.Context) (strategy.Settings, error)
	Save(ctx context.Context, settings strategy.Settings) (strategy.Settings, error)
}

type BacktestStore interface {
	Save(ctx context.Context, run backtest.Run, trades []backtest.Trade) (backtest.Run, error)
	List(ctx context.Context, query backtest.Query) ([]backtest.Run, error)
	Get(ctx context.Context, id string) (backtest.Run, []backtest.Trade, error)
}

type backtestStoreWithOptions interface {
	SaveWithOptions(ctx context.Context, run backtest.Run, trades []backtest.Trade, options backtest.SaveOptions) (backtest.Run, error)
}

type StrategyComparisonStore interface {
	Save(ctx context.Context, comparison backtest.Comparison) (backtest.Comparison, error)
	List(ctx context.Context, query backtest.Query) ([]backtest.Comparison, error)
	Get(ctx context.Context, id string) (backtest.Comparison, error)
}

type activationStore interface {
	Save(ctx context.Context, record activation.Record) (activation.Record, error)
	List(ctx context.Context, query activation.Query) ([]activation.Record, error)
	SaveLifecycle(ctx context.Context, record activation.LifecycleRecord) (activation.LifecycleRecord, error)
	ListLifecycles(ctx context.Context, query activation.Query) ([]activation.LifecycleRecord, error)
	AdvanceLifecycle(ctx context.Context, id string, state activation.LifecycleState, reason string, actor string) (activation.LifecycleRecord, error)
}

type ReconciliationStore interface {
	List(ctx context.Context, query reconciliation.Query) ([]reconciliation.Run, error)
	Get(ctx context.Context, id string) (reconciliation.Run, error)
}

type ReconciliationRunner interface {
	Run(ctx context.Context) (reconciliation.Run, error)
}

type RiskSettingsStore interface {
	Get(ctx context.Context) (risk.Settings, error)
	Save(ctx context.Context, settings risk.Settings) (risk.Settings, error)
}

type RiskDecisionReader interface {
	Latest(ctx context.Context, symbol string) (risk.Decision, error)
	List(ctx context.Context, query risk.DecisionQuery) ([]risk.Decision, error)
}

type PaperAccountReader interface {
	Get(ctx context.Context) (execution.Account, error)
	Save(ctx context.Context, account execution.Account) error
}

type PaperCycleRunner interface {
	RunOnce(ctx context.Context, request orchestration.ManualRunRequest) (orchestration.ManualRunResult, error)
}

type SafetyStore interface {
	Get(ctx context.Context) (safety.Status, error)
	Save(ctx context.Context, status safety.Status) (safety.Status, error)
}

type AuditStore interface {
	Save(ctx context.Context, event audit.Event) (string, error)
	List(ctx context.Context, query audit.Query) ([]audit.Event, error)
}

type ReportStore interface {
	DailyPnL(ctx context.Context, query execution.Query) ([]execution.DailyPnL, error)
	TradeCounts(ctx context.Context, query execution.Query) ([]execution.DailyPnL, error)
	RejectedReasons(ctx context.Context, query risk.DecisionQuery) ([]risk.RejectionSummary, error)
}

type OrderReader interface {
	ListOrders(ctx context.Context, query execution.Query) ([]execution.Order, error)
}

type TradeReader interface {
	ListTrades(ctx context.Context, query execution.Query) ([]execution.Trade, error)
}

type MetricsReader interface {
	Snapshot() observability.Snapshot
	IncHTTPRequests()
}

type PipelineRunReader interface {
	ListRuns(ctx context.Context, query orchestration.RunQuery) ([]orchestration.Run, error)
}

type StreamStats = observability.StreamStats

type StreamStatsReader interface {
	Stats(ctx context.Context) ([]StreamStats, error)
}

const maxBacktestRequestCandles = 200000

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	if deps.Dashboard != nil {
		mux.HandleFunc("GET /", dashboardRedirectHandler)
		mux.Handle("GET /dashboard/", http.StripPrefix("/dashboard/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if path == "" || path == "/" || strings.HasSuffix(path, "/") || strings.HasSuffix(path, "index.html") {
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
			}
			http.FileServer(deps.Dashboard).ServeHTTP(w, r)
		})))
	}
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /ready", readinessHandler(deps))
	mux.HandleFunc("GET /api/v1/market/candles", candlesHandler(deps))
	mux.HandleFunc("GET /api/v1/market/candles/coverage", candleCoverageHandler(deps))
	mux.HandleFunc("POST /api/v1/market/backfills", createMarketBackfillHandler(deps))
	mux.HandleFunc("GET /api/v1/market/backfills", listMarketBackfillsHandler(deps))
	mux.HandleFunc("GET /api/v1/market/backfills/{id}", getMarketBackfillHandler(deps))
	mux.HandleFunc("POST /api/v1/market/aggregations", createMarketAggregationHandler(deps))
	mux.HandleFunc("GET /api/v1/dashboard/summary", dashboardSummaryHandler(deps))
	mux.HandleFunc("GET /api/v1/signals", signalsHandler(deps))
	mux.HandleFunc("GET /api/v1/strategy/settings", strategySettingsHandler(deps))
	mux.HandleFunc("PUT /api/v1/strategy/settings", updateStrategySettingsHandler(deps))
	mux.HandleFunc("POST /api/v1/strategy/evaluate", evaluateStrategyHandler(deps))
	mux.HandleFunc("POST /api/v1/backtests", createBacktestHandler(deps))
	mux.HandleFunc("POST /api/v1/backtests/optimizations", createBacktestOptimizationHandler(deps))
	mux.HandleFunc("GET /api/v1/backtests", listBacktestsHandler(deps))
	mux.HandleFunc("GET /api/v1/backtests/{id}", getBacktestHandler(deps))
	mux.HandleFunc("POST /api/v1/strategy/comparisons", createStrategyComparisonHandler(deps))
	mux.HandleFunc("GET /api/v1/strategy/comparisons", listStrategyComparisonsHandler(deps))
	mux.HandleFunc("GET /api/v1/strategy/comparisons/{id}", getStrategyComparisonHandler(deps))
	mux.HandleFunc("POST /api/v1/strategy/comparisons/{id}/activate", activateStrategyComparisonHandler(deps))
	mux.HandleFunc("GET /api/v1/strategy/activations", listStrategyActivationsHandler(deps))
	mux.HandleFunc("GET /api/v1/strategy/lifecycle", strategyLifecycleHandler(deps))
	mux.HandleFunc("POST /api/v1/strategy/lifecycle/{id}/advance", advanceStrategyLifecycleHandler(deps))
	mux.HandleFunc("GET /api/v1/risk/settings", riskSettingsHandler(deps))
	mux.HandleFunc("PUT /api/v1/risk/settings", updateRiskSettingsHandler(deps))
	mux.HandleFunc("GET /api/v1/safety/status", safetyStatusHandler(deps))
	mux.HandleFunc("PUT /api/v1/safety/status", updateSafetyStatusHandler(deps))
	mux.HandleFunc("GET /api/v1/audit/events", auditEventsHandler(deps))
	mux.HandleFunc("GET /api/v1/reports/paper/daily-pnl", dailyPnLReportHandler(deps))
	mux.HandleFunc("GET /api/v1/reports/paper/trade-counts", tradeCountsReportHandler(deps))
	mux.HandleFunc("GET /api/v1/reports/risk/rejections", riskRejectionsReportHandler(deps))
	mux.HandleFunc("GET /api/v1/risk-decisions", riskDecisionsHandler(deps))
	mux.HandleFunc("GET /api/v1/paper/account", paperAccountHandler(deps))
	mux.HandleFunc("POST /api/v1/paper/account/reset", resetPaperAccountHandler(deps))
	mux.HandleFunc("POST /api/v1/paper/cycles", manualPaperCycleHandler(deps))
	mux.HandleFunc("GET /api/v1/paper/orders", paperOrdersHandler(deps))
	mux.HandleFunc("GET /api/v1/paper/trades", paperTradesHandler(deps))
	mux.HandleFunc("GET /api/v1/execution/status", executionStatusHandler(deps))
	mux.HandleFunc("GET /metrics", metricsHandler(deps))
	mux.HandleFunc("GET /api/v1/ops/pipeline-runs", pipelineRunsHandler(deps))
	mux.HandleFunc("GET /api/v1/reconciliation/runs", reconciliationRunsHandler(deps))
	mux.HandleFunc("POST /api/v1/reconciliation/runs", createReconciliationRunHandler(deps))
	mux.HandleFunc("GET /api/v1/reconciliation/runs/{id}", reconciliationRunHandler(deps))
	mux.HandleFunc("GET /api/v1/ops/streams", streamStatsHandler(deps))
	mux.HandleFunc("GET /api/v1/strategies/templates", strategyTemplatesHandler())
	mux.HandleFunc("GET /api/v1/strategies/templates/{id}", strategyTemplateDetailHandler())
	mux.HandleFunc("POST /api/v1/strategies/pine/validate", pine.ValidateHandler())
	mux.HandleFunc("POST /api/v1/strategies/pine", pine.SaveHandler(deps.PineRepository))
	mux.HandleFunc("GET /api/v1/strategies/pine", pine.ListHandler(deps.PineRepository))
	mux.HandleFunc("GET /api/v1/strategies/pine/{id}", pine.GetHandler(deps.PineRepository))

	handler := http.Handler(mux)
	handler = APIKeyAuthMiddleware(deps.Auth, handler)
	handler = RateLimitMiddleware(deps.Security, handler)
	handler = MaxBodyMiddleware(deps.Security, handler)
	handler = metricsMiddleware(deps.Metrics, handler)
	handler = CorrelationIDMiddleware(handler)
	handler = SecurityHeadersMiddleware(deps.Security, handler)
	handler = RecoverMiddleware(handler)
	return handler
}

func decodeStrictJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain a single JSON object")
	}
	return nil
}

func validateBacktestCandleLimit(candleLimit int) error {
	if candleLimit > maxBacktestRequestCandles {
		return fmt.Errorf("backtest range is too large: requested_candles=%d max_candles=%d", candleLimit, maxBacktestRequestCandles)
	}
	return nil
}

func strategyTemplatesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"data": strategy.PredefinedTemplates()})
	}
}

func strategyTemplateDetailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.PathValue("id"))
		tmpl, ok := strategy.GetPredefinedTemplate(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "strategy template not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": tmpl})
	}
}

func strategySettingsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.StrategySettings == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy settings repository unavailable"})
			return
		}
		settings, err := deps.StrategySettings.Get(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query strategy settings"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": settings})
	}
}

func updateStrategySettingsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.StrategySettings == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy settings repository unavailable"})
			return
		}
		var settings strategy.Settings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		settings = settings.Normalized()
		if err := settings.Validate(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		saved, err := deps.StrategySettings.Save(r.Context(), settings)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save strategy settings"})
			return
		}
		recordAudit(r.Context(), deps.Audit, "strategy.settings_changed", "operator", "strategy settings changed", saved)
		writeJSON(w, http.StatusOK, map[string]any{"data": saved})
	}
}

func evaluateStrategyHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.StrategySettings == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy settings repository unavailable"})
			return
		}
		if deps.Candles == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "candles repository unavailable"})
			return
		}
		if deps.SignalStore == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "signal store unavailable"})
			return
		}
		settings, err := deps.StrategySettings.Get(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query strategy settings"})
			return
		}
		if err := settings.Validate(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stored strategy settings are invalid"})
			return
		}
		candles, err := deps.Candles.List(r.Context(), marketdata.CandleQuery{
			Symbol:   settings.Symbol,
			Interval: settings.Interval,
			Limit:    settings.LookbackLimit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query strategy candles"})
			return
		}
		evaluator, err := strategy.NewEvaluatorFromSettings(settings)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stored strategy settings are invalid"})
			return
		}
		signal, err := evaluator.Evaluate(candles)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		id, err := deps.SignalStore.Save(r.Context(), signal)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save strategy signal"})
			return
		}
		signal.ID = id
		writeJSON(w, http.StatusOK, map[string]any{"data": signal})
	}
}

func createBacktestHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Backtests == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "backtest repository unavailable"})
			return
		}
		if deps.Candles == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "candles repository unavailable"})
			return
		}
		var request backtest.Request
		if err := decodeStrictJSON(r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		request = request.Normalize()
		if err := request.Validate(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		candleLimit, err := marketdata.ExpectedCandleCount(request.From, request.To, request.Interval)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := validateBacktestCandleLimit(candleLimit); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":              "backtest range is too large",
				"requested_candles":  candleLimit,
				"max_candles":        maxBacktestRequestCandles,
				"requested_interval": request.Interval,
			})
			return
		}
		candles, err := deps.Candles.List(r.Context(), marketdata.CandleQuery{
			Symbol:   request.Symbol,
			Interval: request.Interval,
			From:     request.From,
			To:       request.To,
			Limit:    candleLimit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query backtest candles"})
			return
		}
		requiredCandles := request.RequiredCandles()
		if len(candles) < requiredCandles {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":             "not enough candles for selected backtest range",
				"required_candles":  requiredCandles,
				"available_candles": len(candles),
			})
			return
		}
		run, trades, err := backtest.NewEngine().Run(request, candles)
		if err != nil {
			var candleErr backtest.CandleSeriesError
			if errors.As(err, &candleErr) {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"error":              candleErr.Error(),
					"candle_diagnostics": candleErr.Diagnostics,
				})
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		saveEquityCurve := request.SaveEquityCurve == nil || *request.SaveEquityCurve
		var saved backtest.Run
		if store, ok := deps.Backtests.(backtestStoreWithOptions); ok {
			saved, err = store.SaveWithOptions(r.Context(), run, trades, backtest.SaveOptions{SaveEquityCurve: saveEquityCurve})
		} else {
			saved, err = deps.Backtests.Save(r.Context(), run, trades)
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save backtest"})
			return
		}
		recordAudit(r.Context(), deps.Audit, "backtest.created", "operator", "backtest created", saved)
		writeJSON(w, http.StatusCreated, map[string]any{"data": saved, "trades": trades})
	}
}

func createBacktestOptimizationHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Candles == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "candles repository unavailable"})
			return
		}
		var request backtest.OptimizationRequest
		if err := decodeStrictJSON(r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		request = request.Normalize()
		if err := request.Validate(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		candleLimit, err := marketdata.ExpectedCandleCount(request.From, request.To, request.Interval)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := validateBacktestCandleLimit(candleLimit); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":              "backtest range is too large",
				"requested_candles":  candleLimit,
				"max_candles":        maxBacktestRequestCandles,
				"requested_interval": request.Interval,
			})
			return
		}
		candles, err := deps.Candles.List(r.Context(), marketdata.CandleQuery{
			Symbol:   request.Symbol,
			Interval: request.Interval,
			From:     request.From,
			To:       request.To,
			Limit:    candleLimit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query optimization candles"})
			return
		}
		requiredCandles := request.RequiredCandles()
		if len(candles) < requiredCandles {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":             "not enough candles for selected optimization range",
				"required_candles":  requiredCandles,
				"available_candles": len(candles),
			})
			return
		}
		result, err := backtest.NewOptimizer(backtest.NewEngine()).Optimize(request, candles)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		recordAudit(r.Context(), deps.Audit, "backtest.optimization_created", "operator", "backtest optimization created", result)
		writeJSON(w, http.StatusCreated, map[string]any{"data": result})
	}
}

func candleCoverageHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Candles == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "candles repository unavailable"})
			return
		}
		query, err := parseCandleCoverageQuery(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		coverage, err := deps.Candles.Coverage(r.Context(), query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query candle coverage"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": coverage})
	}
}

type createBackfillRequest struct {
	Symbol       string    `json:"symbol"`
	BaseInterval string    `json:"base_interval"`
	From         time.Time `json:"from"`
	To           time.Time `json:"to"`
}

func createMarketBackfillHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Backfills == nil || deps.BackfillRunner == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "backfill service unavailable"})
			return
		}
		var request createBackfillRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if request.Symbol == "" {
			request.Symbol = "BTCUSDT"
		}
		if request.BaseInterval == "" {
			request.BaseInterval = "1m"
		}
		if request.To.IsZero() {
			request.To = time.Now().UTC()
		}
		if request.From.IsZero() {
			request.From = request.To.AddDate(-2, 0, 0)
		}
		if _, err := marketdata.ExpectedCandleCount(request.From, request.To, request.BaseInterval); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		now := time.Now().UTC()
		job, err := deps.Backfills.Create(r.Context(), marketdata.BackfillJob{
			Symbol:       strings.ToUpper(strings.TrimSpace(request.Symbol)),
			BaseInterval: strings.TrimSpace(request.BaseInterval),
			From:         request.From.UTC(),
			To:           request.To.UTC(),
			NextOpenTime: request.From.UTC(),
			Status:       marketdata.BackfillStatusPending,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create backfill job"})
			return
		}
		go func(jobID string) {
			_, _ = deps.BackfillRunner.Resume(context.Background(), jobID)
		}(job.ID)
		writeJSON(w, http.StatusAccepted, map[string]any{"data": job})
	}
}

func listMarketBackfillsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Backfills == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "backfill repository unavailable"})
			return
		}
		limit, err := parseLimit(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		jobs, err := deps.Backfills.List(r.Context(), marketdata.BackfillJobQuery{
			Symbol: r.URL.Query().Get("symbol"),
			Limit:  limit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query backfill jobs"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": jobs})
	}
}

func getMarketBackfillHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Backfills == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "backfill repository unavailable"})
			return
		}
		id := r.PathValue("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "backfill id is required"})
			return
		}
		job, err := deps.Backfills.Get(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query backfill job"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": job})
	}
}

func createMarketAggregationHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Aggregator == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "aggregation service unavailable"})
			return
		}
		var request marketdata.AggregationRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		result, err := deps.Aggregator.Aggregate(r.Context(), request)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"data": result})
	}
}

func listBacktestsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Backtests == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "backtest repository unavailable"})
			return
		}
		limit, err := parseLimit(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		runs, err := deps.Backtests.List(r.Context(), backtest.Query{
			Symbol: r.URL.Query().Get("symbol"),
			Limit:  limit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query backtests"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": runs})
	}
}

func getBacktestHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Backtests == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "backtest repository unavailable"})
			return
		}
		id := r.PathValue("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "backtest id is required"})
			return
		}
		run, trades, err := deps.Backtests.Get(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query backtest"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": run, "trades": trades})
	}
}

func createStrategyComparisonHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.StrategyComparisons == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy comparison repository unavailable"})
			return
		}
		if deps.Candles == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "candles repository unavailable"})
			return
		}
		var request backtest.ComparisonRequest
		if err := decodeStrictJSON(r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		request = request.Normalize()
		if err := request.Validate(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		candleLimit, err := marketdata.ExpectedCandleCount(request.From, request.To, request.Interval)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := validateBacktestCandleLimit(candleLimit); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":              "backtest range is too large",
				"requested_candles":  candleLimit,
				"max_candles":        maxBacktestRequestCandles,
				"requested_interval": request.Interval,
			})
			return
		}
		candles, err := deps.Candles.List(r.Context(), marketdata.CandleQuery{
			Symbol:   request.Symbol,
			Interval: request.Interval,
			From:     request.From,
			To:       request.To,
			Limit:    candleLimit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query comparison candles"})
			return
		}
		requiredCandles := request.RequiredCandles()
		if len(candles) < requiredCandles {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":             "not enough candles for selected comparison range",
				"required_candles":  requiredCandles,
				"available_candles": len(candles),
			})
			return
		}
		comparison, err := backtest.NewComparator(backtest.NewEngine()).Compare(request, candles)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		saved, err := deps.StrategyComparisons.Save(r.Context(), comparison)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save strategy comparison"})
			return
		}
		recordAudit(r.Context(), deps.Audit, "strategy.comparison_created", "operator", "strategy comparison created", saved)
		writeJSON(w, http.StatusCreated, map[string]any{"data": saved})
	}
}

func listStrategyComparisonsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.StrategyComparisons == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy comparison repository unavailable"})
			return
		}
		limit, err := parseLimit(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		comparisons, err := deps.StrategyComparisons.List(r.Context(), backtest.Query{
			Symbol: r.URL.Query().Get("symbol"),
			Limit:  limit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query strategy comparisons"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": comparisons})
	}
}

func getStrategyComparisonHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.StrategyComparisons == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy comparison repository unavailable"})
			return
		}
		id := r.PathValue("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "strategy comparison id is required"})
			return
		}
		comparison, err := deps.StrategyComparisons.Get(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query strategy comparison"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": comparison})
	}
}

func activateStrategyComparisonHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.StrategyComparisons == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy comparison repository unavailable"})
			return
		}
		if deps.StrategySettings == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy settings repository unavailable"})
			return
		}
		if deps.StrategyActivations == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy activation repository unavailable"})
			return
		}
		var request activation.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil && r.Body != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		request.ComparisonID = r.PathValue("id")
		service := activation.NewService(activation.Dependencies{
			Comparisons: deps.StrategyComparisons,
			Settings:    deps.StrategySettings,
			Activations: deps.StrategyActivations,
		})
		record, err := service.Activate(r.Context(), request)
		if err != nil {
			if errors.Is(err, activation.ErrActivationGate) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		recordAudit(r.Context(), deps.Audit, "strategy.activated", record.Actor, "strategy activated from comparison evidence", record)
		writeJSON(w, http.StatusOK, map[string]any{"data": record})
	}
}

func listStrategyActivationsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.StrategyActivations == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy activation repository unavailable"})
			return
		}
		limit, err := parseLimit(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		records, err := deps.StrategyActivations.List(r.Context(), activation.Query{
			StrategyName: r.URL.Query().Get("strategy_name"),
			Limit:        limit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query strategy activations"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": records})
	}
}

func strategyLifecycleHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.StrategyActivations == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy lifecycle repository unavailable"})
			return
		}
		limit, err := parseLimit(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		records, err := deps.StrategyActivations.ListLifecycles(r.Context(), activation.Query{
			StrategyName: r.URL.Query().Get("strategy_name"),
			Symbol:       r.URL.Query().Get("symbol"),
			Interval:     r.URL.Query().Get("interval"),
			Limit:        limit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query strategy lifecycle"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": records})
	}
}

type advanceLifecycleRequest struct {
	State     activation.LifecycleState `json:"state"`
	Reason    string                    `json:"reason"`
	UpdatedBy string                    `json:"updated_by"`
}

func advanceStrategyLifecycleHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.StrategyActivations == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "strategy lifecycle repository unavailable"})
			return
		}
		id := r.PathValue("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "strategy lifecycle id is required"})
			return
		}
		var request advanceLifecycleRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if request.State == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "state is required"})
			return
		}
		if request.UpdatedBy == "" {
			request.UpdatedBy = "operator"
		}
		record, err := deps.StrategyActivations.AdvanceLifecycle(r.Context(), id, request.State, request.Reason, request.UpdatedBy)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to advance strategy lifecycle"})
			return
		}
		recordAudit(r.Context(), deps.Audit, "strategy.lifecycle_advanced", request.UpdatedBy, "strategy lifecycle advanced", record)
		writeJSON(w, http.StatusOK, map[string]any{"data": record})
	}
}

func riskSettingsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.RiskSettings == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "risk settings repository unavailable"})
			return
		}
		settings, err := deps.RiskSettings.Get(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query risk settings"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": settings})
	}
}

func updateRiskSettingsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.RiskSettings == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "risk settings repository unavailable"})
			return
		}
		var settings risk.Settings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		settings = settings.Normalize()
		if err := settings.Validate(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		saved, err := deps.RiskSettings.Save(r.Context(), settings)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save risk settings"})
			return
		}
		recordAudit(r.Context(), deps.Audit, "risk.settings_changed", "operator", "risk settings changed", saved)
		writeJSON(w, http.StatusOK, map[string]any{"data": saved})
	}
}

func safetyStatusHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Safety == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "safety repository unavailable"})
			return
		}
		status, err := deps.Safety.Get(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query safety status"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": status})
	}
}

func updateSafetyStatusHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Safety == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "safety repository unavailable"})
			return
		}
		var status safety.Status
		if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		saved, err := deps.Safety.Save(r.Context(), status)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save safety status"})
			return
		}
		if deps.Audit != nil {
			details, _ := json.Marshal(saved)
			summary := "kill switch disabled"
			if saved.KillSwitchActive {
				summary = "kill switch enabled"
			}
			_, _ = deps.Audit.Save(r.Context(), audit.Event{
				EventType:   "safety.status_changed",
				Actor:       saved.UpdatedBy,
				Summary:     summary,
				DetailsJSON: string(details),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": saved})
	}
}

func auditEventsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Audit == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "audit repository unavailable"})
			return
		}
		query, err := parseAuditQuery(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		events, err := deps.Audit.List(r.Context(), query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query audit events"})
			return
		}
		if wantsCSV(r) {
			writeAuditCSV(w, events)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": events})
	}
}

func dailyPnLReportHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Reports == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports repository unavailable"})
			return
		}
		query, err := parseExecutionQuery(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		rows, err := deps.Reports.DailyPnL(r.Context(), query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query daily pnl report"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": rows})
	}
}

func tradeCountsReportHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Reports == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports repository unavailable"})
			return
		}
		query, err := parseExecutionQuery(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		rows, err := deps.Reports.TradeCounts(r.Context(), query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query trade count report"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": rows})
	}
}

func riskRejectionsReportHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Reports == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reports repository unavailable"})
			return
		}
		query, err := parseDecisionQuery(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		rows, err := deps.Reports.RejectedReasons(r.Context(), query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query risk rejection report"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": rows})
	}
}

func dashboardRedirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/dashboard/", http.StatusTemporaryRedirect)
}

func metricsMiddleware(metrics MetricsReader, next http.Handler) http.Handler {
	if metrics == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics.IncHTTPRequests()
		next.ServeHTTP(w, r)
	})
}

func dashboardSummaryHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := defaultString(r.URL.Query().Get("symbol"), "BTCUSDT")
		interval := defaultString(r.URL.Query().Get("interval"), "1m")
		summary := map[string]any{
			"symbol":   symbol,
			"interval": interval,
		}
		if deps.Candles != nil {
			candles, _ := deps.Candles.List(r.Context(), marketdata.CandleQuery{Symbol: symbol, Interval: interval, Limit: 1, Desc: true})
			if len(candles) > 0 {
				summary["latest_price"] = candles[0].Close
				summary["latest_candle_time"] = candles[0].OpenTime
			}
		}
		if deps.Signals != nil {
			if signal, err := deps.Signals.Latest(r.Context(), symbol); err == nil {
				summary["latest_signal"] = signal
			}
		}
		if deps.RiskDecisions != nil {
			if decision, err := deps.RiskDecisions.Latest(r.Context(), symbol); err == nil {
				summary["latest_risk_decision"] = decision
			}
		}
		if deps.PaperAccount != nil {
			if account, err := deps.PaperAccount.Get(r.Context()); err == nil {
				summary["paper_account"] = account
			}
		}
		if deps.Orders != nil {
			if orders, err := deps.Orders.ListOrders(r.Context(), execution.Query{Symbol: symbol, Limit: 1}); err == nil && len(orders) > 0 {
				summary["latest_paper_order"] = orders[0]
			}
		}
		if deps.Trades != nil {
			if trades, err := deps.Trades.ListTrades(r.Context(), execution.Query{Symbol: symbol, Limit: 1}); err == nil && len(trades) > 0 {
				summary["latest_paper_trade"] = trades[0]
			}
		}
		writeJSON(w, http.StatusOK, summary)
	}
}

func signalsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Signals == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "signals repository unavailable"})
			return
		}
		query, err := parseSignalQuery(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		signals, err := deps.Signals.List(r.Context(), query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query signals"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": signals})
	}
}

func riskDecisionsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.RiskDecisions == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "risk decisions repository unavailable"})
			return
		}
		query, err := parseDecisionQuery(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		decisions, err := deps.RiskDecisions.List(r.Context(), query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query risk decisions"})
			return
		}
		if wantsCSV(r) {
			writeRiskDecisionsCSV(w, decisions)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": decisions})
	}
}

func paperAccountHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.PaperAccount == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "paper account repository unavailable"})
			return
		}
		account, err := deps.PaperAccount.Get(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query paper account"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": account})
	}
}

func resetPaperAccountHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.PaperAccount == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "paper account repository unavailable"})
			return
		}
		var account execution.Account
		if err := json.NewDecoder(r.Body).Decode(&account); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		account.BaseAsset = strings.ToUpper(strings.TrimSpace(account.BaseAsset))
		account.QuoteAsset = strings.ToUpper(strings.TrimSpace(account.QuoteAsset))
		if account.BaseAsset == "" || account.QuoteAsset == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "base_asset and quote_asset are required"})
			return
		}
		if account.BaseBalance < 0 || account.QuoteBalance < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "balances cannot be negative"})
			return
		}
		account.UpdatedAt = time.Now().UTC()
		if err := deps.PaperAccount.Save(r.Context(), account); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to reset paper account"})
			return
		}
		recordAudit(r.Context(), deps.Audit, "paper.account_reset", "operator", "paper account reset", account)
		writeJSON(w, http.StatusOK, map[string]any{"data": account})
	}
}

func manualPaperCycleHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.PaperCycleRunner == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "paper cycle runner unavailable"})
			return
		}
		var request orchestration.ManualRunRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		request = request.Normalize()
		result, err := deps.PaperCycleRunner.RunOnce(r.Context(), request)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		recordAudit(r.Context(), deps.Audit, "paper.manual_cycle_run", "operator", "manual paper cycle run", result)
		writeJSON(w, http.StatusCreated, map[string]any{"data": result})
	}
}

func paperOrdersHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Orders == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "paper orders repository unavailable"})
			return
		}
		query, err := parseExecutionQuery(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		orders, err := deps.Orders.ListOrders(r.Context(), query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query paper orders"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": orders})
	}
}

func paperTradesHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Trades == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "paper trades repository unavailable"})
			return
		}
		query, err := parseExecutionQuery(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		trades, err := deps.Trades.ListTrades(r.Context(), query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query paper trades"})
			return
		}
		if wantsCSV(r) {
			writeTradesCSV(w, trades)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": trades})
	}
}

func executionStatusHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := deps.ExecutionStatus
		if status.Mode == "" {
			status = execution.Status{
				Mode:               "paper",
				PaperEnabled:       false,
				ExchangeAdapter:    "binance_disabled",
				LiveTradingEnabled: false,
				RetryAttempts:      1,
				Timeout:            "0s",
				LastError:          "binance live trading is disabled",
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": status})
	}
}

func metricsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Metrics == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "metrics unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, deps.Metrics.Snapshot())
	}
}

func pipelineRunsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.PipelineRuns == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "pipeline runs repository unavailable"})
			return
		}
		limit, err := parseLimit(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		runs, err := deps.PipelineRuns.ListRuns(r.Context(), orchestration.RunQuery{
			Status: r.URL.Query().Get("status"),
			Limit:  limit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query pipeline runs"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": runs})
	}
}

func reconciliationRunsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.ReconciliationRuns == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reconciliation repository unavailable"})
			return
		}
		limit, err := parseLimit(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		runs, err := deps.ReconciliationRuns.List(r.Context(), reconciliation.Query{Limit: limit})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query reconciliation runs"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": runs})
	}
}

func createReconciliationRunHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.ReconciliationRunner == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reconciliation runner unavailable"})
			return
		}
		run, err := deps.ReconciliationRunner.Run(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to run reconciliation"})
			return
		}
		if run.Status == reconciliation.StatusMismatch {
			if hasCriticalReconciliationMismatch(run) {
				recordAudit(r.Context(), deps.Audit, "reconciliation.critical_mismatch", "system", "critical reconciliation mismatch detected", run)
				armKillSwitchForReconciliation(r.Context(), deps, run)
			} else {
				recordAudit(r.Context(), deps.Audit, "reconciliation.mismatch", "system", "reconciliation mismatch detected", run)
			}
		} else {
			recordAudit(r.Context(), deps.Audit, "reconciliation.matched", "system", "reconciliation matched", run)
		}
		writeJSON(w, http.StatusCreated, map[string]any{"data": run})
	}
}

func hasCriticalReconciliationMismatch(run reconciliation.Run) bool {
	for _, mismatch := range run.Mismatches {
		if strings.EqualFold(mismatch.Severity, "critical") {
			return true
		}
	}
	return false
}

func armKillSwitchForReconciliation(ctx context.Context, deps Dependencies, run reconciliation.Run) {
	if deps.Safety == nil {
		return
	}
	status := safety.Status{
		KillSwitchActive: true,
		Reason:           "critical reconciliation mismatch detected",
		UpdatedBy:        "reconciliation",
		UpdatedAt:        time.Now().UTC(),
	}
	saved, err := deps.Safety.Save(ctx, status)
	if err != nil {
		recordAudit(ctx, deps.Audit, "reconciliation.kill_switch_failed", "system", "failed to arm kill switch after critical reconciliation mismatch", map[string]any{
			"run":   run,
			"error": err.Error(),
		})
		return
	}
	recordAudit(ctx, deps.Audit, "safety.status_changed", "reconciliation", "kill switch enabled by critical reconciliation mismatch", saved)
}

func reconciliationRunHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.ReconciliationRuns == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reconciliation repository unavailable"})
			return
		}
		id := r.PathValue("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reconciliation run id is required"})
			return
		}
		run, err := deps.ReconciliationRuns.Get(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query reconciliation run"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": run})
	}
}

func streamStatsHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Streams == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "stream stats unavailable"})
			return
		}
		stats, err := deps.Streams.Stats(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query stream stats"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": stats})
	}
}

func candlesHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Candles == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "candles repository unavailable"})
			return
		}

		query, err := parseCandleQuery(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		candles, err := deps.Candles.List(r.Context(), query)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query candles"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": candles})
	}
}

func parseCandleQuery(r *http.Request) (marketdata.CandleQuery, error) {
	values := r.URL.Query()
	limit := 500
	if rawLimit := values.Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			return marketdata.CandleQuery{}, errInvalid("limit must be a positive integer")
		}
		if parsed > 1000 {
			parsed = 1000
		}
		limit = parsed
	}

	from, err := parseOptionalTime(values.Get("from"))
	if err != nil {
		return marketdata.CandleQuery{}, errInvalid("from must be RFC3339")
	}
	to, err := parseOptionalTime(values.Get("to"))
	if err != nil {
		return marketdata.CandleQuery{}, errInvalid("to must be RFC3339")
	}

	symbol := values.Get("symbol")
	if symbol == "" {
		return marketdata.CandleQuery{}, errInvalid("symbol is required")
	}
	interval := values.Get("interval")
	if interval == "" {
		return marketdata.CandleQuery{}, errInvalid("interval is required")
	}

	return marketdata.CandleQuery{
		Symbol:   symbol,
		Interval: interval,
		From:     from,
		To:       to,
		Limit:    limit,
	}, nil
}

func parseCandleCoverageQuery(r *http.Request) (marketdata.CandleQuery, error) {
	values := r.URL.Query()
	from, err := parseOptionalTime(values.Get("from"))
	if err != nil {
		return marketdata.CandleQuery{}, errInvalid("from must be RFC3339")
	}
	to, err := parseOptionalTime(values.Get("to"))
	if err != nil {
		return marketdata.CandleQuery{}, errInvalid("to must be RFC3339")
	}
	symbol := values.Get("symbol")
	if symbol == "" {
		return marketdata.CandleQuery{}, errInvalid("symbol is required")
	}
	interval := values.Get("interval")
	if interval == "" {
		return marketdata.CandleQuery{}, errInvalid("interval is required")
	}
	return marketdata.CandleQuery{Symbol: symbol, Interval: interval, From: from, To: to}, nil
}

func parseSignalQuery(r *http.Request) (strategy.SignalQuery, error) {
	limit, err := parseLimit(r)
	if err != nil {
		return strategy.SignalQuery{}, err
	}
	from, to, err := parseTimeRange(r)
	if err != nil {
		return strategy.SignalQuery{}, err
	}
	return strategy.SignalQuery{Symbol: r.URL.Query().Get("symbol"), From: from, To: to, Limit: limit}, nil
}

func parseDecisionQuery(r *http.Request) (risk.DecisionQuery, error) {
	limit, err := parseLimit(r)
	if err != nil {
		return risk.DecisionQuery{}, err
	}
	from, to, err := parseTimeRange(r)
	if err != nil {
		return risk.DecisionQuery{}, err
	}
	return risk.DecisionQuery{Symbol: r.URL.Query().Get("symbol"), From: from, To: to, Limit: limit}, nil
}

func parseExecutionQuery(r *http.Request) (execution.Query, error) {
	limit, err := parseLimit(r)
	if err != nil {
		return execution.Query{}, err
	}
	from, to, err := parseTimeRange(r)
	if err != nil {
		return execution.Query{}, err
	}
	return execution.Query{Symbol: r.URL.Query().Get("symbol"), From: from, To: to, Limit: limit}, nil
}

func parseAuditQuery(r *http.Request) (audit.Query, error) {
	limit, err := parseLimit(r)
	if err != nil {
		return audit.Query{}, err
	}
	from, to, err := parseTimeRange(r)
	if err != nil {
		return audit.Query{}, err
	}
	return audit.Query{
		EventType: r.URL.Query().Get("event_type"),
		Actor:     r.URL.Query().Get("actor"),
		From:      from,
		To:        to,
		Limit:     limit,
	}, nil
}

func parseLimit(r *http.Request) (int, error) {
	limit := 100
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			return 0, errInvalid("limit must be a positive integer")
		}
		if parsed > 1000 {
			parsed = 1000
		}
		limit = parsed
	}
	return limit, nil
}

func parseTimeRange(r *http.Request) (time.Time, time.Time, error) {
	from, err := parseOptionalTime(r.URL.Query().Get("from"))
	if err != nil {
		return time.Time{}, time.Time{}, errInvalid("from must be RFC3339")
	}
	to, err := parseOptionalTime(r.URL.Query().Get("to"))
	if err != nil {
		return time.Time{}, time.Time{}, errInvalid("to must be RFC3339")
	}
	return from, to, nil
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func wantsCSV(r *http.Request) bool {
	return strings.EqualFold(r.URL.Query().Get("format"), "csv")
}

func writeAuditCSV(w http.ResponseWriter, events []audit.Event) {
	writeCSV(w, []string{"id", "event_type", "actor", "summary", "created_at"}, func(writer *csv.Writer) {
		for _, event := range events {
			_ = writer.Write([]string{event.ID, event.EventType, event.Actor, event.Summary, event.CreatedAt.Format(time.RFC3339)})
		}
	})
}

func writeTradesCSV(w http.ResponseWriter, trades []execution.Trade) {
	writeCSV(w, []string{"id", "order_id", "symbol", "side", "quantity", "price", "fee", "created_at"}, func(writer *csv.Writer) {
		for _, trade := range trades {
			_ = writer.Write([]string{
				trade.ID,
				trade.OrderID,
				trade.Symbol,
				string(trade.Side),
				strconv.FormatFloat(trade.Quantity, 'f', -1, 64),
				strconv.FormatFloat(trade.Price, 'f', -1, 64),
				strconv.FormatFloat(trade.Fee, 'f', -1, 64),
				trade.CreatedAt.Format(time.RFC3339),
			})
		}
	})
}

func writeRiskDecisionsCSV(w http.ResponseWriter, decisions []risk.Decision) {
	writeCSV(w, []string{"id", "signal_id", "symbol", "signal_side", "decision", "reason", "evaluated_at"}, func(writer *csv.Writer) {
		for _, decision := range decisions {
			_ = writer.Write([]string{
				decision.ID,
				decision.SignalID,
				decision.Symbol,
				string(decision.SignalSide),
				string(decision.Decision),
				decision.Reason,
				decision.EvaluatedAt.Format(time.RFC3339),
			})
		}
	})
}

func writeCSV(w http.ResponseWriter, header []string, writeRows func(*csv.Writer)) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	writer := csv.NewWriter(w)
	_ = writer.Write(header)
	writeRows(writer)
	writer.Flush()
}

func recordAudit(ctx context.Context, store AuditStore, eventType string, actor string, summary string, details any) {
	if store == nil {
		return
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		encoded = []byte("{}")
	}
	_, _ = store.Save(ctx, audit.Event{
		EventType:   eventType,
		Actor:       actor,
		Summary:     summary,
		DetailsJSON: string(encoded),
	})
}

type errInvalid string

func (e errInvalid) Error() string {
	return string(e)
}
