package manager

import (
	"fmt"

	agentregistry "code-agent-gateway/client/internal/agent/registry"
	"code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/client/internal/snapshot"
	"code-agent-gateway/common/domain"
)

type Manager struct {
	registry *agentregistry.Registry
}

func New(registry *agentregistry.Registry) *Manager {
	return &Manager{registry: registry}
}

func (m *Manager) Threads(runtimeName string) ([]domain.Thread, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return nil, err
	}
	return runtime.ListThreads()
}

func (m *Manager) Environment(runtimeName string) ([]domain.EnvironmentResource, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return nil, err
	}
	return runtime.ListEnvironment()
}

func (m *Manager) CreateThread(runtimeName string, params types.CreateThreadParams) (domain.Thread, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return domain.Thread{}, err
	}
	return runtime.CreateThread(params)
}

func (m *Manager) StartTurn(runtimeName string, params types.StartTurnParams) (types.StartTurnResult, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return types.StartTurnResult{}, err
	}
	return runtime.StartTurn(params)
}

func (m *Manager) Snapshot(runtimeName string) (snapshot.Snapshot, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	return snapshot.Build(runtime)
}

func (m *Manager) resolveRuntime(runtimeName string) (types.Runtime, error) {
	if m.registry == nil {
		return nil, fmt.Errorf("agent runtime registry is not configured")
	}
	runtime, ok := m.registry.Get(runtimeName)
	if !ok {
		return nil, fmt.Errorf("runtime %q is not registered", runtimeName)
	}
	return runtime, nil
}
