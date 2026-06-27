package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"sentra/internal/activation"
	"sentra/internal/audit"
	"sentra/internal/backtest"
	"sentra/internal/execution"
	"sentra/internal/marketdata"
	"sentra/internal/observability"
	"sentra/internal/orchestration"
	"sentra/internal/reconciliation"
	"sentra/internal/risk"
	"sentra/internal/safety"
	"sentra/internal/strategy"
)

func TestCorrelationIDMiddlewareReusesRequestID(t *testing.T) {
	handler := CorrelationIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := CorrelationIDFromContext(r.Context())
		if got != "request-123" {
			t.Fatalf("expected request-123, got %q", got)
		}
	}))

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Request-ID", "request-123")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Header().Get("X-Request-ID") != "request-123" {
		t.Fatalf("expected response request ID, got %q", response.Header().Get("X-Request-ID"))
	}
}

func TestSecurityHeadersAreApplied(t *testing.T) {
	router := NewRouter(Dependencies{})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("expected nosniff header")
	}
	if response.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("expected frame deny header")
	}
	if response.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("expected no-referrer header")
	}
	if response.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("expected CSP header")
	}
}

func TestSecurityHeadersAllowDashboardAssetsAndAPIRequests(t *testing.T) {
	router := NewRouter(Dependencies{})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	csp := response.Header().Get("Content-Security-Policy")
	for _, directive := range []string{"default-src 'self'", "script-src 'self'", "style-src 'self'", "connect-src 'self'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, directive) {
			t.Fatalf("expected CSP directive %q in %q", directive, csp)
		}
	}
}

func TestRecoverMiddlewareReturnsJSONError(t *testing.T) {
	handler := RecoverMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("sensitive panic detail")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "internal server error") {
		t.Fatalf("expected generic error, got %q", response.Body.String())
	}
	if strings.Contains(response.Body.String(), "sensitive panic detail") {
		t.Fatal("panic detail leaked in response")
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected JSON response, got %q", response.Header().Get("Content-Type"))
	}
}

func TestRequestBodyLimitRejectsLargeBody(t *testing.T) {
	router := NewRouter(Dependencies{
		Security: SecurityConfig{
			MaxRequestBodyBytes: 4,
		},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/anything", strings.NewReader("12345"))
	router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", response.Code)
	}
}

func TestRateLimitRejectsProtectedRouteAfterLimit(t *testing.T) {
	router := NewRouter(Dependencies{
		Auth: AuthConfig{
			Enabled:     true,
			AdminAPIKey: "secret",
		},
		Security: SecurityConfig{
			RateLimitRequestsPerMinute: 1,
		},
	})

	for index := 0; index < 2; index++ {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
		request.Header.Set("X-API-Key", "secret")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if index == 1 && response.Code != http.StatusTooManyRequests {
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("expected 429, got %d body %s", response.Code, string(body))
		}
		if index == 1 && response.Header().Get("Retry-After") == "" {
			t.Fatalf("expected Retry-After header on 429 response")
		}
	}
}

func TestAPIKeyAuthAllowsPublicHealthRoutes(t *testing.T) {
	router := NewRouter(Dependencies{
		Auth: AuthConfig{
			Enabled:     true,
			AdminAPIKey: "secret",
		},
		Postgres: fakePinger{},
		Redis:    fakePinger{},
	})

	for _, path := range []string{"/health", "/ready", "/metrics"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code == http.StatusUnauthorized {
			t.Fatalf("expected %s to be public, got unauthorized", path)
		}
	}
}

func TestAPIKeyAuthRejectsProtectedRoutesWithoutKey(t *testing.T) {
	router := NewRouter(Dependencies{
		Auth: AuthConfig{
			Enabled:     true,
			AdminAPIKey: "secret",
		},
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", response.Code)
	}
}

func TestAPIKeyAuthAllowsProtectedRoutesWithValidKey(t *testing.T) {
	router := NewRouter(Dependencies{
		Auth: AuthConfig{
			Enabled:     true,
			AdminAPIKey: "secret",
		},
		Candles: &fakeCandleReader{},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/market/candles?symbol=BTCUSDT&interval=1m", nil)
	request.Header.Set("X-API-Key", "secret")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code == http.StatusUnauthorized {
		t.Fatal("expected valid API key to pass auth")
	}
}

func TestHealthAndReadinessRoutes(t *testing.T) {
	router := NewRouter(Dependencies{
		Postgres: fakePinger{},
		Redis:    fakePinger{},
	})

	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("expected health 200, got %d", health.Code)
	}

	ready := httptest.NewRecorder()
	router.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if ready.Code != http.StatusOK {
		t.Fatalf("expected ready 200, got %d", ready.Code)
	}
}

func TestDashboardRoutesServeStaticUI(t *testing.T) {
	dashboardFS, err := fs.Sub(fstest.MapFS{
		"dashboard/index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Sentra Dashboard</title>")},
		"dashboard/app.js":     &fstest.MapFile{Data: []byte("console.log('dashboard')")},
	}, "dashboard")
	if err != nil {
		t.Fatalf("failed to create dashboard fs: %v", err)
	}
	router := NewRouter(Dependencies{Dashboard: http.FS(dashboardFS)})

	root := httptest.NewRecorder()
	router.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	if root.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected root redirect, got %d", root.Code)
	}
	if root.Header().Get("Location") != "/dashboard/" {
		t.Fatalf("expected dashboard redirect, got %q", root.Header().Get("Location"))
	}

	index := httptest.NewRecorder()
	router.ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/dashboard/", nil))
	if index.Code != http.StatusOK {
		t.Fatalf("expected dashboard index 200, got %d", index.Code)
	}
	if !strings.Contains(index.Body.String(), "Sentra Dashboard") {
		t.Fatalf("expected dashboard HTML, got %q", index.Body.String())
	}

	asset := httptest.NewRecorder()
	router.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/dashboard/app.js", nil))
	if asset.Code != http.StatusOK {
		t.Fatalf("expected dashboard asset 200, got %d", asset.Code)
	}
}

func TestCandlesRouteReturnsRepositoryCandles(t *testing.T) {
	store := &fakeCandleReader{
		candles: []marketdata.Candle{
			{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
				OpenTime: time.Unix(10, 0).UTC(),
				Close:    "100.00",
			},
		},
	}
	router := NewRouter(Dependencies{
		Postgres: fakePinger{},
		Redis:    fakePinger{},
		Candles:  store,
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/market/candles?symbol=BTCUSDT&interval=1m&from=1970-01-01T00:00:10Z&to=1970-01-01T00:01:10Z&limit=50", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if store.query.Symbol != "BTCUSDT" || store.query.Interval != "1m" || store.query.Limit != 50 {
		t.Fatalf("unexpected candle query: %+v", store.query)
	}

	var body struct {
		Data []marketdata.Candle `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].Close != "100.00" {
		t.Fatalf("unexpected response body: %+v", body)
	}
}

func TestCandleCoverageRouteReturnsRangeAndCount(t *testing.T) {
	from := time.Unix(10, 0).UTC()
	to := time.Unix(20, 0).UTC()
	store := &fakeCandleReader{
		coverage: marketdata.Coverage{
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			Count:     5,
			FirstTime: &from,
			LastTime:  &to,
		},
	}
	router := NewRouter(Dependencies{Candles: store})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/market/candles/coverage?symbol=BTCUSDT&interval=1m&from=1970-01-01T00:00:10Z&to=1970-01-01T00:00:20Z", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if store.query.Symbol != "BTCUSDT" || store.query.Interval != "1m" {
		t.Fatalf("unexpected coverage query: %+v", store.query)
	}
	var body struct {
		Data marketdata.Coverage `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode coverage: %v", err)
	}
	if body.Data.Count != 5 {
		t.Fatalf("expected count 5, got %+v", body.Data)
	}
}

func TestDashboardSummaryRouteReturnsOperationalSnapshot(t *testing.T) {
	router := NewRouter(Dependencies{
		Postgres:      fakePinger{},
		Redis:         fakePinger{},
		Candles:       &fakeCandleReader{candles: []marketdata.Candle{{Symbol: "BTCUSDT", Interval: "1m", Close: "100.00", OpenTime: time.Unix(10, 0).UTC()}}},
		Signals:       &fakeSignalReader{latest: strategy.Signal{ID: "signal-1", Symbol: "BTCUSDT", Side: strategy.SideBuy}},
		RiskDecisions: &fakeRiskDecisionReader{latest: risk.Decision{ID: "risk-1", Symbol: "BTCUSDT", Decision: risk.DecisionApproved}},
		PaperAccount:  &fakePaperAccountReader{account: execution.Account{BaseAsset: "BTC", QuoteAsset: "USDT", BaseBalance: 0.1, QuoteBalance: 900}},
		Orders:        &fakeOrderReader{orders: []execution.Order{{ID: "order-1", Symbol: "BTCUSDT", Status: execution.OrderStatusFilled}}},
		Trades:        &fakeTradeReader{trades: []execution.Trade{{ID: "trade-1", Symbol: "BTCUSDT", Price: 100}}},
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary?symbol=BTCUSDT&interval=1m", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["symbol"] != "BTCUSDT" {
		t.Fatalf("expected symbol BTCUSDT, got %#v", body["symbol"])
	}
	if body["latest_price"] != "100.00" {
		t.Fatalf("expected latest price 100.00, got %#v", body["latest_price"])
	}
}

func TestOperationalReadRoutesReturnData(t *testing.T) {
	router := NewRouter(Dependencies{
		Postgres:      fakePinger{},
		Redis:         fakePinger{},
		Signals:       &fakeSignalReader{signals: []strategy.Signal{{ID: "signal-1", Symbol: "BTCUSDT", Side: strategy.SideBuy}}},
		RiskDecisions: &fakeRiskDecisionReader{decisions: []risk.Decision{{ID: "risk-1", Symbol: "BTCUSDT", Decision: risk.DecisionApproved}}},
		PaperAccount:  &fakePaperAccountReader{account: execution.Account{BaseAsset: "BTC", QuoteAsset: "USDT", QuoteBalance: 1000}},
		Orders:        &fakeOrderReader{orders: []execution.Order{{ID: "order-1", Symbol: "BTCUSDT"}}},
		Trades:        &fakeTradeReader{trades: []execution.Trade{{ID: "trade-1", Symbol: "BTCUSDT"}}},
	})

	paths := []string{
		"/api/v1/signals?symbol=BTCUSDT&limit=10",
		"/api/v1/risk-decisions?symbol=BTCUSDT&limit=10",
		"/api/v1/paper/account",
		"/api/v1/paper/orders?symbol=BTCUSDT&limit=10",
		"/api/v1/paper/trades?symbol=BTCUSDT&limit=10",
	}

	for _, path := range paths {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("expected %s to return 200, got %d", path, response.Code)
		}
	}
}

func TestRiskSettingsRoutesReadAndUpdate(t *testing.T) {
	settingsStore := &fakeRiskSettingsStore{settings: risk.DefaultSettings()}
	router := NewRouter(Dependencies{RiskSettings: settingsStore})

	read := httptest.NewRecorder()
	router.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/risk/settings", nil))
	if read.Code != http.StatusOK {
		t.Fatalf("expected settings read 200, got %d", read.Code)
	}

	body := []byte(`{"enabled":true,"max_signal_strength":100,"min_signal_strength":1,"max_quote_amount":250,"max_daily_loss":50,"max_daily_trades":5,"allow_buy":true,"allow_sell":false,"cooldown_seconds":300}`)
	update := httptest.NewRecorder()
	router.ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/api/v1/risk/settings", bytes.NewReader(body)))
	if update.Code != http.StatusOK {
		t.Fatalf("expected settings update 200, got %d body %s", update.Code, update.Body.String())
	}
	if settingsStore.settings.MaxQuoteAmount != 250 || settingsStore.settings.AllowSell {
		t.Fatalf("settings were not updated: %+v", settingsStore.settings)
	}
}

func TestRiskSettingsUpdateRejectsInvalidPayload(t *testing.T) {
	router := NewRouter(Dependencies{RiskSettings: &fakeRiskSettingsStore{settings: risk.DefaultSettings()}})

	body := []byte(`{"enabled":true,"min_signal_strength":-1}`)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/v1/risk/settings", bytes.NewReader(body)))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", response.Code)
	}
}

func TestPaperAccountResetRouteUpdatesAccount(t *testing.T) {
	accountStore := &fakePaperAccountStore{account: execution.Account{BaseAsset: "BTC", QuoteAsset: "USDT", QuoteBalance: 10000}}
	router := NewRouter(Dependencies{PaperAccount: accountStore})
	body := []byte(`{"base_asset":"BTC","quote_asset":"USDT","base_balance":0.5,"quote_balance":2500}`)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/paper/account/reset", bytes.NewReader(body)))

	if response.Code != http.StatusOK {
		t.Fatalf("expected reset 200, got %d body %s", response.Code, response.Body.String())
	}
	if accountStore.saved.BaseAsset != "BTC" || accountStore.saved.QuoteAsset != "USDT" {
		t.Fatalf("expected reset assets saved, got %+v", accountStore.saved)
	}
	if accountStore.saved.BaseBalance != 0.5 || accountStore.saved.QuoteBalance != 2500 {
		t.Fatalf("expected reset balances saved, got %+v", accountStore.saved)
	}
}

func TestManualPaperCycleRouteRunsPipeline(t *testing.T) {
	runner := &fakePaperCycleRunner{}
	router := NewRouter(Dependencies{PaperCycleRunner: runner})
	body := []byte(`{"symbol":"btcusdt","interval":"1m"}`)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/paper/cycles", bytes.NewReader(body)))

	if response.Code != http.StatusCreated {
		t.Fatalf("expected cycle 201, got %d body %s", response.Code, response.Body.String())
	}
	if runner.request.Symbol != "BTCUSDT" || runner.request.Interval != "1m" {
		t.Fatalf("expected normalized request, got %+v", runner.request)
	}
}

func TestExecutionStatusRouteReportsDisabledLiveAdapter(t *testing.T) {
	router := NewRouter(Dependencies{
		ExecutionStatus: execution.Status{
			Mode:               "paper",
			PaperEnabled:       true,
			ExchangeAdapter:    "binance_disabled",
			LiveTradingEnabled: false,
			RetryAttempts:      3,
			Timeout:            "5s",
			LastError:          "binance live trading is disabled",
		},
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/execution/status", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected execution status 200, got %d body %s", response.Code, response.Body.String())
	}
	var body struct {
		Data execution.Status `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if body.Data.LiveTradingEnabled {
		t.Fatalf("expected live trading disabled, got %+v", body.Data)
	}
	if body.Data.Mode != "paper" || body.Data.ExchangeAdapter != "binance_disabled" {
		t.Fatalf("unexpected execution status: %+v", body.Data)
	}
}

func TestSafetyStatusRoutesReadAndUpdateWithAudit(t *testing.T) {
	safetyStore := &fakeSafetyStore{status: safety.Status{KillSwitchActive: false}}
	auditStore := &fakeAuditStore{}
	router := NewRouter(Dependencies{Safety: safetyStore, Audit: auditStore})
	body := []byte(`{"kill_switch_active":true,"reason":"maintenance","updated_by":"operator"}`)

	read := httptest.NewRecorder()
	router.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/safety/status", nil))
	if read.Code != http.StatusOK {
		t.Fatalf("expected safety read 200, got %d", read.Code)
	}

	update := httptest.NewRecorder()
	router.ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/api/v1/safety/status", bytes.NewReader(body)))
	if update.Code != http.StatusOK {
		t.Fatalf("expected safety update 200, got %d body %s", update.Code, update.Body.String())
	}
	if !safetyStore.status.KillSwitchActive || len(auditStore.events) != 1 {
		t.Fatalf("expected safety active and one audit event, got status %+v events %d", safetyStore.status, len(auditStore.events))
	}
}

func TestAuditEventsRouteReturnsEvents(t *testing.T) {
	router := NewRouter(Dependencies{Audit: &fakeAuditStore{events: []audit.Event{{ID: "audit-1", EventType: "safety.status_changed"}}}})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?limit=10", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected audit 200, got %d", response.Code)
	}
}

func TestCSVExportsReturnCSV(t *testing.T) {
	router := NewRouter(Dependencies{
		Audit:         &fakeAuditStore{events: []audit.Event{{ID: "audit-1", EventType: "safety.status_changed", Actor: "operator"}}},
		Trades:        &fakeTradeReader{trades: []execution.Trade{{ID: "trade-1", Symbol: "BTCUSDT", Side: strategy.SideBuy}}},
		RiskDecisions: &fakeRiskDecisionReader{decisions: []risk.Decision{{ID: "risk-1", Symbol: "BTCUSDT", Decision: risk.DecisionRejected, Reason: "blocked"}}},
	})

	for _, path := range []string{
		"/api/v1/audit/events?format=csv",
		"/api/v1/paper/trades?format=csv",
		"/api/v1/risk-decisions?format=csv",
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("expected %s 200, got %d", path, response.Code)
		}
		if !strings.Contains(response.Header().Get("Content-Type"), "text/csv") {
			t.Fatalf("expected CSV content type for %s, got %q", path, response.Header().Get("Content-Type"))
		}
	}
}

func TestReportRoutesReturnPaperAndRiskSummaries(t *testing.T) {
	reports := &fakeReportStore{
		pnl:        []execution.DailyPnL{{Day: time.Unix(10, 0).UTC(), NetPnL: 12.5, TradeCount: 2}},
		rejections: []risk.RejectionSummary{{Reason: "buy disabled", Count: 3}},
	}
	router := NewRouter(Dependencies{Reports: reports})

	for _, path := range []string{
		"/api/v1/reports/paper/daily-pnl?symbol=BTCUSDT&limit=10",
		"/api/v1/reports/paper/trade-counts?symbol=BTCUSDT&limit=10",
		"/api/v1/reports/risk/rejections?symbol=BTCUSDT&limit=10",
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("expected %s 200, got %d body %s", path, response.Code, response.Body.String())
		}
	}
}

func TestSensitiveActionsWriteAuditEvents(t *testing.T) {
	auditStore := &fakeAuditStore{}
	router := NewRouter(Dependencies{
		Audit:            auditStore,
		RiskSettings:     &fakeRiskSettingsStore{settings: risk.DefaultSettings()},
		PaperAccount:     &fakePaperAccountStore{account: execution.Account{BaseAsset: "BTC", QuoteAsset: "USDT", QuoteBalance: 10000}},
		PaperCycleRunner: &fakePaperCycleRunner{},
	})

	requests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPut, "/api/v1/risk/settings", `{"enabled":true,"max_signal_strength":100,"min_signal_strength":1,"max_quote_amount":250,"max_daily_loss":50,"max_daily_trades":5,"allow_buy":true,"allow_sell":false,"cooldown_seconds":300}`},
		{http.MethodPost, "/api/v1/paper/account/reset", `{"base_asset":"BTC","quote_asset":"USDT","base_balance":0,"quote_balance":10000}`},
		{http.MethodPost, "/api/v1/paper/cycles", `{"symbol":"BTCUSDT","interval":"1m"}`},
	}

	for _, item := range requests {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(item.method, item.path, strings.NewReader(item.body)))
		if response.Code >= 400 {
			t.Fatalf("expected %s success, got %d body %s", item.path, response.Code, response.Body.String())
		}
	}
	if len(auditStore.events) != len(requests) {
		t.Fatalf("expected %d audit events, got %d", len(requests), len(auditStore.events))
	}
}

func TestStrategySettingsRoutesReadUpdateAndEvaluate(t *testing.T) {
	settingsStore := &fakeStrategySettingsStore{settings: strategy.DefaultSettings()}
	signalStore := &fakeSignalStore{}
	router := NewRouter(Dependencies{
		Postgres:         fakePinger{},
		Redis:            fakePinger{},
		Candles:          &fakeCandleReader{candles: smaCandles()},
		StrategySettings: settingsStore,
		SignalStore:      signalStore,
	})

	read := httptest.NewRecorder()
	router.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/strategy/settings", nil))
	if read.Code != http.StatusOK {
		t.Fatalf("expected settings read 200, got %d", read.Code)
	}

	updateBody := []byte(`{"strategy_name":"sma-crossover","version":"v1","symbol":"BTCUSDT","interval":"1m","fast_period":3,"slow_period":5,"lookback_limit":10}`)
	update := httptest.NewRecorder()
	router.ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/api/v1/strategy/settings", bytes.NewReader(updateBody)))
	if update.Code != http.StatusOK {
		t.Fatalf("expected settings update 200, got %d body %s", update.Code, update.Body.String())
	}
	if settingsStore.settings.FastPeriod != 3 || settingsStore.settings.SlowPeriod != 5 {
		t.Fatalf("settings were not updated: %+v", settingsStore.settings)
	}

	evaluate := httptest.NewRecorder()
	router.ServeHTTP(evaluate, httptest.NewRequest(http.MethodPost, "/api/v1/strategy/evaluate", nil))
	if evaluate.Code != http.StatusOK {
		t.Fatalf("expected evaluate 200, got %d body %s", evaluate.Code, evaluate.Body.String())
	}
	if len(signalStore.saved) != 1 {
		t.Fatalf("expected saved signal, got %d", len(signalStore.saved))
	}
}

func TestStrategySettingsUpdateRejectsInvalidPayload(t *testing.T) {
	router := NewRouter(Dependencies{
		StrategySettings: &fakeStrategySettingsStore{settings: strategy.DefaultSettings()},
	})

	body := []byte(`{"strategy_name":"sma-crossover","version":"v1","symbol":"BTCUSDT","interval":"1m","fast_period":20,"slow_period":10,"lookback_limit":100}`)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/v1/strategy/settings", bytes.NewReader(body)))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", response.Code)
	}
}

func TestBacktestRoutesCreateListAndReadRun(t *testing.T) {
	store := &fakeBacktestStore{}
	router := NewRouter(Dependencies{
		Candles:   &fakeCandleReader{candles: smaCandles()},
		Backtests: store,
	})

	body := []byte(`{"strategy_name":"sma-crossover","version":"v1","symbol":"BTCUSDT","interval":"1m","from":"1970-01-01T00:00:00Z","to":"1970-01-01T00:30:00Z","fast_period":3,"slow_period":5,"starting_balance":1000,"fee_rate":0.001}`)
	create := httptest.NewRecorder()
	router.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/backtests", bytes.NewReader(body)))
	if create.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d body %s", create.Code, create.Body.String())
	}
	var createBody map[string]any
	if err := json.Unmarshal(create.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	data, ok := createBody["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object in response body: %+v", createBody)
	}
	if data["total_fees"] == nil {
		t.Fatalf("expected total_fees in response body: %+v", data)
	}
	if data["round_trip_cost_percent"] == nil {
		t.Fatalf("expected round_trip_cost_percent in response body: %+v", data)
	}
	if len(store.runs) != 1 {
		t.Fatalf("expected stored run, got %d", len(store.runs))
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/backtests?symbol=BTCUSDT&limit=10", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", list.Code)
	}

	read := httptest.NewRecorder()
	router.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/backtests/run-1", nil))
	if read.Code != http.StatusOK {
		t.Fatalf("expected read 200, got %d", read.Code)
	}
}

func TestBacktestRouteCreatesExecutableTemplateRun(t *testing.T) {
	store := &fakeBacktestStore{}
	router := NewRouter(Dependencies{
		Candles:   &fakeCandleReader{candles: templateBacktestCandles(520)},
		Backtests: store,
	})

	body := []byte(`{"strategy_name":"multi-factor-momentum","version":"v1","symbol":"BTCUSDT","interval":"1m","from":"1970-01-01T00:01:40Z","to":"1970-01-01T08:41:40Z","fast_period":50,"slow_period":200,"rsi_period":14,"starting_balance":1000,"fee_rate":0.001,"position_sizing_mode":"percent_equity","position_size_value":10}`)
	create := httptest.NewRecorder()
	router.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/backtests", bytes.NewReader(body)))
	if create.Code != http.StatusCreated {
		t.Fatalf("expected executable template backtest create 201, got %d body %s", create.Code, create.Body.String())
	}
	if len(store.runs) != 1 {
		t.Fatalf("expected stored template run, got %d", len(store.runs))
	}
	if store.runs[0].StrategyName != strategy.StrategyMultiFactorMomentum {
		t.Fatalf("expected strategy %q, got %q", strategy.StrategyMultiFactorMomentum, store.runs[0].StrategyName)
	}
}

func TestStrategyComparisonRoutesCreateListAndReadComparison(t *testing.T) {
	store := &fakeComparisonStore{}
	router := NewRouter(Dependencies{
		Candles:             &fakeCandleReader{candles: smaCandles()},
		StrategyComparisons: store,
	})

	body := []byte(`{"symbol":"BTCUSDT","interval":"1m","from":"1970-01-01T00:00:00Z","to":"1970-01-01T00:30:00Z","fast_period":3,"slow_period":5,"rsi_period":3,"rsi_oversold":30,"rsi_overbought":70,"starting_balance":1000,"fee_rate":0.001}`)
	create := httptest.NewRecorder()
	router.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/strategy/comparisons", bytes.NewReader(body)))
	if create.Code != http.StatusCreated {
		t.Fatalf("expected comparison create 201, got %d body %s", create.Code, create.Body.String())
	}
	if len(store.comparisons) != 1 {
		t.Fatalf("expected stored comparison, got %d", len(store.comparisons))
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/strategy/comparisons?symbol=BTCUSDT&limit=10", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("expected comparison list 200, got %d", list.Code)
	}

	read := httptest.NewRecorder()
	router.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/strategy/comparisons/comparison-1", nil))
	if read.Code != http.StatusOK {
		t.Fatalf("expected comparison read 200, got %d", read.Code)
	}
}

func TestBacktestOptimizationRouteRunsSMAGrid(t *testing.T) {
	router := NewRouter(Dependencies{
		Candles: &fakeCandleReader{candles: smaCandles()},
	})

	body := []byte(`{"symbol":"BTCUSDT","interval":"1m","from":"1970-01-01T00:00:00Z","to":"1970-01-01T00:30:00Z","fast_periods":[2,3],"slow_periods":[4,5],"starting_balance":1000,"fee_rate":0.001,"position_sizing_mode":"percent_equity","position_size_value":10,"regime_filter_enabled":true,"regime_filter_period":3,"regime_min_atr_percent":0.01,"regime_max_atr_percent":10,"shorting_enabled":true}`)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/backtests/optimizations", bytes.NewReader(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("expected optimization create 201, got %d body %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"total_combinations":4`) {
		t.Fatalf("expected optimization results in response, got %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"regime_filter_enabled":true`) || !strings.Contains(response.Body.String(), `"shorting_enabled":true`) {
		t.Fatalf("expected hardened execution controls in optimization response, got %s", response.Body.String())
	}
}

func TestStrategyActivationRouteActivatesComparisonWinner(t *testing.T) {
	comparisonStore := &fakeComparisonStore{
		comparisons: []backtest.Comparison{comparisonForActivation()},
	}
	settingsStore := &fakeStrategySettingsStore{settings: strategy.DefaultSettings()}
	activationStore := &fakeActivationStore{}
	auditStore := &fakeAuditStore{}
	router := NewRouter(Dependencies{
		StrategyComparisons: comparisonStore,
		StrategySettings:    settingsStore,
		StrategyActivations: activationStore,
		Audit:               auditStore,
	})

	body := []byte(`{"actor":"operator"}`)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/strategy/comparisons/comparison-1/activate", bytes.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("expected activation 200, got %d body %s", response.Code, response.Body.String())
	}
	if settingsStore.settings.StrategyName != strategy.StrategyRSIMeanReversion {
		t.Fatalf("expected RSI settings activated, got %+v", settingsStore.settings)
	}
	if len(activationStore.records) != 1 {
		t.Fatalf("expected activation history record, got %d", len(activationStore.records))
	}
	if len(auditStore.events) != 1 || auditStore.events[0].EventType != "strategy.activated" {
		t.Fatalf("expected strategy activation audit event, got %+v", auditStore.events)
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/strategy/activations?limit=10", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("expected activation list 200, got %d", list.Code)
	}
}

func TestStrategyActivationRouteRejectsWeakComparisonWinner(t *testing.T) {
	comparison := comparisonForActivation()
	comparison.Results[0].ValidationStatus = "weak_profit_factor"
	comparison.Results[0].ValidationReason = "profit factor must be greater than 1.2"
	comparisonStore := &fakeComparisonStore{
		comparisons: []backtest.Comparison{comparison},
	}
	settingsStore := &fakeStrategySettingsStore{settings: strategy.DefaultSettings()}
	activationStore := &fakeActivationStore{}
	auditStore := &fakeAuditStore{}
	router := NewRouter(Dependencies{
		StrategyComparisons: comparisonStore,
		StrategySettings:    settingsStore,
		StrategyActivations: activationStore,
		Audit:               auditStore,
	})

	body := []byte(`{"actor":"operator"}`)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/strategy/comparisons/comparison-1/activate", bytes.NewReader(body)))
	if response.Code != http.StatusConflict {
		t.Fatalf("expected activation conflict, got %d body %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "activation gate blocked") {
		t.Fatalf("expected activation gate error, got %s", response.Body.String())
	}
	if settingsStore.settings.StrategyName == strategy.StrategyRSIMeanReversion {
		t.Fatalf("expected strategy settings not to change, got %+v", settingsStore.settings)
	}
	if len(activationStore.records) != 0 {
		t.Fatalf("expected no activation history record, got %d", len(activationStore.records))
	}
	if len(auditStore.events) != 0 {
		t.Fatalf("expected no audit event, got %+v", auditStore.events)
	}
}

func TestStrategyLifecycleRoutesListAndAdvance(t *testing.T) {
	activationStore := &fakeActivationStore{
		lifecycles: []activation.LifecycleRecord{{
			ID:           "lifecycle-1",
			StrategyName: strategy.StrategyRSIMeanReversion,
			Symbol:       "BTCUSDT",
			Interval:     "1m",
			State:        activation.StateValidated,
		}},
	}
	router := NewRouter(Dependencies{StrategyActivations: activationStore, Audit: &fakeAuditStore{}})

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/strategy/lifecycle?limit=10", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("expected lifecycle list 200, got %d body %s", list.Code, list.Body.String())
	}

	body := []byte(`{"state":"PAPER_TRADING","reason":"paper soak started","updated_by":"operator"}`)
	advance := httptest.NewRecorder()
	router.ServeHTTP(advance, httptest.NewRequest(http.MethodPost, "/api/v1/strategy/lifecycle/lifecycle-1/advance", bytes.NewReader(body)))
	if advance.Code != http.StatusOK {
		t.Fatalf("expected lifecycle advance 200, got %d body %s", advance.Code, advance.Body.String())
	}
	if activationStore.lifecycles[0].State != activation.StatePaperTrading {
		t.Fatalf("expected paper trading state, got %+v", activationStore.lifecycles[0])
	}
}

func TestReconciliationRoutesReturnRuns(t *testing.T) {
	store := &fakeReconciliationStore{
		runs: []reconciliation.Run{{ID: "reconciliation-1", Status: reconciliation.StatusMismatch}},
	}
	router := NewRouter(Dependencies{ReconciliationRuns: store})

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/reconciliation/runs?limit=10", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("expected reconciliation list 200, got %d", list.Code)
	}

	detail := httptest.NewRecorder()
	router.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/api/v1/reconciliation/runs/reconciliation-1", nil))
	if detail.Code != http.StatusOK {
		t.Fatalf("expected reconciliation detail 200, got %d", detail.Code)
	}
}

func TestReconciliationRunRouteTriggersRunnerAndAuditsMismatch(t *testing.T) {
	store := &fakeReconciliationStore{}
	auditStore := &fakeAuditStore{}
	router := NewRouter(Dependencies{
		ReconciliationRuns:   store,
		ReconciliationRunner: store,
		Audit:                auditStore,
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/reconciliation/runs", nil))
	if response.Code != http.StatusCreated {
		t.Fatalf("expected reconciliation create 201, got %d body %s", response.Code, response.Body.String())
	}
	if store.runCount != 1 {
		t.Fatalf("expected runner called once, got %d", store.runCount)
	}
	if len(auditStore.events) != 1 || auditStore.events[0].EventType != "reconciliation.mismatch" {
		t.Fatalf("expected mismatch audit event, got %+v", auditStore.events)
	}
}

func TestReconciliationRunRouteArmsKillSwitchForCriticalMismatch(t *testing.T) {
	store := &fakeReconciliationStore{
		nextRun: reconciliation.Run{
			ID:     "reconciliation-critical",
			Status: reconciliation.StatusMismatch,
			Mismatches: []reconciliation.Mismatch{{
				Kind:          reconciliation.MismatchOrder,
				Key:           "paper-client-1",
				InternalValue: "submitted",
				ExternalValue: "missing",
				Severity:      "critical",
			}},
		},
	}
	auditStore := &fakeAuditStore{}
	safetyStore := &fakeSafetyStore{}
	router := NewRouter(Dependencies{
		ReconciliationRuns:   store,
		ReconciliationRunner: store,
		Audit:                auditStore,
		Safety:               safetyStore,
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/reconciliation/runs", nil))
	if response.Code != http.StatusCreated {
		t.Fatalf("expected reconciliation create 201, got %d body %s", response.Code, response.Body.String())
	}
	if !safetyStore.status.KillSwitchActive {
		t.Fatalf("expected critical reconciliation mismatch to arm kill switch, got %+v", safetyStore.status)
	}
	if safetyStore.status.Reason != "critical reconciliation mismatch detected" {
		t.Fatalf("expected reconciliation safety reason, got %+v", safetyStore.status)
	}
	if len(auditStore.events) != 2 {
		t.Fatalf("expected mismatch and kill-switch audit events, got %+v", auditStore.events)
	}
	if auditStore.events[0].EventType != "reconciliation.critical_mismatch" {
		t.Fatalf("expected critical mismatch audit first, got %+v", auditStore.events)
	}
	if auditStore.events[1].EventType != "safety.status_changed" {
		t.Fatalf("expected safety audit second, got %+v", auditStore.events)
	}
}

func TestBacktestCreateRejectsInvalidPayload(t *testing.T) {
	router := NewRouter(Dependencies{Backtests: &fakeBacktestStore{}, Candles: &fakeCandleReader{}})

	body := []byte(`{"symbol":"","interval":"1m","fast_period":5,"slow_period":3,"starting_balance":1000}`)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/backtests", bytes.NewReader(body)))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", response.Code)
	}
}

func TestBacktestCreateReturnsPreflightDetailsForInsufficientCandles(t *testing.T) {
	router := NewRouter(Dependencies{
		Backtests: &fakeBacktestStore{},
		Candles:   &fakeCandleReader{candles: smaCandles()[:3]},
	})

	body := []byte(`{"strategy_name":"sma-crossover","version":"v1","symbol":"BTCUSDT","interval":"1m","from":"1970-01-01T00:00:00Z","to":"1970-01-01T00:30:00Z","fast_period":3,"slow_period":5,"starting_balance":1000,"fee_rate":0.001}`)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/backtests", bytes.NewReader(body)))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", response.Code)
	}
	var bodyMap map[string]any
	if err := json.NewDecoder(response.Body).Decode(&bodyMap); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyMap["required_candles"] != float64(6) || bodyMap["available_candles"] != float64(3) {
		t.Fatalf("expected preflight counts, got %+v", bodyMap)
	}
}

func TestBacktestCreateRejectsOversizedRangeBeforeQuery(t *testing.T) {
	candles := &fakeCandleReader{candles: smaCandles()}
	router := NewRouter(Dependencies{
		Backtests: &fakeBacktestStore{},
		Candles:   candles,
	})

	body := []byte(`{"strategy_name":"sma-crossover","version":"v1","symbol":"BTCUSDT","interval":"1m","from":"2010-01-01T00:00:00Z","to":"2026-01-01T00:00:00Z","fast_period":3,"slow_period":5,"starting_balance":1000,"fee_rate":0.001}`)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/backtests", bytes.NewReader(body)))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d body %s", response.Code, response.Body.String())
	}
	if candles.query.Limit != 0 {
		t.Fatalf("expected oversized range to be rejected before candle query, got query %+v", candles.query)
	}
	if !strings.Contains(response.Body.String(), "backtest range is too large") {
		t.Fatalf("expected oversized range error, got %s", response.Body.String())
	}
}

func TestBacktestCreateReturnsCandleDiagnosticsForDirtyData(t *testing.T) {
	candles := smaCandles()
	candles[4].OpenTime = candles[3].OpenTime
	router := NewRouter(Dependencies{
		Backtests: &fakeBacktestStore{},
		Candles:   &fakeCandleReader{candles: candles},
	})

	body := []byte(`{"strategy_name":"sma-crossover","version":"v1","symbol":"BTCUSDT","interval":"1m","from":"1970-01-01T00:00:00Z","to":"1970-01-01T00:30:00Z","fast_period":3,"slow_period":5,"starting_balance":1000,"fee_rate":0.001}`)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/backtests", bytes.NewReader(body)))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d body %s", response.Code, response.Body.String())
	}
	var bodyMap map[string]any
	if err := json.NewDecoder(response.Body).Decode(&bodyMap); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	diagnostics, ok := bodyMap["candle_diagnostics"].(map[string]any)
	if !ok {
		t.Fatalf("expected candle_diagnostics object, got %+v", bodyMap)
	}
	if diagnostics["duplicate_count"] == float64(0) {
		t.Fatalf("expected duplicate count diagnostics, got %+v", diagnostics)
	}
}

func TestObservabilityRoutesReturnMetricsRunsAndStreams(t *testing.T) {
	metrics := observability.NewRegistry()
	metrics.IncPipelineCompleted()
	router := NewRouter(Dependencies{
		Postgres:     fakePinger{},
		Redis:        fakePinger{},
		Metrics:      metrics,
		PipelineRuns: &fakePipelineRunReader{runs: []orchestration.Run{{Key: "run-1", Status: "completed"}}},
		Streams:      &fakeStreamStatsReader{stats: []StreamStats{{Name: "stream:market-data", Pending: 2}}},
	})

	paths := []string{
		"/metrics",
		"/api/v1/ops/pipeline-runs?status=completed&limit=10",
		"/api/v1/ops/streams",
	}

	for _, path := range paths {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("expected %s to return 200, got %d", path, response.Code)
		}
	}
}

func TestStrategyTemplateRoutesReturnCatalogAndDetails(t *testing.T) {
	router := NewRouter(Dependencies{})

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/strategies/templates", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("expected template list to return 200, got %d", list.Code)
	}
	var listBody struct {
		Data []strategy.StrategyTemplate `json:"data"`
	}
	if err := json.NewDecoder(list.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode template list: %v", err)
	}
	if len(listBody.Data) != 10 {
		t.Fatalf("expected 10 templates, got %d", len(listBody.Data))
	}

	detail := httptest.NewRecorder()
	router.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/api/v1/strategies/templates/multi-factor-momentum", nil))
	if detail.Code != http.StatusOK {
		t.Fatalf("expected template detail to return 200, got %d", detail.Code)
	}
	var detailBody struct {
		Data strategy.StrategyTemplate `json:"data"`
	}
	if err := json.NewDecoder(detail.Body).Decode(&detailBody); err != nil {
		t.Fatalf("decode template detail: %v", err)
	}
	if detailBody.Data.ID != strategy.StrategyMultiFactorMomentum {
		t.Fatalf("expected multi-factor template, got %q", detailBody.Data.ID)
	}
	if detailBody.Data.SupportStatus != strategy.TemplateExecutablePine {
		t.Fatalf("expected executable Pine support, got %q", detailBody.Data.SupportStatus)
	}

	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/v1/strategies/templates/unknown", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("expected unknown template to return 404, got %d", missing.Code)
	}
}

type fakePinger struct{}

func (fakePinger) Ping(ctx context.Context) error {
	return nil
}

type fakeCandleReader struct {
	query    marketdata.CandleQuery
	candles  []marketdata.Candle
	coverage marketdata.Coverage
}

func (f *fakeCandleReader) List(ctx context.Context, query marketdata.CandleQuery) ([]marketdata.Candle, error) {
	f.query = query
	return f.candles, nil
}

func (f *fakeCandleReader) Coverage(ctx context.Context, query marketdata.CandleQuery) (marketdata.Coverage, error) {
	f.query = query
	return f.coverage, nil
}

type fakeSignalReader struct {
	latest  strategy.Signal
	signals []strategy.Signal
}

func (f *fakeSignalReader) Latest(ctx context.Context, symbol string) (strategy.Signal, error) {
	return f.latest, nil
}

func (f *fakeSignalReader) List(ctx context.Context, query strategy.SignalQuery) ([]strategy.Signal, error) {
	return f.signals, nil
}

type fakeRiskDecisionReader struct {
	latest    risk.Decision
	decisions []risk.Decision
}

type fakeRiskSettingsStore struct {
	settings risk.Settings
}

func (f *fakeRiskSettingsStore) Get(ctx context.Context) (risk.Settings, error) {
	return f.settings, nil
}

func (f *fakeRiskSettingsStore) Save(ctx context.Context, settings risk.Settings) (risk.Settings, error) {
	f.settings = settings
	return settings, nil
}

func (f *fakeRiskDecisionReader) Latest(ctx context.Context, symbol string) (risk.Decision, error) {
	return f.latest, nil
}

func (f *fakeRiskDecisionReader) List(ctx context.Context, query risk.DecisionQuery) ([]risk.Decision, error) {
	return f.decisions, nil
}

type fakePaperAccountReader struct {
	account execution.Account
}

func (f *fakePaperAccountReader) Get(ctx context.Context) (execution.Account, error) {
	return f.account, nil
}

func (f *fakePaperAccountReader) Save(ctx context.Context, account execution.Account) error {
	f.account = account
	return nil
}

type fakePaperAccountStore struct {
	account execution.Account
	saved   execution.Account
}

func (f *fakePaperAccountStore) Get(ctx context.Context) (execution.Account, error) {
	return f.account, nil
}

func (f *fakePaperAccountStore) Save(ctx context.Context, account execution.Account) error {
	f.saved = account
	f.account = account
	return nil
}

type fakePaperCycleRunner struct {
	request orchestration.ManualRunRequest
}

func (f *fakePaperCycleRunner) RunOnce(ctx context.Context, request orchestration.ManualRunRequest) (orchestration.ManualRunResult, error) {
	f.request = request
	return orchestration.ManualRunResult{
		Status: "executed",
		Signal: strategy.Signal{ID: "signal-1", Symbol: request.Symbol, Side: strategy.SideBuy},
		Decision: risk.Decision{
			ID:       "risk-1",
			Symbol:   request.Symbol,
			Decision: risk.DecisionApproved,
		},
		Execution: &execution.Result{
			Order: execution.Order{ID: "order-1", Symbol: request.Symbol},
			Trade: execution.Trade{ID: "trade-1", Symbol: request.Symbol},
		},
	}, nil
}

type fakeSafetyStore struct {
	status safety.Status
}

func (f *fakeSafetyStore) Get(ctx context.Context) (safety.Status, error) {
	return f.status, nil
}

func (f *fakeSafetyStore) Save(ctx context.Context, status safety.Status) (safety.Status, error) {
	f.status = status
	return status, nil
}

func (f *fakeSafetyStore) IsKillSwitchActive(ctx context.Context) (bool, error) {
	return f.status.KillSwitchActive, nil
}

type fakeAuditStore struct {
	events []audit.Event
	query  audit.Query
}

func (f *fakeAuditStore) Save(ctx context.Context, event audit.Event) (string, error) {
	f.events = append(f.events, event)
	return "audit-1", nil
}

func (f *fakeAuditStore) List(ctx context.Context, query audit.Query) ([]audit.Event, error) {
	f.query = query
	return f.events, nil
}

type fakeReportStore struct {
	pnl        []execution.DailyPnL
	rejections []risk.RejectionSummary
}

func (f *fakeReportStore) DailyPnL(ctx context.Context, query execution.Query) ([]execution.DailyPnL, error) {
	return f.pnl, nil
}

func (f *fakeReportStore) TradeCounts(ctx context.Context, query execution.Query) ([]execution.DailyPnL, error) {
	return f.pnl, nil
}

func (f *fakeReportStore) RejectedReasons(ctx context.Context, query risk.DecisionQuery) ([]risk.RejectionSummary, error) {
	return f.rejections, nil
}

type fakeOrderReader struct {
	orders []execution.Order
}

func (f *fakeOrderReader) ListOrders(ctx context.Context, query execution.Query) ([]execution.Order, error) {
	return f.orders, nil
}

type fakeTradeReader struct {
	trades []execution.Trade
}

func (f *fakeTradeReader) ListTrades(ctx context.Context, query execution.Query) ([]execution.Trade, error) {
	return f.trades, nil
}

type fakePipelineRunReader struct {
	runs []orchestration.Run
}

func (f *fakePipelineRunReader) ListRuns(ctx context.Context, query orchestration.RunQuery) ([]orchestration.Run, error) {
	return f.runs, nil
}

type fakeStreamStatsReader struct {
	stats []StreamStats
}

func (f *fakeStreamStatsReader) Stats(ctx context.Context) ([]StreamStats, error) {
	return f.stats, nil
}

type fakeStrategySettingsStore struct {
	settings strategy.Settings
}

func (f *fakeStrategySettingsStore) Get(ctx context.Context) (strategy.Settings, error) {
	return f.settings, nil
}

func (f *fakeStrategySettingsStore) Save(ctx context.Context, settings strategy.Settings) (strategy.Settings, error) {
	f.settings = settings
	return settings, nil
}

type fakeSignalStore struct {
	saved []strategy.Signal
}

func (f *fakeSignalStore) Save(ctx context.Context, signal strategy.Signal) (string, error) {
	f.saved = append(f.saved, signal)
	return "signal-1", nil
}

func smaCandles() []marketdata.Candle {
	base := time.Unix(100, 0).UTC()
	closes := []string{"10", "10", "10", "10", "10", "12", "14", "16", "18", "20", "22", "24", "26", "28", "30", "32", "34", "36"}
	candles := make([]marketdata.Candle, 0, len(closes))
	for index, closeValue := range closes {
		openTime := base.Add(time.Duration(index) * time.Minute)
		candles = append(candles, marketdata.Candle{
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Open:      closeValue,
			High:      closeValue,
			Low:       closeValue,
			Close:     closeValue,
			IsClosed:  true,
		})
	}
	return candles
}

func templateBacktestCandles(count int) []marketdata.Candle {
	base := time.Unix(100, 0).UTC()
	candles := make([]marketdata.Candle, 0, count)
	for index := 0; index < count; index++ {
		openTime := base.Add(time.Duration(index) * time.Minute)
		closeValue := strconv.FormatFloat(100+float64(index)*0.1, 'f', 2, 64)
		candles = append(candles, marketdata.Candle{
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Open:      closeValue,
			High:      closeValue,
			Low:       closeValue,
			Close:     closeValue,
			Volume:    "1000",
			IsClosed:  true,
		})
	}
	return candles
}

type fakeBacktestStore struct {
	runs   []backtest.Run
	trades []backtest.Trade
}

type fakeComparisonStore struct {
	comparisons []backtest.Comparison
}

func (f *fakeComparisonStore) Save(ctx context.Context, comparison backtest.Comparison) (backtest.Comparison, error) {
	comparison.ID = "comparison-1"
	for index := range comparison.Results {
		comparison.Results[index].ComparisonID = comparison.ID
	}
	f.comparisons = append(f.comparisons, comparison)
	return comparison, nil
}

func (f *fakeComparisonStore) List(ctx context.Context, query backtest.Query) ([]backtest.Comparison, error) {
	return f.comparisons, nil
}

func (f *fakeComparisonStore) Get(ctx context.Context, id string) (backtest.Comparison, error) {
	if len(f.comparisons) == 0 {
		return backtest.Comparison{}, nil
	}
	return f.comparisons[0], nil
}

type fakeActivationStore struct {
	records    []activation.Record
	lifecycles []activation.LifecycleRecord
}

func (f *fakeActivationStore) Save(ctx context.Context, record activation.Record) (activation.Record, error) {
	record.ID = "activation-1"
	f.records = append(f.records, record)
	return record, nil
}

func (f *fakeActivationStore) List(ctx context.Context, query activation.Query) ([]activation.Record, error) {
	return f.records, nil
}

func (f *fakeActivationStore) SaveLifecycle(ctx context.Context, record activation.LifecycleRecord) (activation.LifecycleRecord, error) {
	record.ID = "lifecycle-1"
	f.lifecycles = append(f.lifecycles, record)
	return record, nil
}

func (f *fakeActivationStore) ListLifecycles(ctx context.Context, query activation.Query) ([]activation.LifecycleRecord, error) {
	return f.lifecycles, nil
}

func (f *fakeActivationStore) AdvanceLifecycle(ctx context.Context, id string, state activation.LifecycleState, reason string, actor string) (activation.LifecycleRecord, error) {
	for index := range f.lifecycles {
		if f.lifecycles[index].ID == id {
			f.lifecycles[index].State = state
			f.lifecycles[index].Reason = reason
			f.lifecycles[index].UpdatedBy = actor
			return f.lifecycles[index], nil
		}
	}
	return activation.LifecycleRecord{}, nil
}

type fakeReconciliationStore struct {
	runs     []reconciliation.Run
	runCount int
	nextRun  reconciliation.Run
}

func (f *fakeReconciliationStore) List(ctx context.Context, query reconciliation.Query) ([]reconciliation.Run, error) {
	return f.runs, nil
}

func (f *fakeReconciliationStore) Get(ctx context.Context, id string) (reconciliation.Run, error) {
	for _, run := range f.runs {
		if run.ID == id {
			return run, nil
		}
	}
	return reconciliation.Run{}, nil
}

func (f *fakeReconciliationStore) Run(ctx context.Context) (reconciliation.Run, error) {
	f.runCount++
	run := f.nextRun
	if run.ID == "" {
		run = reconciliation.Run{
			ID:     "reconciliation-created",
			Status: reconciliation.StatusMismatch,
			Mismatches: []reconciliation.Mismatch{{
				Kind:     reconciliation.MismatchBalance,
				Key:      "USDT",
				Severity: "warning",
			}},
		}
	}
	f.runs = append(f.runs, run)
	return run, nil
}

func comparisonForActivation() backtest.Comparison {
	return backtest.Comparison{
		ID:                "comparison-1",
		Symbol:            "BTCUSDT",
		Interval:          "1m",
		ExecutionFillMode: backtest.ExecutionFillModeNextOpen,
		WinnerStrategy:    strategy.StrategyRSIMeanReversion,
		CreatedAt:         time.Now().UTC(),
		Results: []backtest.ComparisonResult{
			{
				ID:                          "result-1",
				Rank:                        1,
				StrategyName:                strategy.StrategyRSIMeanReversion,
				Version:                     "v1",
				FastPeriod:                  9,
				SlowPeriod:                  21,
				RSIPeriod:                   14,
				RSIOversold:                 30,
				RSIOverbought:               70,
				MaxDrawdown:                 12,
				TotalTrades:                 140,
				ProfitFactor:                1.4,
				Expectancy:                  0.2,
				ExcessReturnPercent:         3.2,
				ValidationStatus:            "candidate",
				ExecutionFillMode:           backtest.ExecutionFillModeNextOpen,
				TrainValidationStatus:       "candidate",
				TestValidationStatus:        "candidate",
				WalkForwardFolds:            4,
				WalkForwardPasses:           4,
				WalkForwardValidationStatus: "candidate",
			},
		},
	}
}

func (f *fakeBacktestStore) Save(ctx context.Context, run backtest.Run, trades []backtest.Trade) (backtest.Run, error) {
	run.ID = "run-1"
	f.runs = append(f.runs, run)
	for index := range trades {
		trades[index].RunID = run.ID
	}
	f.trades = append(f.trades, trades...)
	return run, nil
}

func (f *fakeBacktestStore) List(ctx context.Context, query backtest.Query) ([]backtest.Run, error) {
	return f.runs, nil
}

func (f *fakeBacktestStore) Get(ctx context.Context, id string) (backtest.Run, []backtest.Trade, error) {
	if len(f.runs) == 0 {
		return backtest.Run{}, nil, nil
	}
	return f.runs[0], f.trades, nil
}
