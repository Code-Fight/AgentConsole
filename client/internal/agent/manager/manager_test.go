package manager

import (
	"strings"
	"testing"

	agentregistry "code-agent-gateway/client/internal/agent/registry"
)

func TestSnapshotReturnsErrorWhenRuntimeMissing(t *testing.T) {
	mgr := New(agentregistry.New())

	_, err := mgr.Snapshot("missing")
	if err == nil {
		t.Fatal("expected error for missing runtime")
	}
	if !strings.Contains(err.Error(), `runtime "missing" is not registered`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
