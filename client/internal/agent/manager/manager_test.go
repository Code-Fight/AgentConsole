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
	runtime := &stubRuntime{
		threadRuntimeSettings: domain.ThreadRuntimeSettings{
			ThreadID: "thread-01",
			Preferences: domain.ThreadRuntimePreferences{
				Model:          "gpt-5.4",
				ApprovalPolicy: "on-request",
				SandboxMode:    "workspace-write",
			},
			Options: domain.ThreadRuntimeOptions{
				Models: []domain.ThreadRuntimeModelOption{
					{ID: "gpt-5.4", DisplayName: "GPT-5.4", IsDefault: true},
				},
				ApprovalPolicies: []string{"on-request", "never"},
				SandboxModes:     []string{"workspace-write", "danger-full-access"},
			},
		},
	}
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

	if err := mgr.UpsertMCPServer("fake", "github", map[string]any{"command": "npx"}); err != nil {
		t.Fatal(err)
	}
	if runtime.lastMCPID != "github" || runtime.lastMCPConfig["command"] != "npx" {
		t.Fatalf("unexpected mcp upsert: id=%q config=%#v", runtime.lastMCPID, runtime.lastMCPConfig)
	}

	if err := mgr.SetMCPServerEnabled("fake", "github", false); err != nil {
		t.Fatal(err)
	}
	if runtime.lastMCPEnabledID != "github" || runtime.lastMCPEnabled {
		t.Fatalf("unexpected mcp enable toggle: id=%q enabled=%v", runtime.lastMCPEnabledID, runtime.lastMCPEnabled)
	}

	if err := mgr.RemoveMCPServer("fake", "github"); err != nil {
		t.Fatal(err)
	}
	if runtime.lastRemovedMCPID != "github" {
		t.Fatalf("unexpected mcp remove target: %q", runtime.lastRemovedMCPID)
	}

	if err := mgr.InstallPlugin("fake", agenttypes.InstallPluginParams{
		PluginID:        "plugin-a",
		MarketplacePath: "/tmp/codex/marketplace",
		PluginName:      "plugin-a",
	}); err != nil {
		t.Fatal(err)
	}
	if runtime.lastInstalledPlugin.PluginID != "plugin-a" || runtime.lastInstalledPlugin.MarketplacePath != "/tmp/codex/marketplace" {
		t.Fatalf("unexpected plugin install params: %+v", runtime.lastInstalledPlugin)
	}

	if err := mgr.SetPluginEnabled("fake", "plugin-a", false); err != nil {
		t.Fatal(err)
	}
	if runtime.lastPluginEnabledID != "plugin-a" || runtime.lastPluginEnabled {
		t.Fatalf("unexpected plugin enable toggle: id=%q enabled=%v", runtime.lastPluginEnabledID, runtime.lastPluginEnabled)
	}

	if err := mgr.UninstallPlugin("fake", "plugin-a"); err != nil {
		t.Fatal(err)
	}
	if runtime.lastPluginID != "plugin-a" {
		t.Fatalf("unexpected plugin uninstall target: %q", runtime.lastPluginID)
	}

	settings, err := mgr.ReadThreadRuntimeSettings("fake", "thread-01")
	if err != nil {
		t.Fatal(err)
	}
	if settings.ThreadID != "thread-01" || settings.Preferences.Model != "gpt-5.4" {
		t.Fatalf("unexpected runtime settings read result: %+v", settings)
	}

	updated, err := mgr.UpdateThreadRuntimeSettings("fake", agenttypes.UpdateThreadRuntimeSettingsParams{
		ThreadID: "thread-01",
		Patch: domain.ThreadRuntimePreferencePatch{
			Model:       ptrToString("gpt-5.2"),
			SandboxMode: ptrToString("read-only"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Preferences.Model != "gpt-5.2" || updated.Preferences.SandboxMode != "read-only" {
		t.Fatalf("unexpected runtime settings update result: %+v", updated)
	}
	if runtime.lastThreadRuntimeReadID != "thread-01" || runtime.lastThreadRuntimeUpdate.ThreadID != "thread-01" {
		t.Fatalf("runtime settings routing did not hit runtime: read=%q update=%+v", runtime.lastThreadRuntimeReadID, runtime.lastThreadRuntimeUpdate)
	}
}

type stubRuntime struct {
	lastSkillNameOrPath     string
	lastSkillEnabled        bool
	lastPluginID            string
	lastMCPID               string
	lastMCPConfig           map[string]any
	lastMCPEnabledID        string
	lastMCPEnabled          bool
	lastRemovedMCPID        string
	lastInstalledPlugin     agenttypes.InstallPluginParams
	lastPluginEnabledID     string
	lastPluginEnabled       bool
	threadRuntimeSettings   domain.ThreadRuntimeSettings
	lastThreadRuntimeReadID string
	lastThreadRuntimeUpdate agenttypes.UpdateThreadRuntimeSettingsParams
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

func (s *stubRuntime) UpsertMCPServer(serverID string, config map[string]any) error {
	s.lastMCPID = serverID
	s.lastMCPConfig = config
	return nil
}

func (s *stubRuntime) RemoveMCPServer(serverID string) error {
	s.lastRemovedMCPID = serverID
	return nil
}

func (s *stubRuntime) SetMCPServerEnabled(serverID string, enabled bool) error {
	s.lastMCPEnabledID = serverID
	s.lastMCPEnabled = enabled
	return nil
}

func (s *stubRuntime) ReloadMCPServers() error {
	return nil
}

func (s *stubRuntime) InstallPlugin(params agenttypes.InstallPluginParams) error {
	s.lastInstalledPlugin = params
	return nil
}

func (s *stubRuntime) SetPluginEnabled(pluginID string, enabled bool) error {
	s.lastPluginEnabledID = pluginID
	s.lastPluginEnabled = enabled
	return nil
}

func (s *stubRuntime) UninstallPlugin(pluginID string) error {
	s.lastPluginID = pluginID
	return nil
}

func (s *stubRuntime) ReadThreadRuntimeSettings(threadID string) (domain.ThreadRuntimeSettings, error) {
	s.lastThreadRuntimeReadID = threadID
	settings := s.threadRuntimeSettings
	if settings.ThreadID == "" {
		settings.ThreadID = threadID
	}
	return settings, nil
}

func (s *stubRuntime) UpdateThreadRuntimeSettings(params agenttypes.UpdateThreadRuntimeSettingsParams) (domain.ThreadRuntimeSettings, error) {
	s.lastThreadRuntimeUpdate = params
	next := s.threadRuntimeSettings
	if next.ThreadID == "" {
		next.ThreadID = params.ThreadID
	}
	if params.Patch.Model != nil {
		next.Preferences.Model = *params.Patch.Model
	}
	if params.Patch.ApprovalPolicy != nil {
		next.Preferences.ApprovalPolicy = *params.Patch.ApprovalPolicy
	}
	if params.Patch.SandboxMode != nil {
		next.Preferences.SandboxMode = *params.Patch.SandboxMode
	}
	s.threadRuntimeSettings = next
	return next, nil
}

func ptrToString(value string) *string {
	return &value
}
