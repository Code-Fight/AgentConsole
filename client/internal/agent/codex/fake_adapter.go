package codex

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	agenttypes "code-agent-gateway/client/internal/agent/types"
	"code-agent-gateway/common/domain"
)

type FakeAdapter struct {
	mu          sync.RWMutex
	threads     []domain.Thread
	turns       map[string]domain.Turn
	environment []domain.EnvironmentResource
	nextThread  int
	nextTurn    int
}

func NewFakeAdapter() *FakeAdapter {
	return &FakeAdapter{
		threads:     []domain.Thread{},
		turns:       map[string]domain.Turn{},
		environment: []domain.EnvironmentResource{},
		nextThread:  1,
		nextTurn:    1,
	}
}

func (a *FakeAdapter) SeedSnapshot(threads []domain.Thread, environment []domain.EnvironmentResource) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.threads = append([]domain.Thread(nil), threads...)
	a.turns = map[string]domain.Turn{}
	a.environment = append([]domain.EnvironmentResource(nil), environment...)
	a.nextThread = len(a.threads) + 1
}

func (a *FakeAdapter) ListThreads() ([]domain.Thread, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return append([]domain.Thread(nil), a.threads...), nil
}

func (a *FakeAdapter) ListEnvironment() ([]domain.EnvironmentResource, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return append([]domain.EnvironmentResource(nil), a.environment...), nil
}

func (a *FakeAdapter) SetSkillEnabled(nameOrPath string, enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for idx := range a.environment {
		if a.environment[idx].Kind != domain.EnvironmentKindSkill || a.environment[idx].ResourceID != nameOrPath {
			continue
		}
		a.environment[idx].Status = enabledStatus(enabled)
		return nil
	}

	return fmt.Errorf("skill %q not found", nameOrPath)
}

func (a *FakeAdapter) CreateSkill(params agenttypes.CreateSkillParams) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return "", fmt.Errorf("skill name is required")
	}
	slug := normalizeSkillSlug(name)
	if slug == "" {
		return "", fmt.Errorf("skill name is invalid")
	}
	homeDir, err := resolveUserHomeDir()
	if err != nil {
		return "", err
	}
	resourceID := filepath.Join(homeDir, ".codex", "skills", slug, "SKILL.md")

	resource := domain.EnvironmentResource{
		ResourceID:  resourceID,
		Kind:        domain.EnvironmentKindSkill,
		DisplayName: name,
		Status:      domain.EnvironmentResourceStatusDisabled,
		Details: map[string]any{
			"path":    resourceID,
			"enabled": false,
		},
	}

	for idx := range a.environment {
		if a.environment[idx].Kind != domain.EnvironmentKindSkill || a.environment[idx].ResourceID != resourceID {
			continue
		}
		a.environment[idx] = mergeFakeEnvironmentResource(a.environment[idx], resource)
		return resourceID, nil
	}

	a.environment = append(a.environment, resource)
	return resourceID, nil
}

func (a *FakeAdapter) DeleteSkill(nameOrPath string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	selector := strings.TrimSpace(nameOrPath)
	if selector == "" {
		return fmt.Errorf("skill id is required")
	}

	resourceID := selector
	if !isPathLikeResourceID(selector) {
		slug := normalizeSkillSlug(selector)
		if slug == "" {
			return fmt.Errorf("skill name is invalid")
		}
		homeDir, err := resolveUserHomeDir()
		if err != nil {
			return err
		}
		resourceID = filepath.Join(homeDir, ".codex", "skills", slug, "SKILL.md")
	}

	filtered := make([]domain.EnvironmentResource, 0, len(a.environment))
	removed := false
	for _, resource := range a.environment {
		if resource.Kind == domain.EnvironmentKindSkill && resource.ResourceID == resourceID {
			removed = true
			continue
		}
		filtered = append(filtered, resource)
	}
	if !removed {
		return fmt.Errorf("skill %q not found", nameOrPath)
	}
	a.environment = filtered
	return nil
}

func (a *FakeAdapter) UpsertMCPServer(serverID string, config map[string]any) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	resource := domain.EnvironmentResource{
		ResourceID:  serverID,
		Kind:        domain.EnvironmentKindMCP,
		DisplayName: serverID,
		Status:      domain.EnvironmentResourceStatusEnabled,
		Details:     cloneFakeDetails(config),
	}
	for idx := range a.environment {
		if a.environment[idx].Kind != domain.EnvironmentKindMCP || a.environment[idx].ResourceID != serverID {
			continue
		}
		a.environment[idx] = mergeFakeEnvironmentResource(a.environment[idx], resource)
		return nil
	}
	a.environment = append(a.environment, resource)
	return nil
}

func (a *FakeAdapter) RemoveMCPServer(serverID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	filtered := make([]domain.EnvironmentResource, 0, len(a.environment))
	removed := false
	for _, resource := range a.environment {
		if resource.Kind == domain.EnvironmentKindMCP && resource.ResourceID == serverID {
			removed = true
			continue
		}
		filtered = append(filtered, resource)
	}
	if !removed {
		return fmt.Errorf("mcp server %q not found", serverID)
	}
	a.environment = filtered
	return nil
}

func (a *FakeAdapter) SetMCPServerEnabled(serverID string, enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for idx := range a.environment {
		if a.environment[idx].Kind != domain.EnvironmentKindMCP || a.environment[idx].ResourceID != serverID {
			continue
		}
		a.environment[idx].Status = enabledStatus(enabled)
		if a.environment[idx].Details == nil {
			a.environment[idx].Details = map[string]any{}
		}
		a.environment[idx].Details["enabled"] = enabled
		return nil
	}

	return fmt.Errorf("mcp server %q not found", serverID)
}

func (a *FakeAdapter) InstallPlugin(params agenttypes.InstallPluginParams) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for idx := range a.environment {
		if a.environment[idx].Kind != domain.EnvironmentKindPlugin || a.environment[idx].ResourceID != params.PluginID {
			continue
		}
		a.environment[idx].Status = domain.EnvironmentResourceStatusEnabled
		a.environment[idx].RestartRequired = true
		if a.environment[idx].Details == nil {
			a.environment[idx].Details = map[string]any{}
		}
		a.environment[idx].Details["marketplacePath"] = params.MarketplacePath
		a.environment[idx].Details["pluginName"] = params.PluginName
		return nil
	}

	a.environment = append(a.environment, domain.EnvironmentResource{
		ResourceID:      params.PluginID,
		Kind:            domain.EnvironmentKindPlugin,
		DisplayName:     params.PluginName,
		Status:          domain.EnvironmentResourceStatusEnabled,
		RestartRequired: true,
		Details: map[string]any{
			"marketplacePath": params.MarketplacePath,
			"pluginName":      params.PluginName,
		},
	})
	return nil
}

func (a *FakeAdapter) SetPluginEnabled(pluginID string, enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for idx := range a.environment {
		if a.environment[idx].Kind != domain.EnvironmentKindPlugin || a.environment[idx].ResourceID != pluginID {
			continue
		}
		a.environment[idx].Status = enabledStatus(enabled)
		a.environment[idx].RestartRequired = true
		if a.environment[idx].Details == nil {
			a.environment[idx].Details = map[string]any{}
		}
		a.environment[idx].Details["enabled"] = enabled
		return nil
	}

	return fmt.Errorf("plugin %q not found", pluginID)
}

func (a *FakeAdapter) UninstallPlugin(pluginID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for idx := range a.environment {
		if a.environment[idx].Kind != domain.EnvironmentKindPlugin || a.environment[idx].ResourceID != pluginID {
			continue
		}
		a.environment[idx].Status = domain.EnvironmentResourceStatusUnknown
		a.environment[idx].RestartRequired = true
		return nil
	}

	return fmt.Errorf("plugin %q not found", pluginID)
}

func (a *FakeAdapter) CreateThread(params agenttypes.CreateThreadParams) (domain.Thread, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	thread := domain.Thread{
		ThreadID: fmt.Sprintf("thread-%02d", a.nextThread),
		Status:   domain.ThreadStatusIdle,
		Title:    params.Title,
	}
	a.nextThread++
	a.threads = append(a.threads, thread)

	return thread, nil
}

func (a *FakeAdapter) ReadThread(threadID string) (domain.Thread, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	index := a.findThreadIndex(threadID)
	if index < 0 {
		return domain.Thread{}, fmt.Errorf("thread %q not found", threadID)
	}

	return a.threads[index], nil
}

func (a *FakeAdapter) ResumeThread(threadID string) (domain.Thread, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	index := a.findThreadIndex(threadID)
	if index < 0 {
		return domain.Thread{}, fmt.Errorf("thread %q not found", threadID)
	}

	a.threads[index].Status = domain.ThreadStatusIdle
	return a.threads[index], nil
}

func (a *FakeAdapter) ArchiveThread(threadID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	index := a.findThreadIndex(threadID)
	if index < 0 {
		return fmt.Errorf("thread %q not found", threadID)
	}

	a.threads[index].Status = domain.ThreadStatusNotLoaded
	return nil
}

func (a *FakeAdapter) StartTurn(params agenttypes.StartTurnParams) (agenttypes.StartTurnResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.findThreadIndex(params.ThreadID) < 0 {
		return agenttypes.StartTurnResult{}, fmt.Errorf("thread %q not found", params.ThreadID)
	}

	result := agenttypes.StartTurnResult{
		TurnID:   fmt.Sprintf("turn-%02d", a.nextTurn),
		ThreadID: params.ThreadID,
		Deltas: []agenttypes.TurnDelta{
			{Sequence: 1, Delta: "assistant: thinking"},
			{Sequence: 2, Delta: "assistant: done"},
		},
	}
	a.nextTurn++
	a.turns[result.TurnID] = domain.Turn{
		TurnID:   result.TurnID,
		ThreadID: params.ThreadID,
		Status:   domain.TurnStatusCompleted,
	}

	return result, nil
}

func (a *FakeAdapter) SteerTurn(params agenttypes.SteerTurnParams) (agenttypes.SteerTurnResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	turn, ok := a.turns[params.TurnID]
	if !ok || turn.ThreadID != params.ThreadID {
		return agenttypes.SteerTurnResult{}, fmt.Errorf("turn %q not found", params.TurnID)
	}

	return agenttypes.SteerTurnResult{
		TurnID:   params.TurnID,
		ThreadID: params.ThreadID,
		Deltas: []agenttypes.TurnDelta{
			{Sequence: 1, Delta: "assistant: steer accepted"},
			{Sequence: 2, Delta: "assistant: updated"},
		},
	}, nil
}

func (a *FakeAdapter) InterruptTurn(params agenttypes.InterruptTurnParams) (domain.Turn, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	turn, ok := a.turns[params.TurnID]
	if !ok || turn.ThreadID != params.ThreadID {
		return domain.Turn{}, fmt.Errorf("turn %q not found", params.TurnID)
	}

	turn.Status = domain.TurnStatusInterrupted
	a.turns[params.TurnID] = turn
	return turn, nil
}

func (a *FakeAdapter) findThreadIndex(threadID string) int {
	for idx, thread := range a.threads {
		if thread.ThreadID == threadID {
			return idx
		}
	}

	return -1
}

func cloneFakeDetails(details map[string]any) map[string]any {
	if len(details) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(details))
	for key, value := range details {
		cloned[key] = value
	}
	return cloned
}

func mergeFakeEnvironmentResource(current domain.EnvironmentResource, next domain.EnvironmentResource) domain.EnvironmentResource {
	if next.DisplayName == "" {
		next.DisplayName = current.DisplayName
	}
	if next.Status == "" {
		next.Status = current.Status
	}
	if next.Details == nil {
		next.Details = cloneFakeDetails(current.Details)
	}
	return next
}
