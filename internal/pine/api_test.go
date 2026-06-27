package pine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type failingPineStore struct {
	err error
}

func (s failingPineStore) Save(ctx context.Context, strategy PineStrategy) (PineStrategy, error) {
	return PineStrategy{}, s.err
}

func (s failingPineStore) List(ctx context.Context, query Query) ([]PineStrategy, error) {
	return nil, s.err
}

func (s failingPineStore) Get(ctx context.Context, id string) (PineStrategy, error) {
	return PineStrategy{}, s.err
}

func TestSaveHandlerDoesNotExposeStorageErrorDetails(t *testing.T) {
	handler := SaveHandler(failingPineStore{err: errors.New("upsert pine strategy: password authentication failed")})
	requestBody := `{
		"name": "safe error test",
		"pine_code": "ema50 = ta.ema(close, 50)\nbuy = close > ema50\nif buy\n    strategy.entry(\"LONG\", strategy.long)"
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/strategies/pine", bytes.NewBufferString(requestBody))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d body %s", response.Code, response.Body.String())
	}
	assertGenericError(t, response.Body.String(), "failed to save pine strategy")
}

func TestListHandlerDoesNotExposeStorageErrorDetails(t *testing.T) {
	handler := ListHandler(failingPineStore{err: errors.New("list pine strategies: connection refused password=secret")})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/strategies/pine", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d body %s", response.Code, response.Body.String())
	}
	assertGenericError(t, response.Body.String(), "failed to list pine strategies")
}

func assertGenericError(t *testing.T, body string, expected string) {
	t.Helper()
	var payload map[string]string
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["error"] != expected {
		t.Fatalf("expected generic error %q, got %q", expected, payload["error"])
	}
	for _, leaked := range []string{"password", "upsert", "connection refused", "secret"} {
		if strings.Contains(strings.ToLower(body), leaked) {
			t.Fatalf("response leaked storage detail %q in body %s", leaked, body)
		}
	}
}
