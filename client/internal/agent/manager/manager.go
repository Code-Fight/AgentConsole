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

func (m *Manager) ReadThread(runtimeName string, threadID string) (domain.Thread, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return domain.Thread{}, err
	}
	return runtime.ReadThread(threadID)
}

func (m *Manager) ResumeThread(runtimeName string, threadID string) (domain.Thread, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return domain.Thread{}, err
	}
	return runtime.ResumeThread(threadID)
}

func (m *Manager) ArchiveThread(runtimeName string, threadID string) error {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return err
	}
	return runtime.ArchiveThread(threadID)
}

func (m *Manager) StartTurn(runtimeName string, params types.StartTurnParams) (types.StartTurnResult, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return types.StartTurnResult{}, err
	}
	return runtime.StartTurn(params)
}

func (m *Manager) SteerTurn(runtimeName string, params types.SteerTurnParams) (types.SteerTurnResult, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return types.SteerTurnResult{}, err
	}
	return runtime.SteerTurn(params)
}

func (m *Manager) InterruptTurn(runtimeName string, params types.InterruptTurnParams) (domain.Turn, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return domain.Turn{}, err
	}
	return runtime.InterruptTurn(params)
}

func (m *Manager) RespondApproval(runtimeName string, requestID string, decision string, answers map[string]any) error {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return err
	}

	responder, ok := runtime.(types.RuntimeApprovalResponder)
	if !ok {
		return fmt.Errorf("runtime %q does not support approval responses", runtimeName)
	}

	return responder.RespondApproval(requestID, decision, answers)
}

func (m *Manager) SetSkillEnabled(runtimeName string, nameOrPath string, enabled bool) error {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return err
	}

	configurator, ok := runtime.(types.RuntimeSkillConfigurator)
	if !ok {
		return fmt.Errorf("runtime %q does not support skill configuration", runtimeName)
	}

	return configurator.SetSkillEnabled(nameOrPath, enabled)
}

func (m *Manager) CreateSkill(runtimeName string, params types.CreateSkillParams) (string, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return "", err
	}

	manager, ok := runtime.(types.RuntimeSkillManager)
	if !ok {
		return "", fmt.Errorf("runtime %q does not support skill scaffolding", runtimeName)
	}

	return manager.CreateSkill(params)
}

func (m *Manager) DeleteSkill(runtimeName string, nameOrPath string) error {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return err
	}

	manager, ok := runtime.(types.RuntimeSkillManager)
	if !ok {
		return fmt.Errorf("runtime %q does not support skill scaffolding", runtimeName)
	}

	return manager.DeleteSkill(nameOrPath)
}

func (m *Manager) UpsertMCPServer(runtimeName string, serverID string, config map[string]any) error {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return err
	}

	mcpManager, ok := runtime.(types.RuntimeMCPManager)
	if !ok {
		return fmt.Errorf("runtime %q does not support mcp configuration", runtimeName)
	}

	return mcpManager.UpsertMCPServer(serverID, config)
}

func (m *Manager) RemoveMCPServer(runtimeName string, serverID string) error {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return err
	}

	mcpManager, ok := runtime.(types.RuntimeMCPManager)
	if !ok {
		return fmt.Errorf("runtime %q does not support mcp configuration", runtimeName)
	}

	return mcpManager.RemoveMCPServer(serverID)
}

func (m *Manager) SetMCPServerEnabled(runtimeName string, serverID string, enabled bool) error {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return err
	}

	mcpManager, ok := runtime.(types.RuntimeMCPManager)
	if !ok {
		return fmt.Errorf("runtime %q does not support mcp configuration", runtimeName)
	}

	return mcpManager.SetMCPServerEnabled(serverID, enabled)
}

func (m *Manager) InstallPlugin(runtimeName string, params types.InstallPluginParams) error {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return err
	}

	pluginManager, ok := runtime.(types.RuntimePluginManager)
	if !ok {
		return fmt.Errorf("runtime %q does not support plugin install", runtimeName)
	}

	return pluginManager.InstallPlugin(params)
}

func (m *Manager) SetPluginEnabled(runtimeName string, pluginID string, enabled bool) error {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return err
	}

	pluginManager, ok := runtime.(types.RuntimePluginManager)
	if !ok {
		return fmt.Errorf("runtime %q does not support plugin enablement", runtimeName)
	}

	return pluginManager.SetPluginEnabled(pluginID, enabled)
}

func (m *Manager) ApplyConfig(runtimeName string, document domain.AgentConfigDocument) (types.ApplyConfigResult, error) {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return types.ApplyConfigResult{}, err
	}

	configManager, ok := runtime.(types.RuntimeConfigManager)
	if !ok {
		return types.ApplyConfigResult{}, fmt.Errorf("runtime %q does not support config apply", runtimeName)
	}

	return configManager.ApplyConfig(document)
}

func (m *Manager) UninstallPlugin(runtimeName string, pluginID string) error {
	runtime, err := m.resolveRuntime(runtimeName)
	if err != nil {
		return err
	}

	pluginManager, ok := runtime.(types.RuntimePluginManager)
	if !ok {
		return fmt.Errorf("runtime %q does not support plugin uninstall", runtimeName)
	}

	return pluginManager.UninstallPlugin(pluginID)
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
