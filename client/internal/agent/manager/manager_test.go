package manager

import (
	"strings"
	"testing"

	agentregistry "code-agent-gateway/client/internal/agent/registry"
	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
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

func TestManagerRoutesCreateThreadAndStartTurnToRuntime(t *testing.T) {
	reg := agentregistry.New()
	reg.Register("fake", &stubRuntime{})
	mgr := New(reg)

	thread, err := mgr.CreateThread("fake", agenttypes.CreateThreadParams{Title: "Investigate flaky test"})
	if err != nil {
		t.Fatal(err)
	}
	if thread.ThreadID != "thread-01" {
		t.Fatalf("unexpected thread: %+v", thread)
	}

	result, err := mgr.StartTurn("fake", agenttypes.StartTurnParams{
		ThreadID: "thread-01",
		Input:    "run tests",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TurnID != "turn-01" || result.ThreadID != "thread-01" {
		t.Fatalf("unexpected turn result: %+v", result)
	}
}

type stubRuntime struct{}

func (s *stubRuntime) ListThreads() ([]domain.Thread, error) {
	return nil, nil
}

func (s *stubRuntime) ListEnvironment() ([]domain.EnvironmentResource, error) {
	return nil, nil
}

func (s *stubRuntime) CreateThread(params agenttypes.CreateThreadParams) (domain.Thread, error) {
	return domain.Thread{
		ThreadID:  "thread-01",
		MachineID: "machine-01",
		Status:    domain.ThreadStatusIdle,
		Title:     params.Title,
	}, nil
}

func (s *stubRuntime) StartTurn(params agenttypes.StartTurnParams) (agenttypes.StartTurnResult, error) {
	return agenttypes.StartTurnResult{
		TurnID:   "turn-01",
		ThreadID: params.ThreadID,
		Deltas: []agenttypes.TurnDelta{
			{Sequence: 1, Delta: "assistant: thinking"},
			{Sequence: 2, Delta: "assistant: done"},
		},
	}, nil
}
