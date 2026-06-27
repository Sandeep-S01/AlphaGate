package observability

import "sync/atomic"

// Snapshot represents a point-in-time view of observability metrics.
type Snapshot struct {
	HTTPRequests      uint64 `json:"http_requests"`
	PipelineCompleted uint64 `json:"pipeline_completed"`
	PipelineFailed    uint64 `json:"pipeline_failed"`

	// Strategy metrics
	StrategyEvaluations   uint64 `json:"strategy_evaluations"`
	StrategyEvaluationErr uint64 `json:"strategy_evaluation_errors"`

	// Trade execution metrics
	TradeExecutions       uint64 `json:"trade_executions"`
	TradeExecutionErr     uint64 `json:"trade_execution_errors"`

	// Backtest metrics
	BacktestRequests      uint64 `json:"backtest_requests"`
	BacktestRequestErr    uint64 `json:"backtest_request_errors"`

	// API metrics (these would typically be histograms in a real implementation)
	APIRequestCount       uint64 `json:"api_request_count"`
	APIRequestErr         uint64 `json:"api_request_errors"`

	// Database metrics
	DBOpenConnections     uint64 `json:"db_open_connections"`
	DBInUseConnections    uint64 `json:"db_in_use_connections"`
	DBIdleConnections     uint64 `json:"db_idle_conensions"`
	DBWaitCount           uint64 `json:"db_wait_count"`
	DBWaitDuration        uint64 `json:"db_wait_duration_ms"`
	DBMaxWaitDuration     uint64 `json:"db_max_wait_duration_ms"`
}

// Registry holds and updates observability metrics.
type Registry struct {
	httpRequests      atomic.Uint64
	pipelineCompleted atomic.Uint64
	pipelineFailed    atomic.Uint64

	// Strategy metrics
	strategyEvaluations   atomic.Uint64
	strategyEvaluationErr atomic.Uint64

	// Trade execution metrics
	tradeExecutions       atomic.Uint64
	tradeExecutionErr     atomic.Uint64

	// Backtest metrics
	backtestRequests      atomic.Uint64
	backtestRequestErr    atomic.Uint64

	// API metrics
	apiRequestCount       atomic.Uint64
	apiRequestErr         atomic.Uint64

	// Database metrics
	dbOpenConnections     atomic.Uint64
	dbInUseConnections    atomic.Uint64
	dbIdleConnections     atomic.Uint64
	dbWaitCount           atomic.Uint64
	dbWaitDuration        atomic.Uint64
	dbMaxWaitDuration     atomic.Uint64
}

// NewRegistry creates a new metrics registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// HTTP request metrics
func (r *Registry) IncHTTPRequests() {
	r.httpRequests.Add(1)
}

func (r *Registry) IncHTTPRequestErr() {
	r.apiRequestErr.Add(1)
}

// Pipeline metrics
func (r *Registry) IncPipelineCompleted() {
	r.pipelineCompleted.Add(1)
}

func (r *Registry) IncPipelineFailed() {
	r.pipelineFailed.Add(1)
}

// Strategy evaluation metrics
func (r *Registry) IncStrategyEvaluations() {
	r.strategyEvaluations.Add(1)
}

func (r *Registry) IncStrategyEvaluationErr() {
	r.strategyEvaluationErr.Add(1)
}

// Trade execution metrics
func (r *Registry) IncTradeExecutions() {
	r.tradeExecutions.Add(1)
}

func (r *Registry) IncTradeExecutionErr() {
	r.tradeExecutionErr.Add(1)
}

// Backtest metrics
func (r *Registry) IncBacktestRequests() {
	r.backtestRequests.Add(1)
}

func (r *Registry) IncBacktestRequestErr() {
	r.backtestRequestErr.Add(1)
}

// API request metrics
func (r *Registry) IncAPIRequestCount() {
	r.apiRequestCount.Add(1)
}

func (r *Registry) IncAPIRequestErr() {
	r.apiRequestErr.Add(1)
}

// Database metrics
func (r *Registry) SetDBOpenConnections(connections uint64) {
	r.dbOpenConnections.Store(connections)
}

func (r *Registry) SetDBInUseConnections(connections uint64) {
	r.dbInUseConnections.Store(connections)
}

func (r *Registry) SetDBIdleConnections(connections uint64) {
	r.dbIdleConnections.Store(connections)
}

func (r *Registry) AddDBWaitCount(count uint64) {
	r.dbWaitCount.Add(count)
}

func (r *Registry) AddDBWaitDuration(durationMs uint64) {
	r.dbWaitDuration.Add(durationMs)
}

func (r *Registry) SetDBMaxWaitDuration(durationMs uint64) {
	r.dbMaxWaitDuration.Store(durationMs)
}

// Snapshot returns a point-in-time copy of all metrics.
func (r *Registry) Snapshot() Snapshot {
	return Snapshot{
		HTTPRequests:      r.httpRequests.Load(),
		PipelineCompleted: r.pipelineCompleted.Load(),
		PipelineFailed:    r.pipelineFailed.Load(),
		StrategyEvaluations:   r.strategyEvaluations.Load(),
		StrategyEvaluationErr: r.strategyEvaluationErr.Load(),
		TradeExecutions:       r.tradeExecutions.Load(),
		TradeExecutionErr:     r.tradeExecutionErr.Load(),
		BacktestRequests:      r.backtestRequests.Load(),
		BacktestRequestErr:    r.backtestRequestErr.Load(),
		APIRequestCount:       r.apiRequestCount.Load(),
		APIRequestErr:         r.apiRequestErr.Load(),
		DBOpenConnections:     r.dbOpenConnections.Load(),
		DBInUseConnections:    r.dbInUseConnections.Load(),
		DBIdleConnections:     r.dbIdleConnections.Load(),
		DBWaitCount:           r.dbWaitCount.Load(),
		DBWaitDuration:        r.dbWaitDuration.Load(),
		DBMaxWaitDuration:     r.dbMaxWaitDuration.Load(),
	}
}
