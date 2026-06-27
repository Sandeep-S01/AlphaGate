package observability

import "testing"

func TestRegistrySnapshotIncludesCounters(t *testing.T) {
	registry := NewRegistry()

	registry.IncHTTPRequests()
	registry.IncHTTPRequests()
	registry.IncPipelineCompleted()
	registry.IncPipelineFailed()

	snapshot := registry.Snapshot()

	if snapshot.HTTPRequests != 2 {
		t.Fatalf("expected 2 HTTP requests, got %d", snapshot.HTTPRequests)
	}
	if snapshot.PipelineCompleted != 1 {
		t.Fatalf("expected 1 completed pipeline, got %d", snapshot.PipelineCompleted)
	}
	if snapshot.PipelineFailed != 1 {
		t.Fatalf("expected 1 failed pipeline, got %d", snapshot.PipelineFailed)
	}
}
