package manager

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	agentregistry "code-agent-gateway/client/internal/agent/registry"
	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

type supervisorRuntimeFactory func(context.Context, agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error)

func (f supervisorRuntimeFactory) Start(ctx context.Context, spec agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error) {
	return f(ctx, spec)
}

type noopRuntime struct{}

func (noopRuntime) ListThreads() ([]domain.Thread, error) { return nil, nil }
func (noopRuntime) ListEnvironment() ([]domain.EnvironmentResource, error) { return nil, nil }
func (noopRuntime) CreateThread(agenttypes.CreateThreadParams) (domain.Thread, error) {
	return domain.Thread{}, nil
}
func (noopRuntime) ReadThread(string) (domain.Thread, error) { return domain.Thread{}, nil }
func (noopRuntime) ResumeThread(string) (domain.Thread, error) { return domain.Thread{}, nil }
func (noopRuntime) ArchiveThread(string) error { return nil }
func (noopRuntime) StartTurn(agenttypes.StartTurnParams) (agenttypes.StartTurnResult, error) {
	return agenttypes.StartTurnResult{}, nil
}
func (noopRuntime) SteerTurn(agenttypes.SteerTurnParams) (agenttypes.SteerTurnResult, error) {
	return agenttypes.SteerTurnResult{}, nil
}
func (noopRuntime) InterruptTurn(agenttypes.InterruptTurnParams) (domain.Turn, error) {
	return domain.Turn{}, nil
}

func TestNewSupervisorRejectsUnsafePersistedAgentID(t *testing.T) {
	managedAgentsDir := t.TempDir()
	recordDir := filepath.Join(managedAgentsDir, "agent-01")
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		t.Fatal(err)
	}

	record := map[string]any{
		"agentId":     "../escape",
		"agentType":   "codex",
		"displayName": "Tampered",
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recordDir, managedAgentMetadataName), data, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = NewSupervisor(context.Background(), managedAgentsDir, agentregistry.New(), map[domain.AgentType]agenttypes.RuntimeFactory{
		domain.AgentTypeCodex: supervisorRuntimeFactory(func(context.Context, agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error) {
			return noopRuntime{}, func() error { return nil }, nil
		}),
	})
	if err == nil {
		t.Fatal("expected unsafe persisted agentId to be rejected")
	}
}

func TestDeleteAgentRejectsUnsafeAgentID(t *testing.T) {
	managedAgentsDir := t.TempDir()
	supervisor, err := NewSupervisor(context.Background(), managedAgentsDir, agentregistry.New(), map[domain.AgentType]agenttypes.RuntimeFactory{
		domain.AgentTypeCodex: supervisorRuntimeFactory(func(context.Context, agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error) {
			return noopRuntime{}, func() error { return nil }, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	sentinelPath := filepath.Join(filepath.Dir(managedAgentsDir), "sentinel")
	if err := os.MkdirAll(sentinelPath, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := supervisor.DeleteAgent("../sentinel"); err == nil {
		t.Fatal("expected unsafe agentId to be rejected")
	}
	if _, err := os.Stat(sentinelPath); err != nil {
		t.Fatalf("expected sentinel to remain after rejected delete: %v", err)
	}
}

func TestNewSupervisorRollsBackStartedAgentsWhenBootstrapFails(t *testing.T) {
	managedAgentsDir := t.TempDir()
	writeRecord := func(agentID string) {
		t.Helper()
		recordDir := filepath.Join(managedAgentsDir, agentID)
		if err := os.MkdirAll(recordDir, 0o755); err != nil {
			t.Fatal(err)
		}
		record := managedAgentRecord{
			AgentID:     agentID,
			AgentType:   domain.AgentTypeCodex,
			DisplayName: agentID,
		}
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(recordDir, managedAgentMetadataName), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeRecord("agent-01")
	writeRecord("agent-02")

	registry := agentregistry.New()
	cleanupCalled := false
	_, err := NewSupervisor(context.Background(), managedAgentsDir, registry, map[domain.AgentType]agenttypes.RuntimeFactory{
		domain.AgentTypeCodex: supervisorRuntimeFactory(func(_ context.Context, spec agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error) {
			if spec.AgentID == "agent-01" {
				return noopRuntime{}, func() error {
					cleanupCalled = true
					return nil
				}, nil
			}
			return nil, nil, errors.New("boom")
		}),
	})
	if err == nil {
		t.Fatal("expected bootstrap failure")
	}
	if !cleanupCalled {
		t.Fatal("expected already-started runtime to be cleaned up")
	}
	if names := registry.Names(); len(names) != 0 {
		t.Fatalf("expected registry rollback after bootstrap failure, got %v", names)
	}
}

func TestSupervisorRestartAgentRestartsRunningAgent(t *testing.T) {
	managedAgentsDir := t.TempDir()
	startCount := 0
	cleanupCount := 0
	supervisor, err := NewSupervisor(context.Background(), managedAgentsDir, agentregistry.New(), map[domain.AgentType]agenttypes.RuntimeFactory{
		domain.AgentTypeCodex: supervisorRuntimeFactory(func(context.Context, agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error) {
			startCount++
			return noopRuntime{}, func() error {
				cleanupCount++
				return nil
			}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := supervisor.RestartAgent("agent-01"); err != nil {
		t.Fatal(err)
	}
	if startCount < 2 {
		t.Fatalf("expected restart to start runtime again, got %d", startCount)
	}
	if cleanupCount < 1 {
		t.Fatalf("expected restart to cleanup prior runtime, got %d", cleanupCount)
	}
}

func TestSupervisorRestartAgentStartsStoppedAgent(t *testing.T) {
	managedAgentsDir := t.TempDir()
	supervisor, err := NewSupervisor(context.Background(), managedAgentsDir, agentregistry.New(), map[domain.AgentType]agenttypes.RuntimeFactory{
		domain.AgentTypeCodex: supervisorRuntimeFactory(func(context.Context, agenttypes.ManagedAgentSpec) (agenttypes.Runtime, func() error, error) {
			return noopRuntime{}, func() error { return nil }, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := supervisor.StopAll(); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.RestartAgent("agent-01"); err != nil {
		t.Fatal(err)
	}

	agents := supervisor.AgentInstances()
	if len(agents) != 1 {
		t.Fatalf("expected one managed agent, got %d", len(agents))
	}
	if agents[0].Status != domain.AgentInstanceStatusRunning {
		t.Fatalf("expected running status after restart, got %+v", agents[0])
	}
}
