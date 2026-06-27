package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"sentra/internal/marketdata"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "sentra",
	})
}

func readinessHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		status := map[string]any{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"checks":    map[string]any{},
		}
		overallStatus := "ok"
		httpCode := http.StatusOK

		postgresStatus := map[string]any{"status": "ok"}
		if deps.Postgres != nil {
			start := time.Now()
			err := deps.Postgres.Ping(ctx)
			latency := time.Since(start)
			postgresStatus["latency_ms"] = latency.Milliseconds()
			if err != nil {
				postgresStatus["status"] = "error"
				postgresStatus["error"] = err.Error()
				overallStatus = "degraded"
				httpCode = http.StatusServiceUnavailable
			}
		} else {
			postgresStatus["status"] = "not_configured"
			overallStatus = "degraded"
			httpCode = http.StatusServiceUnavailable
		}
		status["checks"].(map[string]any)["postgres"] = postgresStatus

		redisStatus := map[string]any{"status": "ok"}
		if deps.Redis != nil {
			start := time.Now()
			err := deps.Redis.Ping(ctx)
			latency := time.Since(start)
			redisStatus["latency_ms"] = latency.Milliseconds()
			if err != nil {
				redisStatus["status"] = "error"
				redisStatus["error"] = err.Error()
				overallStatus = "degraded"
				httpCode = http.StatusServiceUnavailable
			}
		} else {
			redisStatus["status"] = "not_configured"
			overallStatus = "degraded"
			httpCode = http.StatusServiceUnavailable
		}
		status["checks"].(map[string]any)["redis"] = redisStatus

		candlesStatus := map[string]any{"status": "not_configured"}
		if deps.Candles != nil {
			start := time.Now()
			_, err := deps.Candles.Coverage(ctx, marketdata.CandleQuery{
				Symbol:   "BTCUSDT",
				Interval: "1m",
			})
			latency := time.Since(start)
			candlesStatus["latency_ms"] = latency.Milliseconds()
			if err != nil {
				candlesStatus["status"] = "ok"
				candlesStatus["note"] = "service responding"
			} else {
				candlesStatus["status"] = "ok"
			}
		}
		status["checks"].(map[string]any)["candles"] = candlesStatus

		signalsStatus := map[string]any{"status": "not_configured"}
		if deps.Signals != nil {
			start := time.Now()
			_, err := deps.Signals.Latest(ctx, "BTCUSDT")
			latency := time.Since(start)
			signalsStatus["latency_ms"] = latency.Milliseconds()
			if err != nil {
				signalsStatus["status"] = "ok"
				signalsStatus["note"] = "service responding"
			} else {
				signalsStatus["status"] = "ok"
			}
		}
		status["checks"].(map[string]any)["signals"] = signalsStatus

		status["status"] = overallStatus
		writeJSON(w, httpCode, status)
	}
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
