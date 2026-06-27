package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type contextKey string

const correlationIDKey contextKey = "correlation_id"

func CorrelationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-Request-ID")
		if correlationID == "" {
			correlationID = newCorrelationID()
		}

		w.Header().Set("X-Request-ID", correlationID)
		ctx := context.WithValue(r.Context(), correlationIDKey, correlationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CorrelationIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(correlationIDKey).(string)
	return value
}

func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type SecurityConfig struct {
	MaxRequestBodyBytes        int64
	RateLimitRequestsPerMinute int
	// CORS settings
	CORSEnabled          bool
	CORSAllowedOrigins   []string
	CORSAllowedMethods   []string
	CORSAllowedHeaders   []string
	CORSExposedHeaders   []string
	CORSAllowCredentials bool
	CORSMaxAge           int
	// Security headers configurability
	DisableSecurityHeaders bool
	CustomSecurityHeaders  map[string]string
}

func SecurityHeadersMiddleware(cfg SecurityConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip security headers if disabled
		if cfg.DisableSecurityHeaders {
			next.ServeHTTP(w, r)
			return
		}

		// Set standard security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")

		// Set Content-Security-Policy header if not customized
		if len(cfg.CustomSecurityHeaders) == 0 {
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' ws: wss:; img-src 'self' data:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
		} else {
			// Apply custom security headers
			for key, value := range cfg.CustomSecurityHeaders {
				w.Header().Set(key, value)
			}
		}

		// Set CORS headers if enabled
		if cfg.CORSEnabled {
			if len(cfg.CORSAllowedOrigins) > 0 {
				w.Header().Set("Access-Control-Allow-Origin", strings.Join(cfg.CORSAllowedOrigins, ","))
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			if len(cfg.CORSAllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.CORSAllowedMethods, ","))
			}

			if len(cfg.CORSAllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.CORSAllowedHeaders, ","))
			}

			if len(cfg.CORSExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.CORSExposedHeaders, ","))
			}

			if cfg.CORSAllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if cfg.CORSMaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.CORSMaxAge))
			}
		}

		next.ServeHTTP(w, r)
	})
}

func MaxBodyMiddleware(cfg SecurityConfig, next http.Handler) http.Handler {
	limit := cfg.MaxRequestBodyBytes
	if limit <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			defer r.Body.Close()
			if r.ContentLength > limit {
				writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func RateLimitMiddleware(cfg SecurityConfig, next http.Handler) http.Handler {
	if cfg.RateLimitRequestsPerMinute <= 0 {
		return next
	}
	limiter := newFixedWindowLimiter(cfg.RateLimitRequestsPerMinute, time.Minute)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = r.RemoteAddr
		}
		if !limiter.allow(key, time.Now()) {
			w.Header().Set("Retry-After", "60")
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

type fixedWindowLimiter struct {
	limit  int
	window time.Duration
	mu     sync.Mutex
	state  map[string]rateState
}

type rateState struct {
	windowStart time.Time
	count       int
}

func newFixedWindowLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		limit:  limit,
		window: window,
		state:  make(map[string]rateState),
	}
}

func (l *fixedWindowLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	state := l.state[key]
	if state.windowStart.IsZero() || now.Sub(state.windowStart) >= l.window {
		l.state[key] = rateState{windowStart: now, count: 1}
		return true
	}
	if state.count >= l.limit {
		return false
	}
	state.count++
	l.state[key] = state
	return true
}

func newCorrelationID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "request"
	}
	return hex.EncodeToString(bytes[:])
}
