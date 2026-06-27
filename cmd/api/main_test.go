package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestRunWiresSharedMetricsRegistryToRouter(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}

	text := string(source)
	if strings.Contains(text, "Metrics:             observability.NewRegistry()") {
		t.Fatalf("router must use the shared metrics registry, not create a new one")
	}
	if !regexp.MustCompile(`Metrics:\s+metrics`).MatchString(text) {
		t.Fatalf("expected router dependencies to receive the shared metrics registry")
	}
}
