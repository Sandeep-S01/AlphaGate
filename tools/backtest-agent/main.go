package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type agentConfig struct {
	BaseURL      string
	Symbol       string
	Interval     string
	OutputDir    string
	Timeout      time.Duration
	RateLimit    bool
	MaxBodyBytes int64
}

type resultStatus string

const (
	statusPass    resultStatus = "PASS"
	statusFail    resultStatus = "FAIL"
	statusWarn    resultStatus = "WARN"
	statusBlocked resultStatus = "BLOCKED"
)

type scenarioResult struct {
	Level       string         `json:"level"`
	Name        string         `json:"name"`
	Status      resultStatus   `json:"status"`
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	HTTPStatus  int            `json:"http_status"`
	LatencyMS   int64          `json:"latency_ms"`
	Evidence    string         `json:"evidence"`
	Error       string         `json:"error,omitempty"`
	Request     map[string]any `json:"request,omitempty"`
	Response    map[string]any `json:"response,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt time.Time      `json:"completed_at"`
}

type report struct {
	GeneratedAt time.Time        `json:"generated_at"`
	BaseURL     string           `json:"base_url"`
	Symbol      string           `json:"symbol"`
	Interval    string           `json:"interval"`
	Summary     summary          `json:"summary"`
	Scenarios   []scenarioResult `json:"scenarios"`
	Findings    []finding        `json:"findings"`
}

type summary struct {
	Total   int `json:"total"`
	Pass    int `json:"pass"`
	Warn    int `json:"warn"`
	Fail    int `json:"fail"`
	Blocked int `json:"blocked"`
}

type finding struct {
	Severity       string `json:"severity"`
	Area           string `json:"area"`
	Finding        string `json:"finding"`
	Recommendation string `json:"recommendation"`
}

type client struct {
	baseURL string
	http    *http.Client
	apiKey  string
}

type httpResult struct {
	status  int
	headers http.Header
	body    []byte
	latency time.Duration
}

type templateItem struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SupportStatus string `json:"support_status"`
}

type candle struct {
	OpenTime  time.Time `json:"OpenTime"`
	CloseTime time.Time `json:"CloseTime"`
}

func main() {
	cfg := agentConfig{}
	flag.StringVar(&cfg.BaseURL, "base-url", "http://127.0.0.1:8080", "Sentra API base URL")
	flag.StringVar(&cfg.Symbol, "symbol", "BTCUSDT", "symbol to test")
	flag.StringVar(&cfg.Interval, "interval", "1m", "interval to test")
	flag.StringVar(&cfg.OutputDir, "output-dir", "docs", "directory for report artifacts")
	flag.DurationVar(&cfg.Timeout, "timeout", 20*time.Second, "per-request timeout")
	flag.BoolVar(&cfg.RateLimit, "rate-limit", true, "run rate-limit scenario")
	flag.Int64Var(&cfg.MaxBodyBytes, "max-body-bytes", 10*1024*1024, "maximum response body bytes to read")
	flag.Parse()

	ctx := context.Background()
	rep, err := run(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "backtest agent failed: %v\n", err)
		os.Exit(1)
	}
	if err := writeReports(cfg.OutputDir, rep); err != nil {
		fmt.Fprintf(os.Stderr, "write report: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Backtest E2E agent complete: %d pass, %d warn, %d fail, %d blocked\n", rep.Summary.Pass, rep.Summary.Warn, rep.Summary.Fail, rep.Summary.Blocked)
	fmt.Printf("Reports: %s, %s\n", filepath.Join(cfg.OutputDir, "backtesting_e2e_report.json"), filepath.Join(cfg.OutputDir, "backtesting_e2e_report.md"))
	if rep.Summary.Fail > 0 {
		os.Exit(2)
	}
}

func run(ctx context.Context, cfg agentConfig) (report, error) {
	c := client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
		apiKey: fmt.Sprintf("backtest-agent-%d", time.Now().UnixNano()),
	}
	rep := report{
		GeneratedAt: time.Now().UTC(),
		BaseURL:     cfg.BaseURL,
		Symbol:      cfg.Symbol,
		Interval:    cfg.Interval,
	}

	add := func(res scenarioResult) {
		rep.Scenarios = append(rep.Scenarios, res)
	}

	add(simpleGET(ctx, c, "L0-Min", "Health endpoint returns OK", "/health", []int{http.StatusOK}))
	add(simpleGET(ctx, c, "L0-Min", "Readiness endpoint returns OK", "/ready", []int{http.StatusOK}))

	templates, tmplResult := fetchTemplates(ctx, c)
	add(tmplResult)
	executable := executableTemplates(templates)
	blocked := blockedTemplates(templates)

	candles, candleResult := fetchCandles(ctx, c, cfg.Symbol, cfg.Interval, 600)
	add(candleResult)
	from, to, rangeOK := candleRange(candles)

	add(postJSON(ctx, c, "L1-Easy", "Malformed JSON is rejected", "/api/v1/backtests", []byte(`{"strategy_name":`), []int{http.StatusBadRequest}, nil))
	add(postBacktest(ctx, c, "L1-Easy", "Unknown strategy is rejected", backtestPayload(cfg.Symbol, cfg.Interval, from, to, "unknown-strategy"), []int{http.StatusBadRequest}, rangeOK))
	add(postBacktest(ctx, c, "L2-Medium", "Data-blocked template is rejected", backtestPayload(cfg.Symbol, cfg.Interval, from, to, firstOr(blocked, "funding-rate-arbitrage")), []int{http.StatusBadRequest}, rangeOK))

	futureFrom := time.Now().UTC().Add(24 * time.Hour)
	futureTo := futureFrom.Add(2 * time.Hour)
	add(postBacktest(ctx, c, "L2-Medium", "Future empty range returns insufficient-candles error", backtestPayload(cfg.Symbol, cfg.Interval, futureFrom, futureTo, "sma-crossover"), []int{http.StatusBadRequest}, true))

	add(postBacktest(ctx, c, "L3-Hard", "Native SMA backtest creates a run", backtestPayload(cfg.Symbol, cfg.Interval, from, to, "sma-crossover"), []int{http.StatusCreated}, rangeOK))
	add(postBacktest(ctx, c, "L3-Hard", "Native RSI backtest creates a run", backtestPayload(cfg.Symbol, cfg.Interval, from, to, "rsi-mean-reversion"), []int{http.StatusCreated}, rangeOK))
	add(postBacktest(ctx, c, "L3-Hard", "Native trend-pullback backtest creates a run", backtestPayload(cfg.Symbol, cfg.Interval, from, to, "btc-trend-pullback"), []int{http.StatusCreated}, rangeOK))

	for _, id := range executable {
		add(postBacktest(ctx, c, "L4-Template", "Executable template backtest creates a run: "+id, backtestPayload(cfg.Symbol, cfg.Interval, from, to, id), []int{http.StatusCreated}, rangeOK))
	}

	add(postBacktest(ctx, c, "L5-Extreme", "Oversized historical range is handled without server error", backtestPayload(cfg.Symbol, cfg.Interval, time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().UTC(), "sma-crossover"), []int{http.StatusBadRequest, http.StatusCreated}, true))
	add(postBacktest(ctx, c, "L5-Extreme", "Invalid economics are rejected", invalidEconomicsPayload(cfg.Symbol, cfg.Interval, from, to), []int{http.StatusBadRequest}, rangeOK))

	if cfg.RateLimit {
		add(rateLimitProbe(ctx, c))
	}

	rep.Summary = summarize(rep.Scenarios)
	rep.Findings = deriveFindings(rep.Scenarios, templates)
	return rep, nil
}

func simpleGET(ctx context.Context, c client, level, name, path string, expected []int) scenarioResult {
	start := time.Now().UTC()
	res, err := c.do(ctx, http.MethodGet, path, nil, nil)
	return evaluateHTTP(level, name, http.MethodGet, path, start, nil, res, err, expected, "")
}

func fetchTemplates(ctx context.Context, c client) ([]templateItem, scenarioResult) {
	start := time.Now().UTC()
	path := "/api/v1/strategies/templates"
	res, err := c.do(ctx, http.MethodGet, path, nil, nil)
	result := evaluateHTTP("L1-Easy", "Template catalog returns all predefined strategies", http.MethodGet, path, start, nil, res, err, []int{http.StatusOK}, "")
	var body struct {
		Data []templateItem `json:"data"`
	}
	if err == nil && res.status == http.StatusOK {
		if decodeErr := json.Unmarshal(res.body, &body); decodeErr != nil {
			result.Status = statusFail
			result.Error = "invalid JSON: " + decodeErr.Error()
		} else if len(body.Data) != 10 {
			result.Status = statusFail
			result.Evidence = fmt.Sprintf("expected 10 templates, got %d", len(body.Data))
		} else {
			result.Evidence = "catalog contains 10 templates"
		}
	}
	return body.Data, result
}

func fetchCandles(ctx context.Context, c client, symbol, interval string, limit int) ([]candle, scenarioResult) {
	path := fmt.Sprintf("/api/v1/market/candles?symbol=%s&interval=%s&limit=%d", symbol, interval, limit)
	start := time.Now().UTC()
	res, err := c.do(ctx, http.MethodGet, path, nil, nil)
	result := evaluateHTTP("L1-Easy", "Market candle seed data is available", http.MethodGet, path, start, nil, res, err, []int{http.StatusOK}, "")
	var body struct {
		Data []candle `json:"data"`
	}
	if err == nil && res.status == http.StatusOK {
		if decodeErr := json.Unmarshal(res.body, &body); decodeErr != nil {
			result.Status = statusFail
			result.Error = "invalid JSON: " + decodeErr.Error()
		} else if len(body.Data) < 260 {
			result.Status = statusBlocked
			result.Evidence = fmt.Sprintf("only %d candles available; executable templates need roughly 200-450 candles", len(body.Data))
		} else {
			result.Evidence = fmt.Sprintf("%d candles available", len(body.Data))
		}
	}
	return body.Data, result
}

func postBacktest(ctx context.Context, c client, level, name string, payload map[string]any, expected []int, enabled bool) scenarioResult {
	if !enabled {
		now := time.Now().UTC()
		return scenarioResult{Level: level, Name: name, Status: statusBlocked, Evidence: "skipped because seed candle range was unavailable", Request: payload, StartedAt: now, CompletedAt: now}
	}
	raw, _ := json.Marshal(payload)
	return postJSON(ctx, c, level, name, "/api/v1/backtests", raw, expected, payload)
}

func postJSON(ctx context.Context, c client, level, name, path string, body []byte, expected []int, req map[string]any) scenarioResult {
	start := time.Now().UTC()
	res, err := c.do(ctx, http.MethodPost, path, map[string]string{"Content-Type": "application/json"}, body)
	return evaluateHTTP(level, name, http.MethodPost, path, start, req, res, err, expected, "")
}

func rateLimitProbe(ctx context.Context, c client) scenarioResult {
	level := "L5-Extreme"
	name := "Protected API rate limit eventually returns 429"
	path := "/api/v1/strategies/templates"
	start := time.Now().UTC()
	seen429 := false
	lastStatus := 0
	total := 130
	for i := 0; i < total; i++ {
		res, err := c.do(ctx, http.MethodGet, path, nil, nil)
		if err != nil {
			return scenarioResult{Level: level, Name: name, Status: statusFail, Method: http.MethodGet, Path: path, HTTPStatus: lastStatus, Error: err.Error(), StartedAt: start, CompletedAt: time.Now().UTC()}
		}
		lastStatus = res.status
		if res.status == http.StatusTooManyRequests {
			seen429 = true
			break
		}
	}
	status := statusPass
	evidence := "received 429 after burst requests"
	if !seen429 {
		status = statusWarn
		evidence = fmt.Sprintf("no 429 after %d requests; last status %d. Rate limit may be disabled or higher than expected.", total, lastStatus)
	}
	return scenarioResult{Level: level, Name: name, Status: status, Method: http.MethodGet, Path: path, HTTPStatus: lastStatus, LatencyMS: time.Since(start).Milliseconds(), Evidence: evidence, StartedAt: start, CompletedAt: time.Now().UTC()}
}

func (c client) do(ctx context.Context, method, path string, headers map[string]string, body []byte) (httpResult, error) {
	url := c.baseURL + path
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return httpResult{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-Request-ID", fmt.Sprintf("backtest-agent-%d", rand.Int63()))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		return httpResult{latency: time.Since(start)}, err
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, 10*1024*1024)
	raw, readErr := io.ReadAll(limited)
	if readErr != nil {
		return httpResult{status: resp.StatusCode, headers: resp.Header, latency: time.Since(start)}, readErr
	}
	return httpResult{status: resp.StatusCode, headers: resp.Header, body: raw, latency: time.Since(start)}, nil
}

func evaluateHTTP(level, name, method, path string, start time.Time, req map[string]any, res httpResult, err error, expected []int, evidence string) scenarioResult {
	out := scenarioResult{
		Level:       level,
		Name:        name,
		Method:      method,
		Path:        path,
		HTTPStatus:  res.status,
		LatencyMS:   res.latency.Milliseconds(),
		Request:     req,
		StartedAt:   start,
		CompletedAt: time.Now().UTC(),
	}
	if err != nil {
		out.Status = statusFail
		out.Error = err.Error()
		return out
	}
	if containsStatus(expected, res.status) {
		out.Status = statusPass
	} else if res.status >= 500 {
		out.Status = statusFail
	} else {
		out.Status = statusWarn
	}
	out.Evidence = evidence
	if out.Evidence == "" {
		out.Evidence = summarizeBody(res.body)
	}
	out.Response = parseMap(res.body)
	if out.Evidence == "" || strings.HasPrefix(out.Evidence, "{") || strings.HasPrefix(out.Evidence, "data present") {
		if summary := summarizeBacktestResponse(out.Response); summary != "" {
			out.Evidence = summary
		}
	}
	return out
}

func summarizeBacktestResponse(response map[string]any) string {
	data, ok := response["data"].(map[string]any)
	if !ok {
		return ""
	}
	strategyName, _ := data["strategy_name"].(string)
	returnPercent, okReturn := numberValue(data["return_percent"])
	totalTrades, okTrades := numberValue(data["total_trades"])
	totalFees, okFees := numberValue(data["total_fees"])
	roundTripCost, okCost := numberValue(data["round_trip_cost_percent"])
	validationStatus, _ := data["validation_status"].(string)
	if strategyName == "" || !okReturn || !okTrades {
		return ""
	}
	if !okFees {
		totalFees = 0
	}
	if !okCost {
		roundTripCost = 0
	}
	return fmt.Sprintf("strategy=%s return=%.2f%% trades=%.0f fees=%.2f roundTripCost=%.2f%% validation=%s", strategyName, returnPercent, totalTrades, totalFees, roundTripCost, validationStatus)
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func backtestPayload(symbol, interval string, from, to time.Time, strategyName string) map[string]any {
	return map[string]any{
		"strategy_name":              strategyName,
		"version":                    "v1",
		"symbol":                     symbol,
		"interval":                   interval,
		"from":                       from.Format(time.RFC3339),
		"to":                         to.Format(time.RFC3339),
		"fast_period":                9,
		"slow_period":                21,
		"rsi_period":                 14,
		"rsi_oversold":               30,
		"rsi_overbought":             70,
		"starting_balance":           1000,
		"fee_rate":                   0.001,
		"slippage_rate":              0.0005,
		"position_sizing_mode":       "percent_equity",
		"position_size_value":        10,
		"cooldown_bars":              1,
		"min_holding_bars":           1,
		"atr_exit_enabled":           true,
		"atr_period":                 14,
		"atr_stop_multiplier":        2,
		"atr_take_profit_multiplier": 3,
		"execution_fill_mode":        "next_open",
		"shorting_enabled":           false,
		"save_equity_curve":          false,
	}
}

func invalidEconomicsPayload(symbol, interval string, from, to time.Time) map[string]any {
	payload := backtestPayload(symbol, interval, from, to, "sma-crossover")
	payload["starting_balance"] = -100
	payload["fee_rate"] = -0.01
	payload["position_size_value"] = 999
	return payload
}

func candleRange(candles []candle) (time.Time, time.Time, bool) {
	if len(candles) == 0 {
		return time.Time{}, time.Time{}, false
	}
	sort.Slice(candles, func(i, j int) bool { return candles[i].OpenTime.Before(candles[j].OpenTime) })
	from := candles[0].OpenTime.UTC()
	to := candles[len(candles)-1].CloseTime.UTC()
	return from, to, !from.IsZero() && to.After(from)
}

func executableTemplates(templates []templateItem) []string {
	var ids []string
	for _, tmpl := range templates {
		if tmpl.SupportStatus == "executable_pine" || tmpl.SupportStatus == "executable_native" {
			ids = append(ids, tmpl.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func blockedTemplates(templates []templateItem) []string {
	var ids []string
	for _, tmpl := range templates {
		if tmpl.SupportStatus == "blocked_by_data" || tmpl.SupportStatus == "template_only" {
			ids = append(ids, tmpl.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func summarize(results []scenarioResult) summary {
	out := summary{Total: len(results)}
	for _, res := range results {
		switch res.Status {
		case statusPass:
			out.Pass++
		case statusWarn:
			out.Warn++
		case statusFail:
			out.Fail++
		case statusBlocked:
			out.Blocked++
		}
	}
	return out
}

func deriveFindings(results []scenarioResult, templates []templateItem) []finding {
	var findings []finding
	for _, res := range results {
		if res.Status == statusFail {
			findings = append(findings, finding{
				Severity:       "HIGH",
				Area:           res.Level,
				Finding:        fmt.Sprintf("%s failed: %s", res.Name, firstNonEmpty(res.Error, res.Evidence)),
				Recommendation: "Fix this before treating backtesting as production-ready.",
			})
		}
		if res.Status == statusWarn {
			findings = append(findings, finding{
				Severity:       "MEDIUM",
				Area:           res.Level,
				Finding:        fmt.Sprintf("%s produced warning: %s", res.Name, res.Evidence),
				Recommendation: "Review expected behavior and tighten assertions or API semantics.",
			})
		}
	}
	dataBlocked := 0
	for _, tmpl := range templates {
		if tmpl.SupportStatus == "blocked_by_data" {
			dataBlocked++
		}
	}
	if dataBlocked > 0 {
		findings = append(findings, finding{
			Severity:       "MEDIUM",
			Area:           "Strategy Templates",
			Finding:        fmt.Sprintf("%d predefined templates are intentionally blocked by missing market-data/execution infrastructure.", dataBlocked),
			Recommendation: "Keep them visible but disabled in backtesting until L2/order-flow/funding/pairs/grid state support exists.",
		})
	}
	if len(findings) == 0 {
		findings = append(findings, finding{
			Severity:       "INFO",
			Area:           "Backtesting",
			Finding:        "All configured E2E scenarios passed.",
			Recommendation: "Run this agent in CI and before strategy-engine changes.",
		})
	}
	return findings
}

func writeReports(outputDir string, rep report) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "backtesting_e2e_report.json"), raw, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "backtesting_e2e_report.md"), []byte(markdownReport(rep)), 0644)
}

func markdownReport(rep report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Backtesting E2E Micro-Agent Report\n\n")
	fmt.Fprintf(&b, "Generated: `%s`\n\n", rep.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "Base URL: `%s`\n\n", rep.BaseURL)
	fmt.Fprintf(&b, "Summary: **%d pass**, **%d warn**, **%d fail**, **%d blocked** out of **%d** scenarios.\n\n", rep.Summary.Pass, rep.Summary.Warn, rep.Summary.Fail, rep.Summary.Blocked, rep.Summary.Total)
	fmt.Fprintf(&b, "## Scenario Matrix\n\n| Level | Scenario | Status | HTTP | Latency | Evidence |\n| --- | --- | --- | --- | ---: | --- |\n")
	for _, res := range rep.Scenarios {
		fmt.Fprintf(&b, "| %s | %s | %s | %d | %dms | %s |\n", escapeMD(res.Level), escapeMD(res.Name), res.Status, res.HTTPStatus, res.LatencyMS, escapeMD(firstNonEmpty(res.Error, res.Evidence)))
	}
	fmt.Fprintf(&b, "\n## Findings\n\n")
	for _, f := range rep.Findings {
		fmt.Fprintf(&b, "- **%s / %s:** %s Recommendation: %s\n", f.Severity, f.Area, f.Finding, f.Recommendation)
	}
	return b.String()
}

func parseMap(raw []byte) map[string]any {
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func summarizeBody(raw []byte) string {
	if len(raw) == 0 {
		return "empty response body"
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err == nil {
		if errText, ok := decoded["error"].(string); ok {
			return errText
		}
		if data, ok := decoded["data"]; ok {
			return fmt.Sprintf("data present (%T)", data)
		}
	}
	text := strings.TrimSpace(string(raw))
	if len(text) > 180 {
		text = text[:180] + "..."
	}
	return text
}

func containsStatus(expected []int, status int) bool {
	for _, value := range expected {
		if value == status {
			return true
		}
	}
	return false
}

func firstOr(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return values[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func escapeMD(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}
