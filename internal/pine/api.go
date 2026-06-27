package pine

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type SaveRequest struct {
	Name     string `json:"name"`
	PineCode string `json:"pine_code"`
}

type PineSaver interface {
	Save(ctx context.Context, strategy PineStrategy) (PineStrategy, error)
}

type PineLister interface {
	List(ctx context.Context, query Query) ([]PineStrategy, error)
}

type PineGetter interface {
	Get(ctx context.Context, id string) (PineStrategy, error)
}

func writeJSON(w http.ResponseWriter, status int, val any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(val)
}

// ValidateHandler checks the syntax and validity of Pine Script.
func ValidateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		var req struct {
			PineCode string `json:"pine_code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		parser := NewParser(req.PineCode)
		res := parser.Parse()

		var infoList []IndicatorInfo
		for name, def := range res.Config.Indicators {
			infoList = append(infoList, IndicatorInfo{
				Name:   name,
				Type:   def.Type,
				Source: def.Source,
				Params: def.Params,
			})
		}

		validationResult := ValidationResult{
			Valid:      len(res.Errors) == 0,
			Config:     res.Config,
			Warnings:   res.Warnings,
			Errors:     res.Errors,
			Indicators: infoList,
			Rules:      res.Config.Rules,
		}

		writeJSON(w, http.StatusOK, validationResult)
	}
}

// SaveHandler parses and saves/updates a Pine Script strategy in the DB.
func SaveHandler(repo PineSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		var req SaveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		if strings.TrimSpace(req.Name) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "strategy name is required"})
			return
		}
		if strings.TrimSpace(req.PineCode) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pine code is required"})
			return
		}

		// Validate first
		parser := NewParser(req.PineCode)
		res := parser.Parse()
		if len(res.Errors) > 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":  "pine script compilation failed",
				"errors": res.Errors,
			})
			return
		}

		strategy := PineStrategy{
			Name:            req.Name,
			PineCode:        req.PineCode,
			ConvertedConfig: res.Config,
		}

		saved, err := repo.Save(r.Context(), strategy)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save pine strategy"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": saved})
	}
}

// ListHandler lists all saved Pine Script strategies.
func ListHandler(repo PineLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		strategies, err := repo.List(r.Context(), Query{Limit: 100})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list pine strategies"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": strategies})
	}
}

// GetHandler retrieves a specific Pine strategy details.
func GetHandler(repo PineGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		// The ID is usually part of the URL path, e.g. /api/v1/strategies/pine/{id}
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request path"})
			return
		}
		id := parts[len(parts)-1]

		strategy, err := repo.Get(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "strategy not found"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": strategy})
	}
}
