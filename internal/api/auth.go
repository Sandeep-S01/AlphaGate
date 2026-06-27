package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type AuthConfig struct {
	Enabled     bool
	AdminAPIKey string
}

func APIKeyAuthMiddleware(cfg AuthConfig, next http.Handler) http.Handler {
	if !cfg.Enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		provided := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(cfg.AdminAPIKey)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isPublicPath(path string) bool {
	switch path {
	case "/health", "/ready", "/metrics":
		return true
	default:
		return path == "/" || strings.HasPrefix(path, "/dashboard/")
	}
}
