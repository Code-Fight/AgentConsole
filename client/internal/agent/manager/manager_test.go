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

func TestManagerRoutesThreadAndTurnOperationsToRuntime(t *testing.T) {
	reg := agentregistry.New()
	runtime := &stubRuntime{}
	reg.Register("fake", runtime)
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

	readThread, err := mgr.ReadThread("fake", "thread-01")
	if err != nil {
		t.Fatal(err)
	}
	if readThread.ThreadID != "thread-01" {
		t.Fatalf("unexpected read thread: %+v", readThread)
	}

	resumedThread, err := mgr.ResumeThread("fake", "thread-01")
	if err != nil {
		t.Fatal(err)
	}
	if resumedThread.ThreadID != "thread-01" || resumedThread.Status != domain.ThreadStatusIdle {
		t.Fatalf("unexpected resumed thread: %+v", resumedThread)
	}

	if err := mgr.ArchiveThread("fake", "thread-01"); err != nil {
		t.Fatal(err)
	}

	steerResult, err := mgr.SteerTurn("fake", agenttypes.SteerTurnParams{
		ThreadID: "thread-01",
		TurnID:   "turn-01",
		Input:    "try a smaller patch",
	})
	if err != nil {
		t.Fatal(err)
	}
	if steerResult.TurnID != "turn-01" || steerResult.ThreadID != "thread-01" {
		t.Fatalf("unexpected steer result: %+v", steerResult)
	}

	interruptedTurn, err := mgr.InterruptTurn("fake", agenttypes.InterruptTurnParams{
		ThreadID: "thread-01",
		TurnID:   "turn-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if interruptedTurn.TurnID != "turn-01" || interruptedTurn.Status != domain.TurnStatusInterrupted {
		t.Fatalf("unexpected interrupted turn: %+v", interruptedTurn)
	}

	if err := mgr.SetSkillEnabled("fake", "skill-a", false); err != nil {
		t.Fatal(err)
	}
	if runtime.lastSkillNameOrPath != "skill-a" || runtime.lastSkillEnabled {
		t.Fatalf("unexpected skill mutation: nameOrPath=%q enabled=%v", runtime.lastSkillNameOrPath, runtime.lastSkillEnabled)
	}

	if err := mgr.UninstallPlugin("fake", "plugin-a"); err != nil {
		t.Fatal(err)
	}
	if runtime.lastPluginID != "plugin-a" {
		t.Fatalf("unexpected plugin uninstall target: %q", runtime.lastPluginID)
	}
}

type stubRuntime struct {
	lastSkillNameOrPath string
	lastSkillEnabled    bool
	lastPluginID        string
}

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

func (s *stubRuntime) ReadThread(threadID string) (domain.Thread, error) {
	return domain.Thread{
		ThreadID:  threadID,
		MachineID: "machine-01",
		Status:    domain.ThreadStatusIdle,
		Title:     "Investigate flaky test",
	}, nil
}

func (s *stubRuntime) ResumeThread(threadID string) (domain.Thread, error) {
	return domain.Thread{
		ThreadID:  threadID,
		MachineID: "machine-01",
		Status:    domain.ThreadStatusIdle,
		Title:     "Investigate flaky test",
	}, nil
}

func (s *stubRuntime) ArchiveThread(string) error {
	return nil
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

func (s *stubRuntime) SteerTurn(params agenttypes.SteerTurnParams) (agenttypes.SteerTurnResult, error) {
	return agenttypes.SteerTurnResult{
		TurnID:   params.TurnID,
		ThreadID: params.ThreadID,
		Deltas: []agenttypes.TurnDelta{
			{Sequence: 3, Delta: "assistant: adjusted"},
		},
	}, nil
}

func (s *stubRuntime) InterruptTurn(params agenttypes.InterruptTurnParams) (domain.Turn, error) {
	return domain.Turn{
		TurnID:   params.TurnID,
		ThreadID: params.ThreadID,
		Status:   domain.TurnStatusInterrupted,
	}, nil
}

func (s *stubRuntime) SetSkillEnabled(nameOrPath string, enabled bool) error {
	s.lastSkillNameOrPath = nameOrPath
	s.lastSkillEnabled = enabled
	return nil
}

func (s *stubRuntime) UninstallPlugin(pluginID string) error {
	s.lastPluginID = pluginID
	return nil
}
